package app

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
)

const defaultBackendURL = "http://localhost:8080"

type Config struct {
	BackendURL       string `json:"backend_url"`
	CurrentProjectID *int   `json:"current_project_id,omitempty"`
}

func configPath(repoRoot string) string {
	return filepath.Join(repoRoot, "cli-proto", ".config", "config.json")
}

func loadConfig(path string) (Config, bool, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		cfg := Config{BackendURL: defaultBackendURL}
		return cfg, true, saveConfig(path, cfg)
	}
	if err != nil {
		return Config{}, false, err
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, false, err
	}
	if strings.TrimSpace(cfg.BackendURL) == "" {
		cfg.BackendURL = defaultBackendURL
		return cfg, true, saveConfig(path, cfg)
	}
	cfg.BackendURL = strings.TrimSpace(cfg.BackendURL)
	return cfg, false, nil
}

func saveConfig(path string, cfg Config) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o644)
}
