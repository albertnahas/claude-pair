package session

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"
)

// UptermSession represents session info from `upterm session current -o json`.
type UptermSession struct {
	SessionID    string `json:"sessionID"`
	Host         string `json:"host"`
	Command      string `json:"command"`
	ForceCommand string `json:"forceCommand"`
	ClientCount  int    `json:"clientCount"`
}

// Upterm wraps upterm commands for session sharing.
type Upterm struct {
	sessionID   string
	adminSocket string
	proc        *exec.Cmd
}

func NewUpterm(sessionID string) *Upterm {
	return &Upterm{
		sessionID:   sessionID,
		adminSocket: adminSocketPath(sessionID),
	}
}

// Host starts an upterm session sharing a tmux session running claude.
// It uses --force-command so all clients attach to the same tmux session.
func (u *Upterm) Host(tmuxSessionName, projectDir, claudeName string) (*exec.Cmd, error) {
	// Build the claude command
	claudeCmd := "claude"
	if claudeName != "" {
		claudeCmd = fmt.Sprintf("claude --name %q", claudeName)
	}

	// Use tmux with force-command so host and all guests share one view
	hostCmd := fmt.Sprintf("cd %s && tmux new-session -s %s '%s'",
		projectDir, tmuxSessionName, claudeCmd)
	forceCmd := fmt.Sprintf("tmux attach-session -t %s", tmuxSessionName)

	args := []string{
		"host",
		"--accept",
		"--force-command", forceCmd,
		"--", "bash", "-c", hostCmd,
	}

	u.proc = exec.Command("upterm", args...)
	u.proc.Stdin = stdinFd()
	u.proc.Stdout = stdoutFd()
	u.proc.Stderr = stderrFd()

	if err := u.proc.Start(); err != nil {
		return nil, fmt.Errorf("starting upterm: %w", err)
	}

	return u.proc, nil
}

// WaitReady polls until upterm session info is available.
func (u *Upterm) WaitReady() error {
	deadline := time.Now().Add(20 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := u.GetSessionInfo(); err == nil {
			return nil
		}
		time.Sleep(500 * time.Millisecond)
	}
	return fmt.Errorf("upterm session did not become ready within 20s")
}

// GetSessionInfo returns the current session info.
func (u *Upterm) GetSessionInfo() (*UptermSession, error) {
	out, err := exec.Command("upterm", "session", "current", "-o", "json").CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("getting session info: %w (%s)", err, string(out))
	}

	var info UptermSession
	if err := json.Unmarshal(out, &info); err != nil {
		return nil, fmt.Errorf("parsing session info: %w", err)
	}
	return &info, nil
}

// GetSSHCommand returns the SSH command guests use to join.
func (u *Upterm) GetSSHCommand() (string, error) {
	info, err := u.GetSessionInfo()
	if err != nil {
		return "", err
	}
	// The sessionID from upterm is the full ssh connection string
	return info.SessionID, nil
}

// Kill terminates the upterm session.
func (u *Upterm) Kill() {
	if u.proc != nil && u.proc.Process != nil {
		_ = u.proc.Process.Signal(os.Interrupt)
		// Give it a moment to clean up
		done := make(chan error, 1)
		go func() { done <- u.proc.Wait() }()
		select {
		case <-done:
		case <-time.After(3 * time.Second):
			_ = u.proc.Process.Kill()
		}
	}
}

// HasUpterm checks if upterm is installed.
func HasUpterm() bool {
	_, err := exec.LookPath("upterm")
	return err == nil
}

// HasClaude checks if claude CLI is installed.
func HasClaude() bool {
	_, err := exec.LookPath("claude")
	return err == nil
}

func adminSocketPath(sessionID string) string {
	if runtime.GOOS == "darwin" {
		return filepath.Join(homeDir(), "Library", "Application Support", "upterm", sessionID+".sock")
	}
	// Linux: /run/user/<UID>/upterm/
	return filepath.Join("/run/user", fmt.Sprintf("%d", os.Getuid()), "upterm", sessionID+".sock")
}
