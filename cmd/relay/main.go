package main

import (
	"bufio"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"syscall"

	"golang.org/x/term"
	"remote-cli/internal/relay"
)

func main() {
	addr := envOr("RELAY_ADDR", ":8080")
	dbPath := envOr("RELAY_DB", "relay.db")
	jwtSecret := envOr("RELAY_JWT_SECRET", "change-me-at-least-32-chars-long!!")
	relayURL := envOr("RELAY_URL", "http://localhost"+addr)
	staticDir := envOr("RELAY_STATIC_DIR", "pwa/dist")

	if jwtSecret == "change-me-at-least-32-chars-long!!" {
		log.Println("WARNING: using default RELAY_JWT_SECRET — set a strong secret in production")
	}

	db, err := relay.NewDB(dbPath)
	if err != nil {
		log.Fatalf("db: %v", err)
	}

	auth := relay.NewAuth(db, jwtSecret)

	hash, err := db.GetAdminPasswordHash()
	if err != nil {
		log.Fatalf("db: %v", err)
	}
	if hash == "" {
		// Allow non-interactive setup via env var (useful for Docker).
		if pw := os.Getenv("RELAY_ADMIN_PASSWORD"); pw != "" {
			hashed, err := auth.HashPassword(pw)
			if err != nil {
				log.Fatalf("hash password: %v", err)
			}
			if err := db.SetAdminPassword(hashed); err != nil {
				log.Fatalf("set admin password: %v", err)
			}
			log.Println("admin password set from RELAY_ADMIN_PASSWORD")
		} else if err := setupAdmin(db, auth); err != nil {
			log.Fatalf("setup: %v", err)
		}
	}

	hub := relay.NewHub()
	pairing := relay.NewPairingStore()
	srv := relay.NewServer(db, auth, hub, pairing, relayURL, staticDir)

	log.Printf("relay listening on %s", addr)
	if err := http.ListenAndServe(addr, srv.Routes()); err != nil {
		log.Fatal(err)
	}
}

func setupAdmin(db *relay.DB, auth *relay.Auth) error {
	fmt.Print("First run — set admin password: ")
	var password string
	if term.IsTerminal(int(syscall.Stdin)) {
		b, err := term.ReadPassword(int(syscall.Stdin))
		fmt.Println()
		if err != nil {
			return err
		}
		password = string(b)
	} else {
		scanner := bufio.NewScanner(os.Stdin)
		scanner.Scan()
		password = strings.TrimSpace(scanner.Text())
	}
	if password == "" {
		return fmt.Errorf("password cannot be empty")
	}
	hashed, err := auth.HashPassword(password)
	if err != nil {
		return err
	}
	return db.SetAdminPassword(hashed)
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
