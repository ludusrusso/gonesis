package cron

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ludusrusso/wildgecu/pkg/provider"
)

// mockProvider returns a fixed response for testing.
type mockProvider struct {
	response string
}

func (m *mockProvider) Generate(_ context.Context, _ *provider.GenerateParams) (*provider.Response, error) {
	return &provider.Response{
		Message: provider.Message{
			Role:    provider.RoleModel,
			Content: m.response,
		},
	}, nil
}

type errorProvider struct {
	err error
}

func (p *errorProvider) Generate(_ context.Context, _ *provider.GenerateParams) (*provider.Response, error) {
	return nil, p.err
}

func TestExecute(t *testing.T) {
	t.Run("writes result file with scheduled source frontmatter", func(t *testing.T) {
		resultsDir := t.TempDir()
		cfg := &ExecutorConfig{
			Provider: &mockProvider{response: "Here is your summary"},
			Results:  resultsDir,
			Logger:   slog.Default(),
		}
		job := &CronJob{Name: "daily-summary", Schedule: "0 9 * * *", Prompt: "Summarize my day"}

		Execute(context.Background(), cfg, job)

		matches, err := filepath.Glob(filepath.Join(resultsDir, "daily-summary-*.md"))
		if err != nil {
			t.Fatalf("Glob failed: %v", err)
		}
		if len(matches) != 1 {
			t.Fatalf("expected 1 result file, got %d", len(matches))
		}
		data, err := os.ReadFile(matches[0])
		if err != nil {
			t.Fatalf("ReadFile failed: %v", err)
		}
		body := string(data)
		for _, want := range []string{"source: scheduled", "status: success", "started_at:", "duration:", "Here is your summary"} {
			if !strings.Contains(body, want) {
				t.Errorf("output missing %q:\n%s", want, body)
			}
		}

		filename := filepath.Base(matches[0])
		if !strings.HasPrefix(filename, "daily-summary-") || !strings.HasSuffix(filename, ".md") {
			t.Errorf("unexpected filename %q", filename)
		}
	})
}

type blockingProvider struct{}

func (p *blockingProvider) Generate(ctx context.Context, _ *provider.GenerateParams) (*provider.Response, error) {
	<-ctx.Done()
	return nil, ctx.Err()
}

type recordingProvider struct {
	deadline    time.Time
	hadDeadline bool
	response    string
}

func (p *recordingProvider) Generate(ctx context.Context, _ *provider.GenerateParams) (*provider.Response, error) {
	p.deadline, p.hadDeadline = ctx.Deadline()
	return &provider.Response{
		Message: provider.Message{Role: provider.RoleModel, Content: p.response},
	}, nil
}

func TestExecuteTimeout(t *testing.T) {
	t.Run("cancels provider when deadline fires and records timeout", func(t *testing.T) {
		resultsDir := t.TempDir()
		cfg := &ExecutorConfig{
			Provider: &blockingProvider{},
			Results:  resultsDir,
			Logger:   slog.Default(),
		}
		job := &CronJob{
			Name:     "slow",
			Schedule: "0 9 * * *",
			Timeout:  20 * time.Millisecond,
			Prompt:   "hang",
		}

		start := time.Now()
		Execute(context.Background(), cfg, job)
		elapsed := time.Since(start)

		if elapsed > 2*time.Second {
			t.Fatalf("Execute did not respect timeout; ran for %s", elapsed)
		}

		matches, err := filepath.Glob(filepath.Join(resultsDir, "slow-*.md"))
		if err != nil {
			t.Fatalf("Glob failed: %v", err)
		}
		if len(matches) != 1 {
			t.Fatalf("expected 1 result file, got %d", len(matches))
		}
		data, _ := os.ReadFile(matches[0])
		body := string(data)
		if !strings.Contains(body, "Timed out") {
			t.Errorf("expected timeout marker in result, got %q", body)
		}
		if !strings.Contains(body, "status: timeout") {
			t.Errorf("expected status: timeout, got %q", body)
		}
	})

	t.Run("no timeout means no deadline", func(t *testing.T) {
		rec := &recordingProvider{response: "ok"}
		cfg := &ExecutorConfig{
			Provider: rec,
			Results:  t.TempDir(),
			Logger:   slog.Default(),
		}
		job := &CronJob{Name: "n", Schedule: "0 9 * * *", Prompt: "p"}
		Execute(context.Background(), cfg, job)
		if rec.hadDeadline {
			t.Errorf("expected no deadline when Timeout is zero")
		}
	})

	t.Run("timeout set means provider sees deadline", func(t *testing.T) {
		rec := &recordingProvider{response: "ok"}
		cfg := &ExecutorConfig{
			Provider: rec,
			Results:  t.TempDir(),
			Logger:   slog.Default(),
		}
		job := &CronJob{Name: "n", Schedule: "0 9 * * *", Timeout: time.Second, Prompt: "p"}
		Execute(context.Background(), cfg, job)
		if !rec.hadDeadline {
			t.Errorf("expected provider ctx to carry a deadline")
		}
	})
}

