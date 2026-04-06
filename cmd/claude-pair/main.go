package main

import (
	"fmt"
	"os"
	"strings"

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
		noRecord   bool
		name       string
		allowUsers []string
		web        bool
	)

	cmd := &cobra.Command{
		Use:   "host",
		Short: "Start a shared Claude Code session",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := session.HostConfig{
				ProjectDir: mustGetwd(),
				Record:     !noRecord,
				Name:       name,
				AllowUsers: allowUsers,
				Web:        web,
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

	return cmd
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
