package relay

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"golang.org/x/crypto/bcrypt"
	"remote-cli/internal/protocol"
)

type Server struct {
	db        *DB
	auth      *Auth
	hub       *Hub
	pairing   *PairingStore
	relayURL  string
	staticDir string
	upgrader  websocket.Upgrader
}

func NewServer(db *DB, auth *Auth, hub *Hub, pairing *PairingStore, relayURL, staticDir string) *Server {
	return &Server{
		db:        db,
		auth:      auth,
		hub:       hub,
		pairing:   pairing,
		relayURL:  relayURL,
		staticDir: staticDir,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		},
	}
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/ws/agent", s.handleAgentWS)
	mux.HandleFunc("/ws/phone", s.handlePhoneWS)
	mux.HandleFunc("/api/auth/login", s.handleLogin)
	mux.HandleFunc("/api/pair/request", s.handlePairRequest)
	mux.HandleFunc("/api/pair/redeem", s.handlePairRedeem)
	mux.HandleFunc("/api/devices/", s.handleDeviceActions)
	if s.staticDir != "" {
		mux.Handle("/", s.spaHandler())
	}
	return mux
}

// spaHandler serves the PWA; unknown paths fall back to index.html for client-side routing.
func (s *Server) spaHandler() http.Handler {
	fs := http.Dir(s.staticDir)
	fileServer := http.FileServer(fs)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f, err := fs.Open(r.URL.Path)
		if err != nil {
			// File not found — serve index.html for SPA routing.
			http.ServeFile(w, r, s.staticDir+"/index.html")
			return
		}
		f.Close()
		fileServer.ServeHTTP(w, r)
	})
}

// ── Agent WebSocket ───────────────────────────────────────────────────────────

func (s *Server) handleAgentWS(w http.ResponseWriter, r *http.Request) {
	ws, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	c := &conn{
		id:   newID(),
		kind: kindPairing,
		ws:   ws,
		send: make(chan []byte, 64),
	}
	s.hub.register(c)
	go c.writePump()

	defer func() {
		s.hub.unregister(c)
		close(c.send)
		if c.kind == kindAgent {
			// Terminate any sessions that were running on this device.
			for _, sid := range s.hub.sessionsForDevice(c.deviceID) {
				s.hub.sendToSessionOwner(sid, protocol.SessionEndedMsg{
					Type:      protocol.TypeSessionEnded,
					SessionID: sid,
					Reason:    "agent disconnected",
				})
				s.hub.unregisterSession(sid)
			}
			s.hub.setDeviceIdle(c.deviceID)
			if err := s.db.UpdateDeviceLastSeen(c.deviceID); err != nil {
				log.Printf("update last_seen %s: %v", c.deviceID, err)
			}
			s.broadcastDeviceUpdate(c.deviceID, "offline")
		}
	}()

	c.push(protocol.ConnectedMsg{Type: protocol.TypeConnected, ConnectionID: c.id})

	for {
		_, raw, err := ws.ReadMessage()
		if err != nil {
			return
		}
		var env protocol.Envelope
		if err := json.Unmarshal(raw, &env); err != nil {
			continue
		}
		switch env.Type {
		case protocol.TypeDeviceRegister:
			s.handleDeviceRegister(c, raw)
		case protocol.TypeDeviceHeartbeat:
			// keepalive
		default:
			if c.kind == kindAgent {
				s.routeAgentMessage(env.Type, raw)
			} else {
				c.push(protocol.ErrorMsg{
					Type:    protocol.TypeError,
					Code:    "unauthorized",
					Message: "send device.register first",
				})
			}
		}
	}
}

func (s *Server) handleDeviceRegister(c *conn, raw []byte) {
	var msg protocol.DeviceRegisterMsg
	if err := json.Unmarshal(raw, &msg); err != nil {
		return
	}
	device, err := s.db.GetDeviceByID(msg.DeviceID)
	if err != nil || device == nil {
		c.push(protocol.ErrorMsg{Type: protocol.TypeError, Code: "auth_failed", Message: "device not found"})
		return
	}
	if bcrypt.CompareHashAndPassword([]byte(device.TokenHash), []byte(msg.DeviceToken)) != nil {
		c.push(protocol.ErrorMsg{Type: protocol.TypeError, Code: "auth_failed", Message: "invalid token"})
		return
	}
	if !s.hub.upgradeToAgent(c.id, device.ID) {
		return
	}
	log.Printf("agent registered: %s (%s)", device.Name, device.ID)
	s.broadcastDeviceUpdate(device.ID, "online")
}

