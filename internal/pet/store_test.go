package pet_test

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/leekli/tamagotchi-go/internal/pet"
)

func TestFileStoreLoadMissingFileIsNotAnError(t *testing.T) {
	t.Parallel()

	store := pet.NewFileStore(filepath.Join(t.TempDir(), "save.json"))

	p, ok, err := store.Load()

	require.NoError(t, err)
	assert.False(t, ok)
	assert.Zero(t, p)
}

func TestFileStoreRoundTripsSaveAndLoad(t *testing.T) {
	t.Parallel()

	store := pet.NewFileStore(filepath.Join(t.TempDir(), "nested", "save.json"))
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	want := pet.New(now)

	require.NoError(t, store.Save(want))

	got, ok, err := store.Load()

	require.NoError(t, err)
	require.True(t, ok)
	assert.True(t, want.CreatedAt.Equal(got.CreatedAt))
	assert.True(t, want.LastSeenAt.Equal(got.LastSeenAt))
	assert.True(t, want.HappinessLastSeenAt.Equal(got.HappinessLastSeenAt))
	assert.True(t, want.LastCleanedAt.Equal(got.LastCleanedAt))
	assert.Equal(t, want.Hunger, got.Hunger)
	assert.Equal(t, want.Happiness, got.Happiness)
	assert.Equal(t, want.Weight, got.Weight)
}

// TestFileStoreRoundTripsHappinessProgress proves partial progress toward the
// next Happiness point survives a save and load, so quitting and relaunching
// never restarts a step that was part-way done.
func TestFileStoreRoundTripsHappinessProgress(t *testing.T) {
	t.Parallel()

	store := pet.NewFileStore(filepath.Join(t.TempDir(), "save.json"))
	born := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	// 4 minutes on: one whole Happiness point lost at 3:00, and the 1:00 since
	// is progress toward the next.
	want := pet.New(born).Advance(born.Add(4 * time.Minute))
	require.Equal(t, time.Minute, want.HappinessProgress, "sanity check on the setup")

	require.NoError(t, store.Save(want))
	got, ok, err := store.Load()

	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, want.HappinessProgress, got.HappinessProgress)
	assert.Equal(t, want.Happiness, got.Happiness)
}

// TestFileStoreRoundTripsCareMistakesAndEmptySpells proves the Care mistake
// tally and both Stats' Empty spells survive a save and load, in each phase: one
// spell lapsed and counted, the other still inside its grace window. It checks
// behaviour, not stored fields: the loaded Pet must report the same Attention
// calls and go on to count the same next mistake at the same instant.
func TestFileStoreRoundTripsCareMistakesAndEmptySpells(t *testing.T) {
	t.Parallel()

	store := pet.NewFileStore(filepath.Join(t.TempDir(), "save.json"))
	born := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	p := pet.New(born)
	p.LastCleanedAt = born.Add(24 * time.Hour) // no Mess, so both Stats decay normally
	p.Hunger = 1                               // Empty 3:00, counted at 7:00
	// Happiness starts full: Empty 12:00, window running until 16:00.

	now := born.Add(13 * time.Minute)
	want := p.Advance(now)
	require.Equal(t, 1, want.CareMistakes, "sanity check: Hunger's spell has lapsed")
	require.Equal(t, pet.AttentionWindowRunning, want.HappinessAttention(now), "sanity check: Happiness's is running")

	require.NoError(t, store.Save(want))
	got, ok, err := store.Load()

	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, want.CareMistakes, got.CareMistakes)
	assert.Equal(t, pet.AttentionLapsed, got.HungerAttention(now))
	assert.Equal(t, pet.AttentionWindowRunning, got.HappinessAttention(now))

	end := born.Add(16 * time.Minute)
	assert.Equal(t, 1, got.Advance(end.Add(-time.Nanosecond)).CareMistakes, "Happiness's window has not yet expired")
	assert.Equal(t, 2, got.Advance(end).CareMistakes, "it expires at the same instant it would have without the save")
	assert.Equal(t, want.Advance(end).CareMistakes, got.Advance(end).CareMistakes)
}

