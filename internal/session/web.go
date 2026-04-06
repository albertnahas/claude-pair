package session

import (
	"fmt"
	"os"
	"os/exec"
	"time"
)

const ttydDefaultPort = 7681

// WebViewer manages a ttyd process for browser-based session viewing.
type WebViewer struct {
	proc *exec.Cmd
	died chan struct{}
}

// HasTtyd checks if ttyd is installed.
func HasTtyd() bool {
	_, err := exec.LookPath("ttyd")
	return err == nil
}

// Start launches ttyd in read-only mode attached to the given tmux session.
// Returns the URL viewers can open in a browser.
func (w *WebViewer) Start(tmuxSessionName string) (string, error) {
	w.died = make(chan struct{})

	w.proc = exec.Command("ttyd",
		"-R",
		"-p", fmt.Sprintf("%d", ttydDefaultPort),
		"tmux", "attach-session", "-r", "-t", tmuxSessionName,
	)
	w.proc.Stdout = os.Stderr // ttyd logs to stderr; route to our stderr
	w.proc.Stderr = os.Stderr

	if err := w.proc.Start(); err != nil {
		return "", fmt.Errorf("starting ttyd: %w", err)
	}

	go func() {
		_ = w.proc.Wait()
		close(w.died)
	}()

	// Give ttyd a moment to bind the port before reporting success.
	select {
	case <-w.died:
		return "", fmt.Errorf("ttyd exited immediately — port %d may already be in use", ttydDefaultPort)
	case <-time.After(500 * time.Millisecond):
	}

	return fmt.Sprintf("http://localhost:%d", ttydDefaultPort), nil
}

// Stop kills the ttyd process.
func (w *WebViewer) Stop() {
	if w.proc == nil || w.proc.Process == nil {
		return
	}
	_ = w.proc.Process.Signal(os.Interrupt)
	select {
	case <-w.died:
	case <-time.After(3 * time.Second):
		_ = w.proc.Process.Kill()
		<-w.died
	}
}
