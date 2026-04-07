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

**join** — Join a session from within Claude Code. Opens a new Terminal window with the SSH connection.

If the user provides a link (e.g., `/pair join ssh TOKEN@uptermd.upterm.dev`), extract the SSH command and open it:
```bash
osascript -e 'tell application "Terminal" to do script "ssh TOKEN@uptermd.upterm.dev"'
```

If no link is provided (`/pair join`), first discover sessions on the local network:
```bash
claude-pair discover --json 2>/dev/null || true
```
If that doesn't work or isn't available, tell the user to provide the SSH link: `/pair join ssh TOKEN@host`

If a session is found, open it in a new Terminal window using the osascript command above.

Tell the user: "Opened a new Terminal window with the shared session."

**discover** — Scan the local network and open the first found session in a new Terminal window:
```bash
claude-pair discover --json 2>/dev/null
```
If sessions are found, pick the first one and open it with osascript. Otherwise tell the user no sessions were found.

**stop** — Run `claude-pair stop` and confirm the session has ended.

**status** — Run `claude-pair status` and display the output. Highlight the join command so the user can easily copy and share it.

Keep responses concise. Always show the SSH join command on its own line for easy copying.
