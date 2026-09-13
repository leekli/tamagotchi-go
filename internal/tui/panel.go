package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Panel is a labelled, bordered content region rendered to an exact width
// and height regardless of what it is given to render — the fixed-dimension
// contract every Bordered Dashboard Screen relies on (see ADR-0007). Width
// and Height are the Panel's total outer size, border and padding included.
// Caption, when non-empty, is spliced into the top border; a Screen's sole
// Panel leaves it empty (see CONTEXT.md's Panel entry).
type Panel struct {
	Width, Height int
	Caption       string
	Border        lipgloss.AdaptiveColor
}

// panelBorderWidth is the border (1 column/row) Lip Gloss adds on each side
// of a Panel's declared Width()/Height() — which already accounts for
// Padding(1) internally, so only the border itself is subtracted here.
const panelBorderWidth = 2

// Render draws content inside p: shorter or fewer-lined content is padded up
// to p's exact size, longer or wider content is truncated down to it, so the
// result is always exactly p.Width columns by p.Height rows regardless of
// input. Width()/Height() alone only pad; MaxWidth()/MaxHeight() (applied
// after the border, per Lip Gloss's own render order) is what guarantees the
// upper bound.
func (p Panel) Render(content string) string {
	innerWidth := p.Width - panelBorderWidth
	innerHeight := p.Height - panelBorderWidth

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(p.Border).
		Padding(1).
		Width(innerWidth).
		Height(innerHeight).
		MaxWidth(p.Width).
		MaxHeight(p.Height).
		Render(content)

	if p.Caption == "" {
		return box
	}

	lines := strings.Split(box, "\n")
	if len(lines) == 0 {
		return box
	}
	lines[0] = lipgloss.NewStyle().Foreground(p.Border).Render(p.topBorderLine())
	return strings.Join(lines, "\n")
}

// topBorderLine builds the plain-text top border with Caption spliced in
// between a dash and a space on each side, e.g. "╭─ PET ──────────╮" —
// always exactly Width runes, so it lines up with every other border row.
func (p Panel) topBorderLine() string {
	prefix := "─ " + p.Caption + " "
	dashes := p.Width - 2 - lipgloss.Width(prefix)
	if dashes < 0 {
		dashes = 0
	}
	return "╭" + prefix + strings.Repeat("─", dashes) + "╮"
}
