package cli

import (
	"github.com/spf13/cobra"

	"github.com/agenticraptor/undo-anything/internal/timeline"
)

func newTimelineCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "timeline [dir]",
		Aliases: []string{"tui"},
		Short:   "Browse snapshots interactively and restore with one key",
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
			return timeline.Run(s, m, cfg)
		},
	}
}
