package daemon

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/ludusrusso/wildgecu/pkg/cron"
)

// CronTestRequest is the first message on a cron-test streaming connection.
type CronTestRequest struct {
	Type    string `json:"type"` // "cron.test"
	Name    string `json:"name"`
	Timeout string `json:"timeout,omitempty"` // Go duration string; empty = use job default
}

// CronTestEvent is a server→client message on a cron-test streaming connection.
// The event sequence is: zero or more stdout/stderr events followed by one
// terminal done event carrying the exit code.
type CronTestEvent struct {
	Type     string `json:"type"` // "stdout" | "stderr" | "done"
	Data     string `json:"data,omitempty"`
	ExitCode int    `json:"exit_code,omitempty"`
	Duration string `json:"duration,omitempty"`
	Error    string `json:"error,omitempty"`
}

// Reserved exit codes surfaced to the CLI.
const (
	cronTestExitNoJob       = 126
	cronTestExitDaemonError = 125
)

// cronTestRunner executes a job manually and returns its outcome. Kept as an
// interface so tests can swap in a stub without touching the real scheduler.
type cronTestRunner func(ctx context.Context, job *cron.CronJob, opts cron.TestOptions) cron.TestResult

// handleCronTestConn runs a single cron-test streaming request on an accepted
// socket connection. It loads the job from disk, invokes the runner, and
// emits NDJSON events terminated by a "done" event. The daemon context is
// used so that a disconnecting client does not cancel an in-flight job.
func handleCronTestConn(ctx context.Context, conn io.Writer, raw json.RawMessage, cronsDir string, run cronTestRunner, logger *slog.Logger) {
	var encMu sync.Mutex
	encoder := json.NewEncoder(conn)
	send := func(ev CronTestEvent) {
		encMu.Lock()
		defer encMu.Unlock()
		if err := encoder.Encode(ev); err != nil {
			logger.Debug("cron-test send error (client likely detached)", "error", err)
		}
	}

	var req CronTestRequest
	if err := json.Unmarshal(raw, &req); err != nil {
		send(CronTestEvent{Type: "done", ExitCode: cronTestExitDaemonError, Error: fmt.Sprintf("bad request: %v", err)})
		return
	}
	if req.Name == "" {
		send(CronTestEvent{Type: "done", ExitCode: cronTestExitNoJob, Error: "missing name"})
		return
	}

	path := filepath.Join(cronsDir, cron.Filename(req.Name))
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		send(CronTestEvent{Type: "done", ExitCode: cronTestExitNoJob, Error: fmt.Sprintf("no job named %q", req.Name)})
		return
	}
	if err != nil {
		send(CronTestEvent{Type: "done", ExitCode: cronTestExitNoJob, Error: fmt.Sprintf("read %s: %v", req.Name, err)})
		return
	}
	job, err := cron.Parse(data)
	if err != nil {
		send(CronTestEvent{Type: "done", ExitCode: cronTestExitNoJob, Error: fmt.Sprintf("job %q has config errors: %v", req.Name, err)})
		return
	}

	var override time.Duration
	if req.Timeout != "" {
		d, perr := time.ParseDuration(req.Timeout)
		if perr != nil {
			send(CronTestEvent{Type: "done", ExitCode: cronTestExitDaemonError, Error: fmt.Sprintf("bad timeout %q: %v", req.Timeout, perr)})
			return
		}
		if d < 0 {
			send(CronTestEvent{Type: "done", ExitCode: cronTestExitDaemonError, Error: fmt.Sprintf("timeout must be non-negative, got %q", req.Timeout)})
			return
		}
		override = d
	}

	opts := cron.TestOptions{
		OnStdout: func(s string) { send(CronTestEvent{Type: "stdout", Data: s}) },
		OnStderr: func(s string) { send(CronTestEvent{Type: "stderr", Data: s}) },
		Timeout:  override,
	}

	res := run(ctx, job, opts)
	send(CronTestEvent{Type: "done", ExitCode: res.ExitCode, Duration: res.Duration.String()})
}
