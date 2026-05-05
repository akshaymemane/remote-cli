package agent

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"remote-cli/internal/protocol"
)

// claudeSession drives a real `claude` subprocess for a session.
// Tools execute automatically inside the subprocess (--print mode).
// Text and tool events are forwarded to the phone via the shared send channel.
type claudeSession struct {
	id      string
	send    chan<- []byte
	msgCh   chan string   // incoming user messages
	stop    chan struct{} // close to request shutdown
	stopped chan struct{} // closed when goroutine exits
	once    sync.Once
	bridge  *permissionBridge // nil if permission bridge failed to start
}

func newClaudeSession(id string, send chan<- []byte) *claudeSession {
	s := &claudeSession{
		id:      id,
		send:    send,
		msgCh:   make(chan string, 8),
		stop:    make(chan struct{}),
		stopped: make(chan struct{}),
	}
	go s.run()
	return s
}

func (s *claudeSession) deliver(content string) {
	select {
	case s.msgCh <- content:
	case <-s.stop:
	default:
		log.Printf("session %s: queue full, dropping message", s.id)
	}
}

func (s *claudeSession) sessionID() string { return s.id }

func (s *claudeSession) end(_ string) {
	s.once.Do(func() { close(s.stop) })
	<-s.stopped
}

func (s *claudeSession) approveTool(toolUseID string) {
	if s.bridge != nil {
		s.bridge.resolve(toolUseID, true, "")
	}
}

func (s *claudeSession) denyTool(toolUseID, reason string) {
	if s.bridge != nil {
		s.bridge.resolve(toolUseID, false, reason)
	}
}

func (s *claudeSession) push(msg any) {
	b, _ := json.Marshal(msg)
	select {
	case s.send <- b:
	case <-s.stopped:
	}
}

func (s *claudeSession) pushError(code, message string) {
	s.push(protocol.ErrorMsg{
		Type:      protocol.TypeError,
		SessionID: s.id,
		Code:      code,
		Message:   message,
	})
}

// ── Main goroutine ────────────────────────────────────────────────────────────

