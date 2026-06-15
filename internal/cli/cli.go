// Package cli wires together the undo-anything command-line interface on top of
// cobra. Each subcommand lives in its own file; this file builds the root
// command and is the single entry point called by main.
package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/agenticraptor/undo-anything/internal/buildinfo"
)

// Execute runs the root command and returns a process exit code.
func Execute() int {
	root := newRootCmd()
	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, dangerStyle.Render("error:")+" "+err.Error())
		return 1
	}
	return 0
}

func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "ua",
		Short: "A universal local time machine for any folder",
		Long: titleStyle.Render("undo-anything") + " — a universal local time machine for any folder.\n\n" +
			"It snapshots your folder on every save (content-addressed and deduplicated),\n" +
			"so you can restore any file — or the whole tree — to any point in time with\n" +
			"one command. Zero config, fully local, and every restore is itself undoable.",
		SilenceUsage:  true,
		SilenceErrors: true,
		Version:       buildinfo.Version,
		CompletionOptions: cobra.CompletionOptions{
			HiddenDefaultCmd: true,
		},
	}
	root.SetVersionTemplate(buildinfo.String() + "\n")

	root.AddCommand(
		newInitCmd(),
		newWatchCmd(),
		newSnapshotCmd(),
		newLogCmd(),
		newTimelineCmd(),
		newShowCmd(),
		newDiffCmd(),
		newRestoreCmd(),
		newStatusCmd(),
		newPruneCmd(),
		newDaemonCmd(),
		newDoctorCmd(),
		newVersionCmd(),
	)
	return root
}
