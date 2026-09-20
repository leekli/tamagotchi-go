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
	"github.com/leekli/tamagotchi-go/internal/tui"
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

// adultFrames are the Adult-stage's two idle poses: bulkier and more
// detailed than Teen's, within the same three-row art box, continuing the
// same Marutchi-lineage silhouette rather than a new design — a Pet reads as
// one creature growing up, not changing species.
var adultFrames = [][]string{
	art.MustLoad("adult-walk-1.txt"),
	art.MustLoad("adult-walk-2.txt"),
}

// deathArt is the Death Stage's art: a single static gravestone pose, no
// bob or animation, the same stillness eggArt uses — a Pet that has died is
// as inanimate as one that hasn't hatched yet.
var deathArt = art.MustLoad("gravestone.txt")

// messArt is the small pile glyph shown while the Pet HasMess — a status
// indicator, not a set piece, so it's kept tiny.
var messArt = art.MustLoad("mess.txt")

const (
	// artBoxHeight leaves a row of bob headroom above and below the art, the
	// same technique welcome/character.go uses for the Character's bob, so
	// hatching, growing, or bobbing never shifts the rest of the stack.
	artBoxHeight = 5

	// bobFramesPerStep paces the gentle idle bob shared by every hatched
	// Stage. babyPoseFramesPerStep, teenPoseFramesPerStep, and
	// adultPoseFramesPerStep separately pace each Stage's own
	// pose-alternation walk cycle.
	bobFramesPerStep       = 4
	babyPoseFramesPerStep  = 8
	teenPoseFramesPerStep  = 8
	adultPoseFramesPerStep = 8
)

// alternatingPose returns the current pose out of frames for frame, advancing
// one step every framesPerStep frames — the pose-cycling behind both Baby's
// and Teen's idle walk cycles, so a further alternating-pose Stage needs no
// copy of this same one-liner.
func alternatingPose(frames [][]string, framesPerStep, frame int) []string {
	return frames[anim.Cycle(frame, framesPerStep, len(frames))]
}

// hatchedBob returns the vertical bob offset for frame shared by every
// hatched Stage (Baby, Child, Teen, and Adult), the same anim.Bob pattern
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

// meterBar renders value (0..pet.MaxStat) as a graded block bar, e.g.
// "▓▓▓░" for 3 of 4.
func meterBar(value int) string {
	return strings.Repeat("▓", value) + strings.Repeat("░", pet.MaxStat-value)
}

// meterStyle returns the style a Meter's value should render in: good
// (green) for 3-4 of pet.MaxStat, fair (amber) for 2, low (danger) for 0-1.
// Segment count remains the primary signal in meterBar; colour here is
// additive, never the only one, so grading still reads correctly once
// colour degrades to 16-colour, monochrome, or NO_COLOR terminals.
func meterStyle(value int, good, fair, low lipgloss.Style) lipgloss.Style {
	switch {
	case value >= pet.MaxStat-1:
		return good
	case value == pet.MaxStat-2:
		return fair
	default:
		return low
	}
}

// renderMeter draws label as a graded block-bar Meter, e.g.
// "Hunger     ▓▓▓░ 3/4".
func renderMeter(label string, value int, good, fair, low lipgloss.Style) string {
	style := meterStyle(value, good, fair, low)
	return style.Render(fmt.Sprintf("%-9s  %s %d/%d", label, meterBar(value), value, pet.MaxStat))
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

// renderSickLine draws the Sick indicator. It is a text label as well as the
// Danger colour, so it still reads on a monochrome or NO_COLOR terminal, and it
// fits the STATS panel's 19-column content width.
func renderSickLine(style lipgloss.Style) string {
	return style.Render("[+] Sick")
}

// renderMessLine draws the Mess indicator glyph.
func renderMessLine(style lipgloss.Style) string {
	return style.Render(strings.Join(messArt, "\n"))
}

// renderTab draws one Icon-bar tab: label centred inside a fixed-size
// bordered box (tabWidth x tabHeight, no vertical padding — see
// docs/adr/0007), filled with selectedStyle's Accent background and dark
// foreground when selected, outline-only in normal otherwise. The border
// itself stays border regardless of selection: only the label's own style
// carries the "this is the pressable one" signal.
func renderTab(label string, selected bool, normal, selectedStyle lipgloss.Style, border lipgloss.AdaptiveColor) string {
	style := normal
	if selected {
		style = selectedStyle
	}
	field := lipgloss.PlaceHorizontal(tabFieldWidth, lipgloss.Center, label)
	panel := tui.Panel{Width: tabWidth, Height: tabHeight, PaddingX: 1, PaddingY: 0, Border: border}
	return panel.Render(style.Render(field))
}

// renderTabRow lays out labels as a row of bordered tabs, the selected index
// highlighted, each wrapped in its zoneID so a click can be tested against
// exactly its cells (nil-manager guarded, the same pattern
// welcome/welcome.go's chip uses). The row is always exactly iconRowWidth
// columns, regardless of how many tabs it holds, so switching between the
// Feed/Play/Clean bar and the narrower Meal/Snack chooser never shifts
// anything else on screen.
func renderTabRow(labels []string, zoneIDs []string, selected int, normal, selectedStyle lipgloss.Style, border lipgloss.AdaptiveColor) string {
	parts := make([]string, 0, 2*len(labels)-1)
	for i, label := range labels {
		if i > 0 {
			parts = append(parts, tabGap)
		}
		tab := renderTab(label, i == selected, normal, selectedStyle, border)
		if zone.DefaultManager != nil {
			tab = zone.Mark(zoneIDs[i], tab)
		}
		parts = append(parts, tab)
	}
	row := lipgloss.JoinHorizontal(lipgloss.Top, parts...)
	return lipgloss.PlaceHorizontal(iconRowWidth, lipgloss.Center, row)
}

// renderIconBar draws the Care-action icon bar (Feed, Play, Clean, Cure) as a row
// of bordered tabs.
func renderIconBar(selected int, normal, selectedStyle lipgloss.Style, border lipgloss.AdaptiveColor) string {
	return renderTabRow(iconLabel[:], iconZoneID[:], selected, normal, selectedStyle, border)
}

// renderFeedChoice draws Feed's inline Meal/Snack chooser, using the same
// tab-row layout as renderIconBar.
func renderFeedChoice(selected int, normal, selectedStyle lipgloss.Style, border lipgloss.AdaptiveColor) string {
	return renderTabRow(feedOptionLabel[:], feedZoneID[:], selected, normal, selectedStyle, border)
}
