package agent

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/gorilla/websocket"
	"remote-cli/internal/protocol"
)

// Run connects to the relay and keeps a persistent agent connection,
// reconnecting automatically on any error.
func Run(cfg *Config) error {
	if cfg.DeviceToken == "" {
		return fmt.Errorf("not paired — run 'remote-cli pair --relay <url>' first")
	}
	wsURL := httpToWS(cfg.RelayURL) + "/ws/agent"
	log.Printf("agent starting as %q, relay: %s", cfg.DeviceName, cfg.RelayURL)
	for {
		if err := connect(cfg, wsURL); err != nil {
			log.Printf("connection lost: %v — reconnecting in 5s", err)
		}
		time.Sleep(5 * time.Second)
	}
}

func connect(cfg *Config, wsURL string) error {
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		return err
	}
	defer ws.Close()

	// Read the connected message.
	ws.SetReadDeadline(time.Now().Add(10 * time.Second))
	_, raw, err := ws.ReadMessage()
	ws.SetReadDeadline(time.Time{})
	if err != nil {
		return fmt.Errorf("handshake: %w", err)
	}
	var env protocol.Envelope
	if json.Unmarshal(raw, &env) != nil || env.Type != protocol.TypeConnected {
		return fmt.Errorf("expected connected, got: %s", raw)
	}

	// Shared write channel — all goroutines push here; one pump drains it.
	send := make(chan []byte, 128)

	writeErr := make(chan error, 1)
	go func() {
		for msg := range send {
			if err := ws.WriteMessage(websocket.TextMessage, msg); err != nil {
				writeErr <- err
				return
			}
		}
	}()

	// Authenticate.
	reg, _ := json.Marshal(protocol.DeviceRegisterMsg{
		Type:        protocol.TypeDeviceRegister,
		DeviceID:    cfg.DeviceID,
		DeviceToken: cfg.DeviceToken,
	})
	send <- reg

	log.Printf("connected as %q (%s)", cfg.DeviceName, cfg.DeviceID)

	mgr := newSessionMgr(send)

	defer func() {
		mgr.cleanup()
		close(send) // safe: session goroutine has exited after cleanup
	}()

	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	readErr := make(chan error, 1)
	go func() {
		for {
			_, msg, err := ws.ReadMessage()
			if err != nil {
				readErr <- err
				return
			}
			handleRelayMsg(mgr, msg)
		}
	}()

	for {
		select {
		case err := <-readErr:
			return err
		case err := <-writeErr:
			return err
		case <-ticker.C:
			hb, _ := json.Marshal(protocol.Envelope{Type: protocol.TypeDeviceHeartbeat})
			select {
			case send <- hb:
			default:
			}
		}
	}
}

func handleRelayMsg(mgr *sessionMgr, raw []byte) {
	var env protocol.Envelope
	if err := json.Unmarshal(raw, &env); err != nil {
		return
	}
	switch env.Type {
	case protocol.TypeSessionStart:
		var msg protocol.RelaySessionStartMsg
		if json.Unmarshal(raw, &msg) == nil && msg.SessionID != "" {
			mgr.start(msg.SessionID)
		}

	case protocol.TypeSessionEnd:
		var msg protocol.RelaySessionEndMsg
		if json.Unmarshal(raw, &msg) == nil && msg.SessionID != "" {
			mgr.end(msg.SessionID, msg.Reason)
		}

	case protocol.TypeUserMessage:
		var msg protocol.UserMessageMsg
		if json.Unmarshal(raw, &msg) == nil && msg.SessionID != "" {
			log.Printf("message.user session=%s content=%q", msg.SessionID, msg.Content)
			mgr.deliver(msg.SessionID, msg.Content)
		}

	case protocol.TypeToolUseApprove, protocol.TypeToolUseDeny:
		// Phase 3: forward to active SDK session
		log.Printf("tool_use decision received (phase 3): %s", env.Type)

	default:
		log.Printf("relay → agent: %s", env.Type)
	}
}