// TestFileStoreRoundTripsSickness proves Sickness survives a save and load, and
// so does the restarted sickness clock of a Cured Pet. It checks behaviour: the
// loaded Pet must be Sick when it was, stay Sick through a Clean, and fall Sick
// again at the same instant it would have without the save.
func TestFileStoreRoundTripsSickness(t *testing.T) {
	t.Parallel()

	born := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	t.Run("a Sick Pet is still Sick after loading, and Clean still does not cure it", func(t *testing.T) {
		t.Parallel()
		store := pet.NewFileStore(filepath.Join(t.TempDir(), "save.json"))
		at := born.Add(11 * time.Minute)
		sick := pet.New(born).Advance(at)
		require.True(t, sick.Sick(at), "sanity check")

		require.NoError(t, store.Save(sick))
		got, ok, err := store.Load()

		require.NoError(t, err)
		require.True(t, ok)
		assert.True(t, got.Sick(at))
		assert.True(t, got.Clean(at).Sick(at), "the loaded Pet is Sick from a stored record, not just from its Mess")
	})

	t.Run("a Cured Pet keeps its restarted sickness clock", func(t *testing.T) {
		t.Parallel()
		store := pet.NewFileStore(filepath.Join(t.TempDir(), "save.json"))
		curedAt := born.Add(12 * time.Minute)
		cured := pet.New(born).Advance(curedAt).Cure(curedAt)
		require.False(t, cured.Sick(curedAt), "sanity check")

		require.NoError(t, store.Save(cured))
		got, ok, err := store.Load()

		require.NoError(t, err)
		require.True(t, ok)
		sickAgain := curedAt.Add(pet.SickAfterMess)
		assert.False(t, got.Sick(sickAgain.Add(-time.Nanosecond)))
		assert.True(t, got.Sick(sickAgain), "Sick again exactly a wait after the Cure, as without the save")
	})
}

// TestFileStoreLoadOldSaveHasNoSickness: a save from before Sickness existed
// loads as not Sick and never Cured, so a Pet whose Mess is still young is not
// suddenly Sick.
func TestFileStoreLoadOldSaveHasNoSickness(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "save.json")
	createdAt := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	body := `{
		"schema_version": 1,
		"created_at": "` + createdAt.Format(time.RFC3339Nano) + `",
		"last_seen_at": "` + createdAt.Format(time.RFC3339Nano) + `",
		"hunger": 3,
		"happiness": 3,
		"weight": 2
	}`
	require.NoError(t, os.WriteFile(path, []byte(body), 0o600))

	got, ok, err := pet.NewFileStore(path).Load()

	require.NoError(t, err)
	require.True(t, ok)
	assert.True(t, got.SickSince.IsZero())
	assert.True(t, got.LastCuredAt.IsZero())
	assert.False(t, got.Sick(createdAt.Add(9*time.Minute)), "not Sick before the usual 10:00")
	assert.True(t, got.Sick(createdAt.Add(10*time.Minute)), "and Sick at it, as for any Pet")
}

// TestFileStoreLoadOldSaveWithNoEmptyStatsHasNoTally: a save from before Care
// mistakes existed, with both Stats above 0, loads with nothing owed.
func TestFileStoreLoadOldSaveWithNoEmptyStatsHasNoTally(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "save.json")
	createdAt := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	body := `{
		"schema_version": 1,
		"created_at": "` + createdAt.Format(time.RFC3339Nano) + `",
		"last_seen_at": "` + createdAt.Format(time.RFC3339Nano) + `",
		"hunger": 3,
		"happiness": 2,
		"weight": 2
	}`
	require.NoError(t, os.WriteFile(path, []byte(body), 0o600))

	got, ok, err := pet.NewFileStore(path).Load()

	require.NoError(t, err)
	require.True(t, ok)
	assert.Zero(t, got.CareMistakes)
	assert.Equal(t, pet.AttentionNone, got.HungerAttention(createdAt))
	assert.Equal(t, pet.AttentionNone, got.HappinessAttention(createdAt))
}

