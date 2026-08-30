package cli

import (
	"bytes"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/leekli/tamagotchi-go/internal/tui"
)

func noopStarter(*tui.App) error { return nil }

func TestRunVersionPrintsAndExitsZero(t *testing.T) {
	var out, errOut bytes.Buffer

	code := run([]string{"--version"}, &out, &errOut, func(*tui.App) error {
		t.Fatal("program should not start when --version is given")
		return nil
	})

	assert.Equal(t, 0, code)
	assert.Contains(t, out.String(), "tamagotchi-go dev (commit none, built unknown)")
	assert.Empty(t, errOut.String())
}

func TestRunUnknownFlagExitsTwo(t *testing.T) {
	var out, errOut bytes.Buffer

	code := run([]string{"--wat"}, &out, &errOut, noopStarter)

	assert.Equal(t, 2, code)
	assert.Contains(t, errOut.String(), "flag provided but not defined")
}

func TestRunHelpExitsZero(t *testing.T) {
	var out, errOut bytes.Buffer

	code := run([]string{"-h"}, &out, &errOut, noopStarter)

	assert.Equal(t, 0, code)
	assert.Contains(t, errOut.String(), "-no-color")
}

func TestRunStartsProgramAndReportsSuccess(t *testing.T) {
	var out, errOut bytes.Buffer
	started := false

	code := run(nil, &out, &errOut, func(app *tui.App) error {
		started = true
		require.NotNil(t, app)
		assert.Equal(t, tui.WelcomeScreenID, app.Current().ID())
		return nil
	})

	assert.Equal(t, 0, code)
	assert.True(t, started)
	assert.Empty(t, errOut.String())
}

func TestRunReportsProgramFailure(t *testing.T) {
	var out, errOut bytes.Buffer

	code := run([]string{"--no-color"}, &out, &errOut, func(*tui.App) error {
		return errors.New("boom")
	})

	assert.Equal(t, 1, code)
	assert.Contains(t, errOut.String(), "tamagotchi-go: boom")
}

func TestRunExportedEntrypointDelegates(t *testing.T) {
	// The exported Run wires the real program starter; --version returns before
	// it is ever called, so this exercises Run without needing a terminal.
	var out, errOut bytes.Buffer

	code := Run([]string{"--version"}, &out, &errOut)

	assert.Equal(t, 0, code)
	assert.Contains(t, out.String(), "tamagotchi-go")
}

func TestScreenFactoriesCoverEveryScreenID(t *testing.T) {
	factories := ScreenFactories()

	for _, id := range tui.AllScreenIDs() {
		factory, ok := factories[id]
		require.Truef(t, ok, "no factory registered for ScreenID %q", id)

		screen := factory()
		require.NotNil(t, screen)
		assert.Equalf(t, id, screen.ID(), "factory for %q built a Screen reporting ID %q", id, screen.ID())
	}

	assert.Len(t, factories, len(tui.AllScreenIDs()), "factory map has entries with no matching ScreenID")
}
