package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/albertnahas/claude-pair/internal/session"
)

func main() {
	root := &cobra.Command{
		Use:   "claude-pair",
		Short: "Real-time pair programming with shared Claude Code sessions",
	}

	root.AddCommand(hostCmd(), joinCmd(), stopCmd(), doctorCmd(), statusCmd(), discoverCmd())

	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}

func hostCmd() *cobra.Command {
	var (
		noRecord     bool
		name         string
		allowUsers   []string
		web          bool
		discoverable bool
	)

	cmd := &cobra.Command{
		Use:   "host",
		Short: "Start a shared Claude Code session",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := session.HostConfig{
				ProjectDir:   mustGetwd(),
				Record:       !noRecord,
				Name:         name,
				AllowUsers:   allowUsers,
				Web:          web,
				Discoverable: discoverable,
			}
			mgr, err := session.NewManager(cfg)
			if err != nil {
				return err
			}
			return mgr.Host()
		},
	}

	cmd.Flags().BoolVar(&noRecord, "no-record", false, "Disable session recording")
	cmd.Flags().StringVar(&name, "name", "", "Human-readable session name")
	cmd.Flags().StringSliceVar(&allowUsers, "allow", nil, "Restrict access to GitHub users (e.g., --allow alice --allow bob)")
	cmd.Flags().BoolVar(&web, "web", false, "Launch a browser-accessible viewer via ttyd")
	cmd.Flags().BoolVar(&discoverable, "discoverable", false, "Advertise session on local network via mDNS")

	return cmd
}

func discoverCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "discover",
		Short: "Scan local network for advertised sessions and join one",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("Scanning for sessions on local network...")

			disc := &session.Discovery{}
			sessions, err := disc.Discover(3 * time.Second)
			if err != nil {
				return fmt.Errorf("scan failed: %w", err)
			}

			if len(sessions) == 0 {
				fmt.Println("No sessions found on local network.")
				return nil
			}

			fmt.Println()
			for i, s := range sessions {
				access := "[open]"
				if !s.Open {
					access = fmt.Sprintf("[restricted: %s]", strings.Join(s.AllowUsers, ", "))
				}
				project := filepath.Base(s.Project)
				fmt.Printf("  %d. %s — %s %s — %s\n", i+1, s.Host, project, access, s.JoinCmd)
			}
			fmt.Println()

			var choice int
			if len(sessions) == 1 {
				fmt.Printf("Join session? [y/q]: ")
				scanner := bufio.NewScanner(os.Stdin)
				scanner.Scan()
				input := strings.TrimSpace(scanner.Text())
				if input != "y" && input != "Y" {
					return nil
				}
				choice = 1
			} else {
				fmt.Printf("Join session [1-%d, q to quit]: ", len(sessions))
				scanner := bufio.NewScanner(os.Stdin)
				scanner.Scan()
				input := strings.TrimSpace(scanner.Text())
				if input == "q" || input == "Q" || input == "" {
					return nil
				}
				n, err := strconv.Atoi(input)
				if err != nil || n < 1 || n > len(sessions) {
					return fmt.Errorf("invalid selection: %s", input)
				}
				choice = n
			}

			selected := sessions[choice-1]
			fmt.Println("Joining session...")
			return session.Join(selected.JoinCmd, "")
		},
	}
}

func joinCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                "join <ssh command...>",
		Short:              "Join a session as navigator",
		Args:               cobra.MinimumNArgs(1),
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return session.Join(strings.Join(args, " "), "")
		},
	}
	return cmd
}

func stopCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "stop",
		Short: "End the current session (host only)",
		RunE: func(cmd *cobra.Command, args []string) error {
			return session.Stop()
		},
	}
}

func doctorCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Check dependencies and connectivity",
		Run: func(cmd *cobra.Command, args []string) {
			session.Doctor()
		},
	}
}

func statusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show current session info",
		RunE: func(cmd *cobra.Command, args []string) error {
			return session.Status()
		},
	}
}

func mustGetwd() string {
	dir, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	return dir
}
