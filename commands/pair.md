---
description: "Start, stop, join, or check a pair programming session. Share your live Claude Code terminal with a colleague over SSH. Use /pair to start sharing, /pair join to hop into a session, /pair stop to end, /pair status to see the join link."
allowed-tools: ["Bash"]
argument-hint: "[start|stop|status|join|discover] [--allow user] [--web]"
---

Parse the argument. The first word is the subcommand (default: `start`). Any remaining words are flags or arguments.

**start** (default) — Build the command:
```
claude-pair host --discoverable --no-record --bg [extra flags]
```
Supported extra flags the user can pass:
- `--allow <user>` (repeatable) — restrict to GitHub users
- `--web` — launch browser viewer on localhost:7681
- `--name <name>` — session name

Examples:
- `/pair` → start with defaults
- `/pair --allow alice --web` → restricted + web viewer

After running, wait ~5 seconds, then run `claude-pair status` to retrieve the join link. Display the SSH join command prominently. If status shows no join command yet, wait 3 more seconds and retry once.

**join** — Join a session from within Claude Code. Opens a new Terminal window.

If the user provides a link (e.g., `/pair join ssh TOKEN@uptermd.upterm.dev`), extract the SSH target (the user@host part). If no link provided, run `claude-pair status` to get the join link from the active session.

Open it in a new Terminal window by creating a temp script and opening it:
```bash
SCRIPT=$(mktemp /tmp/claude-pair-join-XXXXX.sh)
echo '#!/bin/bash' > "$SCRIPT"
echo 'ssh -o StrictHostKeyChecking=accept-new -o UserKnownHostsFile=/dev/null TOKEN@uptermd.upterm.dev' >> "$SCRIPT"
chmod +x "$SCRIPT"
open -a Terminal "$SCRIPT"
```

Tell the user: "Opened a new Terminal window with the shared session."

**discover** — Scan the local network for sessions. Run `claude-pair discover` in background mode or tell the user to run `cpd` in a terminal, as discover requires interactive stdin.

**stop** — Run `claude-pair stop` and confirm the session has ended.

**status** — Run `claude-pair status` and display the output. Highlight the join command so the user can easily copy and share it.

Keep responses concise. Always show the SSH join command on its own line for easy copying.
