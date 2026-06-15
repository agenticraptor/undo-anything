module github.com/agenticraptor/undo-anything

go 1.25

// Pin a patched-minimum toolchain: builds from source use at least this Go
// release, which carries the standard-library security fixes govulncheck flags
// for older toolchains (GO-2025-3750, GO-2025-3956, GO-2026-4602).
toolchain go1.25.8

require (
	github.com/BurntSushi/toml v1.3.2
	github.com/charmbracelet/bubbles v0.18.0
	github.com/charmbracelet/bubbletea v0.25.0
	github.com/charmbracelet/lipgloss v0.10.0
	github.com/fsnotify/fsnotify v1.7.0
	github.com/spf13/cobra v1.8.0
)

require (
	github.com/aymanbagabas/go-osc52/v2 v2.0.1 // indirect
	github.com/containerd/console v1.0.4-0.20230313162750-1ae8d489ac81 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/lucasb-eyer/go-colorful v1.2.0 // indirect
	github.com/mattn/go-isatty v0.0.18 // indirect
	github.com/mattn/go-localereader v0.0.1 // indirect
	github.com/mattn/go-runewidth v0.0.15 // indirect
	github.com/muesli/ansi v0.0.0-20211018074035-2e021307bc4b // indirect
	github.com/muesli/cancelreader v0.2.2 // indirect
	github.com/muesli/reflow v0.3.0 // indirect
	github.com/muesli/termenv v0.15.2 // indirect
	github.com/rivo/uniseg v0.4.7 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	golang.org/x/sync v0.1.0 // indirect
	golang.org/x/sys v0.12.0 // indirect
	golang.org/x/term v0.6.0 // indirect
	golang.org/x/text v0.3.8 // indirect
)
