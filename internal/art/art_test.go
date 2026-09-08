package art_test

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/leekli/tamagotchi-go/internal/art"
)

func TestLoadReturnsRowsWithoutTrailingNewline(t *testing.T) {
	t.Parallel()

	rows, err := art.Load("wordmark.txt")
	require.NoError(t, err)
	require.NotEmpty(t, rows)
	assert.NotEqual(t, "", rows[0])
	assert.NotContains(t, rows[len(rows)-1], "\n")
}

func TestLoadKeepsInteriorSpacing(t *testing.T) {
	t.Parallel()

	rows, err := art.Load("marutchi-walk-1.txt")
	require.NoError(t, err)
	require.Len(t, rows, 3)
	// Every row is padded to the same width so the frame is a solid block.
	for _, r := range rows {
		assert.Equal(t, len([]rune(rows[0])), len([]rune(r)))
	}
}

func TestLoadMissingFileErrors(t *testing.T) {
	t.Parallel()

	_, err := art.Load("does-not-exist.txt")
	assert.Error(t, err)
}

func TestMustLoadPanicsOnMissingFile(t *testing.T) {
	t.Parallel()

	assert.Panics(t, func() { art.MustLoad("does-not-exist.txt") })
	assert.NotPanics(t, func() { art.MustLoad("wordmark.txt") })
}

func TestWidth(t *testing.T) {
	t.Parallel()

	assert.Equal(t, 0, art.Width(nil))
	assert.Equal(t, 3, art.Width([]string{"a", "abc", "ab"}))
}

func TestMirrorReversesRowsAndSwapsGlyphs(t *testing.T) {
	t.Parallel()

	in := []string{
		"/--\\",
		"(  )",
		"b  d",
	}
	want := []string{
		"/--\\",
		"(  )",
		"b  d",
	}
	assert.Equal(t, want, art.Mirror(in))
}

func TestMirrorIsSelfInverse(t *testing.T) {
	t.Parallel()

	in := []string{
		" ,--. ",
		"( o o)",
		" >`-' ",
	}
	assert.Equal(t, in, art.Mirror(art.Mirror(in)))
}

func TestMirrorPadsRaggedRows(t *testing.T) {
	t.Parallel()

	out := art.Mirror([]string{"xy", "x"})
	require.Len(t, out, 2)
	assert.Equal(t, "yx", out[0])
	assert.Equal(t, " x", out[1])
}

func TestMirrorSwapsSlashesAndBrackets(t *testing.T) {
	t.Parallel()

	out := art.Mirror([]string{"/<[{"})
	assert.Equal(t, "}]>\\", out[0])
}

func TestMirrorSwapsTheMiddleGlyphOfAnOddWidthRow(t *testing.T) {
	t.Parallel()

	out := art.Mirror([]string{"a/b"})
	assert.Equal(t, "d\\a", out[0])
}

// FuzzMirrorPreservesShape hardens Mirror against arbitrary input, including
// non-ASCII runes and ragged rows the handwritten cases above don't try: for
// any rows, Mirror must never panic, must return exactly as many rows as it
// was given, and must pad every one of them to the input's widest row.
func FuzzMirrorPreservesShape(f *testing.F) {
	f.Add("abc\ndef\nghij")
	f.Add("/<[{\nb  d")
	f.Add("")
	f.Add("a\n\nbb")

	f.Fuzz(func(t *testing.T, blob string) {
		rows := strings.Split(blob, "\n")
		width := art.Width(rows)

		out := art.Mirror(rows)

		require.Len(t, out, len(rows), "Mirror must preserve the number of rows")
		for _, r := range out {
			assert.Equal(t, width, utf8.RuneCountInString(r), "every output row should be padded to the input's widest row")
		}
	})
}
