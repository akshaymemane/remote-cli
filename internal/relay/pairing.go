package relay

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"sync"
	"time"
)

type pairingEntry struct {
	connID string
	expiry time.Time
}

type PairingStore struct {
	mu      sync.Mutex
	byCode  map[string]*pairingEntry // code → entry
	byConn  map[string]string        // connID → code
}

func NewPairingStore() *PairingStore {
	ps := &PairingStore{
		byCode: make(map[string]*pairingEntry),
		byConn: make(map[string]string),
	}
	go ps.cleanupLoop()
	return ps
}

func (ps *PairingStore) Create(connID string) (string, error) {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	// Remove any existing code for this connection.
	if old, ok := ps.byConn[connID]; ok {
		delete(ps.byCode, old)
	}

	code, err := randomCode()
	if err != nil {
		return "", err
	}
	ps.byCode[code] = &pairingEntry{connID: connID, expiry: time.Now().Add(5 * time.Minute)}
	ps.byConn[connID] = code
	return code, nil
}

// Redeem validates and consumes the code. Returns the connection ID on success.
func (ps *PairingStore) Redeem(code string) (string, bool) {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	entry, ok := ps.byCode[code]
	if !ok || time.Now().After(entry.expiry) {
		delete(ps.byCode, code)
		return "", false
	}
	connID := entry.connID
	delete(ps.byCode, code)
	delete(ps.byConn, connID)
	return connID, true
}

func (ps *PairingStore) cleanupLoop() {
	t := time.NewTicker(time.Minute)
	for range t.C {
		ps.mu.Lock()
		for code, e := range ps.byCode {
			if time.Now().After(e.expiry) {
				delete(ps.byConn, e.connID)
				delete(ps.byCode, code)
			}
		}
		ps.mu.Unlock()
	}
}

func randomCode() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(1_000_000))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}
