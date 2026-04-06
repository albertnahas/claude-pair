package session

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/albertnahas/claude-pair/internal/recording"
)

const (
	stateDir      = ".claude-pair"
	stateFile     = "active-session.json"
	sessionPrefix = "claude-pair-"
)

// HostConfig holds configuration for hosting a session.
type HostConfig struct {
	ProjectDir string
	Record     bool
	MaxGuests  int
	Name       string
}

// SessionState persists the active session info for stop/status commands.
type SessionState struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	TmuxName  string `json:"tmux_name"`
	JoinCmd   string `json:"join_cmd"`
	Recording string `json:"recording,omitempty"`
	StartedAt string `json:"started_at"`
	ProjectDir string `json:"project_dir"`
	PID       int    `json:"pid,omitempty"`
}

// Manager orchestrates the session lifecycle.
type Manager struct {
	cfg    HostConfig
	id     string
	tmux   *Tmux
	upterm *Upterm
	rec    *recording.Recorder
}

// NewManager creates a session manager.
func NewManager(cfg HostConfig) (*Manager, error) {
	id := generateID()
	return &Manager{
		cfg:    cfg,
		id:     id,
		tmux:   NewTmux(sessionPrefix + id),
		upterm: NewUpterm(id),
	}, nil
}

// Host starts a shared Claude Code session.
func (m *Manager) Host() error {
	fmt.Println("Starting claude-pair session...")

	tmuxName := sessionPrefix + m.id

	// 1. Start recording if enabled
	var recordingPath string
	if m.cfg.Record {
		recDir := filepath.Join(homeDir(), stateDir, "recordings")
		_ = os.MkdirAll(recDir, 0o755)
		recordingPath = filepath.Join(recDir, m.id+".cast")
		m.rec = recording.New(recordingPath, 200, 50)
		if err := m.rec.WriteHeader(); err != nil {
			fmt.Printf("  Warning: recording failed to start: %v\n", err)
			m.rec = nil
		}
	}

	// 2. Start upterm hosting a tmux+claude session
	fmt.Print("  Starting shared session via upterm... ")
	proc, err := m.upterm.Host(tmuxName, m.cfg.ProjectDir, m.cfg.Name)
	if err != nil {
		return fmt.Errorf("\n  Failed: %w", err)
	}
	fmt.Println("started")

	// 3. Wait for upterm to connect to relay
	fmt.Print("  Connecting to relay... ")
	if err := m.upterm.WaitReady(); err != nil {
		m.upterm.Kill()
		return fmt.Errorf("\n  Failed: %w", err)
	}
	fmt.Println("done")

	// 4. Get the join command
	joinCmd, err := m.upterm.GetSSHCommand()
	if err != nil {
		m.upterm.Kill()
		return fmt.Errorf("getting join command: %w", err)
	}

	// 5. Start pipe-pane for recording if enabled
	if m.rec != nil {
		// Wait briefly for tmux session to be created
		time.Sleep(1 * time.Second)
		if err := m.tmux.PipePaneTo(recordingPath); err != nil {
			fmt.Printf("  Warning: recording pipe-pane failed: %v\n", err)
		}
	}

	// 6. Save session state
	state := SessionState{
		ID:         m.id,
		Name:       m.cfg.Name,
		TmuxName:   tmuxName,
		JoinCmd:    joinCmd,
		Recording:  recordingPath,
		StartedAt:  time.Now().Format(time.RFC3339),
		ProjectDir: m.cfg.ProjectDir,
		PID:        proc.Process.Pid,
	}
	if err := saveState(state); err != nil {
		fmt.Printf("  Warning: could not save session state: %v\n", err)
	}

	// 7. Print invite info
	fmt.Println()
	fmt.Println("  ╔══════════════════════════════════════════════════════╗")
	fmt.Println("  ║              claude-pair session ready               ║")
	fmt.Println("  ╠══════════════════════════════════════════════════════╣")
	fmt.Printf("  ║  Session: %-42s ║\n", m.id)
	fmt.Println("  ╠══════════════════════════════════════════════════════╣")
	fmt.Println("  ║  Share this with your pair:                         ║")
	fmt.Println("  ╚══════════════════════════════════════════════════════╝")
	fmt.Println()
	fmt.Printf("  Join command:\n    claude-pair join %s\n\n", joinCmd)
	if recordingPath != "" {
		fmt.Printf("  Recording to: %s\n\n", recordingPath)
	}

	// 8. Handle signals for clean shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	fmt.Println("  Press Ctrl+C to end the session, or run: claude-pair stop")
	fmt.Println()

	// Wait for upterm process to exit or signal
	done := make(chan error, 1)
	go func() {
		done <- proc.Wait()
	}()

	select {
	case <-sigCh:
		fmt.Println("\n  Shutting down session...")
		m.cleanup()
	case err := <-done:
		m.cleanup()
		if err != nil {
			return nil // Normal exit when session ends
		}
	}

	return nil
}

