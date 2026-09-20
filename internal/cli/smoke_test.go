package cli_test

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/creack/pty"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/leekli/tamagotchi-go/internal/pet"
)

// syncBuffer is an io.Writer safe for concurrent writes and snapshot reads.
type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (s *syncBuffer) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.buf.Write(p)
}

func (s *syncBuffer) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.buf.String()
}

func waitForOutput(t *testing.T, out *syncBuffer, want string) {
	t.Helper()
	waitForOutputTimeout(t, out, want, 5*time.Second)
}

func waitForOutputTimeout(t *testing.T, out *syncBuffer, want string, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if bytes.Contains([]byte(out.String()), []byte(want)) {
			return
		}
		time.Sleep(25 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %q in output; got:\n%s", want, out.String())
}

// buildBinary builds the real binary once and returns its path.
func buildBinary(t *testing.T) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "tamagotchi-go")
	build := exec.Command("go", "build", "-o", bin, "../../cmd/tamagotchi-go")
	build.Stderr = os.Stderr
	require.NoError(t, build.Run(), "building the binary")
	return bin
}

// smokeProcess is the built binary running under a pseudo-terminal.
type smokeProcess struct {
	cmd  *exec.Cmd
	ptmx *os.File
	out  *syncBuffer
}

// startBinary launches bin against savePath under a pseudo-terminal.
func startBinary(t *testing.T, bin, savePath string) *smokeProcess {
	t.Helper()

	cmd := exec.Command(bin, "--save-path", savePath)
	ptmx, err := pty.StartWithSize(cmd, &pty.Winsize{Rows: 30, Cols: 100})
	require.NoError(t, err)
	t.Cleanup(func() { _ = ptmx.Close() })

	// The child queries the terminal on startup (cursor position, background
	// colour). A real terminal answers; our pseudo-terminal does not, so we
	// answer for it — otherwise the program blocks for seconds waiting.
	out := &syncBuffer{}
	go func() {
		b := make([]byte, 4096)
		for {
			n, readErr := ptmx.Read(b)
			if n > 0 {
				chunk := b[:n]
				_, _ = out.Write(chunk)
				if bytes.Contains(chunk, []byte("\x1b[6n")) {
					_, _ = ptmx.Write([]byte("\x1b[1;1R"))
				}
				if bytes.Contains(chunk, []byte("\x1b]11;?")) {
					_, _ = ptmx.Write([]byte("\x1b]11;rgb:2020/2020/2020\x1b\\"))
				}
			}
			if readErr != nil {
				return
			}
		}
	}()
	return &smokeProcess{cmd: cmd, ptmx: ptmx, out: out}
}

func (p *smokeProcess) send(t *testing.T, keys string) {
	t.Helper()
	_, err := p.ptmx.Write([]byte(keys))
	require.NoError(t, err)
}

// quit sends Ctrl+C and requires the binary to exit 0 promptly.
func (p *smokeProcess) quit(t *testing.T) {
	t.Helper()
	p.send(t, "\x03")

	exited := make(chan error, 1)
	go func() { exited <- p.cmd.Wait() }()

	select {
	case err := <-exited:
		require.NoError(t, err, "binary should exit 0 on Ctrl+C")
	case <-time.After(5 * time.Second):
		_ = p.cmd.Process.Kill()
		t.Fatal("binary did not exit after Ctrl+C")
	}
}

// smokeSaveFile is the on-disk JSON shape the smoke test checks against —
// deliberately re-declared rather than importing internal/pet's unexported
// type, so this assertion is a black-box check of the real file, not a
// white-box check of the package that wrote it.
type smokeSaveFile struct {
	SchemaVersion int    `json:"schema_version"`
	CreatedAt     string `json:"created_at"`
	LastSeenAt    string `json:"last_seen_at"`
	LastCleanedAt string `json:"last_cleaned_at"`
	Hunger        int    `json:"hunger"`
	Happiness     int    `json:"happiness"`
	Weight        int    `json:"weight"`
}