// routeAgentMessage routes authenticated agent messages to the owning phone only.
func (s *Server) routeAgentMessage(msgType protocol.MessageType, raw []byte) {
	// Helper to extract session_id from any agent message.
	var sid struct {
		SessionID string `json:"session_id"`
	}
	json.Unmarshal(raw, &sid) //nolint — best-effort

	switch msgType {
	case protocol.TypeSessionStarted:
		var msg protocol.SessionStartedMsg
		if json.Unmarshal(raw, &msg) != nil {
			return
		}
		log.Printf("session started: %s", msg.SessionID)
		s.hub.sendToSessionOwner(msg.SessionID, msg)

	case protocol.TypeSessionEnded:
		var msg protocol.SessionEndedMsg
		if json.Unmarshal(raw, &msg) != nil {
			return
		}
		log.Printf("session ended: %s (%s)", msg.SessionID, msg.Reason)
		deviceID := s.hub.deviceForSession(msg.SessionID)
		s.hub.sendToSessionOwner(msg.SessionID, msg) // send before unregister clears owner
		s.hub.unregisterSession(msg.SessionID)
		s.hub.setDeviceIdle(deviceID)
		// Device is idle again if the agent is still connected.
		if deviceID != "" && s.hub.agentStatus(deviceID) != "offline" {
			s.broadcastDeviceUpdate(deviceID, "online")
		}

	case protocol.TypeAssistantChunk, protocol.TypeToolUseResult:
		s.hub.sendRawToSessionOwner(sid.SessionID, raw)

	case protocol.TypeToolUseRequest:
		var msg protocol.ToolUseRequestMsg
		if json.Unmarshal(raw, &msg) != nil {
			return
		}
		s.hub.registerToolUse(msg.ToolUseID, msg.SessionID)
		s.hub.sendToSessionOwner(msg.SessionID, msg)

	case protocol.TypeError:
		s.hub.sendRawToSessionOwner(sid.SessionID, raw)
	}
}

// ── Phone WebSocket ───────────────────────────────────────────────────────────

func (s *Server) handlePhoneWS(w http.ResponseWriter, r *http.Request) {
	ws, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	c := &conn{
		id:   newID(),
		kind: kindPairing, // upgraded after client.auth
		ws:   ws,
		send: make(chan []byte, 128),
	}
	s.hub.register(c)
	go c.writePump()

	defer func() {
		if c.kind == kindPhone {
			for _, sid := range s.hub.sessionsForOwner(c.id) {
				deviceID := s.hub.deviceForSession(sid)
				if deviceID != "" {
					s.hub.sendToDevice(deviceID, protocol.RelaySessionEndMsg{
						Type:      protocol.TypeSessionEnd,
						SessionID: sid,
						Reason:    "phone disconnected",
					})
				}
			}
		}
		s.hub.unregister(c)
		close(c.send)
	}()

	// First message must be client.auth within 30 seconds.
	ws.SetReadDeadline(time.Now().Add(30 * time.Second))
	_, raw, err := ws.ReadMessage()
	ws.SetReadDeadline(time.Time{})
	if err != nil {
		return
	}
	var authMsg struct {
		Type  protocol.MessageType `json:"type"`
		Token string               `json:"token"`
	}
	if json.Unmarshal(raw, &authMsg) != nil || authMsg.Type != protocol.TypeClientAuth {
		c.push(protocol.ErrorMsg{Type: protocol.TypeError, Code: "auth_required", Message: "send client.auth first"})
		return
	}
	if err := s.auth.ValidateToken(authMsg.Token); err != nil {
		c.push(protocol.ErrorMsg{Type: protocol.TypeError, Code: "auth_failed", Message: "invalid token"})
		return
	}
	s.hub.upgradeToPhone(c.id)
	log.Printf("phone connected: %s", c.id)

	// Push current device list immediately after auth.
	s.sendDeviceList(c)

	for {
		_, raw, err := ws.ReadMessage()
		if err != nil {
			return
		}
		var env protocol.Envelope
		if json.Unmarshal(raw, &env) != nil {
			continue
		}
		s.routePhoneMessage(c, env.Type, raw)
	}
}

