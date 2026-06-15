# FAQ

### How is this different from git?

git is for **intentional, shared history**: you decide when to commit, and you
publish those commits to collaborators. `undo-anything` is a **personal,
automatic safety net** for the moments *between* commits — the uncommitted work
that git, by design, doesn't protect. They complement each other. `undo-anything`
even ignores `.git/` by default so the two never trip over each other.

### Will it bloat my disk?

Usually not. File contents are content-addressed, so identical content is stored
once, and every blob is zlib-compressed. A snapshot only writes the bytes that
actually changed. Run `ua status` to see real numbers — including how much space
you've saved versus naive full copies — and `ua prune` to thin old history and
reclaim space.

### What triggers a snapshot?

Any create/modify/rename/delete under the tracked folder (excluding ignored
paths) marks the tree dirty. After a short **debounce** (default 1.5s of quiet),
one snapshot of the whole tree is taken. If nothing actually changed since the
last snapshot, no new snapshot is recorded.

### Can I track more than one folder?

Yes. Each folder has its own independent `.undo/` store and its own background
watcher. Run `ua init` and `ua daemon start` in each folder you care about.

### Does it follow symlinks or store huge binaries?

No symlinks (only regular files are tracked), and files larger than
`watch.max_file_size_mb` (default 50 MB) are skipped. Both keep the store small
and predictable. Raise the limit in config if you need to.

### What exactly happens when I restore?

1. A **pre-restore safety snapshot** of your current state is captured (unless
   you pass `--no-safety`).
2. The requested files — or the whole tree — are written from the chosen
   snapshot, atomically.
3. With `--clean`, tracked files not present in the target are removed (ignored
   files are never touched).

The safety snapshot's ID is printed, so you can undo the restore with
`ua restore <safety-id>`.

### I restored the wrong thing. Can I undo it?

Yes — that's the point of the safety snapshot. Run `ua log`, find the
`pre-restore` snapshot (labelled "before restore to …"), and restore it. The
`ua timeline` TUI makes this a two-keystroke operation.

### Is my data encrypted?

No. Blobs are deduplicated and compressed but **not encrypted**. Treat `.undo/`
as sensitively as the files it protects, and use `.uaignore` to exclude secrets
you don't want retained.

### Does it phone home?

No. `undo-anything` makes no network connections at all — no telemetry, no
account, no update checks. See the [privacy section](../README.md#privacy) for a
one-liner to verify it yourself.

### How do I stop tracking a folder / remove everything?

Stop the watcher and delete the store:

```bash
ua daemon stop
ua daemon uninstall      # if you installed the OS service
rm -rf .undo             # removes all snapshots and history
```

### Does it work on Windows?

The core (init, snapshot, log, show, diff, restore, timeline, watch) is
cross-platform and tested on Windows in CI. The background **service installer**
currently targets macOS (launchd) and Linux (systemd user units); on Windows,
run `ua watch` in a terminal or wire it into Task Scheduler yourself.
Contributions for native Windows service support are welcome.

### Why "undo-anything" but the command is `ua`?

`ua` is short to type for a tool you reach for reflexively. Think of it the way
`rg` is short for ripgrep.
