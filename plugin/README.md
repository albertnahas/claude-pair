# claude-pair plugin for Claude Code

Adds a `/pair` slash command to Claude Code for starting, stopping, and checking pair programming sessions.

## Install

```sh
claude plugin add /path/to/claude-pair/plugin
```

## Usage

| Command | Action |
|---|---|
| `/pair` or `/pair start` | Start a discoverable shared session and display the SSH join link |
| `/pair stop` | End the active session |
| `/pair status` | Show session info and the join link |

The plugin shells out to the `claude-pair` binary, which must be on your PATH. Run `claude-pair doctor` to verify dependencies.