func TestExecuteTest(t *testing.T) {
	t.Run("streams stdout, records manual source, returns exit 0", func(t *testing.T) {
		resultsDir := t.TempDir()
		cfg := &ExecutorConfig{
			Provider: &mockProvider{response: "streamed output"},
			Results:  resultsDir,
			Logger:   slog.Default(),
		}
		job := &CronJob{Name: "manual", Schedule: "0 9 * * *", Prompt: "run now"}

		var stdoutCapture, stderrCapture strings.Builder
		res := ExecuteTest(context.Background(), cfg, job, TestOptions{
			OnStdout: func(s string) { stdoutCapture.WriteString(s) },
			OnStderr: func(s string) { stderrCapture.WriteString(s) },
		})

		if res.ExitCode != 0 {
			t.Errorf("expected exit 0, got %d", res.ExitCode)
		}
		if res.Duration <= 0 {
			t.Errorf("expected positive duration, got %s", res.Duration)
		}
		if stdoutCapture.String() != "streamed output" {
			t.Errorf("stdout capture mismatch: %q", stdoutCapture.String())
		}
		if stderrCapture.String() != "" {
			t.Errorf("expected empty stderr, got %q", stderrCapture.String())
		}

		matches, _ := filepath.Glob(filepath.Join(resultsDir, "manual-*.md"))
		if len(matches) != 1 {
			t.Fatalf("expected 1 result file, got %d", len(matches))
		}
		data, _ := os.ReadFile(matches[0])
		body := string(data)
		if !strings.Contains(body, "source: manual") {
			t.Errorf("expected source: manual, got %q", body)
		}
		if !strings.Contains(body, "status: success") {
			t.Errorf("expected status: success, got %q", body)
		}
	})

	t.Run("error streams to stderr and returns non-zero", func(t *testing.T) {
		cfg := &ExecutorConfig{
			Provider: &errorProvider{err: errors.New("boom")},
			Results:  t.TempDir(),
			Logger:   slog.Default(),
		}
		job := &CronJob{Name: "err", Schedule: "0 9 * * *", Prompt: "p"}

		var stderrCapture strings.Builder
		res := ExecuteTest(context.Background(), cfg, job, TestOptions{
			OnStderr: func(s string) { stderrCapture.WriteString(s) },
		})

		if res.ExitCode == 0 {
			t.Errorf("expected non-zero exit, got 0")
		}
		if !strings.Contains(stderrCapture.String(), "boom") {
			t.Errorf("expected error surfaced on stderr, got %q", stderrCapture.String())
		}
	})
}

func TestExecuteTestTimeoutPrecedence(t *testing.T) {
	t.Run("override takes precedence over job timeout and default", func(t *testing.T) {
		rec := &recordingProvider{response: "ok"}
		cfg := &ExecutorConfig{Provider: rec, Results: t.TempDir(), Logger: slog.Default()}
		job := &CronJob{Name: "n", Schedule: "0 9 * * *", Timeout: time.Hour, Prompt: "p"}

		before := time.Now()
		ExecuteTest(context.Background(), cfg, job, TestOptions{Timeout: 2 * time.Second})

		if !rec.hadDeadline {
			t.Fatal("expected a deadline")
		}
		budget := rec.deadline.Sub(before)
		if budget > 10*time.Second {
			t.Errorf("override ignored; deadline budget = %s", budget)
		}
	})

	t.Run("job timeout used when no override", func(t *testing.T) {
		rec := &recordingProvider{response: "ok"}
		cfg := &ExecutorConfig{Provider: rec, Results: t.TempDir(), Logger: slog.Default()}
		job := &CronJob{Name: "n", Schedule: "0 9 * * *", Timeout: 2 * time.Second, Prompt: "p"}

		before := time.Now()
		ExecuteTest(context.Background(), cfg, job, TestOptions{})

		if !rec.hadDeadline {
			t.Fatal("expected a deadline")
		}
		budget := rec.deadline.Sub(before)
		if budget > 10*time.Second || budget < time.Second {
			t.Errorf("job timeout not applied; budget = %s", budget)
		}
	})

	t.Run("5-minute default when neither is set", func(t *testing.T) {
		rec := &recordingProvider{response: "ok"}
		cfg := &ExecutorConfig{Provider: rec, Results: t.TempDir(), Logger: slog.Default()}
		job := &CronJob{Name: "n", Schedule: "0 9 * * *", Prompt: "p"}

		before := time.Now()
		ExecuteTest(context.Background(), cfg, job, TestOptions{})

		if !rec.hadDeadline {
			t.Fatal("expected a deadline")
		}
		budget := rec.deadline.Sub(before)
		// default is 5m; should be clearly > 1 minute and ≤ 5 minutes.
		if budget < time.Minute || budget > 6*time.Minute {
			t.Errorf("default timeout not ~5m; budget = %s", budget)
		}
	})
}
