# claude-pair

Share a live Claude Code session with a collaborator over SSH.

![claude-pair demo](demo.png)

## What it does

- Share a live Claude Code session — guest joins with a single `ssh` command, no install needed
- Discover sessions on your local network automatically (`--discoverable` + `claude-pair discover`)
- Restrict access to specific GitHub users via SSH key verification (`--allow`)
- Browser-based viewer via ttyd (`--web`) for non-technical observers
- Session recording in asciinema-compatible format

## Quick start

**Host** (starts the session):
```sh
claude-pair host
```

**Restrict to specific GitHub users:**
```sh
claude-pair host --allow alice --allow bob
```

**Make it discoverable on your local network:**
```sh
claude-pair host --discoverable
```

**Add a browser-accessible viewer:**
```sh
claude-pair host --web
```

**Guest** (joins with the SSH command the host shares):
```sh
ssh TOKEN@uptermd.upterm.dev
```

Or discover sessions on the local network:
```sh
claude-pair discover
```

## Install

```sh
curl -fsSL https://raw.githubusercontent.com/albertnahas/claude-pair/main/install.sh | sh
```

Or with Go:
```sh
go install github.com/albertnahas/claude-pair/cmd/claude-pair@latest
```

```sh
brew install albertnahas/tap/claude-pair
```

The installer handles all dependencies (upterm, tmux, ttyd). Run `claude-pair doctor` to verify.

## How it works

`claude-pair host` creates a tmux session, launches Claude Code inside it, and connects upterm to relay it over SSH. The guest runs a plain `ssh` command — no client install required. The join URL is displayed in the host terminal and embedded in the tmux status bar for easy reference.

## Roadmap

See [ROADMAP.md](ROADMAP.md).

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md).

---

[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
