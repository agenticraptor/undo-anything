# Configuration

`undo-anything` works with **zero configuration**. Everything below is optional
and has a sensible default.

Configuration lives at `.undo/config.toml` inside the tracked folder. `ua init`
writes a documented starter file; you can edit it any time.

## Full reference

```toml
[watch]
# Quiet period (milliseconds) after the last change before a snapshot is taken.
# Editors often write a file several times per save; this coalesces the burst.
debounce_ms = 1500

# Files larger than this (megabytes) are skipped when snapshotting.
max_file_size_mb = 50

[ignore]
# Extra .gitignore-style patterns, layered on top of the built-in defaults.
patterns = ["*.bin", "secrets/"]

# Also honour the folder's own .gitignore.
use_gitignore = true

[retention]
# Keep every snapshot taken within this many hours.
keep_hours = 48

# For snapshots older than keep_hours, keep one per calendar day for this many
# recent days.
keep_daily = 30

# Hard cap on retained snapshots; the newest are always kept.
max_snapshots = 2000
```

## Defaults

| Key | Default | Meaning |
|-----|---------|---------|
| `watch.debounce_ms` | `1500` | Coalesce rapid saves into one snapshot |
| `watch.max_file_size_mb` | `50` | Skip files larger than this |
| `ignore.patterns` | `[]` | Extra ignore rules |
| `ignore.use_gitignore` | `true` | Honour the folder's `.gitignore` |
| `retention.keep_hours` | `48` | Keep-everything window |
| `retention.keep_daily` | `30` | Days of one-per-day retention |
| `retention.max_snapshots` | `2000` | Hard cap (newest kept) |

Unknown keys are ignored, so a config written by a newer version still loads on
an older one.

## Ignore rules

Three sources are combined, **last match wins**:

1. **Built-in defaults** — the store itself (`.undo/`), VCS dirs (`.git/`,
   `.hg/`, `.svn/`), dependency/build dirs (`node_modules/`, `target/`, `dist/`,
   `build/`, `.next/`, `__pycache__/`, …), and OS/editor cruft (`.DS_Store`,
   `*.log`, `*.tmp`, `*.swp`, …).
2. **`ignore.patterns`** from `config.toml`.
3. **`.uaignore`** — a file at the folder root using `.gitignore` syntax.
4. **`.gitignore`** — when `use_gitignore = true`.

### Syntax

The matcher supports the common subset of `.gitignore`:

| Pattern | Matches |
|---------|---------|
| `*.log` | any `.log` file at any depth |
| `build/` | a directory named `build` **and everything under it** |
| `/build` | `build` only at the folder root (anchored) |
| `docs/*.tmp` | `.tmp` files directly under `docs/` |
| `**/cache/**` | anything under any `cache/` directory |
| `!keep.log` | re-include a path excluded by an earlier rule |
| `# comment` | ignored (comment line) |

### Excluding secrets

If there are files you never want stored (even locally), add them to `.uaignore`
or `ignore.patterns`:

```gitignore
# .uaignore
.env
.env.*
secrets/
*.pem
```

Already snapshotted something sensitive? Tighten the ignore rule, then run
`ua prune` to thin old snapshots and garbage-collect now-unreferenced blobs.

## Environment

`undo-anything` reads no secret environment variables and makes no network
calls. The only environment it consults is the standard `HOME` /
`XDG_CONFIG_HOME` when installing an OS service (see below).

## Where the background service is installed

`ua daemon install` writes a per-folder service file:

- **macOS (launchd):** `~/Library/LaunchAgents/undo-anything-<hash>.plist`
- **Linux (systemd user):** `~/.config/systemd/user/undo-anything-<hash>.service`

`<hash>` is derived from the folder's absolute path, so each folder gets its own
service. Remove it with `ua daemon uninstall`.
