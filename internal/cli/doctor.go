package cli

import (
	"fmt"
	"os"
	"runtime"

	"github.com/spf13/cobra"

	"github.com/agenticraptor/undo-anything/internal/buildinfo"
	"github.com/agenticraptor/undo-anything/internal/daemon"
	"github.com/agenticraptor/undo-anything/internal/humanize"
)

func newDoctorCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "doctor [dir]",
		Short: "Check your environment and the store's health",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			fmt.Println(titleStyle.Render("undo-anything doctor"))
			fmt.Println(mutedStyle.Render("  " + buildinfo.String()))
			fmt.Println(mutedStyle.Render(fmt.Sprintf("  %s/%s · %s", runtime.GOOS, runtime.GOARCH, runtime.Version())))
			fmt.Println()

			s, err := openStore(dirArg(args))
			if err != nil {
				fmt.Println(warnLine("No store found here. Run %s to start tracking this folder.", accentStyle.Render("ua init")))
				return nil
			}
			fmt.Println(okLine("Store found at %s", relForDisplay(s.Dir)))

			// Writable check.
			if err := writableCheck(s.Dir); err != nil {
				fmt.Println(warnLine("Store directory is not writable: %v", err))
			} else {
				fmt.Println(okLine("Store directory is writable"))
			}

			// Usage + integrity-ish summary.
			usage, err := s.Usage()
			if err != nil {
				fmt.Println(warnLine("Could not read store usage: %v", err))
			} else {
				fmt.Println(okLine("%d snapshot(s), %d object(s), %s on disk",
					usage.Snapshots, usage.Objects, humanize.Bytes(usage.ObjectBytes)))
			}

			// Daemon.
			if pid, ok := daemon.Status(s); ok {
				fmt.Println(okLine("Background watcher running (pid %d)", pid))
			} else {
				fmt.Println(infoLine("Background watcher not running"))
			}
			return nil
		},
	}
	return cmd
}

func writableCheck(dir string) error {
	f, err := os.CreateTemp(dir, ".ua-doctor-*")
	if err != nil {
		return err
	}
	name := f.Name()
	_ = f.Close()
	return os.Remove(name)
}
