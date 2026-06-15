package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/agenticraptor/undo-anything/internal/restore"
)

func newRestoreCmd() *cobra.Command {
	var (
		clean    bool
		noSafety bool
		assumeY  bool
	)
	cmd := &cobra.Command{
		Use:   "restore <snapshot> [path...]",
		Short: "Restore files or the whole folder to a snapshot",
		Long: "Restore the working folder to a previous snapshot. Pass one or more paths\n" +
			"to restore only those files; pass none to restore the entire tree.\n\n" +
			"Every restore first takes an automatic \"safety\" snapshot of the current\n" +
			"state, so you can always undo the undo.",
		Args: cobra.MinimumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			s, cfg, err := loadConfigured(".")
			if err != nil {
				return err
			}
			m, err := buildMatcher(s, cfg)
			if err != nil {
				return err
			}
			snap, err := s.ReadSnapshot(args[0])
			if err != nil {
				return err
			}
			paths := args[1:]

			opts := restore.Options{
				Safety:      !noSafety,
				Clean:       clean,
				Matcher:     m,
				MaxFileSize: cfg.MaxFileSizeBytes(),
			}

			if len(paths) == 0 {
				prompt := fmt.Sprintf("Restore the entire folder to snapshot %s?", snap.ID)
				if clean {
					prompt = fmt.Sprintf("Restore the entire folder to snapshot %s and remove tracked files not in it?", snap.ID)
				}
				if !confirm(warnLine("%s", prompt), assumeY) {
					fmt.Println(infoLine("Aborted. Nothing changed."))
					return nil
				}
				res, err := restore.Snapshot(s, snap, opts)
				if err != nil {
					return err
				}
				reportRestore(res, snap.ID)
				return nil
			}

			var all restore.Result
			for _, p := range paths {
				res, err := restore.File(s, snap, toRepoRel(s.Root, p), opts)
				if err != nil {
					return err
				}
				all.SafetyID = res.SafetyID
				all.Restored = append(all.Restored, res.Restored...)
				// Only take the safety snapshot once.
				opts.Safety = false
			}
			reportRestore(all, snap.ID)
			return nil
		},
	}
	cmd.Flags().BoolVar(&clean, "clean", false, "remove tracked files not present in the snapshot (whole-tree only)")
	cmd.Flags().BoolVar(&noSafety, "no-safety", false, "skip the automatic pre-restore safety snapshot")
	cmd.Flags().BoolVarP(&assumeY, "yes", "y", false, "do not prompt for confirmation")
	return cmd
}

func reportRestore(res restore.Result, targetID string) {
	fmt.Println(okLine("Restored to snapshot %s", idStyle.Render(targetID)))
	fmt.Println(mutedStyle.Render(fmt.Sprintf("  %d file(s) written, %d removed", len(res.Restored), len(res.Removed))))
	if res.SafetyID != "" {
		fmt.Println()
		fmt.Println(mutedStyle.Render("  Safety snapshot of your previous state: ") + idStyle.Render(res.SafetyID))
		fmt.Println(mutedStyle.Render("  Undo this restore with: ") + accentStyle.Render("ua restore "+res.SafetyID))
	}
}
