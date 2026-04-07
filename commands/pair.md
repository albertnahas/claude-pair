---
description: "Start, stop, or check a pair programming session. Share your live Claude Code terminal with a colleague over SSH. Use /pair to start sharing, /pair stop to end, /pair status to see the join link."
allowed-tools: ["Bash"]
argument-hint: "[start|stop|status]"
---

Parse the argument (default: `start`):

**start** — Run `claude-pair host --discoverable --no-record` in the background:
```
claude-pair host --discoverable --no-record &
```
Wait ~5 seconds for upterm to connect, then run `claude-pair status` to retrieve the join link. Display the SSH join command prominently so the user can share it with their colleague. If `claude-pair status` fails (session not ready yet), wait 3 more seconds and retry once.

**stop** — Run `claude-pair stop` and confirm the session has ended.

**status** — Run `claude-pair status` and display the output. Highlight the join command so the user can easily copy and share it.

Keep responses concise. Always show the SSH join command on its own line so it's easy to copy.