// TestBinaryLaunchesAndQuits builds the real binary and drives it through a
// pseudo-terminal: it must show the Welcome Screen, advance on Enter, show
// the Pet, and exit cleanly on Ctrl+C — saving the Pet to the --save-path
// override along the way. This is the one place proving the whole
// load → run → save-on-quit path works end to end through the real binary,
// not just through fakes.
func TestBinaryLaunchesAndQuits(t *testing.T) {
	if testing.Short() {
		t.Skip("smoke test builds and launches the binary")
	}

	savePath := filepath.Join(t.TempDir(), "save.json")
	proc := startBinary(t, buildBinary(t), savePath)
	out := proc.out

	waitForOutput(t, out, "Press Enter or click to begin")

	proc.send(t, "\r")
	waitForOutput(t, out, "Hunger")

	// The icon bar only appears once the Pet has hatched into a Baby; unlike
	// the in-process integration tests, a separate OS process can't be
	// fast-forwarded by injecting a synthetic tick, so this really does wait
	// out EggDuration in real time.
	waitForOutputTimeout(t, out, "Play", pet.EggDuration+10*time.Second)

	proc.send(t, "f")
	waitForOutput(t, out, "Meal")

	proc.send(t, "m")
	waitForOutput(t, out, "munch munch")

	proc.send(t, "p")
	waitForOutput(t, out, "plays happily")

	proc.send(t, "c")
	waitForOutput(t, out, "tidied up")

	// A freshly hatched Pet is healthy, so Cure does nothing but say so: proof the
	// fourth tab and its hotkey are wired into the real binary.
	proc.send(t, "u")
	waitForOutput(t, out, "feels fine")

	proc.quit(t)

	b, err := os.ReadFile(savePath)
	require.NoError(t, err, "save-on-quit should have written the save file")

	var sf smokeSaveFile
	require.NoError(t, json.Unmarshal(b, &sf))
	assert.Equal(t, 1, sf.SchemaVersion)
	assert.NotEmpty(t, sf.CreatedAt)
	assert.NotEmpty(t, sf.LastSeenAt)
	assert.Equal(t, 4, sf.Hunger)
	assert.Equal(t, 4, sf.Happiness)
	assert.Equal(t, 2, sf.Weight)
	assert.NotEqual(t, sf.CreatedAt, sf.LastCleanedAt,
		"Clean should have persisted a last_cleaned_at well after birth")
}

// TestBinaryShowsTheCauseOfDeathAndRestarts seeds a back-dated save for each
// cause of Death through --save-path, so the real binary loads a Pet that died
// while the game was closed without waiting out its life. It must show the Death
// panel naming the cause, and Restart must hatch a fresh Egg that is saved.
func TestBinaryShowsTheCauseOfDeathAndRestarts(t *testing.T) {
	if testing.Short() {
		t.Skip("smoke test builds and launches the binary")
	}

	bin := buildBinary(t)
	started := time.Now()

	seeds := map[string]struct {
		seed func(now time.Time) pet.Pet
		want string
	}{
		// Cleaned a minute ago but never fed: Hunger emptied at 12:00 and starved it at
		// 22:00, with no Mess to make it Sick first.
		"Starvation": {
			seed: func(now time.Time) pet.Pet {
				p := pet.New(now.Add(-40 * time.Minute))
				p.LastCleanedAt = now.Add(-time.Minute)
				return p
			},
			want: "Cause: Starvation",
		},
		// Left untended for 40 minutes: Sick at 10:00 and dead of it at 18:00, ahead of
		// the 22:00 at which it would have starved.
		"Sickness": {
			seed: func(now time.Time) pet.Pet { return pet.New(now.Add(-40 * time.Minute)) },
			want: "Cause: Sickness",
		},
		// Three Care mistakes already on its saved tally, last seen at 50 minutes and
		// left for two hours: its Adult lasts 14 minutes rather than 20, so it dies of
		// Neglect at 54:30, a full 6 minutes before old age would have come.
		"Neglect": {
			seed: func(now time.Time) pet.Pet {
				born := now.Add(-2 * time.Hour)
				p := pet.New(born)
				p.CareMistakes = 3
				last := born.Add(50 * time.Minute)
				p.LastSeenAt, p.HappinessLastSeenAt, p.LastCleanedAt = last, last, last
				return p
			},
			want: "Cause: Neglect",
		},
		// Last looked after at 59 minutes and left for two hours: it outlived every
		// need but ran out of life at 60:30.
		"Old age": {
			seed: func(now time.Time) pet.Pet {
				born := now.Add(-2 * time.Hour)
				p := pet.New(born)
				last := born.Add(59 * time.Minute)
				p.LastSeenAt, p.HappinessLastSeenAt, p.LastCleanedAt = last, last, last
				return p
			},
			want: "Cause: Old age",
		},
	}

	for name, tt := range seeds {
		t.Run(name, func(t *testing.T) {
			savePath := filepath.Join(t.TempDir(), "save.json")
			require.NoError(t, pet.NewFileStore(savePath).Save(tt.seed(started)))

			proc := startBinary(t, bin, savePath)
			waitForOutput(t, proc.out, "Press Enter or click to begin")
			proc.send(t, "\r")
			waitForOutput(t, proc.out, tt.want)
			assert.Contains(t, proc.out.String(), "hatch a new Egg", "and the Restart prompt is offered")

			// The Meters only appear for a living Pet, and this one was dead on
			// arrival, so they can only mean the Restart has hatched a new Egg.
			proc.send(t, "\r")
			waitForOutput(t, proc.out, "Hunger")
			proc.quit(t)

			saved, ok, err := pet.NewFileStore(savePath).Load()
			require.NoError(t, err)
			require.True(t, ok)
			assert.True(t, saved.DiedAt.IsZero(), "the saved Pet is the new one, with no death")
			assert.Zero(t, saved.CareMistakes)
			assert.Equal(t, pet.StageEgg, saved.Stage(time.Now()))
			assert.True(t, saved.CreatedAt.After(started), "born after the test began, not the seeded Pet")
		})
	}
}
