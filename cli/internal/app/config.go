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

    "cli/internal/agent"

    goconfig "github.com/ridgelines/go-config"
    "gopkg.in/yaml.v3"
)

const defaultConfigPath = ".mch/config.yaml"

type appConfig struct {
    RepositoryRoot string
    ConfigPath     string
    BackendURL     string
    ProjectID      int
    FlowDir        string
    Flow           flowConfig
    FlowHelp       flowHelpConfig
}

type configFile struct {
    BackendURL string `yaml:"backend_url"`
    ProjectID  int    `yaml:"project_id"`
}

type flowConfig struct {
    Version        int               `yaml:"version"`
    Slug           string            `yaml:"slug"`
    Name           string            `yaml:"name"`
    Description    string            `yaml:"description"`
    Help           string            `yaml:"help"`
    Makefile       string            `yaml:"makefile"`
    Steps          []flowStep        `yaml:"steps"`
    UtilityPrompts map[string]string `yaml:"utility_prompts"`
}

type flowStep struct {
    Slug   string `yaml:"slug"`
    Help   string `yaml:"help"`
    Type   string `yaml:"type"`
    Prompt string `yaml:"prompt"`
    Entry  string `yaml:"entry"`
    Exec   string `yaml:"exec"`
    Exit   string `yaml:"exit"`
}

type flowHelpConfig struct {
    Version      int          `yaml:"version"`
    StageModes   []flowOption `yaml:"stage_modes"`
    TaskStatuses []flowOption `yaml:"task_statuses"`
    TaskSteps    []flowOption `yaml:"task_steps"`
}

