package welcome_test

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/leekli/tamagotchi-go/internal/tui"
)

// These tests map one given/when/then group to each sentence of the Phase 2
// Welcome Screen brief.

func TestAcceptance_Wordmark(t *testing.T) {
	t.Parallel()

	t.Run("given the Screen when it renders then the Tamagotchi wordmark art is shown", func(t *testing.T) {
		view := stripANSI(sizedScreen(t, 90, 24).View())
		// Six rows of block-letter art, ending in the rounded feet of the word.
		assert.Contains(t, view, "(____)")
		assert.Contains(t, view, "|_||_|")
	})

	t.Run("given the Screen when it renders then the wordmark rows stay column-aligned", func(t *testing.T) {
		var wm []string
		for _, l := range strings.Split(sizedScreen(t, 90, 24).View(), "\n") {
			if strings.Contains(l, "|") {
				wm = append(wm, stripANSI(l))
			}
		}
		require.Len(t, wm, 5) // the five wordmark rows that carry '|' strokes

		// Every rendered row is the same visible width, so lipgloss centres them
		// all by the same offset and art column N stays screen column N+k on
		// every row — which is what makes the word readable. A ragged block is
		// re-centred row-by-row and the vertical strokes drift.
		for i := 1; i < len(wm); i++ {
			assert.Equal(t, lipgloss.Width(wm[0]), lipgloss.Width(wm[i]), "row %d width", i)
		}
	})
}

func TestAcceptance_ShineSweep(t *testing.T) {
	t.Parallel()

	t.Run("given a fresh Screen when a few frames pass then the wordmark changes as the sweep crosses", func(t *testing.T) {
		s := sizedScreen(t, 90, 24)
		start := wordmarkRegion(s.View())
		s = advance(t, s, 5)
		crossing := wordmarkRegion(s.View())
		assert.NotEqual(t, start, crossing)
	})

	t.Run("given the sweep has finished when more frames pass then the wordmark is static", func(t *testing.T) {
		s := advance(t, sizedScreen(t, 90, 24), 30)
		settled := wordmarkRegion(s.View())
		s = advance(t, s, 15)
		assert.Equal(t, settled, wordmarkRegion(s.View()))
	})

	t.Run("given the sweep has finished when the terminal is resized then the sweep does not replay", func(t *testing.T) {
		s := advance(t, sizedScreen(t, 90, 24), 30)
		before := wordmarkRegion(s.View())
		s, _ = s.Update(tea.WindowSizeMsg{Width: 120, Height: 30})
		s, _ = s.Update(tea.WindowSizeMsg{Width: 90, Height: 24})
		assert.Equal(t, before, wordmarkRegion(s.View()))
	})
}

func TestAcceptance_Character(t *testing.T) {
	t.Parallel()

	t.Run("given a fresh Screen when it renders then the Character sits below the wordmark", func(t *testing.T) {
		lines := strings.Split(stripANSI(sizedScreen(t, 90, 24).View()), "\n")
		wordmarkRow, charRow := -1, -1
		for i, l := range lines {
			if strings.Contains(l, "|_||_|") && wordmarkRow < 0 {
				wordmarkRow = i
			}
			if strings.Contains(l, "( o o)") {
				charRow = i
			}
		}
		require.GreaterOrEqual(t, wordmarkRow, 0)
		require.GreaterOrEqual(t, charRow, 0)
		assert.Less(t, wordmarkRow, charRow, "the Character box is below the wordmark")
	})

	t.Run("given the Screen when many frames pass then the Character wanders across at least two columns", func(t *testing.T) {
		s := sizedScreen(t, 90, 24)
		cols := map[int]bool{}
		for i := 0; i < 40; i++ {
			for _, l := range strings.Split(stripANSI(s.View()), "\n") {
				if idx := strings.Index(l, "( o o)"); idx >= 0 {
					cols[idx] = true
				}
			}
			s = advance(t, s, 3)
		}
		assert.GreaterOrEqual(t, len(cols), 2)
	})
}

func TestAcceptance_BeginPrompt(t *testing.T) {
	t.Parallel()

	t.Run("given the Screen when it renders then the begin prompt text is shown", func(t *testing.T) {
		view := stripANSI(sizedScreen(t, 90, 24).View())
		assert.Contains(t, view, "Press Enter or click to begin")
	})

	t.Run("given the Screen when a pulse period passes then the prompt styling changes", func(t *testing.T) {
		s := sizedScreen(t, 90, 24)
		lineAt := func() string {
			for _, l := range strings.Split(s.View(), "\n") {
				if strings.Contains(l, "Press Enter or click to begin") {
					return l
				}
			}
			return ""
		}
		first := lineAt()
		changed := false
		for i := 0; i < 15 && !changed; i++ {
			s = advance(t, s, 1)
			if lineAt() != first {
				changed = true
			}
		}
		assert.True(t, changed, "the prompt should pulse over about a second")
	})
}

func TestAcceptance_AdvanceAndQuit(t *testing.T) {
	t.Parallel()

	t.Run("given the Screen when Enter is pressed then it navigates to the Next Screen", func(t *testing.T) {
		_, cmd := newScreen(t).Update(tea.KeyMsg{Type: tea.KeyEnter})
		requireNavigatesToNext(t, cmd)
	})

	t.Run("given the Screen when the begin prompt is clicked then it navigates to the Next Screen", func(t *testing.T) {
		requireNavigatesToNext(t, clickBeginPrompt(t, sizedScreen(t, 100, 30)))
	})

	t.Run("given the Screen when a click lands outside the prompt then nothing happens", func(t *testing.T) {
		s := sizedScreen(t, 100, 30)
		beginZone(t, s) // the prompt zone is known; this click still misses it
		_, cmd := s.Update(tea.MouseMsg{
			Action: tea.MouseActionPress, Button: tea.MouseButtonLeft, X: 1, Y: 1,
		})
		assert.Nil(t, cmd)
	})

	t.Run("given the Screen when Ctrl+C is pressed then the Screen itself does not begin", func(t *testing.T) {
		s, cmd := newScreen(t).Update(tea.KeyMsg{Type: tea.KeyCtrlC})
		assert.Nil(t, cmd, "Ctrl+C quitting is the App's job, not a begin")
		assert.Equal(t, tui.WelcomeScreenID, s.ID())
	})

	t.Run("given the Screen when Esc is pressed then it quits", func(t *testing.T) {
		_, cmd := newScreen(t).Update(tea.KeyMsg{Type: tea.KeyEsc})
		require.NotNil(t, cmd, "Esc is the Welcome Screen's own quit key")
		_, isQuit := cmd().(tea.QuitMsg)
		assert.True(t, isQuit, "expected tea.QuitMsg")
	})
}

func TestAcceptance_Layout(t *testing.T) {
	t.Parallel()

	t.Run("given an 80x24 terminal when the Screen renders then it fills the body area exactly", func(t *testing.T) {
		// The App reserves one row for the help bar, so the body is 23 rows.
		view := sizedScreen(t, 80, 23).View()
		assert.Equal(t, 23, strings.Count(view, "\n")+1)
	})

	t.Run("given the Screen when it renders then wordmark, Character and prompt appear in that order", func(t *testing.T) {
		view := stripANSI(sizedScreen(t, 100, 28).View())
		wm := strings.Index(view, "(____)")
		ch := strings.Index(view, "( o o)")
		pr := strings.Index(view, "Press Enter or click to begin")
		require.True(t, wm >= 0 && ch >= 0 && pr >= 0)
		assert.Less(t, wm, ch)
		assert.Less(t, ch, pr)
	})
}
