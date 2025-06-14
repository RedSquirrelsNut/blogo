package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

const configFileName = ".blogo.json"
const defaultDBPath = "/home/dev/go/blogo/feed.db"

func getConfigFilePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	path := filepath.Join(home, configFileName)
	return path, nil
}

type Config struct {
	DBPath      string `json:"db_path"`
	CurrentUser string `json:"current_user"`
	path        string
}

func Read() (*Config, error) {
	path, err := getConfigFilePath()
	if err != nil {
		return nil, err
	}

	cfg := &Config{path: path}
	// Try reading existing config
	data, err := os.ReadFile(cfg.path)
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	// If file existed, unmarshal
	if err == nil {
		if unmarshalErr := json.Unmarshal(data, cfg); unmarshalErr != nil {
			return nil, unmarshalErr
		}
	}

	// Ensure a default DBPath if none provided
	if cfg.DBPath == "" {
		cfg.DBPath = defaultDBPath
	}

	// On first run (file missing), or if we updated defaults, save it
	if err != nil {
		if writeErr := cfg.write(); writeErr != nil {
			return nil, writeErr
		}
	}
	return cfg, nil
}

func (cfg *Config) write() error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	path, err := getConfigFilePath()
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0666)

}

func (cfg *Config) SetUser(user string) error {
	cfg.CurrentUser = user
	return cfg.write()
}
