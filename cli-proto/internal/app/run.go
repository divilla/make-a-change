package app

import (
	"context"
	"errors"
	"flag"
	"io"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

func Run(args []string) error {
	fs := flag.NewFlagSet("mch", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	backendURL := fs.String("backend-url", "", "backend URL")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return errors.New("mch does not accept subcommands")
	}

	repoRoot, err := resolveRepoRoot(context.Background())
	if err != nil {
		return err
	}
	cfgPath := configPath(repoRoot)
	cfg, _, err := loadConfig(cfgPath)
	if err != nil {
		return err
	}
	effective := cfg
	if strings.TrimSpace(*backendURL) != "" {
		effective.BackendURL = strings.TrimSpace(*backendURL)
	}
	api := newAPIClient(effective.BackendURL)
	projects, projectErr := api.ListProjects(context.Background())
	m := newModel(repoRoot, cfgPath, cfg, api, projects, projectErr, CommandCodexRunner{}, ExternalEditor{})
	_, err = tea.NewProgram(m).Run()
	return err
}
