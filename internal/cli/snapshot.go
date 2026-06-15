package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/agenticraptor/undo-anything/internal/humanize"
	"github.com/agenticraptor/undo-anything/internal/snapshot"
	"github.com/agenticraptor/undo-anything/internal/store"
)

func newSnapshotCmd() *cobra.Command {
	var message string
	cmd := &cobra.Command{
		Use:     "snapshot [dir]",
		Aliases: []string{"snap"},
		Short:   "Capture a snapshot of the folder right now",
		Args:    cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			s, cfg, err := loadConfigured(dirArg(args))
			if err != nil {
				return err
			}
			m, err := buildMatcher(s, cfg)
			if err != nil {
				return err
			}
			snap, created, err := snapshot.Create(s, m, snapshot.Options{
				Trigger:     store.TriggerManual,
				Label:       message,
				MaxFileSize: cfg.MaxFileSizeBytes(),
			})
			if err != nil {
				return err
			}
			if !created {
				fmt.Println(infoLine("No changes since the last snapshot %s", idStyle.Render(snap.ID)))
				return nil
			}
			fmt.Println(okLine("Snapshot %s captured", idStyle.Render(snap.ID)))
			fmt.Println(mutedStyle.Render(fmt.Sprintf(
				"  %d files · %s · %d new blob(s) (%s added)",
				snap.Stats.Files,
				humanize.Bytes(snap.Stats.TotalSize),
				snap.Stats.NewBlobs,
				humanize.Bytes(snap.Stats.NewBytes),
			)))
			return nil
		},
	}
	cmd.Flags().StringVarP(&message, "message", "m", "", "label this snapshot")
	return cmd
}
