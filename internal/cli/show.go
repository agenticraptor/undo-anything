package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/agenticraptor/undo-anything/internal/humanize"
	"github.com/agenticraptor/undo-anything/internal/snapshot"
	"github.com/agenticraptor/undo-anything/internal/store"
)

func newShowCmd() *cobra.Command {
	var listFiles bool
	cmd := &cobra.Command{
		Use:   "show <snapshot> [dir]",
		Short: "Show a snapshot's details and what it changed",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(_ *cobra.Command, args []string) error {
			dir := "."
			if len(args) > 1 {
				dir = args[1]
			}
			s, err := openStore(dir)
			if err != nil {
				return err
			}
			snap, err := s.ReadSnapshot(args[0])
			if err != nil {
				return err
			}

			fmt.Println(titleStyle.Render("Snapshot "+snap.ID) + "  " + triggerBadge(snap.Trigger))
			fmt.Println(mutedStyle.Render("  taken    ") + snap.Time.Local().Format("2006-01-02 15:04:05") + mutedStyle.Render(" ("+humanize.RelTime(snap.Time)+")"))
			if snap.Label != "" {
				fmt.Println(mutedStyle.Render("  label    ") + snap.Label)
			}
			if snap.Parent != "" {
				fmt.Println(mutedStyle.Render("  parent   ") + idStyle.Render(snap.Parent))
			}
			fmt.Println(mutedStyle.Render("  files    ") + fmt.Sprintf("%d (%s)", snap.Stats.Files, humanize.Bytes(snap.Stats.TotalSize)))
			fmt.Println()

			if listFiles {
				for _, f := range snap.Files {
					fmt.Printf("  %s  %s\n", mutedStyle.Render(humanize.Bytes(f.Size)), f.Path)
				}
				return nil
			}

			var parent *store.Snapshot
			if snap.Parent != "" {
				parent, _ = s.ReadSnapshot(snap.Parent)
			}
			changes := snapshot.Diff(parent, snap)
			if len(changes) == 0 {
				fmt.Println(mutedStyle.Render("  (no file-level changes vs parent)"))
				return nil
			}
			fmt.Println(headingStyle.Render("Changes vs parent"))
			printChanges(changes)
			return nil
		},
	}
	cmd.Flags().BoolVar(&listFiles, "files", false, "list all files in the snapshot instead of the diff")
	return cmd
}
