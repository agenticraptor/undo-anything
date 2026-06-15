package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/agenticraptor/undo-anything/internal/humanize"
)

func newLogCmd() *cobra.Command {
	var (
		limit   int
		oneline bool
	)
	cmd := &cobra.Command{
		Use:     "log [dir]",
		Aliases: []string{"history", "ls"},
		Short:   "List snapshots, newest first",
		Args:    cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			s, err := openStore(dirArg(args))
			if err != nil {
				return err
			}
			entries, err := s.IndexDesc()
			if err != nil {
				return err
			}
			if len(entries) == 0 {
				fmt.Println(infoLine("No snapshots yet. Run %s to capture one.", accentStyle.Render("ua snapshot")))
				return nil
			}
			total := len(entries)
			truncated := false
			if limit > 0 && len(entries) > limit {
				entries = entries[:limit]
				truncated = true
			}
			for _, e := range entries {
				if oneline {
					fmt.Printf("%s  %s  %s\n", idStyle.Render(e.ID), mutedStyle.Render(humanize.RelTime(e.Time)), e.Label)
					continue
				}
				line := fmt.Sprintf("%s  %s  %s  %s",
					idStyle.Render(e.ID),
					triggerBadge(e.Trigger),
					mutedStyle.Render(humanize.RelTime(e.Time)),
					mutedStyle.Render(fmt.Sprintf("%d files", e.Files)),
				)
				if e.Label != "" {
					line += "  " + e.Label
				}
				fmt.Println(line)
			}
			if truncated && !oneline {
				fmt.Println(mutedStyle.Render(fmt.Sprintf("\nShowing %d of %d. Use --limit 0 for all, or `ua timeline` to browse.", len(entries), total)))
			}
			return nil
		},
	}
	cmd.Flags().IntVarP(&limit, "limit", "n", 20, "maximum snapshots to show (0 = all)")
	cmd.Flags().BoolVar(&oneline, "oneline", false, "compact one-line-per-snapshot output")
	return cmd
}
