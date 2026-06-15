# Contributing to undo-anything

Thanks for your interest in contributing! This project aims to be a small,
focused, dependency-light tool that people **trust with their work** —
contributions that keep it small, safe, and fast are especially appreciated.

## Getting started

```bash
git clone https://github.com/agenticraptor/undo-anything
cd undo-anything
go mod tidy        # fetch dependencies & populate go.sum
make build         # build into ./bin/ua
make test          # run the unit tests (with -race)
```

Requirements:

- Go 1.25 or newer (the pinned toolchain is fetched automatically on build)
- (optional) [`golangci-lint`](https://golangci-lint.run/) for `make lint`
- (optional) [`goreleaser`](https://goreleaser.com/) for `make snapshot`

## Development workflow

1. Fork the repo and create a feature branch from `main`.
2. Make your change, with tests where it makes sense.
3. Run the full check suite locally:
   ```bash
   make fmt vet test
   ```
4. Open a pull request. Fill in the PR template and link any related issue.

CI runs `gofmt`, `go vet`, `golangci-lint`, and the test suite on Linux, macOS,
and Windows. All checks must pass before review.

## Commit messages

We use [Conventional Commits](https://www.conventionalcommits.org/). This keeps
the generated changelog readable and drives semantic-version bumps.

```
feat: add a `restore --interactive` flicker-free preview
fix: handle a file deleted between walk and read
docs: clarify the systemd user-service setup
test: cover prune's daily-retention bucketing
chore: bump bubbletea to v0.26.0
```

## Coding guidelines

- **The store is sacred.** Anything that writes to `.undo/` must be atomic
  (temp file + rename) and must never be able to corrupt existing history. If
  you touch `internal/store`, add tests that prove integrity is preserved.
- **Restores are always undoable.** Any new restore path must take a
  pre-restore safety snapshot by default. Never delete a user's file without a
  recoverable snapshot of it first.
- **Ignored files are never touched.** `--clean` and any future destructive
  operation must respect the ignore matcher so that `node_modules/`, build
  output, and the like are always safe.
- **Keep dependencies minimal.** Prefer the standard library; if a new
  dependency is truly needed, call it out in the PR description.
- **Tolerant walking.** A single unreadable file must never abort a whole
  snapshot — skip it and continue.
- **Format with `gofmt -s`** and keep `go vet` clean.

## Good first issues

- A `restore --interactive` file picker.
- Line-level (not just file-level) diffs in `ua show` for text files.
- A `ua export <id> <dir>` command to materialise a snapshot elsewhere.
- Windows service installation (`sc.exe` / Task Scheduler) to match launchd/systemd.
- A `--json` output mode for `ua status` and `ua log`.

## Reporting bugs & requesting features

Use the [issue templates](https://github.com/agenticraptor/undo-anything/issues/new/choose).
For anything security- or data-loss-related, please follow
[SECURITY.md](SECURITY.md) instead of opening a public issue.

## License

By contributing, you agree that your contributions will be licensed under the
[MIT License](LICENSE).
