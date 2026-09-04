// Package cli wires command-line arguments to the TUI program. It exists so the
// entrypoint stays a one-liner and the wiring is testable without a terminal.
package cli

import (
	"flag"
	"fmt"
	"io"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"

	"github.com/leekli/tamagotchi-go/internal/pet"
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

	initial, store := loadPet(stderr)

	app := tui.NewApp(ScreenFactories(initial, store), tui.WelcomeScreenID)
	if err := start(app); err != nil {
		fmt.Fprintf(stderr, "tamagotchi-go: %v\n", err)
		return 1
	}
	return 0
}

// loadPet resolves the save file and loads the persisted Pet, falling back
// to a fresh Egg when there is no save file yet or it can't be read. This is
// the one synchronous, startup-time load: everything downstream is either
// pure in-memory logic or deferred tea.Cmd I/O, never direct I/O from a
// Screen's Update.
func loadPet(stderr io.Writer) (pet.Pet, pet.Store) {
	path, err := pet.DefaultSavePath()
	if err != nil {
		fmt.Fprintf(stderr, "tamagotchi-go: resolving save path: %v\n", err)
	}
	store := pet.NewFileStore(path)

	loaded, ok, err := store.Load()
	if err != nil {
		fmt.Fprintf(stderr, "tamagotchi-go: loading save file: %v\n", err)
	}
	if !ok {
		return pet.New(time.Now()), store
	}
	return loaded, store
}

func runProgram(app *tui.App) error {
	_, err := tea.NewProgram(app, tea.WithAltScreen(), tea.WithMouseCellMotion()).Run()
	return err
}

// ScreenFactories returns the production Screen wiring: every ScreenID
// mapped to the factory that builds it. The Next Screen's factory closes
// over the already-loaded Pet and Store, since ScreenFactory itself stays
// zero-arg.
//
// Known, acceptable limitation: because the closure captures initial once,
// re-navigating to the Next Screen would reset it to that captured value
// rather than resuming where the player left off. Nothing currently
// navigates away from the Next Screen (ADR-0003: replace semantics, no
// history stack; the Next Screen is presently a terminal destination), so
// this isn't a live bug — just a flag for whoever adds a route back to the
// Welcome Screen.
func ScreenFactories(initial pet.Pet, store pet.Store) map[tui.ScreenID]tui.ScreenFactory {
	return map[tui.ScreenID]tui.ScreenFactory{
		tui.WelcomeScreenID: welcome.New,
		tui.NextScreenID:    func() tui.Screen { return next.New(initial, store) },
	}
}