// routePhoneMessage processes messages from authenticated phone connections.
func (s *Server) routePhoneMessage(c *conn, msgType protocol.MessageType, raw []byte) {
	switch msgType {
	case protocol.TypeDeviceList:
		s.sendDeviceList(c)

	case protocol.TypeSessionStart:
		var msg struct {
			Type     protocol.MessageType `json:"type"`
			DeviceID string               `json:"device_id"`
		}
		if json.Unmarshal(raw, &msg) != nil || msg.DeviceID == "" {
			return
		}
		switch s.hub.agentStatus(msg.DeviceID) {
		case "offline":
			c.push(protocol.ErrorMsg{
				Type:    protocol.TypeError,
				Code:    "device_offline",
				Message: "Device is offline — run 'remote-cli run' on the machine first.",
			})
			return
		case "busy":
			c.push(protocol.ErrorMsg{
				Type:    protocol.TypeError,
				Code:    "device_busy",
				Message: "Device already has an active session.",
			})
			return
		}
		sessionID := newID()
		s.hub.registerSession(sessionID, msg.DeviceID)
		if !s.hub.sendToDevice(msg.DeviceID, protocol.RelaySessionStartMsg{
			Type:      protocol.TypeSessionStart,
			SessionID: sessionID,
		}) {
			s.hub.unregisterSession(sessionID)
			c.push(protocol.ErrorMsg{
				Type:    protocol.TypeError,
				Code:    "device_unreachable",
				Message: "Could not reach device — it may have just disconnected.",
			})
			return
		}
		s.hub.registerSessionOwner(sessionID, c.id)
		s.hub.setDeviceBusy(msg.DeviceID)
		s.broadcastDeviceUpdate(msg.DeviceID, "busy")
		// Let the phone know the session ID it was assigned.
		c.push(protocol.SessionStateMsg{
			Type:      protocol.TypeSessionState,
			DeviceID:  msg.DeviceID,
			SessionID: sessionID,
			Status:    "starting",
		})

	case protocol.TypeSessionEnd:
		var msg struct {
			Type      protocol.MessageType `json:"type"`
			DeviceID  string               `json:"device_id"`
			SessionID string               `json:"session_id"`
		}
		if json.Unmarshal(raw, &msg) != nil || msg.SessionID == "" {
			return
		}
		deviceID := msg.DeviceID
		if deviceID == "" {
			deviceID = s.hub.deviceForSession(msg.SessionID)
		}
		s.hub.sendToDevice(deviceID, protocol.RelaySessionEndMsg{
			Type:      protocol.TypeSessionEnd,
			SessionID: msg.SessionID,
			Reason:    "user ended session",
		})

	case protocol.TypeUserMessage:
		var msg protocol.UserMessageMsg
		if json.Unmarshal(raw, &msg) != nil || msg.SessionID == "" {
			return
		}
		deviceID := s.hub.deviceForSession(msg.SessionID)
		if deviceID == "" {
			c.push(protocol.ErrorMsg{Type: protocol.TypeError, Code: "not_found", Message: "session not found"})
			return
		}
		if !s.hub.sendToDevice(deviceID, msg) {
			c.push(protocol.ErrorMsg{Type: protocol.TypeError, Code: "device_unreachable", Message: "Could not deliver message to device."})
		}

	case protocol.TypeToolUseApprove:
		var msg protocol.ToolUseApproveMsg
		if json.Unmarshal(raw, &msg) != nil || msg.ToolUseID == "" {
			return
		}
		sessionID := s.hub.sessionForToolUse(msg.ToolUseID)
		if sessionID == "" {
			return
		}
		msg.SessionID = sessionID
		s.hub.unregisterToolUse(msg.ToolUseID)
		deviceID := s.hub.deviceForSession(sessionID)
		s.hub.sendToDevice(deviceID, msg)

	case protocol.TypeToolUseDeny:
		var msg protocol.ToolUseDenyMsg
		if json.Unmarshal(raw, &msg) != nil || msg.ToolUseID == "" {
			return
		}
		sessionID := s.hub.sessionForToolUse(msg.ToolUseID)
		if sessionID == "" {
			return
		}
		msg.SessionID = sessionID
		s.hub.unregisterToolUse(msg.ToolUseID)
		deviceID := s.hub.deviceForSession(sessionID)
		s.hub.sendToDevice(deviceID, msg)
	}
}

func (s *Server) sendDeviceList(c *conn) {
	devices, err := s.db.ListDevices()
	if err != nil {
		return
	}
	infos := make([]protocol.DeviceInfo, 0, len(devices))
	for _, d := range devices {
		infos = append(infos, protocol.DeviceInfo{
			ID:       d.ID,
			Name:     d.Name,
			Status:   s.hub.agentStatus(d.ID),
			LastSeen: d.LastSeen,
		})
	}
	c.push(protocol.DeviceListMsg{Type: protocol.TypeDeviceList, Devices: infos})
}

