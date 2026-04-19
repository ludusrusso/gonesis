package cron

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/ludusrusso/wildgecu/pkg/provider"
)

// Source identifies how a cron run was triggered. It is written into the
// result file's frontmatter so manual invocations can be distinguished from
// scheduled ticks.
const (
	SourceScheduled = "scheduled"
	SourceManual    = "manual"
)

// Exit codes reported by ExecuteTest. Zero means the job succeeded; other
// values let the CLI propagate a meaningful shell exit code.
const (
	exitSuccess = 0
	exitError   = 1
	exitTimeout = 124
)

// TestDefaultTimeout is applied to manual runs that have no configured
// job timeout and no explicit override. Scheduled runs keep "no timeout"
// as their default.
const TestDefaultTimeout = 5 * time.Minute

// ExecutorConfig holds the dependencies for executing a cron job.
type ExecutorConfig struct {
	Provider provider.Provider
	Results  string // path to cron-results directory
	Logger   *slog.Logger
}

// TestOptions configures a manual ExecuteTest run. All fields are optional.
type TestOptions struct {
	OnStdout func(string)
	OnStderr func(string)
	Timeout  time.Duration // override; 0 means fall back to job.Timeout then TestDefaultTimeout
}

// TestResult reports the outcome of a manual run.
type TestResult struct {
	ExitCode int
	Duration time.Duration
}

// Execute runs a scheduled cron job: calls the provider with the prompt
// and writes the result to the results directory with source="scheduled".
func Execute(ctx context.Context, cfg *ExecutorConfig, job *CronJob) {
	runJob(ctx, cfg, job, SourceScheduled, job.Timeout, nil, nil)
}

// ExecuteTest runs a job manually (via `cron test`). It streams stdout/stderr
// via the supplied callbacks, applies the timeout precedence
// (opts.Timeout > job.Timeout > TestDefaultTimeout), and records the run in
// the results directory with source="manual".
func ExecuteTest(ctx context.Context, cfg *ExecutorConfig, job *CronJob, opts TestOptions) TestResult {
	timeout := opts.Timeout
	if timeout <= 0 {
		timeout = job.Timeout
	}
	if timeout <= 0 {
		timeout = TestDefaultTimeout
	}
	start := time.Now()
	code := runJob(ctx, cfg, job, SourceManual, timeout, opts.OnStdout, opts.OnStderr)
	return TestResult{ExitCode: code, Duration: time.Since(start)}
}

func runJob(ctx context.Context, cfg *ExecutorConfig, job *CronJob, source string, timeout time.Duration, onStdout, onStderr func(string)) int {
	cfg.Logger.Info("executing cron job", "name", job.Name, "source", source, "timeout", timeout)

	start := time.Now()
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	resp, err := cfg.Provider.Generate(ctx, &provider.GenerateParams{
		Messages: []provider.Message{
			{Role: provider.RoleUser, Content: job.Prompt},
		},
	})
	duration := time.Since(start)

	if err != nil {
		if timeout > 0 && errors.Is(ctx.Err(), context.DeadlineExceeded) {
			msg := fmt.Sprintf("Timed out after %s.\n", timeout)
			cfg.Logger.Error("cron job timed out", "name", job.Name, "timeout", timeout)
			if onStderr != nil {
				onStderr(msg)
			}
			writeResult(cfg, job, source, "timeout", start, duration, msg)
			return exitTimeout
		}
		cfg.Logger.Error("cron job failed", "name", job.Name, "error", err)
		errMsg := fmt.Sprintf("error: %v\n", err)
		if onStderr != nil {
			onStderr(errMsg)
		}
		writeResult(cfg, job, source, "error", start, duration, errMsg)
		return exitError
	}

	content := resp.Message.Content
	if onStdout != nil {
		onStdout(content)
	}

	if filename := writeResult(cfg, job, source, "success", start, duration, content); filename != "" {
		cfg.Logger.Info("cron job completed", "name", job.Name, "file", filename)
	}
	return exitSuccess
}

func writeResult(cfg *ExecutorConfig, job *CronJob, source, status string, start time.Time, duration time.Duration, content string) string {
	ts := time.Now().UTC().Format("20060102-150405")
	filename := fmt.Sprintf("%s-%s.md", job.Name, ts)
	if err := os.MkdirAll(cfg.Results, 0o755); err != nil {
		cfg.Logger.Error("failed to create results dir", "name", job.Name, "error", err)
		return ""
	}

	var buf bytes.Buffer
	buf.WriteString("---\n")
	fmt.Fprintf(&buf, "source: %s\n", source)
	fmt.Fprintf(&buf, "started_at: %s\n", start.UTC().Format(time.RFC3339))
	fmt.Fprintf(&buf, "duration: %s\n", duration)
	fmt.Fprintf(&buf, "status: %s\n", status)
	buf.WriteString("---\n")
	buf.WriteString(content)

	if err := os.WriteFile(filepath.Join(cfg.Results, filename), buf.Bytes(), 0o644); err != nil {
		cfg.Logger.Error("failed to write cron result", "name", job.Name, "error", err)
		return ""
	}
	return filename
}
