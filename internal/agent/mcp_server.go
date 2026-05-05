package agent

import (
	"bufio"
	"encoding/json"
	"fmt"
	"math/rand"
	"net"
	"os"
	"time"
)

// RunMCPPermissionServer runs a minimal MCP stdio server that proxies Claude
// Code tool-permission requests to the agent's permission bridge over a Unix
// socket.  It is spawned as a subprocess by the claude CLI when
// --permission-prompt-tool mcp__approval__request_permission is used.
func RunMCPPermissionServer(socketPath string) error {
	// Retry a few times because the bridge socket may not be ready yet.
	var conn net.Conn
	var err error
	for range 5 {
		conn, err = net.Dial("unix", socketPath)
		if err == nil {
			break
		}
		time.Sleep(200 * time.Millisecond)
	}
	if err != nil {
		return fmt.Errorf("connect to permission bridge: %w", err)
	}
	defer conn.Close()

	bridgeEnc := json.NewEncoder(conn)
	bridgeDec := json.NewDecoder(conn)

	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
	enc := json.NewEncoder(os.Stdout)

	respond := func(id json.RawMessage, result any) {
		enc.Encode(map[string]any{"jsonrpc": "2.0", "id": id, "result": result}) //nolint
	}
	respondErr := func(id json.RawMessage, code int, msg string) {
		enc.Encode(map[string]any{ //nolint
			"jsonrpc": "2.0", "id": id,
			"error": map[string]any{"code": code, "message": msg},
		})
	}

	for scanner.Scan() {
		var req struct {
			ID     json.RawMessage `json:"id"`
			Method string          `json:"method"`
			Params json.RawMessage `json:"params,omitempty"`
		}
		if err := json.Unmarshal(scanner.Bytes(), &req); err != nil {
			continue
		}
		// Notifications have no id — no response needed.
		if req.ID == nil {
			continue
		}

		switch req.Method {
		case "initialize":
			respond(req.ID, map[string]any{
				"protocolVersion": "2024-11-05",
				"capabilities":    map[string]any{"tools": map[string]any{}},
				"serverInfo":      map[string]any{"name": "remote-cli-approval", "version": "1.0"},
			})

		case "tools/list":
			respond(req.ID, map[string]any{
				"tools": []map[string]any{{
					"name":        "request_permission",
					"description": "Forward a Claude Code tool-permission request to the remote user for approval",
					"inputSchema": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"tool_name":  map[string]any{"type": "string"},
							"tool_input": map[string]any{"type": "object"},
						},
						"required": []string{"tool_name"},
					},
				}},
			})

		case "tools/call":
			var params struct {
				Name      string         `json:"name"`
				Arguments map[string]any `json:"arguments"`
			}
			if err := json.Unmarshal(req.Params, &params); err != nil || params.Name != "request_permission" {
				respondErr(req.ID, -32601, "unknown tool")
				continue
			}

			toolName, _ := params.Arguments["tool_name"].(string)
			toolInput, _ := params.Arguments["tool_input"].(map[string]any)
			toolUseID := fmt.Sprintf("perm-%x%x", time.Now().UnixNano(), rand.Int63()) //nolint:gosec

			if err := bridgeEnc.Encode(bridgeRequest{
				ToolUseID: toolUseID,
				ToolName:  toolName,
				ToolInput: toolInput,
			}); err != nil {
				respondErr(req.ID, -32603, "bridge write error")
				continue
			}

			var decision bridgeResponse
			if err := bridgeDec.Decode(&decision); err != nil {
				respondErr(req.ID, -32603, "bridge read error")
				continue
			}

			var behavior map[string]any
			if decision.Allow {
				behavior = map[string]any{"behavior": "allow"}
			} else {
				msg := decision.Message
				if msg == "" {
					msg = "Permission denied by remote user"
				}
				behavior = map[string]any{"behavior": "deny", "message": msg}
			}
			behaviorJSON, _ := json.Marshal(behavior)
			respond(req.ID, map[string]any{
				"content": []map[string]any{{"type": "text", "text": string(behaviorJSON)}},
			})

		default:
			respondErr(req.ID, -32601, "method not found")
		}
	}
	return scanner.Err()
}
