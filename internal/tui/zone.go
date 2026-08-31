package tui

import (
	"sync"

	zone "github.com/lrstanley/bubblezone"
)

var zoneOnce sync.Once

// enableZones starts the process-wide bubblezone manager exactly once. Screens
// mark clickable regions with zone.Mark; App.View scans the composed frame to
// resolve them. Safe to call from every App construction and from tests.
func enableZones() { zoneOnce.Do(zone.NewGlobal) }
