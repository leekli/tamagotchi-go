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
	// HappinessLastSeenAt and LastCleanedAt are absent from any save file
	// written before Care actions existed; their zero time.Time unmarshals
	// fine and is corrected by withLoadDefaults below, so no schema_version
	// bump is needed for either field.
	HappinessLastSeenAt time.Time `json:"happiness_last_seen_at,omitzero"`
	LastCleanedAt       time.Time `json:"last_cleaned_at,omitzero"`
	// HappinessProgress is absent from any save file written before Decay was
	// made exact; its zero value reads as "no partial progress yet", so no
	// schema_version bump is needed for it either.
	HappinessProgress time.Duration `json:"happiness_progress_ns,omitzero"`
	// The Care mistake tally and each Stat's Empty spell are likewise absent from
	// any save file written before neglect was tracked. A missing tally reads as
	// none owed, and a Stat at 0 with no recorded spell gets a fresh grace window
	// from the first Advance (see Pet.startUnrecordedSpells), so an upgrade never
	// counts time from before it. No schema_version bump is needed.
	CareMistakes         int       `json:"care_mistakes,omitzero"`
	HungerEmptySince     time.Time `json:"hunger_empty_since,omitzero"`
	HungerGraceEndsAt    time.Time `json:"hunger_grace_ends_at,omitzero"`
	HappinessEmptySince  time.Time `json:"happiness_empty_since,omitzero"`
	HappinessGraceEndsAt time.Time `json:"happiness_grace_ends_at,omitzero"`
	// SickSince and LastCuredAt are likewise absent from any save file written
	// before Sickness existed; their zero values read as "never Sick, never
	// Cured", so no schema_version bump is needed.
	SickSince   time.Time `json:"sick_since,omitzero"`
	LastCuredAt time.Time `json:"last_cured_at,omitzero"`
	// DiedAt and the cause of death are likewise absent from any save file
	// written before deaths were recorded, or while the Pet is alive. A save
	// without them is alive unless it is past its age (see Pet.Stage). The cause
	// is stored by name, so reordering the Go constants can never change what an
	// existing save means; an unreadable name is read as old age on load.
	DiedAt    time.Time `json:"died_at,omitzero"`
	Cause     string    `json:"cause_of_death,omitempty"`
	Hunger    int       `json:"hunger"`
	Happiness int       `json:"happiness"`
	Weight    int       `json:"weight"`
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

	return withLoadDefaults(Pet{
		CreatedAt:           sf.CreatedAt,
		LastSeenAt:          sf.LastSeenAt,
		HappinessLastSeenAt: sf.HappinessLastSeenAt,
		HappinessProgress:   sf.HappinessProgress,
		CareMistakes:        sf.CareMistakes,
		HungerEmpty:         EmptySpell{Since: sf.HungerEmptySince, GraceEndsAt: sf.HungerGraceEndsAt},
		HappinessEmpty:      EmptySpell{Since: sf.HappinessEmptySince, GraceEndsAt: sf.HappinessGraceEndsAt},
		SickSince:           sf.SickSince,
		LastCuredAt:         sf.LastCuredAt,
		DiedAt:              sf.DiedAt,
		Cause:               causeFromName(sf.Cause),
		LastCleanedAt:       sf.LastCleanedAt,
		Hunger:              sf.Hunger,
		Happiness:           sf.Happiness,
		Weight:              sf.Weight,
	}), true, nil
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
		SchemaVersion:        schemaVersion,
		CreatedAt:            p.CreatedAt,
		LastSeenAt:           p.LastSeenAt,
		HappinessLastSeenAt:  p.HappinessLastSeenAt,
		HappinessProgress:    p.HappinessProgress,
		CareMistakes:         p.CareMistakes,
		HungerEmptySince:     p.HungerEmpty.Since,
		HungerGraceEndsAt:    p.HungerEmpty.GraceEndsAt,
		HappinessEmptySince:  p.HappinessEmpty.Since,
		HappinessGraceEndsAt: p.HappinessEmpty.GraceEndsAt,
		SickSince:            p.SickSince,
		LastCuredAt:          p.LastCuredAt,
		DiedAt:               p.DiedAt,
		Cause:                causeName(p.Cause),
		LastCleanedAt:        p.LastCleanedAt,
		Hunger:               p.Hunger,
		Happiness:            p.Happiness,
		Weight:               p.Weight,
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
