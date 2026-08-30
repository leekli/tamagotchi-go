// Package cli wires command-line arguments to the TUI program. It exists so the
// entrypoint stays a one-liner and the wiring is testable without a terminal.
package cli

import (
	"flag"
	"fmt"
	"io"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"

	"github.com/leekli/tamagotchi-go/internal/tui"
	"github.com/leekli/tamagotchi-go/internal/tui/next"
	"github.com/leekli/tamagotchi-go/internal/tui/welcome"
)

// Build information, overridden via -ldflags at release time.
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

// starter runs a fully wired App to completion. It is a seam: production uses
// runProgram, tests substitute a stub so the parsing and option handling can be
// exercised without a terminal.
type starter func(*tui.App) error

// Run parses args, applies global options, and starts the program. It returns a
// process exit code and never calls os.Exit itself.
func Run(args []string, stdout, stderr io.Writer) int {
	return run(args, stdout, stderr, runProgram)
}

func run(args []string, stdout, stderr io.Writer, start starter) int {
	fs := flag.NewFlagSet("tamagotchi-go", flag.ContinueOnError)
	fs.SetOutput(stderr)
	showVersion := fs.Bool("version", false, "print version information and exit")
	noColor := fs.Bool("no-color", false, "disable colour output")

	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}

	if *showVersion {
		fmt.Fprintf(stdout, "tamagotchi-go %s (commit %s, built %s)\n", version, commit, date)
		return 0
	}

	if *noColor || termenv.EnvNoColor() {
		lipgloss.SetColorProfile(termenv.Ascii)
	}

	app := tui.NewApp(ScreenFactories(), tui.WelcomeScreenID)
	if err := start(app); err != nil {
		fmt.Fprintf(stderr, "tamagotchi-go: %v\n", err)
		return 1
	}
	return 0
}

func runProgram(app *tui.App) error {
	_, err := tea.NewProgram(app, tea.WithAltScreen(), tea.WithMouseCellMotion()).Run()
	return err
}

// ScreenFactories returns the production Screen wiring: every ScreenID mapped to
// the factory that builds it.
func ScreenFactories() map[tui.ScreenID]tui.ScreenFactory {
	return map[tui.ScreenID]tui.ScreenFactory{
		tui.WelcomeScreenID: welcome.New,
		tui.NextScreenID:    next.New,
	}
}
