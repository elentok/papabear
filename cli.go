package main

import (
	"os"

	"github.com/spf13/cobra"
)

func newRootCmd() *cobra.Command {
	cobra.EnableCommandSorting = false

	rootCmd := &cobra.Command{
		Use:           "papabear",
		Short:         "Manage screen time limits and access controls",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	rootCmd.AddCommand(
		newRunCmd(),
		newSetupCmd(),
		newDoctorCmd(),
		newLogsCmd(),
		newStatusCmd(),
		newAskCmd(),
		newAdminCmd("give", "Add time for a user"),
		newAdminCmd("lock", "Lock a user's screen and account"),
		newAdminCmd("unlock", "Set remaining time and allow login"),
		newAdminCmd("hours", "View or set allowed hours for a user"),
		newAdminCmd("say", "Send a spoken and desktop message to a user"),
		newConfigCmd(),
		newCheckLoginCmd(),
	)
	rootCmd.InitDefaultCompletionCmd()

	return rootCmd
}

func newRunCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "run",
		Short: "Start the daemon",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			runDaemon()
		},
	}
}

func newSetupCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "setup",
		Short: "Install system dependencies",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSetup()
		},
	}
}

func newDoctorCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Check system configuration",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			runDoctor()
		},
	}
}

func newLogsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "logs",
		Short: "Follow service logs",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			runLogs()
		},
	}
}

func newStatusCmd() *cobra.Command {
	var compact bool

	cmd := &cobra.Command{
		Use:   "status [user]",
		Short: "Show remaining time for the current or target user",
		Args:  cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			username := ""
			if len(args) == 1 {
				username = args[0]
			}
			runStatus(compact, username)
		},
	}
	cmd.Flags().BoolVar(&compact, "compact", false, "Show only remaining screen time")
	return cmd
}

func newAskCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "ask [minutes]",
		Short: "Request more screen time",
		Args:  cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			minutes := "15"
			if len(args) == 1 {
				minutes = args[0]
			}
			runAsk(minutes)
		},
	}
}

func newAdminCmd(name, short string) *cobra.Command {
	return &cobra.Command{
		Use:   adminUse(name),
		Short: short,
		Run: func(cmd *cobra.Command, args []string) {
			runAdminCommand(name, args)
		},
	}
}

func adminUse(name string) string {
	switch name {
	case "give":
		return "give [user] <duration>"
	case "lock":
		return "lock [user] [duration]"
	case "unlock":
		return "unlock [user] <duration>"
	case "hours":
		return "hours [user] [<day>] [start-end|clear]"
	case "say":
		return "say [user] <message>"
	default:
		return name
	}
}

func newConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Edit or show the compiled config",
		Args:  cobra.NoArgs,
	}

	cmd.AddCommand(
		&cobra.Command{
			Use:   "edit",
			Short: "Open the config in $EDITOR and validate it",
			Args:  cobra.NoArgs,
			Run: func(cmd *cobra.Command, args []string) {
				runConfigEdit()
			},
		},
		&cobra.Command{
			Use:   "show",
			Short: "Print the compiled config with defaults applied",
			Args:  cobra.NoArgs,
			Run: func(cmd *cobra.Command, args []string) {
				runConfigShow()
			},
		},
	)

	return cmd
}

func newCheckLoginCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "check-login",
		Short:  "Check if a user is allowed to log in",
		Hidden: true,
		Args:   cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			os.Exit(runCheckLogin())
		},
	}
	cmd.SetHelpFunc(func(cmd *cobra.Command, args []string) {})
	return cmd
}
