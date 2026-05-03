package agent

import (
	"encoding/json"
	"log"
	"strings"
	"sync"
	"time"

	"remote-cli/internal/protocol"
)

const idleTimeout = time.Hour

// echoSession is the Phase-2 stub: it echoes user messages back word-by-word
// as streamed assistant_chunk messages, simulating real Claude output.
type echoSession struct {
	id      string
	send    chan<- []byte
	msgs    chan string
	stop    chan struct{} // closed to request graceful shutdown
	stopped chan struct{} // closed after goroutine exits
	once    sync.Once
}

func newEchoSession(id string, send chan<- []byte) *echoSession {
	s := &echoSession{
		id:      id,
		send:    send,
		msgs:    make(chan string, 8),
		stop:    make(chan struct{}),
		stopped: make(chan struct{}),
	}
	go s.run()
	return s
}

// end signals the session to stop and waits for the goroutine to exit.
func (s *echoSession) end(reason string) {
	s.once.Do(func() { close(s.stop) })
	<-s.stopped
}

// deliver queues a user message for echoing.
func (s *echoSession) deliver(content string) {
	select {
	case s.msgs <- content:
	case <-s.stop:
	default:
		log.Printf("session %s: message queue full, dropping", s.id)
	}
}

func (s *echoSession) run() {
	defer close(s.stopped)

	idle := time.NewTimer(idleTimeout)
	defer idle.Stop()

	s.push(protocol.SessionStartedMsg{
		Type:      protocol.TypeSessionStarted,
		SessionID: s.id,
	})

	reason := "session ended"
	for {
		select {
		case <-s.stop:
			s.push(protocol.SessionEndedMsg{
				Type:      protocol.TypeSessionEnded,
				SessionID: s.id,
				Reason:    reason,
			})
			return

		case <-idle.C:
			s.push(protocol.SessionEndedMsg{
				Type:      protocol.TypeSessionEnded,
				SessionID: s.id,
				Reason:    "idle timeout",
			})
			return

		case content := <-s.msgs:
			if !idle.Stop() {
				select {
				case <-idle.C:
				default:
				}
			}
			idle.Reset(idleTimeout)
			s.echoBack(content)
		}
	}
}

func (s *echoSession) echoBack(content string) {
	words := strings.Fields(content)
	if len(words) == 0 {
		return
	}
	for i, word := range words {
		text := word
		if i < len(words)-1 {
			text += " "
		}
		s.push(protocol.AssistantChunkMsg{
			Type:         protocol.TypeAssistantChunk,
			SessionID:    s.id,
			ContentBlock: protocol.ContentBlock{Type: "text", Text: text},
			Index:        i,
		})
		time.Sleep(40 * time.Millisecond) // simulate streaming latency
	}
	// trailing newline signals end of turn
	s.push(protocol.AssistantChunkMsg{
		Type:         protocol.TypeAssistantChunk,
		SessionID:    s.id,
		ContentBlock: protocol.ContentBlock{Type: "text", Text: "\n"},
		Index:        len(words),
	})
}

// push serializes msg and writes to the shared send channel.
// Returns false only if the session is stopping AND the channel is full.
func (s *echoSession) push(msg any) {
	b, err := json.Marshal(msg)
	if err != nil {
		return
	}
	select {
	case s.send <- b:
	case <-s.stopped:
		// goroutine has already exited (shouldn't happen since push is called from run())
	}
}

// ── Session interface ─────────────────────────────────────────────────────────

type session interface {
	deliver(content string)
	end(reason string)
	sessionID() string
}

func (s *echoSession) sessionID() string { return s.id }

// ── Session manager ───────────────────────────────────────────────────────────

// sessionMgr tracks the one active session on this agent.
type sessionMgr struct {
	mu      sync.Mutex
	current session
	send    chan<- []byte
}

func newSessionMgr(send chan<- []byte) *sessionMgr {
	return &sessionMgr{send: send}
}

func (m *sessionMgr) start(sessionID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.current != nil {
		m.current.end("replaced by new session")
	}
	log.Printf("session start: %s", sessionID)
	m.current = newClaudeSession(sessionID, m.send)
}

func (m *sessionMgr) deliver(sessionID, content string) {
	m.mu.Lock()
	s := m.current
	m.mu.Unlock()
	if s == nil || s.sessionID() != sessionID {
		return
	}
	s.deliver(content)
}

func (m *sessionMgr) end(sessionID, reason string) {
	m.mu.Lock()
	s := m.current
	if s != nil && s.sessionID() == sessionID {
		m.current = nil
	} else {
		s = nil
	}
	m.mu.Unlock()
	if s != nil {
		log.Printf("session end: %s (%s)", sessionID, reason)
		s.end(reason)
	}
}

// cleanup ends any active session and waits for it to finish.
// Must be called before closing the send channel.
func (m *sessionMgr) cleanup() {
	m.mu.Lock()
	s := m.current
	m.current = nil
	m.mu.Unlock()
	if s != nil {
		s.end("agent disconnected")
	}
}
