//go:build ignore

// Run with: go run scripts/phase2_test.go
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

func main() {
	relayURL := "http://localhost:9191"
	wsBase := "ws://localhost:9191"
	token := mustLogin(relayURL, "testpass")
	fmt.Printf("JWT: %d chars\n", len(token))

	// Connect phone WS
	ws, _, err := websocket.DefaultDialer.Dial(wsBase+"/ws/phone", nil)
	must(err, "phone dial")
	defer ws.Close()

	// Authenticate
	mustSend(ws, map[string]any{"type": "client.auth", "token": token})

	// Receive device list
	msg := recvJSON(ws)
	fmt.Printf("← %s\n", pretty(msg))

	devices, _ := msg["devices"].([]any)
	if len(devices) == 0 {
		log.Fatal("no devices — is the agent running?")
	}
	dev := devices[0].(map[string]any)
	deviceID := dev["id"].(string)
	fmt.Printf("device: %s (%s)\n", dev["name"], deviceID)

	// Start session
	mustSend(ws, map[string]any{"type": "session.start", "device_id": deviceID})

	// session.state (relay ack)
	stateMsg := recvJSON(ws)
	fmt.Printf("← %s\n", pretty(stateMsg))
	sessionID := stateMsg["session_id"].(string)
	fmt.Printf("session_id: %s\n", sessionID)

	// session.started (agent ack)
	startedMsg := recvJSON(ws)
	fmt.Printf("← %s\n", pretty(startedMsg))

	// Send a message
	mustSend(ws, map[string]any{
		"type":       "message.user",
		"session_id": sessionID,
		"content":    "Hello from phone echo test",
	})

	// Collect streaming chunks
	var fullText strings.Builder
	for i := 0; i < 50; i++ {
		ws.SetReadDeadline(time.Now().Add(3 * time.Second))
		chunk := recvJSON(ws)
		ws.SetReadDeadline(time.Time{})
		fmt.Printf("← %s\n", pretty(chunk))
		if block, ok := chunk["content_block"].(map[string]any); ok {
			text, _ := block["text"].(string)
			fullText.WriteString(text)
			if strings.HasSuffix(text, "\n") {
				break
			}
		}
	}

	fmt.Printf("\nFull echoed text: %q\n", fullText.String())

	// End session
	mustSend(ws, map[string]any{
		"type":       "session.end",
		"session_id": sessionID,
		"device_id":  deviceID,
	})

	// session.ended
	ws.SetReadDeadline(time.Now().Add(3 * time.Second))
	ended := recvJSON(ws)
	ws.SetReadDeadline(time.Time{})
	fmt.Printf("← %s\n", pretty(ended))

	fmt.Println("\nPhase 2 test PASSED ✓")
}

func mustLogin(relayURL, password string) string {
	body, _ := json.Marshal(map[string]string{"password": password})
	resp, err := http.Post(relayURL+"/api/auth/login", "application/json", strings.NewReader(string(body)))
	must(err, "login")
	defer resp.Body.Close()
	var r map[string]string
	json.NewDecoder(resp.Body).Decode(&r)
	return r["token"]
}

func mustSend(ws *websocket.Conn, v any) {
	b, _ := json.Marshal(v)
	must(ws.WriteMessage(websocket.TextMessage, b), "send")
}

func recvJSON(ws *websocket.Conn) map[string]any {
	_, raw, err := ws.ReadMessage()
	must(err, "recv")
	var m map[string]any
	json.Unmarshal(raw, &m)
	return m
}

func pretty(m map[string]any) string {
	b, _ := json.Marshal(m)
	return string(b)
}

func must(err error, ctx string) {
	if err != nil {
		log.Fatalf("%s: %v", ctx, err)
	}
}
