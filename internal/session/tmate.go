package session

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// Tmate wraps tmate commands for session sharing.
type Tmate struct {
	SocketPath string
	sessionID  string
}

func NewTmate(sessionID string) *Tmate {
	return &Tmate{
		SocketPath: fmt.Sprintf("/tmp/claude-pair-%s.sock", sessionID),
		sessionID:  sessionID,
	}
}

// Start launches tmate in daemon mode and waits for it to be ready.
func (t *Tmate) Start() error {
	// Start tmate daemon with its own socket
	if err := run("tmate", "-S", t.SocketPath, "new-session", "-d"); err != nil {
		return fmt.Errorf("starting tmate: %w", err)
	}

	// Wait for tmate to establish connection (up to 15s)
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		if err := run("tmate", "-S", t.SocketPath, "wait", "tmate-ready"); err == nil {
			return nil
		}
		time.Sleep(500 * time.Millisecond)
	}
	return fmt.Errorf("tmate did not become ready within 15s")
}

// GetSSHURL returns the read-write SSH URL for the driver.
func (t *Tmate) GetSSHURL() (string, error) {
	return t.display("#{tmate_ssh}")
}

// GetROURL returns the read-only SSH URL for navigators.
func (t *Tmate) GetROURL() (string, error) {
	return t.display("#{tmate_ssh_ro}")
}

// GetWebURL returns the web URL if available.
func (t *Tmate) GetWebURL() (string, error) {
	return t.display("#{tmate_web}")
}

// GetWebROURL returns the read-only web URL if available.
func (t *Tmate) GetWebROURL() (string, error) {
	return t.display("#{tmate_web_ro}")
}

// SendKeys sends keystrokes to the tmate session.
func (t *Tmate) SendKeys(keys string) error {
	return run("tmate", "-S", t.SocketPath, "send-keys", keys, "Enter")
}

// Kill terminates the tmate session and cleans up the socket.
func (t *Tmate) Kill() error {
	_ = run("tmate", "-S", t.SocketPath, "kill-session")
	_ = os.Remove(t.SocketPath)
	return nil
}

// HasTmate checks if tmate is installed.
func HasTmate() bool {
	_, err := exec.LookPath("tmate")
	return err == nil
}

// HasClaude checks if claude CLI is installed.
func HasClaude() bool {
	_, err := exec.LookPath("claude")
	return err == nil
}

func (t *Tmate) display(format string) (string, error) {
	out, err := exec.Command("tmate", "-S", t.SocketPath, "display", "-p", format).Output()
	if err != nil {
		return "", fmt.Errorf("tmate display %s: %w", format, err)
	}
	return strings.TrimSpace(string(out)), nil
}
