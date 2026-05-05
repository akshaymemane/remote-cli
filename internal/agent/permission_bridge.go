package agent

import (
	"bufio"
	"encoding/json"
	"log"
	"net"
	"os"
	"sync"

	"remote-cli/internal/protocol"
)

// permissionBridge listens on a Unix socket for permission requests from the
// MCP server subprocess. It forwards them to the relay as tool_use.request
// messages (with awaiting_approval=true) and blocks until the phone responds.
type permissionBridge struct {
	socketPath string
	listener   net.Listener
	mu         sync.Mutex
	pending    map[string]chan permDecision
	send       chan<- []byte
	sessionID  string
}

type permDecision struct {
	Allow   bool
	Message string
}

// bridgeRequest is the message the MCP server sends to the bridge.
type bridgeRequest struct {
	ToolUseID string         `json:"tool_use_id"`
	ToolName  string         `json:"tool_name"`
	ToolInput map[string]any `json:"tool_input"`
}

// bridgeResponse is the decision the bridge sends back to the MCP server.
type bridgeResponse struct {
	Allow   bool   `json:"allow"`
	Message string `json:"message,omitempty"`
}

func newPermissionBridge(socketPath, sessionID string, send chan<- []byte) (*permissionBridge, error) {
	_ = os.Remove(socketPath)
	l, err := net.Listen("unix", socketPath)
	if err != nil {
		return nil, err
	}
	b := &permissionBridge{
		socketPath: socketPath,
		listener:   l,
		pending:    make(map[string]chan permDecision),
		send:       send,
		sessionID:  sessionID,
	}
	go b.serve()
	return b, nil
}

func (b *permissionBridge) serve() {
	defer b.listener.Close()
	for {
		conn, err := b.listener.Accept()
		if err != nil {
			return // listener closed — session ending
		}
		go b.handleConn(conn)
	}
}

func (b *permissionBridge) handleConn(conn net.Conn) {
	defer conn.Close()
	scanner := bufio.NewScanner(conn)
	scanner.Buffer(make([]byte, 512*1024), 512*1024)
	enc := json.NewEncoder(conn)

	for scanner.Scan() {
		var req bridgeRequest
		if err := json.Unmarshal(scanner.Bytes(), &req); err != nil {
			log.Printf("permission bridge: bad request: %v", err)
			continue
		}

		ch := make(chan permDecision, 1)
		b.mu.Lock()
		b.pending[req.ToolUseID] = ch
		b.mu.Unlock()

		// Ask the phone for approval.
		b.push(protocol.ToolUseRequestMsg{
			Type:             protocol.TypeToolUseRequest,
			SessionID:        b.sessionID,
			ToolUseID:        req.ToolUseID,
			ToolName:         req.ToolName,
			ToolInput:        req.ToolInput,
			AwaitingApproval: true,
		})

		decision := <-ch

		b.mu.Lock()
		delete(b.pending, req.ToolUseID)
		b.mu.Unlock()

		if err := enc.Encode(bridgeResponse{Allow: decision.Allow, Message: decision.Message}); err != nil {
			log.Printf("permission bridge: write response: %v", err)
			return
		}
	}
}

// resolve delivers a phone decision for a pending permission request.
func (b *permissionBridge) resolve(toolUseID string, allow bool, message string) {
	b.mu.Lock()
	ch := b.pending[toolUseID]
	b.mu.Unlock()
	if ch != nil {
		select {
		case ch <- permDecision{Allow: allow, Message: message}:
		default:
		}
	}
}

// denyAll rejects every pending request — called when session ends while
// approvals are still in flight.
func (b *permissionBridge) denyAll() {
	b.mu.Lock()
	ids := make([]string, 0, len(b.pending))
	for id := range b.pending {
		ids = append(ids, id)
	}
	b.mu.Unlock()
	for _, id := range ids {
		b.resolve(id, false, "session ended")
	}
}

func (b *permissionBridge) close() {
	b.denyAll()
	b.listener.Close()
	os.Remove(b.socketPath)
}

func (b *permissionBridge) push(msg any) {
	data, _ := json.Marshal(msg)
	select {
	case b.send <- data:
	default:
		log.Printf("permission bridge: send channel full, dropping approval request")
	}
}
