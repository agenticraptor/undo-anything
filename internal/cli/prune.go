package cli

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/agenticraptor/undo-anything/internal/humanize"
	"github.com/agenticraptor/undo-anything/internal/prune"
)

func newPruneCmd() *cobra.Command {
	var (
		dryRun  bool
		assumeY bool
	)
	cmd := &cobra.Command{
		Use:   "prune [dir]",
		Short: "Thin old snapshots per your retention policy and reclaim space",
		Long: "Apply the retention policy from config (keep recent snapshots, thin older\n" +
			"ones to one per day, enforce a hard cap), then garbage-collect any file\n" +
			"contents no longer referenced by a surviving snapshot.",
		Args: cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			s, cfg, err := loadConfigured(dirArg(args))
			if err != nil {
				return err
			}
			entries, err := s.Index()
			if err != nil {
				return err
			}
			policy := prune.FromConfig(cfg.Retention)
			keep, remove := prune.Plan(entries, policy, time.Now())

			if len(remove) == 0 {
				fmt.Println(okLine("Nothing to prune — %d snapshot(s) all within policy.", len(keep)))
				return nil
			}

			fmt.Println(infoLine("Policy: keep %dh, 1/day for %dd, max %d snapshots",
				cfg.Retention.KeepHours, cfg.Retention.KeepDaily, cfg.Retention.MaxSnapshots))
			fmt.Println(infoLine("%d snapshot record(s) would be removed; %d kept.", len(remove), len(keep)))

			if dryRun {
				fmt.Println(mutedStyle.Render("\nDry run — nothing was changed. Re-run without --dry-run to apply."))
				return nil
			}
			if !confirm(warnLine("Proceed with prune and garbage collection?"), assumeY) {
				fmt.Println(infoLine("Aborted. Nothing changed."))
				return nil
			}

			res, removed, err := prune.Run(s, policy, time.Now())
			if err != nil {
				return err
			}
			fmt.Println(okLine("Pruned %d snapshot(s), removed %d blob(s), reclaimed %s.",
				removed, res.BlobsRemoved, humanize.Bytes(res.BytesReclaimed)))
			return nil
		},
	}
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "show what would be removed without changing anything")
	cmd.Flags().BoolVarP(&assumeY, "yes", "y", false, "do not prompt for confirmation")
	return cmd
}
