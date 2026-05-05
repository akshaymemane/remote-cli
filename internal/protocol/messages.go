package protocol

type MessageType string

const (
	// Agent → Relay
	TypeDeviceRegister  MessageType = "device.register"
	TypeDeviceHeartbeat MessageType = "device.heartbeat"
	TypeSessionStarted  MessageType = "session.started"
	TypeSessionEnded    MessageType = "session.ended"
	TypeAssistantChunk  MessageType = "message.assistant_chunk"
	TypeToolUseRequest  MessageType = "tool_use.request"
	TypeToolUseResult   MessageType = "tool_use.result"
	TypeError           MessageType = "error"

	// Phone → Relay
	TypeClientAuth     MessageType = "client.auth"
	TypeDeviceList     MessageType = "device.list"
	TypeSessionStart   MessageType = "session.start"
	TypeSessionEnd     MessageType = "session.end"
	TypeUserMessage    MessageType = "message.user"
	TypeToolUseApprove MessageType = "tool_use.approve"
	TypeToolUseDeny    MessageType = "tool_use.deny"

	// Relay → Phone
	TypeDeviceUpdate MessageType = "device.update"
	TypeSessionState MessageType = "session.state"

	// Relay ↔ Agent (pairing)
	TypeConnected    MessageType = "connected"
	TypePairCode     MessageType = "pair.code"
	TypePairComplete MessageType = "pair.complete"
)

// ── Shared envelope ───────────────────────────────────────────────────────────

type Envelope struct {
	Type MessageType `json:"type"`
}

// ── Pairing ───────────────────────────────────────────────────────────────────

type ConnectedMsg struct {
	Type         MessageType `json:"type"`
	ConnectionID string      `json:"connection_id"`
}

type DeviceRegisterMsg struct {
	Type        MessageType `json:"type"`
	DeviceID    string      `json:"device_id"`
	DeviceToken string      `json:"device_token"`
}

type PairCodeMsg struct {
	Type MessageType `json:"type"`
	Code string      `json:"code"`
	URL  string      `json:"url"`
}

type PairCompleteMsg struct {
	Type        MessageType `json:"type"`
	DeviceID    string      `json:"device_id"`
	DeviceToken string      `json:"device_token"`
	DeviceName  string      `json:"device_name"`
}

// ── Devices ───────────────────────────────────────────────────────────────────

type DeviceInfo struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Status   string `json:"status"`
	LastSeen *int64 `json:"last_seen,omitempty"`
}

type DeviceListMsg struct {
	Type    MessageType  `json:"type"`
	Devices []DeviceInfo `json:"devices"`
}

type DeviceUpdateMsg struct {
	Type     MessageType `json:"type"`
	DeviceID string      `json:"device_id"`
	Status   string      `json:"status"`
	Name     string      `json:"name,omitempty"`
	LastSeen *int64      `json:"last_seen,omitempty"`
}

// ── Sessions ──────────────────────────────────────────────────────────────────

// RelaySessionStartMsg is sent relay→agent to spawn a session.
type RelaySessionStartMsg struct {
	Type      MessageType `json:"type"`
	SessionID string      `json:"session_id"`
}

// RelaySessionEndMsg is sent relay→agent to terminate a session.
type RelaySessionEndMsg struct {
	Type      MessageType `json:"type"`
	SessionID string      `json:"session_id"`
	Reason    string      `json:"reason,omitempty"`
}

// SessionStartedMsg is sent agent→relay→phone confirming a session is live.
type SessionStartedMsg struct {
	Type      MessageType `json:"type"`
	SessionID string      `json:"session_id"`
}

// SessionEndedMsg is sent agent→relay→phone when a session terminates.
type SessionEndedMsg struct {
	Type      MessageType `json:"type"`
	SessionID string      `json:"session_id"`
	Reason    string      `json:"reason"`
}

type SessionStateMsg struct {
	Type      MessageType `json:"type"`
	DeviceID  string      `json:"device_id"`
	SessionID string      `json:"session_id,omitempty"`
	Status    string      `json:"status"`
}

// ── Messages ──────────────────────────────────────────────────────────────────

type UserMessageMsg struct {
	Type      MessageType `json:"type"`
	SessionID string      `json:"session_id"`
	Content   string      `json:"content"`
}

type ContentBlock struct {
	Type string `json:"type"` // always "text" in Phase 2
	Text string `json:"text"`
}

type AssistantChunkMsg struct {
	Type         MessageType  `json:"type"`
	SessionID    string       `json:"session_id"`
	ContentBlock ContentBlock `json:"content_block"`
	Index        int          `json:"index"`
}

// ── Tool use ──────────────────────────────────────────────────────────────────

type ToolUseRequestMsg struct {
	Type             MessageType    `json:"type"`
	SessionID        string         `json:"session_id"`
	ToolUseID        string         `json:"tool_use_id"`
	ToolName         string         `json:"tool_name"`
	ToolInput        map[string]any `json:"tool_input"`
	AwaitingApproval bool           `json:"awaiting_approval,omitempty"`
}

type ToolUseApproveMsg struct {
	Type      MessageType `json:"type"`
	SessionID string      `json:"session_id,omitempty"`
	ToolUseID string      `json:"tool_use_id"`
}

type ToolUseDenyMsg struct {
	Type      MessageType `json:"type"`
	SessionID string      `json:"session_id,omitempty"`
	ToolUseID string      `json:"tool_use_id"`
	Reason    string      `json:"reason,omitempty"`
}

type ToolUseResultMsg struct {
	Type      MessageType `json:"type"`
	SessionID string      `json:"session_id"`
	ToolUseID string      `json:"tool_use_id"`
	Result    any         `json:"result"`
}

// ── Errors ────────────────────────────────────────────────────────────────────

type ErrorMsg struct {
	Type      MessageType `json:"type"`
	SessionID string      `json:"session_id,omitempty"`
	Code      string      `json:"code"`
	Message   string      `json:"message"`
}
