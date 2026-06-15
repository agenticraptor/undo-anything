package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/agenticraptor/undo-anything/internal/config"
	"github.com/agenticraptor/undo-anything/internal/snapshot"
	"github.com/agenticraptor/undo-anything/internal/store"
)

func newInitCmd() *cobra.Command {
	var noSnapshot bool
	cmd := &cobra.Command{
		Use:   "init [dir]",
		Short: "Start tracking a folder (creates .undo and a first snapshot)",
		Long: "Initialize an undo-anything store in a folder. This creates a hidden\n" +
			".undo directory, writes a default config, and captures a first snapshot\n" +
			"so you immediately have a baseline to restore to.",
		Args: cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			dir := dirArg(args)
			s, err := store.Init(dir)
			if err != nil {
				return err
			}

			// Write a documented default config the first time.
			cfgPath := config.Path(s.Dir)
			if _, statErr := os.Stat(cfgPath); os.IsNotExist(statErr) {
				if err := config.Save(s.Dir, config.Default()); err != nil {
					return err
				}
			}
			cfg, err := config.Load(s.Dir)
			if err != nil {
				return err
			}

			fmt.Println(okLine("Initialized undo-anything in %s", accentStyle.Render(relForDisplay(filepath.Join(s.Root, store.DirName)))))

			if !noSnapshot {
				m, err := buildMatcher(s, cfg)
				if err != nil {
					return err
				}
				snap, _, err := snapshot.Create(s, m, snapshot.Options{
					Trigger:     store.TriggerInit,
					Label:       "initial snapshot",
					MaxFileSize: cfg.MaxFileSizeBytes(),
				})
				if err != nil {
					return err
				}
				fmt.Println(okLine("Captured baseline snapshot %s (%d files)", idStyle.Render(snap.ID), snap.Stats.Files))
			}

			fmt.Println()
			fmt.Println(mutedStyle.Render("Next:"))
			fmt.Printf("  %s   start watching this folder in the background\n", accentStyle.Render("ua daemon start"))
			fmt.Printf("  %s          browse and restore snapshots interactively\n", accentStyle.Render("ua timeline"))
			return nil
		},
	}
	cmd.Flags().BoolVar(&noSnapshot, "no-snapshot", false, "skip the initial baseline snapshot")
	return cmd
}
