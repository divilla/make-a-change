// Package app constructs the terminal shell and coordinates its local configuration and screens.
package app

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	goconfig "github.com/ridgelines/go-config"
	"gopkg.in/yaml.v3"
)

const defaultConfigPath = ".mch/config.yaml"

type appConfig struct {
	RepositoryRoot string
	ConfigPath     string
	BackendURL     string
	ProjectID      int
	Editor         string
}

type configFile struct {
	BackendURL string `yaml:"backend_url"`
	ProjectID  int    `yaml:"project_id"`
	Editor     string `yaml:"editor,omitempty"`
}

func loadRepositoryConfig() (appConfig, error) {
	repoRoot, err := resolveGitRepositoryRoot(context.Background())
	if err != nil {
		return appConfig{BackendURL: defaultBackendURL}, err
	}
	return loadAppConfig(repoRoot)
}

func loadAppConfig(repoRoot string) (appConfig, error) {
	repoRoot = strings.TrimSpace(repoRoot)
	if repoRoot == "" {
		return appConfig{BackendURL: defaultBackendURL}, fmt.Errorf("repository root is required")
	}
	configPath := filepath.Join(repoRoot, defaultConfigPath)
	cfg, err := loadConfigFile(configPath)
	if err != nil {
		return appConfig{RepositoryRoot: repoRoot, ConfigPath: configPath, BackendURL: defaultBackendURL}, err
	}
	cfg.RepositoryRoot = repoRoot
	cfg.ConfigPath = configPath
	return cfg, nil
}

func loadConfigFile(path string) (appConfig, error) {
	if _, err := os.Stat(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return appConfig{}, fmt.Errorf("config file %s is required", path)
		}
		return appConfig{}, err
	}
	cfg := goconfig.NewConfig([]goconfig.Provider{goconfig.NewYAMLFile(path)})
	backendURL, err := cfg.StringOr("backend_url", "")
	if err != nil {
		return appConfig{}, fmt.Errorf("load backend_url from %s: %w", path, err)
	}
	backendURL = strings.TrimSpace(backendURL)
	if backendURL == "" {
		return appConfig{}, fmt.Errorf("backend_url is required in %s", path)
	}
	projectID, err := cfg.IntOr("project_id", 0)
	if err != nil {
		return appConfig{}, fmt.Errorf("load project_id from %s: %w", path, err)
	}
	editor, err := cfg.StringOr("editor", "")
	if err != nil {
		return appConfig{}, fmt.Errorf("load editor from %s: %w", path, err)
	}
	return appConfig{BackendURL: backendURL, ProjectID: projectID, Editor: editor}, nil
}

func resolveGitRepositoryRoot(ctx context.Context) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "rev-parse", "--show-toplevel")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("resolve repository root with git: %w", err)
	}
	repoRoot := strings.TrimSpace(string(output))
	if repoRoot == "" {
		return "", fmt.Errorf("repository root is required")
	}
	return repoRoot, nil
}

func saveAppConfig(path string, cfg appConfig) error {
	backendURL := strings.TrimSpace(cfg.BackendURL)
	if backendURL == "" {
		return fmt.Errorf("backend_url is required")
	}
	body, err := yaml.Marshal(configFile{
		BackendURL: backendURL,
		ProjectID:  cfg.ProjectID,
		Editor:     cfg.Editor,
	})
	if err != nil {
		return err
	}
	if err := replaceFileAtomically(path, 0o644, func(file *os.File) error {
		_, err := file.Write(body)
		return err
	}); err != nil {
		return fmt.Errorf("save config %s: %w", path, err)
	}
	return nil
}

func renderResolvedConfig(cfg appConfig) string {
	var b bytes.Buffer
	writeConfigLine(&b, "repository_root", cfg.RepositoryRoot)
	writeConfigLine(&b, "config_path", cfg.ConfigPath)
	writeConfigLine(&b, "backend_url", cfg.BackendURL)
	fmt.Fprintf(&b, "project_id: %d\n", cfg.ProjectID)
	return strings.TrimRight(b.String(), "\n")
}

func writeConfigLine(b *bytes.Buffer, key string, value string) {
	fmt.Fprintf(b, "%s: %s\n", key, value)
}
