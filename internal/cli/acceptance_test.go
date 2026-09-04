package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/leekli/tamagotchi-go/internal/anim"
	"github.com/leekli/tamagotchi-go/internal/pet"
	"github.com/leekli/tamagotchi-go/internal/tui"
	"github.com/leekli/tamagotchi-go/internal/tui/next"
)

// These tests map one given/when/then group to each user-facing requirement
// of the Pet domain model & Next Screen content plan, mirroring
// internal/tui/welcome/acceptance_test.go's style.

func TestAcceptance_FreshPetOnFirstLaunch(t *testing.T) {
	t.Run("given no save file when the game launches then the Pet starts as a fresh Egg", func(t *testing.T) {
		var errOut bytes.Buffer
		path := filepath.Join(t.TempDir(), "save.json")

		p, _ := loadPet(&errOut, path)

		assert.Equal(t, pet.StageEgg, p.Stage(time.Now()))
		assert.Equal(t, pet.MaxStat, p.Hunger)
		assert.Equal(t, pet.MaxStat, p.Happiness)
	})
}

func TestAcceptance_OfflineDecayOnLaunch(t *testing.T) {
	t.Run("given a save file with elapsed offline time when the game launches then Hunger and Happiness reflect the offline decay", func(t *testing.T) {
		var errOut bytes.Buffer
		path := filepath.Join(t.TempDir(), "save.json")
		store := pet.NewFileStore(path)
		require.NoError(t, store.Save(pet.New(time.Now().Add(-4*time.Minute))))

		p, _ := loadPet(&errOut, path)

		assert.Less(t, p.Hunger, pet.MaxStat)
		assert.Less(t, p.Happiness, pet.MaxStat)
	})
}

func TestAcceptance_CorruptSaveRecovery(t *testing.T) {
	t.Run("given a corrupt save file when the game launches then it starts a fresh Egg instead of crashing", func(t *testing.T) {
		var errOut bytes.Buffer
		path := filepath.Join(t.TempDir(), "save.json")
		require.NoError(t, os.WriteFile(path, []byte("not json"), 0o600))

		p, _ := loadPet(&errOut, path)

		assert.Equal(t, pet.MaxStat, p.Hunger)
		assert.NotEmpty(t, errOut.String(), "a corrupt save should log a notice rather than fail silently")
	})
}

func TestAcceptance_HatchesAfterEggDuration(t *testing.T) {
	t.Run("given the Pet is an Egg when EggDuration has elapsed then it has hatched into a Baby", func(t *testing.T) {
		born := time.Now()
		store := pet.NewFileStore(filepath.Join(t.TempDir(), "save.json"))
		screen, _ := next.New(pet.New(born), store).Update(tea.WindowSizeMsg{Width: 80, Height: 23})

		require.Equal(t, pet.StageEgg, pet.New(born).Stage(born))

		// Not real time: EggDuration has elapsed relative to born, fed through
		// the same anim.TickMsg the Screen tracks "now" from — no sleeping.
		screen, _ = screen.Update(anim.TickMsg{Time: born.Add(pet.EggDuration)})

		ns, ok := screen.(*next.Screen)
		require.True(t, ok)
		assert.Equal(t, pet.StageBaby, ns.Pet().Stage(born.Add(pet.EggDuration)))
	})
}

func TestAcceptance_QuitSavesTheCurrentPet(t *testing.T) {
	t.Run("given the game is running when the player quits then the current Pet state is saved", func(t *testing.T) {
		store := pet.NewFileStore(filepath.Join(t.TempDir(), "save.json"))
		screen := next.New(pet.New(time.Now()), store)

		qh, ok := screen.(tui.QuitHandler)
		require.True(t, ok, "the Next Screen should implement the save-on-quit hook")
		cmd := qh.OnQuit()
		require.NotNil(t, cmd)
		cmd()

		_, ok, err := store.Load()
		require.NoError(t, err)
		assert.True(t, ok, "the Pet should have been saved on quit")
	})
}
