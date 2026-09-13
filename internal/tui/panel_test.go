package tui_test

import (
	"regexp"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/leekli/tamagotchi-go/internal/tui"
)

// panelBorder is a fixed colour used throughout: these tests are about
// dimensions and structure, not colour, so a single AdaptiveColor keeps them
// focused.
var panelBorder = tui.DefaultPalette().Dim

// ansiSeq strips SGR escape codes so caption/content substrings can be
// asserted on regardless of colour profile, mirroring the same helper
// already defined in internal/tui/welcome and internal/tui/next's tests.
var ansiSeq = regexp.MustCompile("\x1b\\[[0-9;]*m")

// everyLineHasWidth asserts every line of rendered is exactly want columns
// wide, and that rendered has exactly wantLines lines.
func everyLineHasWidth(t *testing.T, rendered string, wantLines, want int) {
	t.Helper()
	lines := strings.Split(rendered, "\n")
	require.Len(t, lines, wantLines, "line count")
	for i, line := range lines {
		assert.Equal(t, want, lipgloss.Width(line), "line %d width", i)
	}
}

func TestPanelPadsContentNarrowerThanWidth(t *testing.T) {
	p := tui.Panel{Width: 20, Height: 6, Border: panelBorder}
	rendered := p.Render("hi")
	everyLineHasWidth(t, rendered, 6, 20)
}

func TestPanelPadsContentShorterThanHeight(t *testing.T) {
	p := tui.Panel{Width: 20, Height: 8, Border: panelBorder}
	rendered := p.Render("one line")
	everyLineHasWidth(t, rendered, 8, 20)
}

func TestPanelClampsContentWiderThanWidth(t *testing.T) {
	p := tui.Panel{Width: 12, Height: 5, Border: panelBorder}
	// No spaces to wrap on: this is exactly the adversarial case that would
	// overflow Width() alone without the MaxWidth backstop.
	rendered := p.Render(strings.Repeat("x", 40))
	everyLineHasWidth(t, rendered, 5, 12)
}

func TestPanelClampsContentTallerThanHeight(t *testing.T) {
	p := tui.Panel{Width: 16, Height: 6, Border: panelBorder}
	lines := make([]string, 20)
	for i := range lines {
		lines[i] = "row"
	}
	rendered := p.Render(strings.Join(lines, "\n"))
	everyLineHasWidth(t, rendered, 6, 16)
}

func TestPanelHandlesEmptyContent(t *testing.T) {
	p := tui.Panel{Width: 14, Height: 5, Border: panelBorder}
	rendered := p.Render("")
	everyLineHasWidth(t, rendered, 5, 14)
}

func TestPanelWithNoCaptionHasAPlainTopBorder(t *testing.T) {
	p := tui.Panel{Width: 18, Height: 6, Border: panelBorder}
	rendered := p.Render("content")
	lines := strings.Split(rendered, "\n")
	require.NotEmpty(t, lines)
	assert.NotContains(t, ansiSeq.ReplaceAllString(lines[0], ""), " ")
	assert.True(t, strings.HasPrefix(ansiSeq.ReplaceAllString(lines[0], ""), "╭"))
	assert.True(t, strings.HasSuffix(ansiSeq.ReplaceAllString(lines[0], ""), "╮"))
}

func TestPanelWithACaptionSplicesItIntoTheTopBorderOnly(t *testing.T) {
	p := tui.Panel{Width: 18, Height: 6, Caption: "PET", Border: panelBorder}
	rendered := p.Render("content")
	lines := strings.Split(rendered, "\n")
	require.Len(t, lines, 6)

	top := ansiSeq.ReplaceAllString(lines[0], "")
	assert.Contains(t, top, "PET")
	assert.Equal(t, 18, lipgloss.Width(lines[0]), "captioned top border keeps the exact width")

	for i, line := range lines[1:] {
		assert.NotContains(t, ansiSeq.ReplaceAllString(line, ""), "PET", "line %d should not carry the caption", i+1)
	}
}

func TestPanelCaptionNeverChangesOverallDimensions(t *testing.T) {
	plain := tui.Panel{Width: 24, Height: 7, Border: panelBorder}
	captioned := tui.Panel{Width: 24, Height: 7, Caption: "STATS", Border: panelBorder}

	plainRendered := strings.Split(plain.Render("x"), "\n")
	captionedRendered := strings.Split(captioned.Render("x"), "\n")

	require.Len(t, captionedRendered, len(plainRendered))
	for i := range captionedRendered {
		assert.Equal(t, lipgloss.Width(plainRendered[i]), lipgloss.Width(captionedRendered[i]), "line %d width", i)
	}
}
