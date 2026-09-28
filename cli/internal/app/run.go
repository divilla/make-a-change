package app

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	httpclient "cli/pkg/client"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

// Version is the user-visible mch executable version.
const Version = "0.1"

// Run executes the mch command with the supplied process arguments and output writer.
func Run(args []string, out io.Writer) error {
	return RunWithIO(args, os.Stdin, out)
}

// ProgramOptions supplies controlled dependencies at the complete-program boundary.
type ProgramOptions struct {
	Context        context.Context
	RepositoryRoot string
	ProgramReady   func(ProgramController)
}

// ProgramController exposes orderly external program shutdown for controlled callers.
type ProgramController interface {
	Quit()
}

// RunWithIO executes the complete CLI program with controlled input and output.
func RunWithIO(args []string, in io.Reader, out io.Writer) error {
	return RunProgramWithIO(args, in, out, ProgramOptions{})
}

// RunProgramWithIO executes the complete CLI program with optional controlled dependencies.
func RunProgramWithIO(args []string, in io.Reader, out io.Writer, options ProgramOptions) error {
	fs := flag.NewFlagSet("mch", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	showVersion := fs.Bool("version", false, "print version")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return errors.New("mch does not accept subcommands")
	}
	if *showVersion {
		_, err := fmt.Fprintf(out, "mch %s\n", Version)
		return err
	}

	var cfg appConfig
	var err error
	if options.RepositoryRoot == "" {
		cfg, err = loadRepositoryConfig()
	} else {
		cfg, err = loadAppConfig(options.RepositoryRoot)
	}
	if err != nil {
		return fmt.Errorf("failed to load repository configuration: %w", err)
	}
	ctx := options.Context
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	model := newModelWithConfig(httpclient.NewHTTPClient(cfg.BackendURL), cfg)
	model.ctx = ctx
	lipgloss.SetColorProfile(termenv.ANSI256)
	programOptions := []tea.ProgramOption{
		tea.WithInput(in),
		tea.WithOutput(out),
		tea.WithMouseCellMotion(),
	}
	programOptions = append(programOptions, tea.WithContext(ctx))
	program := tea.NewProgram(model, programOptions...)
	if options.ProgramReady != nil {
		options.ProgramReady(program)
	}
	_, err = program.Run()
	return err
}