func (s *claudeSession) run() {
	defer close(s.stopped)

	reason := "spawn_failed"
	defer func() {
		s.push(protocol.SessionEndedMsg{
			Type:      protocol.TypeSessionEnded,
			SessionID: s.id,
			Reason:    reason,
		})
	}()

	// Set up the permission bridge so the phone can approve/deny tool calls.
	// If it fails we fall back to auto-approval (claude runs without the flag).
	socketPath := filepath.Join(os.TempDir(), "remote-cli-perm-"+s.id+".sock")
	mcpConfigPath := filepath.Join(os.TempDir(), "remote-cli-mcp-"+s.id+".json")
	bridge, bridgeErr := newPermissionBridge(socketPath, s.id, s.send)
	if bridgeErr != nil {
		log.Printf("session %s: permission bridge unavailable (%v) — tools auto-approve", s.id, bridgeErr)
	} else {
		s.bridge = bridge
	}

	args := []string{
		"--input-format", "stream-json",
		"--output-format", "stream-json",
		"--verbose",
		"--include-partial-messages",
		"--no-session-persistence",
	}
	if s.bridge != nil {
		selfPath, exeErr := os.Executable()
		if exeErr == nil {
			mcpCfg := fmt.Sprintf(
				`{"mcpServers":{"approval":{"command":%q,"args":["mcp-server","--socket",%q]}}}`,
				selfPath, socketPath,
			)
			if writeErr := os.WriteFile(mcpConfigPath, []byte(mcpCfg), 0600); writeErr == nil {
				args = append(args,
					"--permission-prompt-tool", "mcp__approval__request_permission",
					"--mcp-config", mcpConfigPath,
				)
				log.Printf("session %s: phone-side tool approval enabled", s.id)
			}
		}
	}

	cmd := exec.Command("claude", args...)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		s.pushError("spawn_failed", fmt.Sprintf("stdin pipe: %v", err))
		return
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		s.pushError("spawn_failed", fmt.Sprintf("stdout pipe: %v", err))
		return
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		s.pushError("spawn_failed", fmt.Sprintf("stderr pipe: %v", err))
		return
	}

	if err := cmd.Start(); err != nil {
		s.pushError("spawn_failed", fmt.Sprintf("start: %v", err))
		return
	}

	reason = "session ended"
	defer func() {
		stdin.Close()
		cmd.Process.Kill()
		cmd.Wait() //nolint — exit code irrelevant
		if s.bridge != nil {
			s.bridge.close()
		}
		os.Remove(mcpConfigPath)
	}()

	log.Printf("session %s: claude pid %d started", s.id, cmd.Process.Pid)

	// Capture claude stderr: log each line and keep last 10 for error context.
	var stderrMu sync.Mutex
	var stderrLines []string
	go func() {
		sc := bufio.NewScanner(stderr)
		for sc.Scan() {
			line := sc.Text()
			log.Printf("session %s: claude stderr: %s", s.id, line)
			stderrMu.Lock()
			if len(stderrLines) < 10 {
				stderrLines = append(stderrLines, line)
			}
			stderrMu.Unlock()
		}
	}()

	s.push(protocol.SessionStartedMsg{
		Type:      protocol.TypeSessionStarted,
		SessionID: s.id,
	})

	// stdoutEvents receives parsed events from the reader goroutine.
	stdoutEvents := make(chan stdoutEvent, 64)
	go readStdout(stdout, stdoutEvents)

	idle := time.NewTimer(idleTimeout)
	defer idle.Stop()

	// lastText tracks cumulative text per message ID to compute deltas.
	lastText := make(map[string]string)
	// thinking tracks whether we're awaiting a claude result (don't send more msgs).
	thinking := false

	for {
		select {
		case <-s.stop:
			reason = "session ended by user"
			return

		case <-idle.C:
			reason = "idle timeout"
			return

		case content := <-s.msgCh:
			if !thinking {
				idle.Reset(idleTimeout)
				log.Printf("session %s: sending to claude: %q", s.id, content)
				if err := writeUserMsg(stdin, content); err != nil {
					reason = fmt.Sprintf("stdin write: %v", err)
					return
				}
				thinking = true
			} else {
				// Re-queue: put it back (best-effort; if full it's dropped).
				// In practice this rarely happens — the session only delivers
				// one message at a time from sessionMgr.deliver.
				go func(c string) { s.deliver(c) }(content)
			}

		case ev, ok := <-stdoutEvents:
			if !ok {
				reason = "claude process exited"
				stderrMu.Lock()
				snap := stderrLines
				stderrMu.Unlock()
				if len(snap) > 0 {
					s.pushError("process_exit", "Claude exited: "+strings.Join(snap, "; "))
				}
				return
			}
			if done := s.handleEvent(ev, lastText, &thinking, idle); done {
				return
			}
		}
	}
}

// handleEvent translates a claude stdout event into protocol messages.
// Returns true if the session should terminate.
func (s *claudeSession) handleEvent(ev stdoutEvent, lastText map[string]string, thinking *bool, _ *time.Timer) bool {
	switch ev.Type {
	case "assistant":
		for _, block := range ev.Content {
			switch block.Type {
			case "text":
				prev := lastText[ev.MsgID]
				if len(block.Text) <= len(prev) {
					continue
				}
				delta := block.Text[len(prev):]
				lastText[ev.MsgID] = block.Text
				log.Printf("session %s: chunk %q", s.id, delta)
				s.push(protocol.AssistantChunkMsg{
					Type:         protocol.TypeAssistantChunk,
					SessionID:    s.id,
					ContentBlock: protocol.ContentBlock{Type: "text", Text: delta},
				})
			case "tool_use":
				s.push(protocol.ToolUseRequestMsg{
					Type:      protocol.TypeToolUseRequest,
					SessionID: s.id,
					ToolUseID: block.ID,
					ToolName:  block.Name,
					ToolInput: block.Input,
				})
			}
		}

	case "user":
		// Tool results generated internally by the subprocess.
		for _, block := range ev.Content {
			if block.Type != "tool_result" {
				continue
			}
			s.push(protocol.ToolUseResultMsg{
				Type:      protocol.TypeToolUseResult,
				SessionID: s.id,
				ToolUseID: block.ToolUseID,
				Result:    block.ToolResultText(),
			})
		}

	case "result":
		// Turn complete.
		for k := range lastText {
			delete(lastText, k)
		}
		*thinking = false
		if ev.IsError {
			// Push error but keep session alive — user can retry after transient errors (rate limit, etc.)
			s.pushError("claude_error", ev.ErrorText)
			return false
		}
		log.Printf("session %s: turn complete (cost $%.4f)", s.id, ev.CostUSD)

	case "system":
		if ev.Subtype == "init" {
			log.Printf("session %s: claude init (tools: %d)", s.id, len(ev.Tools))
		}
	}
	return false
}

