package session

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
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
	logPath     string
}

func NewUpterm(sessionID string) *Upterm {
	logDir := filepath.Join(homeDir(), stateDir, "logs")
	_ = os.MkdirAll(logDir, 0o755)

	return &Upterm{
		sessionID: sessionID,
		logPath:   filepath.Join(logDir, sessionID+".log"),
	}
}

// Host starts an upterm session that creates a tmux session with claude.
// Upterm runs detached (no terminal) so the host can attach to tmux separately.
func (u *Upterm) Host(tmuxSessionName, projectDir, claudeName string) error {
	claudeCmd := "claude"
	if claudeName != "" {
		claudeCmd = fmt.Sprintf("claude --name %q", claudeName)
	}

	// upterm will run tmux in detached mode, then wait forever.
	// The host and guests all attach to the same tmux session independently.
	shellScript := fmt.Sprintf(
		`cd %s && tmux new-session -d -s %s '%s' && while tmux has-session -t %s 2>/dev/null; do sleep 1; done`,
		projectDir, tmuxSessionName, claudeCmd, tmuxSessionName,
	)
	forceCmd := fmt.Sprintf("tmux attach-session -t %s", tmuxSessionName)

	args := []string{
		"host",
		"--accept",
		"--force-command", forceCmd,
		"--", "bash", "-c", shellScript,
	}

	logFile, err := os.Create(u.logPath)
	if err != nil {
		return fmt.Errorf("creating log file: %w", err)
	}

	u.proc = exec.Command("upterm", args...)
	u.proc.Stdout = logFile
	u.proc.Stderr = logFile
	// No stdin — upterm runs headless

	if err := u.proc.Start(); err != nil {
		logFile.Close()
		return fmt.Errorf("starting upterm: %w", err)
	}

	// Don't close logFile — upterm continues writing to it.
	// It will be cleaned up when the process exits.
	return nil
}

// WaitReady polls until upterm session info is available or the process dies.
func (u *Upterm) WaitReady() error {
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		// Check if process died
		if u.proc.ProcessState != nil {
			logs, _ := os.ReadFile(u.logPath)
			return fmt.Errorf("upterm exited unexpectedly:\n%s", string(logs))
		}

		if _, err := u.GetSessionInfo(); err == nil {
			return nil
		}
		time.Sleep(1 * time.Second)
	}
	logs, _ := os.ReadFile(u.logPath)
	return fmt.Errorf("upterm did not become ready within 30s. Logs:\n%s", string(logs))
}

// GetSessionInfo returns the current session info by finding the admin socket.
func (u *Upterm) GetSessionInfo() (*UptermSession, error) {
	socketPath, err := u.findAdminSocket()
	if err != nil {
		return nil, err
	}

	out, err := exec.Command("upterm", "session", "current",
		"--admin-socket", socketPath, "-o", "json").CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("getting session info: %w (%s)", err, string(out))
	}

	var info UptermSession
	if err := json.Unmarshal(out, &info); err != nil {
		return nil, fmt.Errorf("parsing session info: %w", err)
	}
	u.adminSocket = socketPath
	return &info, nil
}

// findAdminSocket discovers the upterm admin socket path.
func (u *Upterm) findAdminSocket() (string, error) {
	// If we already found it, reuse
	if u.adminSocket != "" {
		if _, err := os.Stat(u.adminSocket); err == nil {
			return u.adminSocket, nil
		}
	}

	// Scan the upterm socket directory for .sock files
	socketDir := uptermSocketDir()
	entries, err := os.ReadDir(socketDir)
	if err != nil {
		return "", fmt.Errorf("reading socket dir %s: %w", socketDir, err)
	}

	for _, e := range entries {
		if filepath.Ext(e.Name()) == ".sock" {
			return filepath.Join(socketDir, e.Name()), nil
		}
	}
	return "", fmt.Errorf("no upterm admin socket found in %s", socketDir)
}

func uptermSocketDir() string {
	if runtime.GOOS == "darwin" {
		return filepath.Join(homeDir(), "Library", "Application Support", "upterm")
	}
	return filepath.Join("/run/user", fmt.Sprintf("%d", os.Getuid()), "upterm")
}

// GetSSHCommand returns the SSH command guests use to join.
func (u *Upterm) GetSSHCommand() (string, error) {
	info, err := u.GetSessionInfo()
	if err != nil {
		return "", err
	}
	// Build: ssh <sessionID>@host:port
	// info.Host is like "ssh://uptermd.upterm.dev:22"
	// info.SessionID is the token
	host := strings.TrimPrefix(info.Host, "ssh://")
	// Split host:port
	hostPart := strings.Split(host, ":")[0]
	port := "22"
	if parts := strings.SplitN(host, ":", 2); len(parts) == 2 {
		port = parts[1]
	}
	return fmt.Sprintf("ssh %s@%s -p %s", info.SessionID, hostPart, port), nil
}

// Kill terminates the upterm session.
func (u *Upterm) Kill() {
	if u.proc != nil && u.proc.Process != nil {
		_ = u.proc.Process.Signal(os.Interrupt)
		done := make(chan error, 1)
		go func() { done <- u.proc.Wait() }()
		select {
		case <-done:
		case <-time.After(3 * time.Second):
			_ = u.proc.Process.Kill()
		}
	}
}

// PID returns the upterm process ID.
func (u *Upterm) PID() int {
	if u.proc != nil && u.proc.Process != nil {
		return u.proc.Process.Pid
	}
	return 0
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

