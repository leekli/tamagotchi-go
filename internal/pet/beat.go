package pet

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// BeatInterval is the gap between simulation steps: the slow clock that
// drives Decay and the periodic save. Deliberately not named "Tick" — that
// name already belongs to anim's much faster (~15/sec) animation clock, and
// reusing it for a clock fifty times slower would blur two genuinely
// different things in code and conversation.
const BeatInterval = 20 * time.Second

// BeatMsg is delivered once per BeatInterval. It carries the wall-clock time
// the beat fired, which the Next Screen passes to Pet.Advance.
type BeatMsg struct {
	Time time.Time
}

// Beat returns a command that emits one BeatMsg after a single
// BeatInterval. A Screen re-issues it from Update on every BeatMsg to keep
// the simulation running — the same pattern anim.Tick uses for animation
// frames.
func Beat() tea.Cmd {
	return tea.Tick(BeatInterval, newBeatMsg)
}

// newBeatMsg builds the message tea.Tick delivers once BeatInterval has
// elapsed. Split out from Beat so it's testable on its own, without waiting
// out the real interval.
func newBeatMsg(t time.Time) tea.Msg {
	return BeatMsg{Time: t}
}