// ── stdout reader ─────────────────────────────────────────────────────────────

// stdoutEvent is a parsed representation of a line from claude's stdout.
type stdoutEvent struct {
	Type      string
	Subtype   string
	MsgID     string
	Content   []contentBlock
	IsError   bool
	ErrorText string
	CostUSD   float64
	Tools     []string
}

type contentBlock struct {
	Type      string
	Text      string
	ID        string
	Name      string
	Input     map[string]any
	ToolUseID string
	RawResult any
}

func (b *contentBlock) ToolResultText() string {
	switch v := b.RawResult.(type) {
	case string:
		return v
	case []any:
		out := ""
		for _, item := range v {
			if m, ok := item.(map[string]any); ok {
				if t, _ := m["text"].(string); t != "" {
					out += t
				}
			}
		}
		return out
	default:
		if v != nil {
			b, _ := json.Marshal(v)
			return string(b)
		}
		return ""
	}
}

func readStdout(r io.Reader, out chan<- stdoutEvent) {
	defer close(out)
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 4*1024*1024), 4*1024*1024)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		ev, ok := parseLine(line)
		if ok {
			out <- ev
		}
	}
}

func parseLine(raw []byte) (stdoutEvent, bool) {
	// Use a generic map first to extract top-level fields.
	var m map[string]json.RawMessage
	if err := json.Unmarshal(raw, &m); err != nil {
		return stdoutEvent{}, false
	}

	ev := stdoutEvent{}
	json.Unmarshal(m["type"], &ev.Type)
	json.Unmarshal(m["subtype"], &ev.Subtype)

	switch ev.Type {
	case "assistant", "user":
		var msg struct {
			ID      string `json:"id"`
			Content []struct {
				Type      string         `json:"type"`
				Text      string         `json:"text"`
				Thinking  string         `json:"thinking"`
				ID        string         `json:"id"`
				Name      string         `json:"name"`
				Input     map[string]any `json:"input"`
				ToolUseID string         `json:"tool_use_id"`
				Content   any            `json:"content"`
				IsError   bool           `json:"is_error"`
			} `json:"content"`
		}
		if msgRaw, ok := m["message"]; ok {
			json.Unmarshal(msgRaw, &msg)
		}
		ev.MsgID = msg.ID
		for _, c := range msg.Content {
			ev.Content = append(ev.Content, contentBlock{
				Type:      c.Type,
				Text:      c.Text,
				ID:        c.ID,
				Name:      c.Name,
				Input:     c.Input,
				ToolUseID: c.ToolUseID,
				RawResult: c.Content,
			})
		}

	case "result":
		var res struct {
			IsError    bool    `json:"is_error"`
			Result     string  `json:"result"`
			TotalCost  float64 `json:"total_cost_usd"`
		}
		json.Unmarshal(raw, &res)
		ev.IsError = res.IsError
		ev.ErrorText = res.Result
		ev.CostUSD = res.TotalCost

	case "system":
		if ev.Subtype == "init" {
			var init struct {
				Tools []string `json:"tools"`
			}
			json.Unmarshal(raw, &init)
			ev.Tools = init.Tools
		}
	}

	return ev, true
}

// writeUserMsg writes a stream-json user message to claude's stdin.
func writeUserMsg(w io.Writer, content string) error {
	msg := map[string]any{
		"type": "user",
		"message": map[string]any{
			"role":    "user",
			"content": content,
		},
	}
	b, _ := json.Marshal(msg)
	_, err := fmt.Fprintf(w, "%s\n", b)
	return err
}