// ── HTTP endpoints ────────────────────────────────────────────────────────────

func (s *Server) broadcastDeviceUpdate(deviceID, status string) {
	msg := protocol.DeviceUpdateMsg{
		Type:     protocol.TypeDeviceUpdate,
		DeviceID: deviceID,
		Status:   status,
	}
	if status == "offline" {
		now := time.Now().Unix()
		msg.LastSeen = &now
	}
	s.hub.broadcastToPhones(msg)
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	hash, err := s.db.GetAdminPasswordHash()
	if err != nil || hash == "" || !s.auth.CheckPassword(req.Password, hash) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	token, err := s.auth.IssueToken()
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	jsonResp(w, map[string]string{"token": token})
}

func (s *Server) handlePairRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		ConnectionID string `json:"connection_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.ConnectionID == "" {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	s.hub.mu.RLock()
	c, ok := s.hub.conns[req.ConnectionID]
	s.hub.mu.RUnlock()
	if !ok || c.kind != kindPairing {
		http.Error(w, "connection not found or already registered", http.StatusBadRequest)
		return
	}

	code, err := s.pairing.Create(req.ConnectionID)
	if err != nil {
		http.Error(w, "failed to generate code", http.StatusInternalServerError)
		return
	}

	pairURL := fmt.Sprintf("%s/?pair=%s", strings.TrimRight(s.relayURL, "/"), code)
	c.push(protocol.PairCodeMsg{Type: protocol.TypePairCode, Code: code, URL: pairURL})
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handlePairRedeem(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if err := s.requireAdmin(r); err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	var req struct {
		Code       string `json:"code"`
		DeviceName string `json:"device_name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Code == "" {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	connID, ok := s.pairing.Redeem(req.Code)
	if !ok {
		http.Error(w, "code expired or not found", http.StatusGone)
		return
	}

	deviceID := newID()
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	token := hex.EncodeToString(tokenBytes)

	hash, err := bcrypt.GenerateFromPassword([]byte(token), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	name := req.DeviceName
	if name == "" {
		name = "unnamed-device"
	}
	if err := s.db.CreateDevice(deviceID, name, string(hash)); err != nil {
		http.Error(w, "failed to register device", http.StatusInternalServerError)
		return
	}

	s.hub.sendToConn(connID, protocol.PairCompleteMsg{
		Type:        protocol.TypePairComplete,
		DeviceID:    deviceID,
		DeviceToken: token,
		DeviceName:  name,
	})

	log.Printf("paired device %s (%s)", name, deviceID)
	jsonResp(w, map[string]string{"device_id": deviceID, "device_name": name})
}

// handleDeviceActions handles PATCH /api/devices/{id} (rename) and DELETE /api/devices/{id}.
func (s *Server) handleDeviceActions(w http.ResponseWriter, r *http.Request) {
	if err := s.requireAdmin(r); err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	// Extract device ID from path: /api/devices/{id}
	id := strings.TrimPrefix(r.URL.Path, "/api/devices/")
	if id == "" {
		http.Error(w, "missing device id", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodPatch:
		var req struct {
			Name string `json:"name"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || strings.TrimSpace(req.Name) == "" {
			http.Error(w, "name required", http.StatusBadRequest)
			return
		}
		name := strings.TrimSpace(req.Name)
		if err := s.db.RenameDevice(id, name); err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		s.hub.broadcastToPhones(protocol.DeviceUpdateMsg{
			Type:     protocol.TypeDeviceUpdate,
			DeviceID: id,
			Status:   s.hub.agentStatus(id),
			Name:     name,
		})
		w.WriteHeader(http.StatusNoContent)

	case http.MethodDelete:
		if err := s.db.DeleteDevice(id); err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		s.hub.disconnectDevice(id)
		s.hub.broadcastToPhones(protocol.DeviceUpdateMsg{
			Type:     protocol.TypeDeviceUpdate,
			DeviceID: id,
			Status:   "deleted",
		})
		w.WriteHeader(http.StatusNoContent)

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) requireAdmin(r *http.Request) error {
	auth := r.Header.Get("Authorization")
	if !strings.HasPrefix(auth, "Bearer ") {
		return fmt.Errorf("missing bearer token")
	}
	return s.auth.ValidateToken(auth[7:])
}

func jsonResp(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

func newID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}
