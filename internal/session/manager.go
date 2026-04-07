package session

import (
	crypto_rand "crypto/rand"
	"encoding/hex"
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
	ProjectDir   string
	Record       bool
	Name         string
	AllowUsers   []string
	Web          bool
	Discoverable bool
}

// SessionState persists the active session info for stop/status commands.
type SessionState struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	TmuxName   string `json:"tmux_name"`
	JoinCmd    string `json:"join_cmd"`
	Recording  string `json:"recording,omitempty"`
	StartedAt  string `json:"started_at"`
	ProjectDir string `json:"project_dir"`
	PID        int    `json:"pid,omitempty"`
	WebURL     string `json:"web_url,omitempty"`
}

// Manager orchestrates the session lifecycle.
type Manager struct {
	cfg    HostConfig
	id     string
	tmux   *Tmux
	upterm *Upterm
	rec    *recording.Recorder
	web    *WebViewer
	disc   *Discovery
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

	// 2. Start upterm headless (it creates a detached tmux session)
	fmt.Print("  Starting shared session via upterm... ")
	if err := m.upterm.Host(tmuxName, m.cfg.ProjectDir, m.cfg.Name, m.cfg.AllowUsers); err != nil {
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

	// 4b. Start mDNS advertisement if requested
	if m.cfg.Discoverable {
		m.disc = &Discovery{}
		if err := m.disc.Advertise(joinCmd, m.cfg.ProjectDir, m.cfg.AllowUsers); err != nil {
			fmt.Printf("  Warning: mDNS advertisement failed: %v\n", err)
			m.disc = nil
		} else {
			fmt.Println("  Advertising on local network via mDNS")
		}
	}

	// 5. Wait for tmux session to be created, then configure it
	if err := m.waitTmuxReady(10 * time.Second); err != nil {
		fmt.Printf("  Warning: tmux session not detected: %v\n", err)
	}
	if err := m.tmux.SetWindowSize("latest"); err != nil {
		fmt.Printf("  Warning: could not set tmux window-size: %v\n", err)
	}
	if err := m.tmux.SetStatusBar(joinCmd, m.cfg.AllowUsers); err != nil {
		fmt.Printf("  Warning: could not set tmux status bar: %v\n", err)
	}
	if m.rec != nil {
		if err := m.tmux.PipePaneTo(recordingPath); err != nil {
			fmt.Printf("  Warning: recording pipe-pane failed: %v\n", err)
		}
	}

	// 5b. Start web viewer if requested
	var webURL string
	if m.cfg.Web {
		if !HasTtyd() {
			fmt.Println("  Warning: ttyd not found — skipping web viewer (brew install ttyd)")
		} else {
			fmt.Print("  Starting web viewer via ttyd... ")
			m.web = &WebViewer{}
			url, err := m.web.Start(tmuxName)
			if err != nil {
				fmt.Printf("Warning: %v\n", err)
				m.web = nil
			} else {
				webURL = url
				fmt.Println("started")
			}
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
		PID:        m.upterm.PID(),
		WebURL:     webURL,
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
	if len(m.cfg.AllowUsers) > 0 {
		fmt.Printf("  ║  Access:  restricted to: %-27s ║\n", strings.Join(m.cfg.AllowUsers, ", "))
	} else {
		fmt.Println("  ║  Access:  open (anyone with the link can join)       ║")
	}
	fmt.Println("  ╠══════════════════════════════════════════════════════╣")
	fmt.Println("  ║  Share this with your pair:                         ║")
	fmt.Println("  ╚══════════════════════════════════════════════════════╝")
	fmt.Println()
	fmt.Printf("  Guest runs:\n    %s\n\n", joinCmd)
	if webURL != "" {
		fmt.Printf("  Web viewer:   %s\n\n", webURL)
	}
	if recordingPath != "" {
		fmt.Printf("  Recording to: %s\n\n", recordingPath)
	}

	// 8. Handle signals for clean shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	fmt.Println("  Press Ctrl+C to end the session, or run: claude-pair stop")
	fmt.Println("  Attaching to session...")
	fmt.Println()

	// Attach host to the tmux session (upterm runs in background)
	attachDone := make(chan error, 1)
	go func() {
		attachDone <- m.tmux.AttachSession()
	}()

	select {
	case <-sigCh:
		fmt.Println("\n  Shutting down session...")
		m.cleanup()
	case err := <-attachDone:
		// Host detached from tmux or tmux session ended
		m.cleanup()
		if err != nil {
			return nil
		}
	}

	return nil
}

func (m *Manager) waitTmuxReady(timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if m.tmux.SessionExists() {
			return nil
		}
		time.Sleep(200 * time.Millisecond)
	}
	return fmt.Errorf("tmux session %s not ready after %s", m.tmux.SessionName, timeout)
}

func (m *Manager) cleanup() {
	if m.disc != nil {
		m.disc.Stop()
	}
	if m.web != nil {
		m.web.Stop()
	}
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

	// Auto-accept host key to avoid the fingerprint prompt
	fullArgs := append([]string{
		"-o", "StrictHostKeyChecking=accept-new",
		"-o", "UserKnownHostsFile=/dev/null",
	}, sshArgs...)

	cmd := exec.Command("ssh", fullArgs...)
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
	fmt.Printf("Join:       %s\n", state.JoinCmd)
	if state.WebURL != "" {
		fmt.Printf("Web:        %s\n", state.WebURL)
	}
	if state.Recording != "" {
		fmt.Printf("Recording:  %s\n", state.Recording)
	}
	return nil
}

// Doctor checks all dependencies.
func Doctor() {
	required := []struct {
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
	for _, c := range required {
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

	ttydStatus := "not installed"
	if HasTtyd() {
		ttydStatus = "OK"
	}
	fmt.Printf("\n  Optional:\n")
	fmt.Printf("  %-10s %-15s (required for --web; install: brew install ttyd)\n", "ttyd", ttydStatus)

	fmt.Println()
	if allOK {
		fmt.Println("  All required dependencies satisfied. Ready to pair!")
	} else {
		fmt.Println("  Install missing dependencies before using claude-pair.")
	}
}

// --- helpers ---

func generateID() string {
	b := make([]byte, 6)
	_, _ = crypto_rand.Read(b)
	return hex.EncodeToString(b)
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
	return os.WriteFile(statePath(), data, 0o600)
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
