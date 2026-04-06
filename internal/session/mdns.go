package session

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/hashicorp/mdns"
)

const mdnsService = "_claude-pair._tcp"

// DiscoveredSession represents a session found on the local network.
type DiscoveredSession struct {
	Name       string
	JoinCmd    string
	Project    string
	Host       string
	AllowUsers []string
	Open       bool
}

// Discovery manages mDNS advertisement and scanning.
type Discovery struct {
	server *mdns.Server
}

// Advertise registers an mDNS service for the current session.
func (d *Discovery) Advertise(joinCmd, projectDir string, allowUsers []string) error {
	hostname, _ := os.Hostname()

	allowVal := "open"
	if len(allowUsers) > 0 {
		allowVal = strings.Join(allowUsers, ",")
	}

	txt := []string{
		"join=" + joinCmd,
		"project=" + projectDir,
		"user=" + hostname,
		"allow=" + allowVal,
	}

	svc, err := mdns.NewMDNSService(hostname, mdnsService, "", "", 1, nil, txt)
	if err != nil {
		return fmt.Errorf("creating mDNS service: %w", err)
	}

	logger := log.New(log.Writer(), "", 0)
	srv, err := mdns.NewServer(&mdns.Config{Zone: svc, Logger: logger})
	if err != nil {
		return fmt.Errorf("starting mDNS server: %w", err)
	}

	d.server = srv
	return nil
}

// Stop shuts down the mDNS advertisement.
func (d *Discovery) Stop() {
	if d.server != nil {
		_ = d.server.Shutdown()
		d.server = nil
	}
}

// Discover scans the local network for claude-pair sessions.
func (d *Discovery) Discover(timeout time.Duration) ([]DiscoveredSession, error) {
	entries := make(chan *mdns.ServiceEntry, 16)
	params := mdns.DefaultParams(mdnsService)
	params.Timeout = timeout
	params.Entries = entries
	// Suppress mDNS library log output during scan
	params.Logger = log.New(log.Writer(), "", 0)

	go func() {
		_ = mdns.Query(params)
		close(entries)
	}()

	var sessions []DiscoveredSession
	seen := map[string]bool{}

	for entry := range entries {
		joinCmd := ""
		project := ""
		host := entry.Host
		allowVal := ""

		for _, field := range entry.InfoFields {
			k, v, ok := strings.Cut(field, "=")
			if !ok {
				continue
			}
			switch k {
			case "join":
				joinCmd = v
			case "project":
				project = v
			case "user":
				host = v
			case "allow":
				allowVal = v
			}
		}

		if joinCmd == "" || seen[joinCmd] {
			continue
		}
		seen[joinCmd] = true

		s := DiscoveredSession{
			Name:    entry.Name,
			JoinCmd: joinCmd,
			Project: project,
			Host:    host,
		}
		if allowVal == "" || allowVal == "open" {
			s.Open = true
		} else {
			s.AllowUsers = strings.Split(allowVal, ",")
		}
		sessions = append(sessions, s)
	}

	return sessions, nil
}
