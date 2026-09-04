package pet_test

import (
	"os"
	"path/filepath"
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
	assert.Equal(t, want.Hunger, got.Hunger)
	assert.Equal(t, want.Happiness, got.Happiness)
	assert.Equal(t, want.Weight, got.Weight)
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
