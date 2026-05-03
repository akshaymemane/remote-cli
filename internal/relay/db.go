package relay

import (
	"database/sql"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type DB struct {
	db *sql.DB
}

func NewDB(path string) (*DB, error) {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1) // SQLite: single writer
	if err := migrate(db); err != nil {
		return nil, err
	}
	return &DB{db: db}, nil
}

func migrate(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS devices (
			id         TEXT PRIMARY KEY,
			name       TEXT    NOT NULL,
			token_hash TEXT    NOT NULL,
			created_at INTEGER NOT NULL,
			last_seen  INTEGER
		);
		CREATE TABLE IF NOT EXISTS admin (
			id            INTEGER PRIMARY KEY CHECK (id = 1),
			password_hash TEXT NOT NULL
		);
	`)
	return err
}

type Device struct {
	ID        string
	Name      string
	TokenHash string
	CreatedAt int64
	LastSeen  *int64
}

func (db *DB) CreateDevice(id, name, tokenHash string) error {
	_, err := db.db.Exec(
		`INSERT INTO devices (id, name, token_hash, created_at) VALUES (?, ?, ?, ?)`,
		id, name, tokenHash, time.Now().Unix(),
	)
	return err
}

func (db *DB) GetDeviceByID(id string) (*Device, error) {
	d := &Device{}
	err := db.db.QueryRow(
		`SELECT id, name, token_hash, created_at, last_seen FROM devices WHERE id = ?`, id,
	).Scan(&d.ID, &d.Name, &d.TokenHash, &d.CreatedAt, &d.LastSeen)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return d, err
}

func (db *DB) ListDevices() ([]Device, error) {
	rows, err := db.db.Query(`SELECT id, name, token_hash, created_at, last_seen FROM devices ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var devices []Device
	for rows.Next() {
		var d Device
		if err := rows.Scan(&d.ID, &d.Name, &d.TokenHash, &d.CreatedAt, &d.LastSeen); err != nil {
			return nil, err
		}
		devices = append(devices, d)
	}
	return devices, rows.Err()
}

func (db *DB) UpdateDeviceLastSeen(id string) error {
	now := time.Now().Unix()
	_, err := db.db.Exec(`UPDATE devices SET last_seen = ? WHERE id = ?`, now, id)
	return err
}

func (db *DB) RenameDevice(id, name string) error {
	_, err := db.db.Exec(`UPDATE devices SET name = ? WHERE id = ?`, name, id)
	return err
}

func (db *DB) DeleteDevice(id string) error {
	_, err := db.db.Exec(`DELETE FROM devices WHERE id = ?`, id)
	return err
}

func (db *DB) SetAdminPassword(hash string) error {
	_, err := db.db.Exec(`INSERT OR REPLACE INTO admin (id, password_hash) VALUES (1, ?)`, hash)
	return err
}

func (db *DB) GetAdminPasswordHash() (string, error) {
	var hash string
	err := db.db.QueryRow(`SELECT password_hash FROM admin WHERE id = 1`).Scan(&hash)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return hash, err
}