type flowOption struct {
    Slug string `yaml:"slug"`
    Help string `yaml:"help"`
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
    cfg.FlowDir = filepath.Join(repoRoot, agent.DefaultDir)
    flow, err := loadFlowConfig(cfg.FlowDir)
    if err != nil {
        return cfg, err
    }
    help, err := loadFlowHelpConfig(cfg.FlowDir, flow.Help)
    if err != nil {
        return cfg, err
    }
    cfg.Flow = flow
    cfg.FlowHelp = help
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
    return appConfig{BackendURL: backendURL, ProjectID: projectID}, nil
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

func loadFlowConfig(flowDir string) (flowConfig, error) {
    path := filepath.Join(flowDir, "flow.yaml")
    content, err := os.ReadFile(path)
    if err != nil {
        return flowConfig{}, fmt.Errorf("load Flow config %s: %w", path, err)
    }
    var flow flowConfig
    if err := yaml.Unmarshal(content, &flow); err != nil {
        return flowConfig{}, fmt.Errorf("parse Flow config %s: %w", path, err)
    }
    if strings.TrimSpace(flow.Slug) == "" {
        return flowConfig{}, fmt.Errorf("flow slug is required in %s", path)
    }
    if strings.TrimSpace(flow.Help) == "" {
        return flowConfig{}, fmt.Errorf("flow help file is required in %s", path)
    }
    if strings.TrimSpace(flow.Makefile) == "" {
        return flowConfig{}, fmt.Errorf("flow Makefile path is required in %s", path)
    }
    if len(flow.Steps) == 0 {
        return flowConfig{}, fmt.Errorf("flow steps are required in %s", path)
    }
    for i, step := range flow.Steps {
        slug := strings.TrimSpace(step.Slug)
        if slug == "" {
            return flowConfig{}, fmt.Errorf("flow step %d slug is required in %s", i+1, path)
        }
        for previousIndex, previousStep := range flow.Steps[:i] {
            if strings.TrimSpace(previousStep.Slug) == slug {
                return flowConfig{}, fmt.Errorf("flow step %d duplicates slug %q from step %d in %s", i+1, slug, previousIndex+1, path)
            }
        }
        if strings.TrimSpace(step.Type) == "" {
            return flowConfig{}, fmt.Errorf("flow step %d type is required in %s", i+1, path)
        }
    }
    return flow, nil
}

func loadFlowHelpConfig(flowDir string, helpPath string) (flowHelpConfig, error) {
    path := filepath.Join(flowDir, helpPath)
    content, err := os.ReadFile(path)
    if err != nil {
        return flowHelpConfig{}, fmt.Errorf("load Flow help config %s: %w", path, err)
    }
    var help flowHelpConfig
    if err := yaml.Unmarshal(content, &help); err != nil {
        return flowHelpConfig{}, fmt.Errorf("parse Flow help config %s: %w", path, err)
    }
    if err := validateFlowOptions(path, "stage_modes", help.StageModes); err != nil {
        return flowHelpConfig{}, err
    }
    if err := validateFlowOptions(path, "task_statuses", help.TaskStatuses); err != nil {
        return flowHelpConfig{}, err
    }
    if err := validateFlowOptions(path, "task_steps", help.TaskSteps); err != nil {
        return flowHelpConfig{}, err
    }
    return help, nil
}

func validateFlowOptions(path string, name string, options []flowOption) error {
    for _, option := range options {
        if strings.TrimSpace(option.Slug) == "" {
            return fmt.Errorf("%s option slug is required in %s", name, path)
        }
    }
    return nil
}

func saveAppConfig(path string, cfg appConfig) error {
    backendURL := strings.TrimSpace(cfg.BackendURL)
    if backendURL == "" {
        return fmt.Errorf("backend_url is required")
    }
    body, err := yaml.Marshal(configFile{
        BackendURL: backendURL,
        ProjectID:  cfg.ProjectID,
    })
    if err != nil {
        return err
    }
    return os.WriteFile(path, body, 0o644)
}

func renderResolvedConfig(cfg appConfig) string {
    var b bytes.Buffer
    writeConfigLine(&b, "repository_root", cfg.RepositoryRoot)
    writeConfigLine(&b, "config_path", cfg.ConfigPath)
    writeConfigLine(&b, "backend_url", cfg.BackendURL)
    fmt.Fprintf(&b, "project_id: %d\n", cfg.ProjectID)
    writeConfigLine(&b, "flow_dir", cfg.FlowDir)
    fmt.Fprintf(&b, "flow:\n")
    fmt.Fprintf(&b, "  version: %d\n", cfg.Flow.Version)
    writeConfigLine(&b, "  slug", cfg.Flow.Slug)
    writeConfigLine(&b, "  name", cfg.Flow.Name)
    writeConfigLine(&b, "  description", cfg.Flow.Description)
    writeConfigLine(&b, "  help", cfg.Flow.Help)
    writeConfigLine(&b, "  makefile", cfg.Flow.Makefile)
    fmt.Fprintf(&b, "  steps:\n")
    for _, step := range cfg.Flow.Steps {
        fmt.Fprintf(&b, "    - slug: %s\n", step.Slug)
        writeConfigLine(&b, "      help", step.Help)
        writeConfigLine(&b, "      type", step.Type)
        writeConfigLine(&b, "      prompt", step.Prompt)
        writeConfigLine(&b, "      entry", step.Entry)
        writeConfigLine(&b, "      exec", step.Exec)
        writeConfigLine(&b, "      exit", step.Exit)
    }
    fmt.Fprintf(&b, "stage_modes:\n")
    writeFlowOptions(&b, cfg.FlowHelp.StageModes)
    fmt.Fprintf(&b, "task_statuses:\n")
    writeFlowOptions(&b, cfg.FlowHelp.TaskStatuses)
    fmt.Fprintf(&b, "task_steps:\n")
    writeFlowOptions(&b, cfg.FlowHelp.TaskSteps)
    return strings.TrimRight(b.String(), "\n")
}

func writeConfigLine(b *bytes.Buffer, key string, value string) {
    fmt.Fprintf(b, "%s: %s\n", key, value)
}

func writeFlowOptions(b *bytes.Buffer, options []flowOption) {
    for _, option := range options {
        fmt.Fprintf(b, "  - slug: %s\n", option.Slug)
        writeConfigLine(b, "    help", option.Help)
    }
}
