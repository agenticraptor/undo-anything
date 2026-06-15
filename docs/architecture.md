# Architecture

`undo-anything` is a small content-addressed store with a file watcher bolted on
top. This document explains how the pieces fit together so you can trust it — and
hack on it.

## The `.undo/` directory

Every tracked folder gets a hidden store directory, much like `.git`:

```
.undo/
├── objects/         content-addressed file contents (sharded, zlib-compressed)
│   └── ab/
│       └── cdef…    a blob named by the SHA-256 of its *uncompressed* contents
├── snapshots/       one JSON manifest per unique snapshot, named <id>.json
├── timeline.log     append-only JSON-lines index of snapshots (newest last)
├── config.toml      per-folder configuration
├── tmp/             scratch space for atomic writes
├── daemon.pid       PID of the background watcher (when running)
└── daemon.log       the background watcher's log
```

The store always lives at the **root** of the tracked folder. Commands locate it
by walking up the directory tree (like git), so they work from any subdirectory.

## Objects (the blob store)

A **blob** is the full contents of one file. Its key is the hex SHA-256 of the
*uncompressed* bytes; on disk it is zlib-compressed and stored at
`objects/<first-2-hex>/<rest>`. The two-character shard keeps directories from
growing unbounded.

Two properties fall out of content-addressing:

- **Deduplication.** Identical content — the same file across many snapshots, or
  two files with the same bytes — is stored exactly once.
- **Integrity.** On read, the decompressed bytes are re-hashed and compared to
  the key. A corrupted object is detected rather than silently returned.

Writes are **atomic**: data is written to `tmp/` and `rename(2)`d into place, so a
crash mid-write can never leave a partial or corrupt object.

## Snapshots (manifests)

A **snapshot** is an immutable manifest describing a whole tracked tree at one
moment:

```jsonc
{
  "id": "92ac1bd72b54",
  "time": "2026-06-15T23:13:15Z",
  "trigger": "watch",            // watch | manual | pre-restore | init
  "label": "",
  "parent": "c12288b7b17a",
  "files": [
    { "path": "a.txt", "hash": "…", "size": 13, "mode": 420, "modtime": "…" }
  ],
  "stats": { "files": 4, "total_size": 98, "new_blobs": 1, "new_bytes": 51 }
}
```

The **id** is the first 12 hex characters of the SHA-256 of the canonical file
list (`path \0 hash \0 mode` for each entry, sorted by path). This means a
snapshot is itself content-addressed: **two identical trees produce the same id**,
so "snapshot on every save" is naturally idempotent — if nothing changed, no new
snapshot is recorded.

Only blobs that don't already exist are written, so the marginal cost of a
snapshot is just the bytes that actually changed plus a small manifest.

## The timeline log

`timeline.log` is an append-only stream of compact JSON records (one per
snapshot), used for fast listing without opening every manifest. It is the
source of truth for ordering: oldest first on disk, presented newest-first in the
UI. `prune` rewrites this file atomically when it thins history.

## The watcher

`internal/watch` wraps [`fsnotify`](https://github.com/fsnotify/fsnotify), which
is **not** recursive, so the watcher:

1. Adds watches to the root and every non-ignored subdirectory.
2. Adds a watch to any newly-created directory as it appears.
3. On any relevant event, marks the tree dirty and resets a **debounce** timer
   (default 1500 ms). Editors often write a file several times per save; the
   debounce coalesces that burst into one snapshot.
4. When the timer fires, it creates one snapshot of the whole tree.

The store's own `.undo/` directory is never watched and never snapshotted, which
prevents an obvious feedback loop.

## Restore & the safety snapshot

`internal/restore` materialises files from a snapshot back onto disk
(atomically, preserving mode and mtime). Two modes:

- **File restore** — rewrite specific paths.
- **Tree restore** — make the working tree match a snapshot. With `--clean`, it
  also removes *tracked* files that weren't in the target snapshot. Ignored files
  are never considered, so build output and `node_modules/` are always safe.

Before doing anything, restore captures a **pre-restore safety snapshot** of the
current state (trigger `pre-restore`). Because that snapshot is just another point
on the timeline, **any restore can be undone** by restoring the safety snapshot it
printed.

## Pruning & garbage collection

`internal/prune` turns the retention policy into a concrete keep-set:

1. Always keep the most recent snapshot.
2. Keep everything within `keep_hours`.
3. For older snapshots, keep one per calendar day for `keep_daily` days.
4. Enforce `max_snapshots` as a hard cap (newest win).

`store.GC` then deletes every snapshot manifest not in the keep-set, rewrites the
timeline, and sweeps any blob no longer referenced by a surviving snapshot. GC is
reference-counted by walking the kept manifests, so a blob shared by a kept
snapshot is never removed.

## Ignore matching

`internal/ignore` implements the common subset of `.gitignore` syntax (comments,
negation, anchored and directory-only patterns, `*`, `?`, and `**`). Patterns are
layered, last-match-wins:

1. Built-in defaults (`.undo/`, `.git/`, `node_modules/`, build dirs, OS cruft…).
2. `ignore.patterns` from `config.toml`.
3. A `.uaignore` file at the folder root.
4. The folder's `.gitignore` (when `use_gitignore = true`).

A directory-only rule (`build/`) matches both the directory and everything nested
beneath it, so ignored subtrees are pruned during the walk and never snapshotted.

## Package map

| Package | Responsibility |
|---------|----------------|
| `internal/store` | Blob CAS, snapshot persistence, timeline index, GC, usage |
| `internal/snapshot` | Walk a tree → manifest; diff two snapshots |
| `internal/ignore` | gitignore-style matching |
| `internal/watch` | Recursive, debounced fsnotify watcher |
| `internal/restore` | File/tree restore + the safety snapshot |
| `internal/prune` | Retention policy → keep-set → GC |
| `internal/daemon` | Background process lifecycle + launchd/systemd install |
| `internal/config` | TOML config load/save with defaults |
| `internal/cli` | cobra commands and styled output |
| `internal/timeline` | Bubble Tea TUI |
| `internal/humanize`, `internal/buildinfo` | Formatting & version metadata |
