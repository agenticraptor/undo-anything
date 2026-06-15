package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/agenticraptor/undo-anything/internal/daemon"
	"github.com/agenticraptor/undo-anything/internal/humanize"
)

func newStatusCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status [dir]",
		Short: "Show watcher status and store statistics",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			s, err := openStore(dirArg(args))
			if err != nil {
				return err
			}
			usage, err := s.Usage()
			if err != nil {
				return err
			}
			entries, err := s.Index()
			if err != nil {
				return err
			}

			fmt.Println(titleStyle.Render("undo-anything") + mutedStyle.Render("  "+relForDisplay(s.Root)))
			fmt.Println()

			// Watcher state.
			if pid, ok := daemon.Status(s); ok {
				fmt.Println(okLine("Watcher running (pid %d)", pid))
			} else {
				fmt.Println(infoLine("Watcher not running — start it with %s", accentStyle.Render("ua daemon start")))
			}

			// Latest snapshot.
			if latest, _ := s.Latest(); latest != nil {
				fmt.Println(infoLine("Last snapshot %s · %s", idStyle.Render(latest.ID), humanize.RelTime(latest.Time)))
			} else {
				fmt.Println(infoLine("No snapshots yet"))
			}

			fmt.Println()
			fmt.Println(headingStyle.Render("Store"))
			fmt.Printf("  %s %s\n", mutedStyle.Render("snapshots   "), humanize.Count(usage.Snapshots))
			fmt.Printf("  %s %s\n", mutedStyle.Render("unique      "), humanize.Count(usage.UniqueSnaps))
			fmt.Printf("  %s %s\n", mutedStyle.Render("objects     "), humanize.Count(usage.Objects))
			fmt.Printf("  %s %s\n", mutedStyle.Render("on disk     "), humanize.Bytes(usage.ObjectBytes))
			fmt.Printf("  %s %s\n", mutedStyle.Render("latest tree "), humanize.Bytes(usage.LogicalSize))

			// Space-efficiency stat: what naive full copies would have cost.
			var totalLogical int64
			for _, e := range entries {
				totalLogical += e.Size
			}
			if usage.ObjectBytes > 0 && totalLogical > usage.ObjectBytes {
				fmt.Println()
				fmt.Println(successStyle.Render(fmt.Sprintf(
					"  %s full copies of every snapshot would be %s — undo-anything uses %s (%s smaller)",
					iconOK,
					humanize.Bytes(totalLogical),
					humanize.Bytes(usage.ObjectBytes),
					humanize.Ratio(totalLogical, usage.ObjectBytes),
				)))
			}
			return nil
		},
	}
	return cmd
}
