package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/agenticraptor/undo-anything/internal/humanize"
	"github.com/agenticraptor/undo-anything/internal/watch"
)

func newWatchCmd() *cobra.Command {
	var (
		quiet    bool
		debounce time.Duration
	)
	cmd := &cobra.Command{
		Use:   "watch [dir]",
		Short: "Watch a folder and snapshot it on every change (foreground)",
		Long: "Run the watcher in the foreground, snapshotting the folder whenever file\n" +
			"activity settles. This is what `ua daemon start` runs in the background.\n" +
			"Press Ctrl-C to stop.",
		Args: cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			s, cfg, err := loadConfigured(dirArg(args))
			if err != nil {
				return err
			}
			m, err := buildMatcher(s, cfg)
			if err != nil {
				return err
			}
			d := debounce
			if d <= 0 {
				d = time.Duration(cfg.Watch.DebounceMs) * time.Millisecond
			}

			ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
			defer stop()

			if !quiet {
				fmt.Println(okLine("Watching %s", accentStyle.Render(relForDisplay(s.Root))))
				fmt.Println(mutedStyle.Render(fmt.Sprintf("  debounce %s · Ctrl-C to stop", d)))
			}

			err = watch.Run(ctx, s, m, watch.Options{
				Debounce:    d,
				MaxFileSize: cfg.MaxFileSizeBytes(),
			}, func(ev watch.Event) {
				ts := time.Now().Format("15:04:05")
				switch {
				case ev.Err != nil:
					fmt.Fprintln(os.Stderr, warnLine("%s snapshot failed: %v", mutedStyle.Render(ts), ev.Err))
				case ev.Created && ev.Snapshot != nil:
					fmt.Printf("%s %s %s  %s\n",
						mutedStyle.Render(ts),
						successStyle.Render(iconOK),
						idStyle.Render(ev.Snapshot.ID),
						mutedStyle.Render(fmt.Sprintf("%d files · %s", ev.Snapshot.Stats.Files, humanize.Bytes(ev.Snapshot.Stats.NewBytes))),
					)
				}
			})
			if err != nil && err != context.Canceled {
				return err
			}
			if !quiet {
				fmt.Println(infoLine("Stopped watching."))
			}
			return nil
		},
	}
	cmd.Flags().BoolVarP(&quiet, "quiet", "q", false, "suppress the startup banner (used by the daemon)")
	cmd.Flags().DurationVar(&debounce, "debounce", 0, "quiet period before a snapshot (overrides config)")
	return cmd
}