func (m *Manager) cleanup() {
	if m.rec != nil {
		m.rec.Close()
	}
	m.upterm.Kill()
	// Kill the tmux session too
	m.tmux.KillSession()
	removeState()
	fmt.Println("  Session ended.")
}

// Join connects to a shared session.
func Join(link string, displayName string) error {
	// The link is an SSH command like: ssh TOKEN@uptermd.upterm.dev
	parts := strings.Fields(link)
	var sshArgs []string

	if len(parts) >= 2 && parts[0] == "ssh" {
		sshArgs = parts[1:]
	} else if strings.Contains(link, "@") {
		sshArgs = []string{link}
	} else {
		return fmt.Errorf("invalid link: expected SSH command (e.g., ssh TOKEN@uptermd.upterm.dev)")
	}

	fmt.Printf("Joining session as %s...\n", nameOrDefault(displayName, "navigator"))

	cmd := exec.Command("ssh", sshArgs...)
	cmd.Stdin = stdinFd()
	cmd.Stdout = stdoutFd()
	cmd.Stderr = stderrFd()
	return cmd.Run()
}

// Stop terminates the active session.
func Stop() error {
	state, err := loadState()
	if err != nil {
		return fmt.Errorf("no active session found")
	}

	// Kill the upterm process
	if state.PID > 0 {
		if proc, err := os.FindProcess(state.PID); err == nil {
			_ = proc.Signal(syscall.SIGTERM)
		}
	}

	// Kill the tmux session
	tmux := NewTmux(state.TmuxName)
	tmux.KillSession()

	removeState()
	fmt.Printf("Session %s stopped.\n", state.ID)
	return nil
}

// Status shows the current session info.
func Status() error {
	state, err := loadState()
	if err != nil {
		fmt.Println("No active session.")
		return nil
	}

	fmt.Printf("Session:    %s\n", state.ID)
	if state.Name != "" {
		fmt.Printf("Name:       %s\n", state.Name)
	}
	fmt.Printf("Started:    %s\n", state.StartedAt)
	fmt.Printf("Project:    %s\n", state.ProjectDir)
	fmt.Printf("Join:       claude-pair join %s\n", state.JoinCmd)
	if state.Recording != "" {
		fmt.Printf("Recording:  %s\n", state.Recording)
	}
	return nil
}

// Doctor checks all dependencies.
func Doctor() {
	checks := []struct {
		name string
		ok   bool
		hint string
	}{
		{"tmux", HasTmux(), "brew install tmux"},
		{"upterm", HasUpterm(), "brew install --cask owenthereal/upterm/upterm"},
		{"claude", HasClaude(), "See https://docs.anthropic.com/en/docs/claude-code"},
		{"ssh", hasSSH(), "Should be pre-installed on macOS/Linux"},
	}

	allOK := true
	for _, c := range checks {
		status := "OK"
		if !c.ok {
			status = "MISSING"
			allOK = false
		}
		fmt.Printf("  %-10s %s", c.name, status)
		if !c.ok {
			fmt.Printf("  (install: %s)", c.hint)
		}
		fmt.Println()
	}

	if allOK {
		fmt.Println("\n  All dependencies satisfied. Ready to pair!")
	} else {
		fmt.Println("\n  Install missing dependencies before using claude-pair.")
	}
}

// --- helpers ---

func generateID() string {
	return fmt.Sprintf("%x", time.Now().UnixNano()%0xFFFFFF)
}

func homeDir() string {
	h, _ := os.UserHomeDir()
	return h
}

func statePath() string {
	return filepath.Join(homeDir(), stateDir, stateFile)
}

func saveState(s SessionState) error {
	dir := filepath.Join(homeDir(), stateDir)
	_ = os.MkdirAll(dir, 0o755)

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(statePath(), data, 0o644)
}

func loadState() (*SessionState, error) {
	data, err := os.ReadFile(statePath())
	if err != nil {
		return nil, err
	}
	var s SessionState
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, err
	}
	return &s, nil
}

func removeState() {
	_ = os.Remove(statePath())
}

func hasSSH() bool {
	_, err := exec.LookPath("ssh")
	return err == nil
}

func nameOrDefault(name, fallback string) string {
	if name != "" {
		return name
	}
	return fallback
}
