---
description: "Start, stop, or check a pair programming session. Share your live Claude Code terminal with a colleague over SSH. Use /pair to start sharing, /pair stop to end, /pair status to see the join link."
allowed-tools: ["Bash"]
argument-hint: "[start|stop|status] [--allow user] [--web]"
---

Parse the argument. The first word is the subcommand (default: `start`). Any remaining words are flags passed through to `claude-pair host`.

**start** (default) — Build the command:
```
claude-pair host --discoverable --no-record --bg [extra flags]
```
Supported extra flags the user can pass:
- `--allow <user>` (repeatable) — restrict to GitHub users
- `--web` — launch browser viewer on localhost:7681
- `--name <name>` — session name

Examples:
- `/pair` → `claude-pair host --discoverable --no-record --bg`
- `/pair start --allow alice --allow bob` → `claude-pair host --discoverable --no-record --bg --allow alice --allow bob`
- `/pair --web` → `claude-pair host --discoverable --no-record --bg --web`
- `/pair --allow alice --web` → `claude-pair host --discoverable --no-record --bg --allow alice --web`

After running, wait ~5 seconds, then run `claude-pair status` to retrieve the join link. Display the SSH join command prominently. If status shows no join command yet, wait 3 more seconds and retry once.

**stop** — Run `claude-pair stop` and confirm the session has ended.

**status** — Run `claude-pair status` and display the output. Highlight the join command so the user can easily copy and share it.

Keep responses concise. Always show the SSH join command on its own line for easy copying.
