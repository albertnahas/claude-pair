# Contributing

Contributions are welcome. Here is how to get started.

## Build

```sh
git clone https://github.com/albertnahas/claude-pair.git
cd claude-pair
go build ./cmd/claude-pair/
```

The binary is placed in the current directory. Move it somewhere on your `$PATH` or use `go install` for a permanent install.

## Test

There is no automated test suite yet (contributions welcome). For manual verification:

1. Run `claude-pair doctor` — all four checks should pass.
2. Open two terminal windows. Run `claude-pair host` in the first.
3. Copy the SSH command it prints and run it in the second terminal.
4. Verify both terminals show the same Claude Code session.
5. Press Ctrl+C in the host terminal and confirm the guest is disconnected cleanly.

## Pull requests

1. Fork the repo and create a branch from `main`.
2. Make your changes. Keep commits focused — one logical change per commit.
3. Run `go fmt ./...` and `go vet ./...` before pushing.
4. Open a PR with a clear description of what changes and why.

For larger changes, open an issue first to discuss the approach.

## Code style

- `go fmt` for formatting — no exceptions
- `go vet` must pass with no warnings
- Keep the dependency footprint small; avoid adding new modules for minor utilities

## Good first issues

Browse issues labeled [`good first issue`](https://github.com/albertnahas/claude-pair/issues?q=is%3Aissue+is%3Aopen+label%3A%22good+first+issue%22) for approachable starting points.
