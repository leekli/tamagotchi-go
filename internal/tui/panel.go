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
// Panel leaves it empty (see CONTEXT.md's Panel entry). PaddingX and
// PaddingY are set independently (not a single uniform value) because a
// small chrome element like an Icon-bar tab wants no vertical padding at
// all (PaddingY: 0) to fit a 3-row height, while every other Panel in this
// app uses 1 on both axes.
type Panel struct {
	Width, Height int
	Caption       string
	PaddingX      int
	PaddingY      int
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
		PaddingLeft(p.PaddingX).
		PaddingRight(p.PaddingX).
		PaddingTop(p.PaddingY).
		PaddingBottom(p.PaddingY).
		Width(innerWidth).
		Height(innerHeight).
		Align(lipgloss.Center, lipgloss.Center).
		MaxWidth(p.Width).
		MaxHeight(p.Height).
		Render(content)

	if p.Caption == "" {
		return box
	}

	lines := strings.Split(box, "\n")
	lines[0] = lipgloss.NewStyle().Foreground(p.Border).Render(p.topBorderLine())
	return strings.Join(lines, "\n")
}

// topBorderLine builds the plain-text top border with Caption spliced in
// between a dash and a space on each side, e.g. "╭─ PET ──────────╮" —
// always exactly Width runes, so it lines up with every other border row
// even if Caption is too long to leave room for any dashes at all (it is
// truncated first, rather than ever widening the line past Width). Its
// corner and fill runes come from the same RoundedBorder Lip Gloss uses to
// build the rest of the box, so the two can never drift apart.
func (p Panel) topBorderLine() string {
	border := lipgloss.RoundedBorder()
	avail := p.Width - lipgloss.Width(border.TopLeft) - lipgloss.Width(border.TopRight)

	prefix := border.Top + " " + p.Caption + " "
	if w := lipgloss.Width(prefix); w > avail {
		prefix = truncateToWidth(prefix, avail)
	}

	dashes := avail - lipgloss.Width(prefix)
	if dashes < 0 {
		dashes = 0
	}
	return border.TopLeft + prefix + strings.Repeat(border.Top, dashes) + border.TopRight
}

// truncateToWidth returns the longest leading-rune prefix of s whose display
// width is at most w. s is assumed to contain only single-width runes (true
// of every caption and border glyph this function is used for), so a rune
// count doubles as a width count here.
func truncateToWidth(s string, w int) string {
	if w <= 0 {
		return ""
	}
	runes := []rune(s)
	if len(runes) <= w {
		return s
	}
	return string(runes[:w])
}
