// Package art loads the game's hand-authored ASCII art. Each piece is a .txt
// file beside this source, embedded into the binary so a built game needs no
// data files at runtime. Art is returned as a slice of rows (trailing newline
// dropped, interior spacing kept) which golden-file tests can compare directly.
//
// Art is ASCII only: widths are measured in runes, one cell per rune.
package art

import (
	"embed"
	"strings"
	"unicode/utf8"
)

//go:embed *.txt
var files embed.FS

// Load returns the rows of the named art file (e.g. "wordmark.txt"). A missing
// file is an authoring mistake caught at first run, not a recoverable error.
func Load(name string) ([]string, error) {
	b, err := files.ReadFile(name)
	if err != nil {
		return nil, err
	}
	return strings.Split(strings.TrimRight(string(b), "\n"), "\n"), nil
}

// MustLoad is [Load] for package-level art vars: it panics if the file is
// missing.
func MustLoad(name string) []string {
	rows, err := Load(name)
	if err != nil {
		panic("art: " + err.Error())
	}
	return rows
}

// Width returns the rune count of the widest row.
func Width(rows []string) int {
	widest := 0
	for _, r := range rows {
		if n := utf8.RuneCountInString(r); n > widest {
			widest = n
		}
	}
	return widest
}

// Mirror returns rows flipped left-to-right. Rows are first padded to a common
// width so ragged art mirrors squarely; then each row's runes are reversed and
// the glyphs with an obvious mirror image are swapped so the result still reads
// as art.
func Mirror(rows []string) []string {
	width := Width(rows)
	out := make([]string, len(rows))
	for i, row := range rows {
		runes := []rune(row + strings.Repeat(" ", width-utf8.RuneCountInString(row)))
		for l, r := 0, len(runes)-1; l < r; l, r = l+1, r-1 {
			runes[l], runes[r] = mirrorRune(runes[r]), mirrorRune(runes[l])
		}
		if len(runes)%2 == 1 {
			mid := len(runes) / 2
			runes[mid] = mirrorRune(runes[mid])
		}
		out[i] = string(runes)
	}
	return out
}

// mirrorRune returns the left-right mirror image of r, or r itself when it has
// no sensible mirror.
func mirrorRune(r rune) rune {
	switch r {
	case '(':
		return ')'
	case ')':
		return '('
	case '/':
		return '\\'
	case '\\':
		return '/'
	case '<':
		return '>'
	case '>':
		return '<'
	case '[':
		return ']'
	case ']':
		return '['
	case '{':
		return '}'
	case '}':
		return '{'
	case 'b':
		return 'd'
	case 'd':
		return 'b'
	default:
		return r
	}
}
