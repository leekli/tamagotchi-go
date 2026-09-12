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
var babyFrames = [][]string{
	art.MustLoad("marutchi-walk-1.txt"),
	art.MustLoad("marutchi-walk-2.txt"),
}

// childArt is the Child-stage's single idle pose. Unlike Baby it doesn't
// alternate poses — just the shared hatchedBob — since a Child is meant to
// read as visually settled next to Baby's fidgeting.
var childArt = art.MustLoad("child.txt")

// teenFrames are the Teen-stage's two idle poses: a visibly bigger, more
// grown-up-looking walk cycle than Child's single static pose, so a Teen
// reads as a further step up in liveliness rather than a repeat of Child.
var teenFrames = [][]string{
	art.MustLoad("teen-walk-1.txt"),
	art.MustLoad("teen-walk-2.txt"),
}

// messArt is the small pile glyph shown while the Pet HasMess — a status
// indicator, not a set piece, so it's kept tiny.
var messArt = art.MustLoad("mess.txt")

const (
	// artBoxHeight leaves a row of bob headroom above and below the art, the
	// same technique welcome/character.go uses for the Character's bob, so
	// hatching, growing, or bobbing never shifts the rest of the stack.
	artBoxHeight = 5

	// bobFramesPerStep paces the gentle idle bob shared by every hatched
	// Stage. babyPoseFramesPerStep and teenPoseFramesPerStep separately pace
	// each Stage's own pose-alternation walk cycle.
	bobFramesPerStep      = 4
	babyPoseFramesPerStep = 8
	teenPoseFramesPerStep = 8
)

// alternatingPose returns the current pose out of frames for frame, advancing
// one step every framesPerStep frames — the pose-cycling behind both Baby's
// and Teen's idle walk cycles, so a further alternating-pose Stage needs no
// copy of this same one-liner.
func alternatingPose(frames [][]string, framesPerStep, frame int) []string {
	return frames[anim.Cycle(frame, framesPerStep, len(frames))]
}

// hatchedBob returns the vertical bob offset for frame shared by every
// hatched Stage (Baby, Child, and Teen), the same anim.Bob pattern
// welcome/character.go uses for the Character's walk bob.
func hatchedBob(frame int) int {
	return anim.Bob(frame, bobFramesPerStep)
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

// renderStageLabel draws the Pet's current Stage as a bare word (e.g.
// "Baby"), so a player doesn't have to infer it from the art alone. Shown
// for every Stage, not just Child.
func renderStageLabel(stage pet.Stage, style lipgloss.Style) string {
	return style.Render(stage.String())
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
