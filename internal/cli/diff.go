package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/agenticraptor/undo-anything/internal/humanize"
	"github.com/agenticraptor/undo-anything/internal/snapshot"
	"github.com/agenticraptor/undo-anything/internal/store"
)

func newDiffCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "diff [<from> [<to>]]",
		Short: "Show what changed between two snapshots",
		Long: "With no arguments, diff the latest snapshot against its parent.\n" +
			"With one argument, diff that snapshot against its parent.\n" +
			"With two arguments, diff <from> against <to>.",
		Args: cobra.MaximumNArgs(2),
		RunE: func(_ *cobra.Command, args []string) error {
			s, err := openStore(".")
			if err != nil {
				return err
			}

			var from, to *store.Snapshot
			switch len(args) {
			case 0:
				latest, err := s.Latest()
				if err != nil || latest == nil {
					return fmt.Errorf("no snapshots to diff")
				}
				if to, err = s.ReadSnapshot(latest.ID); err != nil {
					return err
				}
				if to.Parent != "" {
					from, _ = s.ReadSnapshot(to.Parent)
				}
			case 1:
				if to, err = s.ReadSnapshot(args[0]); err != nil {
					return err
				}
				if to.Parent != "" {
					from, _ = s.ReadSnapshot(to.Parent)
				}
			default:
				if from, err = s.ReadSnapshot(args[0]); err != nil {
					return err
				}
				if to, err = s.ReadSnapshot(args[1]); err != nil {
					return err
				}
			}

			fromID, toID := "∅", "∅"
			if from != nil {
				fromID = from.ID
			}
			if to != nil {
				toID = to.ID
			}
			fmt.Printf("%s %s %s\n\n", idStyle.Render(fromID), mutedStyle.Render(iconArrow), idStyle.Render(toID))

			changes := snapshot.Diff(from, to)
			if len(changes) == 0 {
				fmt.Println(mutedStyle.Render("(no changes)"))
				return nil
			}
			printChanges(changes)
			return nil
		},
	}
	return cmd
}

// printChanges renders a list of path-level changes with colored markers and a
// trailing summary line.
func printChanges(changes []snapshot.Change) {
	var added, removed, modified int
	for _, c := range changes {
		switch c.Kind {
		case snapshot.Added:
			added++
			fmt.Printf("  %s %s %s\n", successStyle.Render("+"), c.Path, mutedStyle.Render(humanize.Bytes(c.NewSize)))
		case snapshot.Removed:
			removed++
			fmt.Printf("  %s %s %s\n", dangerStyle.Render("-"), c.Path, mutedStyle.Render(humanize.Bytes(c.OldSize)))
		case snapshot.Modified:
			modified++
			delta := c.NewSize - c.OldSize
			sign := "+"
			if delta < 0 {
				sign = "-"
				delta = -delta
			}
			fmt.Printf("  %s %s %s\n", warnStyle.Render("~"), c.Path, mutedStyle.Render(fmt.Sprintf("%s%s", sign, humanize.Bytes(delta))))
		}
	}
	fmt.Printf("\n%s\n", mutedStyle.Render(fmt.Sprintf(
		"%d added · %d modified · %d removed", added, modified, removed,
	)))
}
