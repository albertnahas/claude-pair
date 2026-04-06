# Roadmap

## v0.4 — Access control and visibility

- **GitHub auth** (`--allow github:user`): restrict session access to specific GitHub usernames via upterm's ACL, instead of sharing a secret token
- **Web viewer** (`--web`): expose a read-only browser view of the session via [ttyd](https://github.com/tsl0922/ttyd) for stakeholders who don't have SSH
- **Claude Code hooks integration**: emit session lifecycle events (start, join, stop) to Claude Code hooks so other tools can react

## v0.5 — Navigator mode

- **Chat sidebar**: add a tmux split pane for the navigator to send messages and suggestions without interrupting the driver's terminal
- **Tool approval gateway**: route Claude's `PreToolUse` hook events to the navigator for explicit approve/deny before destructive operations execute

## v0.6 — Role switching and replay

- **Driver/navigator handoff**: allow roles to switch mid-session with a single command, transferring terminal control without restarting
- **Session replay viewer**: serve recorded `.cast` files as an interactive web replay so sessions can be reviewed after the fact

## v1.0 — Polished release

- **Claude Code plugin** (`/pair` skill): install `claude-pair` as a Claude Code slash command so pairing can be initiated directly from within a Claude session
- **Polished UX**: stable CLI interface, clear error messages, and documented configuration — ready for broad use
