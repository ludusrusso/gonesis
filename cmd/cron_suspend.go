package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/ludusrusso/wildgecu/pkg/daemon"

	"github.com/spf13/cobra"
)

// exitCodeNoJob is returned when the target cron job does not exist or has
// config errors that prevent it from being loaded.
const exitCodeNoJob = 126

func init() {
	for _, c := range rootCmd.Commands() {
		if c.Use == "cron" {
			c.AddCommand(cronSuspendCmd())
			c.AddCommand(cronResumeCmd())
			return
		}
	}
}

func cronSuspendCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "suspend <name>",
		Short: "Suspend a cron job (pause without deleting its config)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCronToggle(cmd, "cron-suspend", args[0])
		},
	}
}

func cronResumeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "resume <name>",
		Short: "Resume a suspended cron job",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCronToggle(cmd, "cron-resume", args[0])
		},
	}
}

func runCronToggle(cmd *cobra.Command, rpc, name string) error {
	if !daemon.IsRunning() {
		return fmt.Errorf("daemon is not running")
	}
	resp, err := daemon.SendCommand(rpc, map[string]any{"name": name})
	if err != nil {
		return err
	}
	if !resp.OK {
		cmd.SilenceUsage = true
		if strings.Contains(resp.Error, "no job named") || strings.Contains(resp.Error, "config errors") {
			fmt.Fprintln(os.Stderr, resp.Error)
			os.Exit(exitCodeNoJob)
		}
		return fmt.Errorf("%s", resp.Error)
	}
	if msg, ok := resp.Payload.(string); ok && msg != "" {
		fmt.Println(msg)
	}
	return nil
}
