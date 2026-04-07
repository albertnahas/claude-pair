---
description: "Start, stop, or check a pair programming session. Share your live Claude Code terminal with a colleague over SSH. Use /pair to start sharing, /pair stop to end, /pair status to see the join link."
allowed-tools: ["Bash"]
argument-hint: "[start|stop|status]"
---

Parse the argument (default: `start`):

**start** — Run:
```
claude-pair host --discoverable --no-record --bg
```
The `--bg` flag starts the session without attaching to tmux, so it works from within Claude Code. Wait ~5 seconds, then run `claude-pair status` to retrieve the join link. Display the SSH join command prominently. If status shows no join command yet, wait 3 more seconds and retry once.

**stop** — Run `claude-pair stop` and confirm the session has ended.

**status** — Run `claude-pair status` and display the output. Highlight the join command so the user can easily copy and share it.

Keep responses concise. Always show the SSH join command on its own line so it's easy to copy.
