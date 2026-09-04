package welcome

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/leekli/tamagotchi-go/internal/anim"
	"github.com/leekli/tamagotchi-go/internal/art"
)

// marutchiRight holds the two right-facing walk frames; marutchiLeft is their
// mirror image, used when the Character is heading left.
var (
	marutchiRight = [2][]string{
		art.MustLoad("marutchi-walk-1.txt"),
		art.MustLoad("marutchi-walk-2.txt"),
	}
	marutchiLeft = [2][]string{
		art.Mirror(marutchiRight[0]),
		art.Mirror(marutchiRight[1]),
	}
)

const (
	// charBoxWidth is the width of the centred box the Character wanders in.
	charBoxWidth = 40
	// charBoxHeight leaves a row of bob headroom above and below the art.
	charBoxHeight = 5
	// charFramesPerCell is how many animation frames the Character spends
	// moving one column: ~1 cell per 2 frames at anim.FPS.
	charFramesPerCell = 2
)

type facing int

const (
	facingRight facing = iota
	facingLeft
)

// charState is the Character's wander state as a pure function of the animation
// frame count: it walks back and forth across the free width of the box,
// starting centred and heading right, bobbing one row as it goes, forever.
func charState(frame int) (x int, dir facing, bob, step int) {
	span := charBoxWidth - art.Width(marutchiRight[0])
	if span < 1 {
		span = 1
	}

	cell := frame / charFramesPerCell
	// Offset the path by half its span so frame 0 sits centred, heading right.
	period := 2 * span
	phase := ((cell + span/2) % period)

	if phase <= span {
		x, dir = phase, facingRight
	} else {
		x, dir = period-phase, facingLeft
	}

	bob = anim.Bob(frame, charFramesPerCell)
	step = cell % 2
	return x, dir, bob, step
}

// renderCharBox draws the Character at its current wander position inside a
// fixed charBoxWidth x charBoxHeight block, so the stack above and below it
// never shifts as the Character moves or bobs.
func renderCharBox(frame int, style lipgloss.Style) string {
	x, dir, bob, step := charState(frame)

	frameArt := marutchiRight[step]
	if dir == facingLeft {
		frameArt = marutchiLeft[step]
	}

	top := (charBoxHeight-len(frameArt))/2 + bob
	pad := strings.Repeat(" ", x)

	lines := make([]string, charBoxHeight)
	for i, row := range frameArt {
		if y := top + i; y >= 0 && y < charBoxHeight {
			lines[y] = style.Render(pad + row)
		}
	}

	return lipgloss.NewStyle().
		Width(charBoxWidth).
		Height(charBoxHeight).
		MaxWidth(charBoxWidth).
		MaxHeight(charBoxHeight).
		Render(strings.Join(lines, "\n"))
}
