package relay

import (
	"encoding/json"
	"log"
	"sync"

	"github.com/gorilla/websocket"
)

type connKind int

const (
	kindPairing connKind = iota
	kindAgent
	kindPhone
)

type conn struct {
	id       string
	kind     connKind
	deviceID string
	ws       *websocket.Conn
	send     chan []byte
}

func (c *conn) push(msg any) bool {
	b, err := json.Marshal(msg)
	if err != nil {
		return false
	}
	select {
	case c.send <- b:
		return true
	default:
		return false
	}
}

func (c *conn) pushRaw(b []byte) bool {
	select {
	case c.send <- b:
		return true
	default:
		return false
	}
}

func (c *conn) writePump() {
	defer c.ws.Close()
	for msg := range c.send {
		if err := c.ws.WriteMessage(websocket.TextMessage, msg); err != nil {
			log.Printf("ws write %s: %v", c.id, err)
			return
		}
	}
}

// Hub manages all active WebSocket connections and session routing state.
type Hub struct {
	mu           sync.RWMutex
	conns        map[string]*conn   // connID → conn
	byDevice     map[string]*conn   // deviceID → agent conn
	busyDevices  map[string]struct{} // deviceID set — device has an active session
	sessions     map[string]string  // sessionID → deviceID
	sessionOwner map[string]string  // sessionID → phone connID
	toolSessions map[string]string  // toolUseID → sessionID
}

func NewHub() *Hub {
	return &Hub{
		conns:        make(map[string]*conn),
		byDevice:     make(map[string]*conn),
		busyDevices:  make(map[string]struct{}),
		sessions:     make(map[string]string),
		sessionOwner: make(map[string]string),
		toolSessions: make(map[string]string),
	}
}

func (h *Hub) register(c *conn) {
	h.mu.Lock()
	h.conns[c.id] = c
	h.mu.Unlock()
}

func (h *Hub) unregister(c *conn) {
	h.mu.Lock()
	delete(h.conns, c.id)
	if c.kind == kindAgent && c.deviceID != "" {
		if h.byDevice[c.deviceID] == c {
			delete(h.byDevice, c.deviceID)
		}
	}
	h.mu.Unlock()
}

// upgradeToAgent marks a pairing connection as an authenticated agent.
func (h *Hub) upgradeToAgent(connID, deviceID string) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	c, ok := h.conns[connID]
	if !ok || c.kind != kindPairing {
		return false
	}
	c.kind = kindAgent
	c.deviceID = deviceID
	h.byDevice[deviceID] = c
	return true
}

// upgradeToPhone marks a connection as an authenticated phone.
func (h *Hub) upgradeToPhone(connID string) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	c, ok := h.conns[connID]
	if !ok {
		return false
	}
	c.kind = kindPhone
	return true
}

// sendToConn pushes a message to a specific connection by ID.
func (h *Hub) sendToConn(connID string, msg any) bool {
	h.mu.RLock()
	c, ok := h.conns[connID]
	h.mu.RUnlock()
	if !ok {
		return false
	}
	return c.push(msg)
}

// sendToDevice pushes a message to the agent conn for a given device.
func (h *Hub) sendToDevice(deviceID string, msg any) bool {
	h.mu.RLock()
	c, ok := h.byDevice[deviceID]
	h.mu.RUnlock()
	if !ok {
		return false
	}
	return c.push(msg)
}

// broadcastToPhones pushes a message to all authenticated phone connections.
func (h *Hub) broadcastToPhones(msg any) {
	h.mu.RLock()
	phones := h.phones()
	h.mu.RUnlock()
	for _, c := range phones {
		c.push(msg)
	}
}

// phones returns all phone conns. Caller must hold at least h.mu.RLock.
func (h *Hub) phones() []*conn {
	var out []*conn
	for _, c := range h.conns {
		if c.kind == kindPhone {
			out = append(out, c)
		}
	}
	return out
}

// disconnectDevice closes and removes the agent connection for a device.
func (h *Hub) disconnectDevice(deviceID string) {
	h.mu.Lock()
	c, ok := h.byDevice[deviceID]
	if ok {
		delete(h.byDevice, deviceID)
	}
	h.mu.Unlock()
	if ok {
		c.ws.Close()
	}
}

// agentStatus returns "offline", "online", or "busy" for deviceID.
func (h *Hub) agentStatus(deviceID string) string {
	h.mu.RLock()
	_, online := h.byDevice[deviceID]
	_, busy := h.busyDevices[deviceID]
	h.mu.RUnlock()
	if !online {
		return "offline"
	}
	if busy {
		return "busy"
	}
	return "online"
}

func (h *Hub) setDeviceBusy(deviceID string) {
	h.mu.Lock()
	h.busyDevices[deviceID] = struct{}{}
	h.mu.Unlock()
}

func (h *Hub) setDeviceIdle(deviceID string) {
	h.mu.Lock()
	delete(h.busyDevices, deviceID)
	h.mu.Unlock()
}

// sessionsForOwner returns all session IDs owned by a phone connection.
func (h *Hub) sessionsForOwner(connID string) []string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	var ids []string
	for sid, ownerID := range h.sessionOwner {
		if ownerID == connID {
			ids = append(ids, sid)
		}
	}
	return ids
}

// ── Session tracking ──────────────────────────────────────────────────────────

func (h *Hub) registerSession(sessionID, deviceID string) {
	h.mu.Lock()
	h.sessions[sessionID] = deviceID
	h.mu.Unlock()
}

func (h *Hub) registerSessionOwner(sessionID, connID string) {
	h.mu.Lock()
	h.sessionOwner[sessionID] = connID
	h.mu.Unlock()
}

func (h *Hub) unregisterSession(sessionID string) {
	h.mu.Lock()
	delete(h.sessions, sessionID)
	delete(h.sessionOwner, sessionID)
	h.mu.Unlock()
}

func (h *Hub) deviceForSession(sessionID string) string {
	h.mu.RLock()
	id := h.sessions[sessionID]
	h.mu.RUnlock()
	return id
}

// sessionsForDevice returns all session IDs currently mapped to a device.
func (h *Hub) sessionsForDevice(deviceID string) []string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	var ids []string
	for sid, did := range h.sessions {
		if did == deviceID {
			ids = append(ids, sid)
		}
	}
	return ids
}

// sendToSessionOwner pushes a message only to the phone that owns the session.
func (h *Hub) sendToSessionOwner(sessionID string, msg any) bool {
	h.mu.RLock()
	connID := h.sessionOwner[sessionID]
	c := h.conns[connID]
	h.mu.RUnlock()
	if c == nil {
		return false
	}
	return c.push(msg)
}

// sendRawToSessionOwner pushes pre-serialised bytes only to the session owner.
func (h *Hub) sendRawToSessionOwner(sessionID string, b []byte) bool {
	h.mu.RLock()
	connID := h.sessionOwner[sessionID]
	c := h.conns[connID]
	h.mu.RUnlock()
	if c == nil {
		return false
	}
	return c.pushRaw(b)
}

func (h *Hub) registerToolUse(toolUseID, sessionID string) {
	h.mu.Lock()
	h.toolSessions[toolUseID] = sessionID
	h.mu.Unlock()
}

func (h *Hub) sessionForToolUse(toolUseID string) string {
	h.mu.RLock()
	id := h.toolSessions[toolUseID]
	h.mu.RUnlock()
	return id
}

func (h *Hub) unregisterToolUse(toolUseID string) {
	h.mu.Lock()
	delete(h.toolSessions, toolUseID)
	h.mu.Unlock()
}