// TestFileStoreLoadOldSaveWithAnEmptyStatGetsAFreshGraceWindow: a save from
// before spells were recorded can have a Stat at 0 with no record of when it
// emptied. Loading it and catching up must give a fresh window from that first
// Advance, never counting a mistake for time before the upgrade.
func TestFileStoreLoadOldSaveWithAnEmptyStatGetsAFreshGraceWindow(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "save.json")
	createdAt := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	lastSeen := createdAt.Add(time.Hour)
	body := `{
		"schema_version": 1,
		"created_at": "` + createdAt.Format(time.RFC3339Nano) + `",
		"last_seen_at": "` + lastSeen.Format(time.RFC3339Nano) + `",
		"last_cleaned_at": "` + lastSeen.Format(time.RFC3339Nano) + `",
		"hunger": 0,
		"happiness": 4,
		"weight": 2
	}`
	require.NoError(t, os.WriteFile(path, []byte(body), 0o600))

	got, ok, err := pet.NewFileStore(path).Load()
	require.NoError(t, err)
	require.True(t, ok)
	require.Zero(t, got.CareMistakes)

	launchedAt := lastSeen.Add(time.Minute) // the first Advance after the upgrade
	caughtUp := got.Advance(launchedAt)

	assert.Zero(t, caughtUp.CareMistakes, "Hunger was at 0 before the upgrade, but that time is not held against it")
	assert.Equal(t, pet.AttentionWindowRunning, caughtUp.HungerAttention(launchedAt))
	assert.Zero(t, caughtUp.Advance(launchedAt.Add(pet.GraceWindow-time.Nanosecond)).CareMistakes)
	assert.Equal(t, 1, caughtUp.Advance(launchedAt.Add(pet.GraceWindow)).CareMistakes,
		"the window is a full GraceWindow from the first Advance")
}

// TestFileStoreLoadClampsNegativeCareMistakes: a corrupted negative tally is
// treated as none, so it cannot cancel out mistakes still to be counted.
func TestFileStoreLoadClampsNegativeCareMistakes(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "save.json")
	createdAt := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	body := `{
		"schema_version": 1,
		"created_at": "` + createdAt.Format(time.RFC3339Nano) + `",
		"last_seen_at": "` + createdAt.Format(time.RFC3339Nano) + `",
		"care_mistakes": -7,
		"hunger": 4,
		"happiness": 4,
		"weight": 2
	}`
	require.NoError(t, os.WriteFile(path, []byte(body), 0o600))

	got, ok, err := pet.NewFileStore(path).Load()

	require.NoError(t, err)
	require.True(t, ok)
	assert.Zero(t, got.CareMistakes)
}

// TestFileStoreLoadClampsNegativeHappinessProgress proves a corrupted negative
// progress value is treated as none. Left alone, a value below minus one
// interval (here minus 400s against a 180s interval) would make Advance hand
// Happiness points back without any time passing.
func TestFileStoreLoadClampsNegativeHappinessProgress(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "save.json")
	createdAt := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	body := `{
		"schema_version": 1,
		"created_at": "` + createdAt.Format(time.RFC3339Nano) + `",
		"last_seen_at": "` + createdAt.Format(time.RFC3339Nano) + `",
		"happiness_progress_ns": -400000000000,
		"hunger": 4,
		"happiness": 2,
		"weight": 2
	}`
	require.NoError(t, os.WriteFile(path, []byte(body), 0o600))

	got, ok, err := pet.NewFileStore(path).Load()

	require.NoError(t, err)
	require.True(t, ok)
	assert.Zero(t, got.HappinessProgress)
	assert.LessOrEqual(t, got.Advance(createdAt).Happiness, 2,
		"a negative progress value must never raise Happiness")
}

// TestFileStoreLoadDefaultsCareActionFieldsForOldSaveFiles proves a save file
// written before Care actions existed (so its JSON has neither
// last_cleaned_at nor happiness_last_seen_at) loads sanely: LastCleanedAt
// defaults to CreatedAt ("never cleaned since birth", not freshly cleaned at
// the zero time.Time), and HappinessLastSeenAt defaults to LastSeenAt (exactly
// correct, since every such save always advanced Hunger and Happiness decay
// in lockstep).
func TestFileStoreLoadDefaultsCareActionFieldsForOldSaveFiles(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "save.json")
	createdAt := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	lastSeenAt := createdAt.Add(90 * time.Second)
	body := `{
		"schema_version": 1,
		"created_at": "` + createdAt.Format(time.RFC3339Nano) + `",
		"last_seen_at": "` + lastSeenAt.Format(time.RFC3339Nano) + `",
		"hunger": 4,
		"happiness": 4,
		"weight": 2
	}`
	require.NoError(t, os.WriteFile(path, []byte(body), 0o600))

	got, ok, err := pet.NewFileStore(path).Load()

	require.NoError(t, err)
	require.True(t, ok)
	assert.True(t, createdAt.Equal(got.LastCleanedAt),
		"a pre-Mess save should default LastCleanedAt to CreatedAt, not the zero time")
	assert.True(t, got.HasMess(createdAt.Add(pet.MessInterval)),
		"defaulting to CreatedAt should read as messy after MessInterval has passed since birth, not freshly cleaned")
	assert.True(t, lastSeenAt.Equal(got.HappinessLastSeenAt),
		"a pre-Care-actions save should default HappinessLastSeenAt to LastSeenAt")
	assert.Zero(t, got.HappinessProgress,
		"a save written before progress was tracked reads as no partial progress yet")
}

