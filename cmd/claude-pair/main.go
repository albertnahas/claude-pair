package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/albertnahas/claude-pair/internal/session"
)

func main() {
	root := &cobra.Command{
		Use:   "claude-pair",
		Short: "Real-time pair programming with shared Claude Code sessions",
	}

	root.AddCommand(hostCmd(), joinCmd(), stopCmd(), doctorCmd(), statusCmd())

	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}

func hostCmd() *cobra.Command {
	var (
		noRecord  bool
		maxGuests int
		name      string
	)

	cmd := &cobra.Command{
		Use:   "host",
		Short: "Start a shared Claude Code session",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := session.HostConfig{
				ProjectDir: mustGetwd(),
				Record:     !noRecord,
				MaxGuests:  maxGuests,
				Name:       name,
			}
			mgr, err := session.NewManager(cfg)
			if err != nil {
				return err
			}
			return mgr.Host()
		},
	}

	cmd.Flags().BoolVar(&noRecord, "no-record", false, "Disable session recording")
	cmd.Flags().IntVar(&maxGuests, "max-guests", 3, "Maximum concurrent navigators")
	cmd.Flags().StringVar(&name, "name", "", "Human-readable session name")

	return cmd
}

func joinCmd() *cobra.Command {
	var displayName string

	cmd := &cobra.Command{
		Use:   "join <link>",
		Short: "Join a session as navigator (read-only)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return session.Join(args[0], displayName)
		},
	}

	cmd.Flags().StringVar(&displayName, "name", "", "Your display name")

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
