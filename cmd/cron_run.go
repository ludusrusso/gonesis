package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"

	"github.com/ludusrusso/wildgecu/pkg/daemon"

	"github.com/spf13/cobra"
)

// Reserved exit codes for `wildgecu cron test`. Values < 125 are reserved
// for the job's own exit code so scripts can distinguish CLI-level failures
// from job failures.
const (
	exitDaemonUnreachable = 125
	exitNoJobFound        = 126
	exitDetached          = 130
)

func init() {
	for _, c := range rootCmd.Commands() {
		if c.Use == "cron" {
			c.AddCommand(cronTestCmd())
			return
		}
	}
}

func cronTestCmd() *cobra.Command {
	var timeoutFlag string
	cmd := &cobra.Command{
		Use:   "test <name>",
		Short: "Run a cron job now and stream its output",
		Args:  cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			c.SilenceUsage = true
			code := runCronTest(c.OutOrStdout(), c.ErrOrStderr(), args[0], timeoutFlag, daemon.DialSocket)
			if code != 0 {
				os.Exit(code)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&timeoutFlag, "timeout", "", "override timeout (Go duration, e.g. 30s, 5m)")
	return cmd
}

// dialFn opens a connection to the daemon socket. Abstracted for testing.
type dialFn func() (net.Conn, error)

func runCronTest(stdout, stderr io.Writer, name, timeout string, dial dialFn) int {
	conn, err := dial()
	if err != nil {
		fmt.Fprintf(stderr, "daemon unreachable: %v\n", err)
		return exitDaemonUnreachable
	}
	defer conn.Close()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)
	defer signal.Stop(sigCh)

	done := make(chan int, 1)
	go func() {
		done <- streamCronTest(stdout, stderr, conn, name, timeout)
	}()

	select {
	case code := <-done:
		return code
	case <-sigCh:
		fmt.Fprintln(stderr, "detached; job still running")
		return exitDetached
	}
}

func streamCronTest(stdout, stderr io.Writer, conn net.Conn, name, timeout string) int {
	req := daemon.CronTestRequest{Type: "cron.test", Name: name, Timeout: timeout}
	if err := json.NewEncoder(conn).Encode(req); err != nil {
		fmt.Fprintf(stderr, "send request: %v\n", err)
		return exitDaemonUnreachable
	}

	dec := json.NewDecoder(conn)
	for {
		var ev daemon.CronTestEvent
		if err := dec.Decode(&ev); err != nil {
			fmt.Fprintf(stderr, "read event: %v\n", err)
			return exitDaemonUnreachable
		}
		switch ev.Type {
		case "stdout":
			fmt.Fprint(stdout, ev.Data)
		case "stderr":
			fmt.Fprint(stderr, ev.Data)
		case "done":
			if ev.Error != "" {
				fmt.Fprintln(stderr, ev.Error)
			}
			return ev.ExitCode
		}
	}
}