// TestFileStoreLoadClampsWeightOutOfRangeFromOldSaveFiles proves a save file
// written before MaxWeight existed (when Snack's Weight gain was uncapped)
// is corrected to a valid Weight immediately on load, rather than being left
// out of range until the next Feed(Snack) silently snaps it down.
func TestFileStoreLoadClampsWeightOutOfRangeFromOldSaveFiles(t *testing.T) {
	t.Parallel()

	createdAt := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	body := func(weight int) string {
		return `{
			"schema_version": 1,
			"created_at": "` + createdAt.Format(time.RFC3339Nano) + `",
			"last_seen_at": "` + createdAt.Format(time.RFC3339Nano) + `",
			"hunger": 4,
			"happiness": 4,
			"weight": ` + strconv.Itoa(weight) + `
		}`
	}

	t.Run("above MaxWeight", func(t *testing.T) {
		t.Parallel()
		path := filepath.Join(t.TempDir(), "save.json")
		require.NoError(t, os.WriteFile(path, []byte(body(50)), 0o600))

		got, ok, err := pet.NewFileStore(path).Load()

		require.NoError(t, err)
		require.True(t, ok)
		assert.Equal(t, pet.MaxWeight, got.Weight)
	})

	t.Run("below BaseWeight", func(t *testing.T) {
		t.Parallel()
		path := filepath.Join(t.TempDir(), "save.json")
		require.NoError(t, os.WriteFile(path, []byte(body(-3)), 0o600))

		got, ok, err := pet.NewFileStore(path).Load()

		require.NoError(t, err)
		require.True(t, ok)
		assert.Equal(t, pet.BaseWeight, got.Weight)
	})
}

func TestFileStoreSaveCreatesMissingParentDirectory(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "a", "b", "c", "save.json")
	store := pet.NewFileStore(path)

	require.NoError(t, store.Save(pet.New(time.Now())))
	assert.FileExists(t, path)
}

func TestFileStoreSecondSaveOverwritesCleanly(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "save.json")
	store := pet.NewFileStore(path)
	now := time.Now()

	require.NoError(t, store.Save(pet.New(now)))
	require.NoError(t, store.Save(pet.New(now).Advance(now.Add(time.Hour))))

	got, ok, err := store.Load()
	require.NoError(t, err)
	require.True(t, ok)
	assert.True(t, now.Add(time.Hour).Equal(got.LastSeenAt))

	// No stale temp files left behind alongside the save.
	entries, err := os.ReadDir(filepath.Dir(path))
	require.NoError(t, err)
	assert.Len(t, entries, 1)
}

func TestFileStoreLoadCorruptFileReturnsError(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "save.json")
	require.NoError(t, os.WriteFile(path, []byte("{not json"), 0o600))

	store := pet.NewFileStore(path)
	p, ok, err := store.Load()

	assert.Error(t, err)
	assert.False(t, ok)
	assert.Zero(t, p)
}

func TestFileStoreLoadWrongSchemaVersionReturnsError(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "save.json")
	require.NoError(t, os.WriteFile(path, []byte(`{"schema_version":99}`), 0o600))

	store := pet.NewFileStore(path)
	_, ok, err := store.Load()

	assert.Error(t, err)
	assert.False(t, ok)
}

// TestFileStoreLoadUnreadableFileReturnsError covers Load's other read-error
// branch: not "the file doesn't exist" (already covered above) but "the file
// exists and can't be read" — permission denied, in this case. Load must wrap
// and return the error rather than panicking or silently treating it as a
// missing file.
func TestFileStoreLoadUnreadableFileReturnsError(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("chmod-based permission checks don't apply when running as root")
	}
	t.Parallel()

	path := filepath.Join(t.TempDir(), "save.json")
	require.NoError(t, os.WriteFile(path, []byte(`{"schema_version":1}`), 0o600))
	require.NoError(t, os.Chmod(path, 0o000))
	t.Cleanup(func() { _ = os.Chmod(path, 0o600) }) // so t.TempDir() can remove it afterwards

	p, ok, err := pet.NewFileStore(path).Load()

	assert.Error(t, err)
	assert.False(t, ok)
	assert.Zero(t, p)
}

