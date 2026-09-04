package cli

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/leekli/tamagotchi-go/internal/pet"
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
	assert.Contains(t, errOut.String(), "-save-path")
}

func TestRunStartsProgramAndReportsSuccess(t *testing.T) {
	savePath := filepath.Join(t.TempDir(), "save.json")
	var out, errOut bytes.Buffer
	started := false

	code := run([]string{"--save-path", savePath}, &out, &errOut, func(app *tui.App) error {
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
	savePath := filepath.Join(t.TempDir(), "save.json")
	var out, errOut bytes.Buffer

	code := run([]string{"--no-color", "--save-path", savePath}, &out, &errOut, func(*tui.App) error {
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
	store := pet.NewFileStore(filepath.Join(t.TempDir(), "save.json"))
	factories := ScreenFactories(pet.New(time.Now()), store)

	for _, id := range tui.AllScreenIDs() {
		factory, ok := factories[id]
		require.Truef(t, ok, "no factory registered for ScreenID %q", id)

		screen := factory()
		require.NotNil(t, screen)
		assert.Equalf(t, id, screen.ID(), "factory for %q built a Screen reporting ID %q", id, screen.ID())
	}

	assert.Len(t, factories, len(tui.AllScreenIDs()), "factory map has entries with no matching ScreenID")
}

func TestLoadPetStartsAFreshEggWhenNoSaveFileExists(t *testing.T) {
	var errOut bytes.Buffer
	path := filepath.Join(t.TempDir(), "save.json")

	p, store := loadPet(&errOut, path)

	assert.WithinDuration(t, time.Now(), p.CreatedAt, time.Second)
	assert.Equal(t, pet.MaxStat, p.Hunger)
	assert.Equal(t, pet.MaxStat, p.Happiness)
	assert.Empty(t, errOut.String(), "the first-run case is not an error")
	assert.Equal(t, pet.NewFileStore(path), store)
}

func TestLoadPetAppliesOfflineCatchUpOnSuccessfulLoad(t *testing.T) {
	var errOut bytes.Buffer
	path := filepath.Join(t.TempDir(), "save.json")
	store := pet.NewFileStore(path)

	// One decay step's worth of real elapsed time since the game last ran.
	born := time.Now().Add(-4 * time.Minute)
	require.NoError(t, store.Save(pet.New(born)))

	p, _ := loadPet(&errOut, path)

	assert.Equal(t, pet.MaxStat-1, p.Hunger, "offline decay should be applied immediately on load")
	assert.Equal(t, pet.MaxStat-1, p.Happiness)
	assert.Empty(t, errOut.String())
}

func TestLoadPetFallsBackToFreshEggOnCorruptSaveFile(t *testing.T) {
	var errOut bytes.Buffer
	path := filepath.Join(t.TempDir(), "save.json")
	require.NoError(t, os.WriteFile(path, []byte("{not json"), 0o600))

	p, _ := loadPet(&errOut, path)

	assert.Equal(t, pet.MaxStat, p.Hunger, "a corrupt save should start a fresh Egg instead of crashing")
	assert.NotEmpty(t, errOut.String(), "a corrupt save should log a one-line notice")
}

func TestLoadPetUsesDefaultSavePathWhenNoOverrideGiven(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	var errOut bytes.Buffer

	_, store := loadPet(&errOut, "")

	want, err := pet.DefaultSavePath()
	require.NoError(t, err)
	assert.Equal(t, pet.NewFileStore(want), store)
}
