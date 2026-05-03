package agent

import (
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

type Config struct {
	RelayURL    string `toml:"relay_url"`
	DeviceID    string `toml:"device_id"`
	DeviceToken string `toml:"device_token"`
	DeviceName  string `toml:"device_name"`
}

func ConfigPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "remote-cli", "config.toml")
}

func LoadConfig() (*Config, error) {
	var cfg Config
	_, err := toml.DecodeFile(ConfigPath(), &cfg)
	if os.IsNotExist(err) {
		return &Config{}, nil
	}
	return &cfg, err
}

func SaveConfig(cfg *Config) error {
	path := ConfigPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	return toml.NewEncoder(f).Encode(cfg)
}
