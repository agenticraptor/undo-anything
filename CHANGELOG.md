# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.1.0] - 2026-06-15

### Added

- Initial release. 🎉
- Content-addressed, deduplicated, zlib-compressed object store with atomic
  writes and on-read integrity verification.
- Snapshot engine that captures a whole folder, skipping ignored paths, and
  collapses identical states to a single snapshot.
- `init` — start tracking a folder and capture a baseline snapshot.
- `watch` — foreground watcher that snapshots on every (debounced) change.
- `daemon start|stop|status|install|uninstall` — run the watcher in the
  background, or install it as a launchd (macOS) / systemd user (Linux) service.
- `snapshot` — capture a labelled snapshot on demand.
- `log`, `show`, `diff` — browse history and inspect what changed.
- `timeline` — an interactive Bubble Tea TUI to scrub snapshots and restore with
  one key.
- `restore` — restore individual files or the whole tree to any snapshot, with
  an automatic pre-restore safety snapshot so every restore is itself undoable;
  optional `--clean` that only ever removes tracked files.
- `prune` — thin history per a configurable retention policy and garbage-collect
  unreferenced objects.
- `status` and `doctor` — store statistics, a space-savings figure, watcher
  health, and environment checks.
- gitignore-style ignore matching via built-in defaults, a `.uaignore` file, and
  (optionally) the folder's own `.gitignore`.
- Single static binary, zero config, fully local — no network access.

[Unreleased]: https://github.com/agenticraptor/undo-anything/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/agenticraptor/undo-anything/releases/tag/v0.1.0
