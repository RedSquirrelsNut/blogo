package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

const configFileName = ".blogo.json"
const defaultDBPath = "/home/dev/go/blogo/feed.db"

func getConfigFilePath() (string, error) {
	if home, err := os.UserHomeDir(); err != nil {
		return "", err
	} else {
		path := filepath.Join(home, configFileName)
		return path, nil
	}
}

type Config struct {
	DBPath      string `json:"db_path"`
	CurrentUser string `json:"current_user"`
	path        string
}

func Read() (*Config, error) {
	if path, err := getConfigFilePath(); err != nil {
		return nil, err
	} else {
		cfg := &Config{path: path}
		if data, err := os.ReadFile(cfg.path); err != nil && !os.IsNotExist(err) {
			return nil, err
		} else if err != nil {
			if cfg.DBPath == "" {
				cfg.DBPath = defaultDBPath
			}
			if writeErr := cfg.write(); writeErr != nil {
				return nil, writeErr
			}
		} else {
			if unmarshalErr := json.Unmarshal(data, cfg); unmarshalErr != nil {
				return nil, unmarshalErr
			}
		}
		return cfg, nil
	}
}

func (cfg *Config) write() error {
	if data, err := json.MarshalIndent(cfg, "", "  "); err != nil {
		return err
	} else {
		if path, err := getConfigFilePath(); err != nil {
			return err
		} else {
			return os.WriteFile(path, data, 0666)
		}
	}
}

func (cfg *Config) SetUser(user string) error {
	cfg.CurrentUser = user
	return cfg.write()
}
