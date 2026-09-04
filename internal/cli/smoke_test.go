package cli_test

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/creack/pty"
	"github.com/stretchr/testify/require"
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
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if bytes.Contains([]byte(out.String()), []byte(want)) {
			return
		}
		time.Sleep(25 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %q in output; got:\n%s", want, out.String())
}

// TestBinaryLaunchesAndQuits builds the real binary and drives it through a
// pseudo-terminal: it must show the Welcome Screen, advance on Enter, and exit
// cleanly on Ctrl+C.
func TestBinaryLaunchesAndQuits(t *testing.T) {
	if testing.Short() {
		t.Skip("smoke test builds and launches the binary")
	}

	bin := filepath.Join(t.TempDir(), "tamagotchi-go")
	build := exec.Command("go", "build", "-o", bin, "../../cmd/tamagotchi-go")
	build.Stderr = os.Stderr
	require.NoError(t, build.Run(), "building the binary")

	cmd := exec.Command(bin)
	// Keep the save-file load off any real user config directory.
	cmd.Env = append(os.Environ(), "XDG_CONFIG_HOME="+t.TempDir())
	ptmx, err := pty.StartWithSize(cmd, &pty.Winsize{Rows: 30, Cols: 100})
	require.NoError(t, err)
	defer func() { _ = ptmx.Close() }()

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

	waitForOutput(t, out, "Press Enter or click to begin")

	_, err = ptmx.Write([]byte("\r"))
	require.NoError(t, err)
	waitForOutput(t, out, "Hunger")

	_, err = ptmx.Write([]byte{0x03}) // Ctrl+C
	require.NoError(t, err)

	exited := make(chan error, 1)
	go func() { exited <- cmd.Wait() }()

	select {
	case err := <-exited:
		require.NoError(t, err, "binary should exit 0 on Ctrl+C")
	case <-time.After(5 * time.Second):
		_ = cmd.Process.Kill()
		t.Fatal("binary did not exit after Ctrl+C")
	}
}
