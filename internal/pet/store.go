package pet

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// schemaVersion is the current save file schema. A future field addition
// (health, mess, evolution stage, …) bumps this and migrates old files
// rather than failing to load them.
const schemaVersion = 1

// Store loads and saves the single persisted Pet.
type Store interface {
	// Load returns the persisted Pet. ok is false with a nil error when no
	// save file exists yet — the normal first-run case, not an error.
	Load() (Pet, bool, error)
	// Save persists p.
	Save(Pet) error
}

// FileStore is the production Store: a single JSON file on disk.
type FileStore struct {
	Path string
}

// NewFileStore builds a FileStore at path.
func NewFileStore(path string) FileStore {
	return FileStore{Path: path}
}

// DefaultSavePath returns the save file's default location: the user's
// config directory, under "tamagotchi-go/save.json".
func DefaultSavePath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("pet: resolving config directory: %w", err)
	}
	return filepath.Join(dir, "tamagotchi-go", "save.json"), nil
}

// saveFile is the on-disk JSON shape.
type saveFile struct {
	SchemaVersion int       `json:"schema_version"`
	CreatedAt     time.Time `json:"created_at"`
	LastSeenAt    time.Time `json:"last_seen_at"`
	Hunger        int       `json:"hunger"`
	Happiness     int       `json:"happiness"`
	Weight        int       `json:"weight"`
}

// Load implements Store. A missing file is the normal first-run case, not an
// error. A file that exists but is unreadable or fails to parse (corrupt,
// truncated, wrong schema version) returns a descriptive error and never
// panics — the caller treats any such error the same as "no save file".
func (f FileStore) Load() (Pet, bool, error) {
	b, err := os.ReadFile(f.Path)
	if errors.Is(err, os.ErrNotExist) {
		return Pet{}, false, nil
	}
	if err != nil {
		return Pet{}, false, fmt.Errorf("pet: reading save file: %w", err)
	}

	var sf saveFile
	if err := json.Unmarshal(b, &sf); err != nil {
		return Pet{}, false, fmt.Errorf("pet: parsing save file: %w", err)
	}
	if sf.SchemaVersion != schemaVersion {
		return Pet{}, false, fmt.Errorf("pet: save file has schema version %d, want %d", sf.SchemaVersion, schemaVersion)
	}

	return Pet{
		CreatedAt:  sf.CreatedAt,
		LastSeenAt: sf.LastSeenAt,
		Hunger:     sf.Hunger,
		Happiness:  sf.Happiness,
		Weight:     sf.Weight,
	}, true, nil
}

// Save implements Store. It writes atomically enough not to corrupt the file
// on a crash mid-write: write to a temp file in the same directory, then
// rename over the target. The parent directory is created if it doesn't
// exist yet.
func (f FileStore) Save(p Pet) error {
	dir := filepath.Dir(f.Path)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return fmt.Errorf("pet: creating save directory: %w", err)
	}

	b, err := json.Marshal(saveFile{
		SchemaVersion: schemaVersion,
		CreatedAt:     p.CreatedAt,
		LastSeenAt:    p.LastSeenAt,
		Hunger:        p.Hunger,
		Happiness:     p.Happiness,
		Weight:        p.Weight,
	})
	if err != nil {
		return fmt.Errorf("pet: encoding save file: %w", err)
	}

	tmp, err := os.CreateTemp(dir, "save-*.json.tmp")
	if err != nil {
		return fmt.Errorf("pet: creating temp save file: %w", err)
	}
	defer func() { _ = os.Remove(tmp.Name()) }() // no-op once the rename below succeeds

	if _, err := tmp.Write(b); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("pet: writing temp save file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("pet: closing temp save file: %w", err)
	}

	if err := os.Rename(tmp.Name(), f.Path); err != nil {
		return fmt.Errorf("pet: replacing save file: %w", err)
	}
	return nil
}

// SaveCmd returns a tea.Cmd that persists p via store, off the caller's
// Update call stack — the established Bubble Tea pattern for deferred I/O.
// A failed save is not fatal to a running game, so the error is discarded
// rather than surfaced as a message.
func SaveCmd(store Store, p Pet) tea.Cmd {
	return func() tea.Msg {
		_ = store.Save(p)
		return nil
	}
}