// TestFileStoreSaveFailsWhenAParentPathComponentIsAFile exercises Save's
// os.MkdirAll error branch: the save path's parent can't be created because
// something already occupies that name and it isn't a directory.
func TestFileStoreSaveFailsWhenAParentPathComponentIsAFile(t *testing.T) {
	t.Parallel()

	blocker := filepath.Join(t.TempDir(), "blocker")
	require.NoError(t, os.WriteFile(blocker, []byte("not a directory"), 0o600))

	err := pet.NewFileStore(filepath.Join(blocker, "save.json")).Save(pet.New(time.Now()))

	assert.Error(t, err)
}

// TestFileStoreSaveFailsWhenDirectoryIsNotWritable exercises Save's
// os.CreateTemp error branch: the parent directory already exists (so
// MkdirAll is a no-op) but has no write permission, so the temp file can't be
// created.
func TestFileStoreSaveFailsWhenDirectoryIsNotWritable(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("chmod-based permission checks don't apply when running as root")
	}
	t.Parallel()

	dir := t.TempDir()
	require.NoError(t, os.Chmod(dir, 0o500))
	t.Cleanup(func() { _ = os.Chmod(dir, 0o700) }) // so t.TempDir() can remove it afterwards

	err := pet.NewFileStore(filepath.Join(dir, "save.json")).Save(pet.New(time.Now()))

	assert.Error(t, err)
}

// TestFileStoreSaveFailsWhenTargetPathIsADirectory exercises Save's
// os.Rename error branch: the temp file is written successfully, but the
// final rename fails because the target path is already occupied by a
// directory rather than a file.
func TestFileStoreSaveFailsWhenTargetPathIsADirectory(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "save.json")
	require.NoError(t, os.Mkdir(path, 0o750))

	err := pet.NewFileStore(path).Save(pet.New(time.Now()))

	assert.Error(t, err)
}

// failingStore is a pet.Store whose Save always fails, used to prove
// SaveCmd's documented contract: a failed save is discarded, not surfaced.
type failingStore struct{ err error }

func (failingStore) Load() (pet.Pet, bool, error) { return pet.Pet{}, false, nil }

func (f failingStore) Save(pet.Pet) error { return f.err }

// TestSaveCmdDiscardsAFailedSave proves SaveCmd's Cmd still returns its
// documented nil Msg, and does not panic, when the underlying Store fails —
// a failed save must not be fatal to a running game.
func TestSaveCmdDiscardsAFailedSave(t *testing.T) {
	t.Parallel()

	cmd := pet.SaveCmd(failingStore{err: errors.New("disk full")}, pet.New(time.Now()))

	assert.NotPanics(t, func() {
		assert.Nil(t, cmd(), "SaveCmd's message is nil regardless of whether the save succeeded")
	})
}

// FuzzFileStoreLoadNeverPanics hardens Load's JSON parsing beyond the two
// handwritten malformed inputs above: for any bytes written to the save
// file, Load must never panic, and must never report both ok and a non-nil
// error at once.
func FuzzFileStoreLoadNeverPanics(f *testing.F) {
	f.Add([]byte(`{"schema_version":1,"created_at":"2026-01-01T00:00:00Z","last_seen_at":"2026-01-01T00:00:00Z","hunger":4,"happiness":4,"weight":2}`))
	f.Add([]byte("{not json"))
	f.Add([]byte(`{"schema_version":99}`))
	f.Add([]byte(""))
	f.Add([]byte(`null`))
	f.Add([]byte(`{"schema_version":1,"hunger":-999999999999}`))

	f.Fuzz(func(t *testing.T, data []byte) {
		path := filepath.Join(t.TempDir(), "save.json")
		require.NoError(t, os.WriteFile(path, data, 0o600))

		_, ok, err := pet.NewFileStore(path).Load()

		if ok {
			assert.NoError(t, err, "Load must never report both ok and an error")
		}
	})
}

func TestDefaultSavePathIncludesAppDirectory(t *testing.T) {
	t.Parallel()

	path, err := pet.DefaultSavePath()

	require.NoError(t, err)
	assert.Contains(t, path, filepath.Join("tamagotchi-go", "save.json"))
}

func TestSaveCmdPersistsThroughTheStore(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "save.json")
	store := pet.NewFileStore(path)
	p := pet.New(time.Now())

	cmd := pet.SaveCmd(store, p)
	require.NotNil(t, cmd)
	assert.Nil(t, cmd(), "SaveCmd's message is deliberately nil: nothing needs to react to a completed save")

	_, ok, err := store.Load()
	require.NoError(t, err)
	assert.True(t, ok, "SaveCmd should have persisted the Pet")
}
