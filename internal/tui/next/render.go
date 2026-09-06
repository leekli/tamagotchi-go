package next

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	zone "github.com/lrstanley/bubblezone"

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

// messArt is the small pile glyph shown while the Pet HasMess — a status
// indicator, not a set piece, so it's kept tiny.
var messArt = art.MustLoad("mess.txt")

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

// renderMessLine draws the Mess indicator glyph.
func renderMessLine(style lipgloss.Style) string {
	return style.Render(strings.Join(messArt, "\n"))
}

// renderIconBar draws the Care-action icon bar, highlighting selected and
// wrapping each icon in its own bubblezone mark so a click can be tested
// against exactly its cells — the same pattern welcome/prompt.go uses for
// the begin prompt, with the same nil-manager guard for a click arriving
// before the first scan.
func renderIconBar(selected int, normal, selectedStyle lipgloss.Style) string {
	parts := make([]string, numIcons)
	for i := icon(0); i < numIcons; i++ {
		style := normal
		if int(i) == selected {
			style = selectedStyle
		}
		rendered := style.Render(fmt.Sprintf("[ %s ]", iconLabel[i]))
		if zone.DefaultManager != nil {
			rendered = zone.Mark(iconZoneID[i], rendered)
		}
		parts[i] = rendered
	}
	return strings.Join(parts, "  ")
}

// renderFeedChoice draws Feed's inline Meal/Snack chooser, using the same
// selection/zone-marking pattern as renderIconBar.
func renderFeedChoice(selected int, normal, selectedStyle lipgloss.Style) string {
	parts := make([]string, numFeedOptions)
	for i := feedOption(0); i < numFeedOptions; i++ {
		style := normal
		if int(i) == selected {
			style = selectedStyle
		}
		rendered := style.Render(fmt.Sprintf("[ %s ]", feedOptionLabel[i]))
		if zone.DefaultManager != nil {
			rendered = zone.Mark(feedZoneID[i], rendered)
		}
		parts[i] = rendered
	}
	return strings.Join(parts, "  ")
}
