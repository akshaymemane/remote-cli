package agent

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	qrcode "github.com/skip2/go-qrcode"
	"remote-cli/internal/protocol"
)

// Pair runs the interactive pairing flow: connects to the relay, obtains a
// pairing code, renders the QR in the terminal, and waits for confirmation.
func Pair(relayURL string) error {
	cfg, err := LoadConfig()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	if cfg.DeviceToken != "" {
		return fmt.Errorf("already paired as %q — delete %s to reset", cfg.DeviceName, ConfigPath())
	}

	wsURL := httpToWS(relayURL) + "/ws/agent"
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		return fmt.Errorf("connect to relay: %w", err)
	}
	defer ws.Close()

	// Read connection ID.
	ws.SetReadDeadline(time.Now().Add(10 * time.Second))
	_, raw, err := ws.ReadMessage()
	ws.SetReadDeadline(time.Time{})
	if err != nil {
		return fmt.Errorf("read connected msg: %w", err)
	}
	var connected protocol.ConnectedMsg
	if err := json.Unmarshal(raw, &connected); err != nil || connected.Type != protocol.TypeConnected {
		return fmt.Errorf("unexpected message: %s", raw)
	}

	// Ask relay to create a pairing code bound to our WebSocket connection.
	body, _ := json.Marshal(map[string]string{"connection_id": connected.ConnectionID})
	resp, err := http.Post(relayURL+"/api/pair/request", "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("pair request: %w", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("pair request failed: %s", resp.Status)
	}

	// Relay sends the code back over WebSocket.
	ws.SetReadDeadline(time.Now().Add(10 * time.Second))
	_, raw, err = ws.ReadMessage()
	ws.SetReadDeadline(time.Time{})
	if err != nil {
		return fmt.Errorf("read pair code: %w", err)
	}
	var pairCode protocol.PairCodeMsg
	if err := json.Unmarshal(raw, &pairCode); err != nil || pairCode.Type != protocol.TypePairCode {
		return fmt.Errorf("unexpected message: %s", raw)
	}

	printQR(pairCode.URL)
	fmt.Printf("\n  Code: %s\n", pairCode.Code)
	fmt.Printf("  URL:  %s\n\n", pairCode.URL)
	fmt.Println("Open the PWA, tap Add device, and scan the QR code or enter the code above.")
	fmt.Println("Waiting up to 5 minutes...")

	// Wait for pair.complete.
	ws.SetReadDeadline(time.Now().Add(5 * time.Minute))
	_, raw, err = ws.ReadMessage()
	ws.SetReadDeadline(time.Time{})
	if err != nil {
		return fmt.Errorf("pairing timed out: %w", err)
	}
	var complete protocol.PairCompleteMsg
	if err := json.Unmarshal(raw, &complete); err != nil || complete.Type != protocol.TypePairComplete {
		return fmt.Errorf("unexpected message: %s", raw)
	}

	cfg.RelayURL = relayURL
	cfg.DeviceID = complete.DeviceID
	cfg.DeviceToken = complete.DeviceToken
	cfg.DeviceName = complete.DeviceName
	if err := SaveConfig(cfg); err != nil {
		return fmt.Errorf("save config: %w", err)
	}

	fmt.Printf("Paired as %q\n", complete.DeviceName)
	return nil
}

func printQR(content string) {
	q, err := qrcode.New(content, qrcode.Medium)
	if err != nil {
		return
	}
	fmt.Print(q.ToString(false))
}

func httpToWS(u string) string {
	switch {
	case strings.HasPrefix(u, "https://"):
		return "wss://" + u[8:]
	case strings.HasPrefix(u, "http://"):
		return "ws://" + u[7:]
	default:
		return u
	}
}
