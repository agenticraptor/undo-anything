<div align="center">

# ⏪ undo-anything

### A universal local time machine for any folder — with one-key restore.

`undo-anything` quietly snapshots your folder **on every save** — content-addressed,
deduplicated, and fully local — so you can restore any file, or the whole tree,
to **any point in time**. Including the sweeping edit your AI agent made and you
already accepted. **Never lose work again.**

[![CI](https://github.com/agenticraptor/undo-anything/actions/workflows/ci.yml/badge.svg)](https://github.com/agenticraptor/undo-anything/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/agenticraptor/undo-anything?sort=semver)](https://github.com/agenticraptor/undo-anything/releases)
[![Go Reference](https://pkg.go.dev/badge/github.com/agenticraptor/undo-anything.svg)](https://pkg.go.dev/github.com/agenticraptor/undo-anything)
[![Go Report Card](https://goreportcard.com/badge/github.com/agenticraptor/undo-anything)](https://goreportcard.com/report/github.com/agenticraptor/undo-anything)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

</div>

---

> **Try it in one line — no signup, no config, no network:**
>
> ```bash
> go run github.com/agenticraptor/undo-anything/cmd/ua@latest init
> ```
>
> That starts tracking the current folder and captures a baseline you can
> already restore to. Then `ua daemon start` to snapshot on every save.

<!--
  📸 Replace this block with a 15–20s screen-capture GIF of `ua timeline` —
  scrubbing snapshots on the left, the diff updating on the right, then hitting
  `r` to restore. The hero GIF is the single biggest driver of stars; record it
  once and drop it at docs/demo.gif, then uncomment:

  <p align="center"><img src="docs/demo.gif" alt="undo-anything timeline demo" width="760"></p>
-->

```text
⏪ undo-anything timeline   142 snapshots · .../code/api

╭────────────────────────────────────╮ ╭──────────────────────────────────────╮
│ ● 92ac1bd72b54  just now            │ │ Snapshot 92ac1bd72b54                 │
│   4 files                           │ │ taken  2026-06-15 23:13:15 (just now) │
│ ● c12288b7b17a  4m ago              │ │ kind   watch                          │
│   before restore to bd304373        │ │ files  4 (98 B)                       │
│ ● 056d1f13143a  1h ago              │ │                                       │
│   refactor: extract retry client    │ │ Changes vs parent                     │
│ ● bd304373c46f  3h ago              │ │ + internal/retry/retry.go             │
│   initial snapshot                  │ │ ~ internal/payments/client.go         │
│                                     │ │ - internal/payments/old_client.go     │
╰────────────────────────────────────╯ ╰──────────────────────────────────────╯
 ↑/↓ move · r restore · pgup/pgdn scroll · q quit
```

## The problem

You're coding with an AI agent. It makes a big, mostly-good change across a dozen
files. You accept it. Ten minutes later you realize it quietly rewrote a function
you needed — and it's **not in git** (you hadn't committed), **not in your
editor's undo** (you reopened the file), and **not in any diff** you can find.

Existing safety nets don't cover this gap:

| Tool | Why it falls short |
|------|--------------------|
| **git** | Manual, commit-scoped, repo-only. The work you lose is usually the *uncommitted* work. |
| **Time Machine** | macOS-only, not open source, hourly, slow, not per-folder or scriptable. |
| **Editor "local history"** | Locked inside one editor; gone if you switch tools or work outside it. |

`undo-anything` fills it: a tiny, zero-config daemon that snapshots **any** folder
the instant a file changes, and lets you jump back to **any second** — one
command, fully local.

## Why you'll like it

- ⏱️ **Restore any file to any moment.** `ua restore <id> path/to/file` — or the
  whole tree. Time travel for folders.
- 🛟 **Every restore is itself undoable.** Before any restore, `undo-anything`
  captures an automatic *safety snapshot*. Restored the wrong thing? Undo the undo.
- 🤖 **Built for the AI-agent era.** Snapshots run independently of git and your
  editor, so the edits you "accepted and forgot" are always recoverable.
- 🪶 **Tiny and deduplicated.** Identical content is stored once and compressed.
  Hundreds of snapshots typically cost a few MB — see the savings figure in
  `ua status`.
- 🔒 **100% local.** No accounts, no telemetry, no network. Your code never
  leaves your machine. ([See for yourself](#privacy).)
- 🖥️ **A gorgeous timeline TUI.** Scrub history and restore with one key.
- 📦 **One static binary.** Zero config to start; sensible defaults everywhere.

## Install

### `go install`

```bash
go install github.com/agenticraptor/undo-anything/cmd/ua@latest
```

### Pre-built binaries

Grab a binary for your OS/arch from the
[**Releases**](https://github.com/agenticraptor/undo-anything/releases) page
(Linux, macOS, Windows · amd64/arm64).

### Homebrew (macOS / Linux)

```bash
brew install agenticraptor/tap/ua
```

> Available once the Homebrew tap is published — see the note in
> [`.goreleaser.yaml`](.goreleaser.yaml) to enable it.

### From source

```bash
git clone https://github.com/agenticraptor/undo-anything
cd undo-anything
make install
```

## Quickstart

```bash
# 1. Start tracking a folder (creates .undo/ and a baseline snapshot)
cd ~/code/my-project
ua init

# 2. Snapshot automatically on every save, in the background
ua daemon start
#    …now just work. Edit files, let your agent run, whatever.

# 3. Oops — get a single file back to an earlier state
ua log                       # find the snapshot you want
ua restore 056d1f13 internal/payments/client.go

# 4. Or roll the WHOLE folder back, then change your mind and undo that too
ua restore 056d1f13 --clean
ua restore <safety-id-it-printed>

# 5. Browse everything visually and restore with one key
ua timeline
```

Prefer it to start on login? Install it as a service:

```bash
ua daemon install     # launchd on macOS, a systemd user unit on Linux
```

## Usage

```text
ua init [dir]                 Start tracking a folder (+ baseline snapshot)
ua watch [dir]                Watch & snapshot in the foreground (Ctrl-C to stop)
ua snapshot [dir] -m "msg"    Capture a snapshot right now
ua log [dir]                  List snapshots, newest first
ua timeline [dir]             Interactive TUI: browse + one-key restore
ua show <id>                  Show a snapshot's details and its diff
ua diff [<from> [<to>]]       Diff two snapshots (defaults to latest vs parent)
ua restore <id> [path...]     Restore files or the whole tree to a snapshot
ua status [dir]               Watcher status + store statistics
ua prune [dir]                Thin old snapshots and reclaim space
ua daemon start|stop|status|install|uninstall
ua doctor [dir]               Check environment & store health
ua version
```

Selected flags:

| Command | Flag | Description |
|---------|------|-------------|
| `restore` | `--clean` | Remove tracked files not present in the snapshot (whole-tree only) |
| `restore` | `--no-safety` | Skip the automatic pre-restore safety snapshot |
| `restore` | `-y, --yes` | Don't prompt for confirmation |
| `log` | `-n, --limit` | Max snapshots to show (`0` = all) |
| `log` | `--oneline` | Compact, script-friendly output |
| `snapshot` | `-m, --message` | Label the snapshot |
| `watch` | `--debounce` | Quiet period before a snapshot (overrides config) |
| `prune` | `--dry-run` | Show what would be removed without changing anything |

## How it works

```
            file changes
                 │
        fsnotify (debounced)                  ┌──────────────────────────────┐
                 │                            │            .undo/             │
                 ▼                            │                              │
   walk tree (skipping ignored) ─┐           │  objects/   content-addressed │
                 │               ├─ hash ───►│             deduped + zlib     │
        read each file ──────────┘   each    │  snapshots/ immutable manifests│
                 │               file        │  timeline.log  append-only idx │
                 ▼                            └──────────────────────────────┘
      snapshot = manifest of {path → sha256, mode, size}
                 │
   identical tree? → collapse to the same snapshot id (a no-op)
                 │
                 ▼
         ua restore  ──►  (safety snapshot first)  ──►  rewrite files atomically
```

Each unique file content is stored **once**, keyed by its SHA-256 and compressed.
A snapshot is just a small manifest of pointers, so "snapshot on every save" stays
cheap — only changed blobs are ever written. Read
[docs/architecture.md](docs/architecture.md) for the full design.

## Data safety

`undo-anything` is something you trust with unsaved work, so safety is the
headline feature, not an afterthought:

- **Atomic everything.** Objects and manifests are written to a temp file and
  renamed into place. An interrupted snapshot can't corrupt history.
- **Verified on read.** Objects are checked against their content hash when
  restored; corruption is detected, never silently served.
- **Undo the undo.** Every restore takes a pre-restore safety snapshot by
  default and prints its ID, so you can always go back.
- **Ignored files are untouchable.** `--clean` only ever removes *tracked*
  files — your `node_modules/`, build output, and ignored paths are never
  deleted.

See [SECURITY.md](SECURITY.md) for the full policy and how to report issues.

## Configuration

Zero config required. To customize, edit `.undo/config.toml` (created by
`ua init`):

```toml
[watch]
debounce_ms = 1500        # coalesce rapid saves into one snapshot
max_file_size_mb = 50     # skip files larger than this

[ignore]
patterns = ["*.bin", "secrets/"]   # extra .gitignore-style rules
use_gitignore = true               # also honour the folder's .gitignore

[retention]
keep_hours = 48           # keep every snapshot within this window
keep_daily = 30           # then one per day for this many days
max_snapshots = 2000      # hard cap (newest always kept)
```

You can also drop a `.uaignore` file (same syntax as `.gitignore`) in the
folder. The store directory, `.git/`, `node_modules/`, and common build output
are ignored by default. Full reference: [docs/configuration.md](docs/configuration.md).

## Privacy

`undo-anything` runs **entirely on your machine** and makes **no network
connections** — there's no telemetry and no account. You can confirm it yourself:

```bash
# macOS: should show no outbound connections
sudo lsof -i -nP | grep ua || echo "no network — as promised"
```

Snapshots live in `.undo/` next to your files (deduplicated and compressed, but
**not encrypted**) — treat that directory as sensitively as the files it
protects, and exclude secrets via `.uaignore` if you don't want them retained.

## FAQ

A few quick ones — see [docs/faq.md](docs/faq.md) for more.

- **Does this replace git?** No. git is for *intentional, shared* history;
  `undo-anything` is a personal, automatic safety net for the *uncommitted*
  moments in between.
- **Will it bloat my disk?** Rarely — identical content is stored once and
  compressed. `ua status` shows exactly how much you're saving, and `ua prune`
  thins old history.
- **What about huge files / build output?** Files over `max_file_size_mb` and
  anything ignored are skipped automatically.

## Contributing

Contributions are very welcome — see [CONTRIBUTING.md](CONTRIBUTING.md). Good
first issues include an interactive `restore` picker, line-level diffs in
`ua show`, and Windows service installation. Please also read our
[Code of Conduct](CODE_OF_CONDUCT.md).

## License

[MIT](LICENSE) © undo-anything contributors.
