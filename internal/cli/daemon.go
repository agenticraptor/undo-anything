package cli

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/agenticraptor/undo-anything/internal/daemon"
)

func newDaemonCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "daemon",
		Short: "Manage the background watcher (start/stop/status/install)",
	}
	cmd.AddCommand(
		daemonStartCmd(),
		daemonStopCmd(),
		daemonStatusCmd(),
		daemonInstallCmd(),
		daemonUninstallCmd(),
	)
	return cmd
}

func daemonStartCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "start [dir]",
		Short: "Start the background watcher for a folder",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			s, err := openStore(dirArg(args))
			if err != nil {
				return err
			}
			pid, err := daemon.Start(s)
			if errors.Is(err, daemon.ErrAlreadyRunning) {
				fmt.Println(infoLine("Watcher already running (pid %d).", pid))
				return nil
			}
			if err != nil {
				return err
			}
			fmt.Println(okLine("Watcher started (pid %d) for %s", pid, accentStyle.Render(relForDisplay(s.Root))))
			fmt.Println(mutedStyle.Render("  logs: " + relForDisplay(daemon.LogPath(s))))
			return nil
		},
	}
}

func daemonStopCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "stop [dir]",
		Short: "Stop the background watcher for a folder",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			s, err := openStore(dirArg(args))
			if err != nil {
				return err
			}
			err = daemon.Stop(s)
			if errors.Is(err, daemon.ErrNotRunning) {
				fmt.Println(infoLine("Watcher is not running."))
				return nil
			}
			if err != nil {
				return err
			}
			fmt.Println(okLine("Watcher stopped."))
			return nil
		},
	}
}

func daemonStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status [dir]",
		Short: "Report whether the background watcher is running",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			s, err := openStore(dirArg(args))
			if err != nil {
				return err
			}
			if pid, ok := daemon.Status(s); ok {
				fmt.Println(okLine("Running (pid %d).", pid))
			} else {
				fmt.Println(infoLine("Not running."))
			}
			return nil
		},
	}
}

func daemonInstallCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "install [dir]",
		Short: "Install the watcher as an OS service (launchd/systemd) that starts on login",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			s, err := openStore(dirArg(args))
			if err != nil {
				return err
			}
			info, err := daemon.Install(s)
			if info.Path != "" {
				fmt.Println(okLine("Wrote %s service file:", info.Kind))
				fmt.Println(mutedStyle.Render("  " + info.Path))
			}
			if err != nil {
				fmt.Println(warnLine("Service file written but activation needs a manual step: %v", err))
				return nil
			}
			fmt.Println(okLine("Service installed and started. It will run on login."))
			return nil
		},
	}
}

func daemonUninstallCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "uninstall [dir]",
		Short: "Remove the installed OS service",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			s, err := openStore(dirArg(args))
			if err != nil {
				return err
			}
			info, err := daemon.Uninstall(s)
			if err != nil {
				return err
			}
			fmt.Println(okLine("Removed %s service (%s).", info.Kind, info.Label))
			return nil
		},
	}
}
