package next

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/leekli/tamagotchi-go/internal/anim"
	"github.com/leekli/tamagotchi-go/internal/art"
	"github.com/leekli/tamagotchi-go/internal/pet"
)

// eggArt is the Pet's Egg-stage art: static, no idle animation.
var eggArt = art.MustLoad("egg.txt")

// babyFrames are the Baby-stage's two idle poses, reused from the Welcome
// Screen's Marutchi walk cycle: it gives the Pet visual continuity with the
// Character introduced at launch, without new asset-authoring risk.
var babyFrames = [2][]string{
	art.MustLoad("marutchi-walk-1.txt"),
	art.MustLoad("marutchi-walk-2.txt"),
}

const (
	// artBoxHeight leaves a row of bob headroom above and below the art, the
	// same technique welcome/character.go uses for the Character's bob, so
	// hatching or bobbing never shifts the rest of the stack.
	artBoxHeight = 5

	// babyBobFramesPerStep and babyPoseFramesPerStep pace the Baby's idle
	// motion: a gentle bob and an occasional pose change, not a walk cycle —
	// full walking around the Screen is a later feature.
	babyBobFramesPerStep  = 4
	babyPoseFramesPerStep = 8
)

// babyPose returns the Baby's current idle pose for frame.
func babyPose(frame int) []string {
	return babyFrames[(frame/babyPoseFramesPerStep)%len(babyFrames)]
}

// babyBob returns the Baby's vertical bob offset for frame, the same
// anim.Bob pattern welcome/character.go uses for the Character's walk bob.
func babyBob(frame int) int {
	return anim.Bob(frame, babyBobFramesPerStep)
}

// renderArtBox draws frameArt inside a fixed-height block, offset vertically
// by bob, so the stack above and below it never shifts as the Pet bobs or
// hatches from Egg to Baby.
func renderArtBox(frameArt []string, bob int, style lipgloss.Style) string {
	top := (artBoxHeight-len(frameArt))/2 + bob

	lines := make([]string, artBoxHeight)
	for i, row := range frameArt {
		if y := top + i; y >= 0 && y < artBoxHeight {
			lines[y] = row
		}
	}
	return style.Render(strings.Join(lines, "\n"))
}

// renderMeter draws label as a four-pip meter, e.g. "Hunger    [**--]".
func renderMeter(label string, value int, style lipgloss.Style) string {
	pips := strings.Repeat("*", value) + strings.Repeat("-", pet.MaxStat-value)
	return style.Render(fmt.Sprintf("%-10s[%s]", label, pips))
}

// infoLine renders the Pet's Age and Weight on one line.
func infoLine(p pet.Pet, now time.Time) string {
	return fmt.Sprintf("%s   Weight %dg", ageLabel(p.Age(now)), p.Weight)
}

// ageLabel renders age in the human-scale unit the Next Screen shows: whole
// days since birth, starting at "Day 0".
func ageLabel(age time.Duration) string {
	days := int(age / (24 * time.Hour))
	return fmt.Sprintf("Day %d", days)
}
