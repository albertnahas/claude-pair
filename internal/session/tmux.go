package session

import (
	"fmt"
	"os/exec"
	"strings"
)

// Tmux wraps tmux commands for session management.
type Tmux struct {
	SessionName string
}

func NewTmux(sessionName string) *Tmux {
	return &Tmux{SessionName: sessionName}
}

// CreateSession creates a detached tmux session.
func (t *Tmux) CreateSession() error {
	return run("tmux", "new-session", "-d", "-s", t.SessionName, "-x", "200", "-y", "50")
}

// SendKeys sends keystrokes to the tmux session.
func (t *Tmux) SendKeys(keys string) error {
	return run("tmux", "send-keys", "-t", t.SessionName, keys, "Enter")
}

// PipePaneTo starts piping pane output to a file.
func (t *Tmux) PipePaneTo(path string) error {
	return run("tmux", "pipe-pane", "-t", t.SessionName, "-o", fmt.Sprintf("exec cat >> %s", path))
}

// KillSession destroys the tmux session.
func (t *Tmux) KillSession() error {
	return run("tmux", "kill-session", "-t", t.SessionName)
}

// SessionExists checks if a tmux session with the given name exists.
func (t *Tmux) SessionExists() bool {
	err := run("tmux", "has-session", "-t", t.SessionName)
	return err == nil
}

// AttachSession attaches to an existing tmux session.
func (t *Tmux) AttachSession() error {
	cmd := exec.Command("tmux", "attach-session", "-t", t.SessionName)
	cmd.Stdin = stdinFd()
	cmd.Stdout = stdoutFd()
	cmd.Stderr = stderrFd()
	return cmd.Run()
}

// SetStatusBar configures the tmux status bar with a persistent message.
func (t *Tmux) SetStatusBar(joinCmd string) error {
	// Show join command in the bottom status bar so it's always visible
	cmds := [][]string{
		{"set-option", "-t", t.SessionName, "status", "on"},
		{"set-option", "-t", t.SessionName, "status-style", "bg=#1a1a2e,fg=#e0e0e0"},
		{"set-option", "-t", t.SessionName, "status-left", " claude-pair "},
		{"set-option", "-t", t.SessionName, "status-left-style", "bg=#6c5ce7,fg=#ffffff,bold"},
		{"set-option", "-t", t.SessionName, "status-left-length", "15"},
		{"set-option", "-t", t.SessionName, "status-right", fmt.Sprintf(" Pair: %s ", joinCmd)},
		{"set-option", "-t", t.SessionName, "status-right-style", "bg=#2d3436,fg=#74b9ff"},
		{"set-option", "-t", t.SessionName, "status-right-length", "120"},
	}
	for _, args := range cmds {
		if err := run("tmux", args...); err != nil {
			return err
		}
	}
	return nil
}

// HasTmux checks if tmux is installed.
func HasTmux() bool {
	_, err := exec.LookPath("tmux")
	return err == nil
}

func run(name string, args ...string) error {
	out, err := exec.Command(name, args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s %s: %w\n%s", name, strings.Join(args, " "), err, string(out))
	}
	return nil
}
