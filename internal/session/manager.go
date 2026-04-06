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
	stateDir     = ".claude-pair"
	stateFile    = "active-session.json"
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
	ID         string `json:"id"`
	Name       string `json:"name"`
	TmuxName   string `json:"tmux_name"`
	SocketPath string `json:"socket_path"`
	SSHURL     string `json:"ssh_url"`
	ROURL      string `json:"ro_url"`
	Recording  string `json:"recording,omitempty"`
	StartedAt  string `json:"started_at"`
	ProjectDir string `json:"project_dir"`
}

// Manager orchestrates the session lifecycle.
type Manager struct {
	cfg   HostConfig
	id    string
	tmux  *Tmux
	tmate *Tmate
	rec   *recording.Recorder
}

// NewManager creates a session manager.
func NewManager(cfg HostConfig) (*Manager, error) {
	id := generateID()
	return &Manager{
		cfg:   cfg,
		id:    id,
		tmux:  NewTmux(sessionPrefix + id),
		tmate: NewTmate(id),
	}, nil
}

// Host starts a shared Claude Code session.
func (m *Manager) Host() error {
	fmt.Println("Starting claude-pair session...")

	// 1. Start tmate (creates its own tmux-like session)
	fmt.Print("  Connecting to tmate relay... ")
	if err := m.tmate.Start(); err != nil {
		return fmt.Errorf("\n  Failed: %w", err)
	}
	fmt.Println("done")

	// 2. Get sharing URLs
	sshURL, err := m.tmate.GetSSHURL()
	if err != nil {
		m.tmate.Kill()
		return fmt.Errorf("getting SSH URL: %w", err)
	}
	roURL, err := m.tmate.GetROURL()
	if err != nil {
		m.tmate.Kill()
		return fmt.Errorf("getting read-only URL: %w", err)
	}

	// 3. Start recording if enabled
	var recordingPath string
	if m.cfg.Record {
		recDir := filepath.Join(homeDir(), stateDir, "recordings")
		_ = os.MkdirAll(recDir, 0o755)
		recordingPath = filepath.Join(recDir, m.id+".cast")
		m.rec = recording.New(recordingPath, 200, 50)
		if err := m.rec.WriteHeader(); err != nil {
			fmt.Printf("  Warning: recording failed to start: %v\n", err)
			m.rec = nil
		} else {
			// Start piping tmate output to the recording file
			if err := run("tmate", "-S", m.tmate.SocketPath, "pipe-pane", "-o",
				fmt.Sprintf("exec cat >> %s", recordingPath)); err != nil {
				fmt.Printf("  Warning: pipe-pane failed: %v\n", err)
			}
		}
	}

	// 4. Launch Claude Code inside tmate
	fmt.Print("  Launching Claude Code... ")
	claudeCmd := fmt.Sprintf("cd %s && claude", m.cfg.ProjectDir)
	if m.cfg.Name != "" {
		claudeCmd = fmt.Sprintf("cd %s && claude --name %q", m.cfg.ProjectDir, m.cfg.Name)
	}
	if err := m.tmate.SendKeys(claudeCmd); err != nil {
		m.tmate.Kill()
		return fmt.Errorf("launching claude: %w", err)
	}
	fmt.Println("done")

	// 5. Save session state
	state := SessionState{
		ID:         m.id,
		Name:       m.cfg.Name,
		TmuxName:   sessionPrefix + m.id,
		SocketPath: m.tmate.SocketPath,
		SSHURL:     sshURL,
		ROURL:      roURL,
		Recording:  recordingPath,
		StartedAt:  time.Now().Format(time.RFC3339),
		ProjectDir: m.cfg.ProjectDir,
	}
	if err := saveState(state); err != nil {
		fmt.Printf("  Warning: could not save session state: %v\n", err)
	}

	// 6. Print invite info
	fmt.Println()
	fmt.Println("  ╔══════════════════════════════════════════════════════╗")
	fmt.Println("  ║              claude-pair session ready               ║")
	fmt.Println("  ╠══════════════════════════════════════════════════════╣")
	fmt.Printf("  ║  Session: %-42s ║\n", m.id)
	fmt.Println("  ╠══════════════════════════════════════════════════════╣")
	fmt.Println("  ║  Share this with your pair:                         ║")
	fmt.Println("  ╚══════════════════════════════════════════════════════╝")
	fmt.Println()
	fmt.Printf("  Navigator (read-only):\n    claude-pair join %s\n\n", roURL)
	fmt.Printf("  Driver (full control):\n    claude-pair join %s\n\n", sshURL)
	if recordingPath != "" {
		fmt.Printf("  Recording to: %s\n\n", recordingPath)
	}

	// 7. Handle signals for clean shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	fmt.Println("  Press Ctrl+C to end the session, or run: claude-pair stop")
	fmt.Println("  Attaching to session...\n")

	// Attach host to the tmate session
	attachCmd := exec.Command("tmate", "-S", m.tmate.SocketPath, "attach-session")
	attachCmd.Stdin = stdinFd()
	attachCmd.Stdout = stdoutFd()
	attachCmd.Stderr = stderrFd()

	// Run attach in background so we can catch signals
	done := make(chan error, 1)
	go func() {
		done <- attachCmd.Run()
	}()

	select {
	case <-sigCh:
		fmt.Println("\n  Shutting down session...")
		m.cleanup()
	case err := <-done:
		// Session ended (user detached or tmate exited)
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
	m.tmate.Kill()
	removeState()
	fmt.Println("  Session ended.")
}

// Join connects to a shared session as navigator.
func Join(link string, displayName string) error {
	// The link is a tmate SSH URL like: ssh XYZ@lon1.tmate.io
	// Parse it and connect via SSH
	parts := strings.Fields(link)
	var sshArgs []string

	if len(parts) >= 2 && parts[0] == "ssh" {
		sshArgs = parts[1:]
	} else if strings.Contains(link, "@") {
		sshArgs = []string{link}
	} else {
		return fmt.Errorf("invalid link format: expected tmate SSH URL (e.g., ssh XYZ@lon1.tmate.io)")
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

	tmate := &Tmate{SocketPath: state.SocketPath, sessionID: state.ID}
	tmate.Kill()
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
	fmt.Printf("Navigator:  claude-pair join %s\n", state.ROURL)
	fmt.Printf("Driver:     claude-pair join %s\n", state.SSHURL)
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
		{"tmate", HasTmate(), "brew install tmate"},
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
	// Simple 6-char hex ID from timestamp + pid
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
