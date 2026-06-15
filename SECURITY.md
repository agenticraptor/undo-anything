# Security & Data-Safety Policy

`undo-anything` is a tool people trust with their unsaved work, so we treat both
classic security issues **and data-loss bugs** with the same seriousness.

## Reporting a vulnerability or data-loss bug

Please report privately — do **not** open a public issue for security or
data-loss reports.

Use GitHub's **private vulnerability reporting** for this repository:

- Go to the [**Security** tab](https://github.com/agenticraptor/undo-anything/security)
  and click **Report a vulnerability**, or open the form directly:
  <https://github.com/agenticraptor/undo-anything/security/advisories/new>.

This keeps the report confidential between you and the maintainers until a fix
is released. Please include the version (`ua version`), your OS, and the
smallest set of steps that reproduces the problem. We aim to acknowledge reports
within a few days.

## Supported versions

Until the project reaches `v1.0.0`, only the latest released minor version
receives security and data-safety fixes.

| Version | Supported |
|---------|-----------|
| latest `0.x` | ✅ |
| older         | ❌ |

## How undo-anything protects your data

These are the invariants we consider security-critical. A bug in any of them is
treated as a vulnerability:

- **Atomic writes.** Every object and snapshot manifest is written to a temp
  file and atomically renamed into place. An interrupted run can never corrupt
  existing history.
- **Content integrity.** Objects are addressed by the SHA-256 of their contents
  and verified on read; a corrupted object is detected rather than silently
  returned.
- **Undoable restores.** By default every restore first captures a *pre-restore
  safety snapshot*, so any restore can itself be undone.
- **Ignored files are never deleted.** `--clean` and other destructive paths
  only ever touch tracked files, so `node_modules/`, build output, and anything
  matched by your ignore rules are safe.
- **No path traversal.** Restore refuses any manifest entry that would write
  outside the folder — via `..`, an absolute path, or a symlinked parent — so
  restoring an untrusted or tampered `.undo` can never overwrite arbitrary
  files on your system.
- **Decompression-bomb cap.** Objects are read through a hard size ceiling, so a
  crafted blob in a foreign store can't expand to exhaust memory.
- **Owner-only store.** The `.undo` directory and its contents are created
  `0700`/`0600`, so other users on a shared machine can't read your file
  history.
- **Local by default.** `undo-anything` makes no network connections. Your data
  never leaves your machine.

## Scope notes

`undo-anything` stores file contents **unencrypted** inside the `.undo/`
directory (deduplicated and compressed, but not encrypted). Treat `.undo/` with
the same sensitivity as the files it protects, and exclude secrets you don't
want retained via `.uaignore` or your config's ignore patterns.
