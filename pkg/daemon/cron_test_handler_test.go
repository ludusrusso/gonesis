package daemon

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ludusrusso/wildgecu/pkg/cron"
)

func decodeEvents(t *testing.T, buf *bytes.Buffer) []CronTestEvent {
	t.Helper()
	var events []CronTestEvent
	dec := json.NewDecoder(buf)
	for dec.More() {
		var ev CronTestEvent
		if err := dec.Decode(&ev); err != nil {
			t.Fatalf("decode event: %v", err)
		}
		events = append(events, ev)
	}
	return events
}

func runHandler(t *testing.T, dir string, req CronTestRequest, runner cronTestRunner) []CronTestEvent {
	t.Helper()
	raw, _ := json.Marshal(req)
	var buf bytes.Buffer
	handleCronTestConn(context.Background(), &buf, raw, dir, runner, slog.Default())
	return decodeEvents(t, &buf)
}

func TestCronTestHandler(t *testing.T) {
	validJob := "---\nname: foo\ncron: \"0 9 * * *\"\n---\nhello"

	t.Run("emits stdout then done with exit code 0", func(t *testing.T) {
		dir := t.TempDir()
		os.WriteFile(filepath.Join(dir, cron.Filename("foo")), []byte(validJob), 0o644)

		runner := func(_ context.Context, _ *cron.CronJob, opts cron.TestOptions) cron.TestResult {
			opts.OnStdout("hi from run")
			return cron.TestResult{ExitCode: 0, Duration: 123 * time.Millisecond}
		}

		events := runHandler(t, dir, CronTestRequest{Type: "cron.test", Name: "foo"}, runner)

		if len(events) != 2 {
			t.Fatalf("expected 2 events, got %d: %+v", len(events), events)
		}
		if events[0].Type != "stdout" || events[0].Data != "hi from run" {
			t.Errorf("unexpected first event: %+v", events[0])
		}
		if events[1].Type != "done" || events[1].ExitCode != 0 {
			t.Errorf("unexpected done event: %+v", events[1])
		}
		if events[1].Duration == "" {
			t.Errorf("expected non-empty duration")
		}
	})

	t.Run("unknown job returns exit 126", func(t *testing.T) {
		events := runHandler(t, t.TempDir(), CronTestRequest{Type: "cron.test", Name: "ghost"}, nil)
		if len(events) != 1 || events[0].Type != "done" {
			t.Fatalf("expected single done event, got %+v", events)
		}
		if events[0].ExitCode != cronTestExitNoJob {
			t.Errorf("expected exit 126, got %d", events[0].ExitCode)
		}
		if !strings.Contains(events[0].Error, "no job named") {
			t.Errorf("expected 'no job named' error, got %q", events[0].Error)
		}
	})

	t.Run("broken config returns exit 126", func(t *testing.T) {
		dir := t.TempDir()
		os.WriteFile(filepath.Join(dir, cron.Filename("broken")), []byte("---\nname: broken\n---\nno schedule"), 0o644)

		events := runHandler(t, dir, CronTestRequest{Type: "cron.test", Name: "broken"}, nil)
		if len(events) != 1 || events[0].ExitCode != cronTestExitNoJob {
			t.Fatalf("expected exit 126 done event, got %+v", events)
		}
		if !strings.Contains(events[0].Error, "config errors") {
			t.Errorf("expected 'config errors' in error, got %q", events[0].Error)
		}
	})

	t.Run("runner exit code is propagated", func(t *testing.T) {
		dir := t.TempDir()
		os.WriteFile(filepath.Join(dir, cron.Filename("foo")), []byte(validJob), 0o644)

		runner := func(_ context.Context, _ *cron.CronJob, _ cron.TestOptions) cron.TestResult {
			return cron.TestResult{ExitCode: 7, Duration: time.Second}
		}

		events := runHandler(t, dir, CronTestRequest{Type: "cron.test", Name: "foo"}, runner)
		if events[len(events)-1].ExitCode != 7 {
			t.Errorf("expected exit 7, got %d", events[len(events)-1].ExitCode)
		}
	})

	t.Run("timeout override parsed and forwarded to runner", func(t *testing.T) {
		dir := t.TempDir()
		os.WriteFile(filepath.Join(dir, cron.Filename("foo")), []byte(validJob), 0o644)

		var gotTimeout time.Duration
		runner := func(_ context.Context, _ *cron.CronJob, opts cron.TestOptions) cron.TestResult {
			gotTimeout = opts.Timeout
			return cron.TestResult{}
		}

		runHandler(t, dir, CronTestRequest{Type: "cron.test", Name: "foo", Timeout: "45s"}, runner)
		if gotTimeout != 45*time.Second {
			t.Errorf("expected 45s override, got %s", gotTimeout)
		}
	})

	t.Run("invalid timeout is rejected with daemon error", func(t *testing.T) {
		dir := t.TempDir()
		os.WriteFile(filepath.Join(dir, cron.Filename("foo")), []byte(validJob), 0o644)

		called := false
		runner := func(_ context.Context, _ *cron.CronJob, _ cron.TestOptions) cron.TestResult {
			called = true
			return cron.TestResult{}
		}

		events := runHandler(t, dir, CronTestRequest{Type: "cron.test", Name: "foo", Timeout: "not-a-duration"}, runner)
		if called {
			t.Errorf("runner should not be called for invalid timeout")
		}
		if events[0].ExitCode != cronTestExitDaemonError {
			t.Errorf("expected exit 125, got %d", events[0].ExitCode)
		}
	})

	t.Run("missing name returns exit 126", func(t *testing.T) {
		events := runHandler(t, t.TempDir(), CronTestRequest{Type: "cron.test"}, nil)
		if events[0].ExitCode != cronTestExitNoJob {
			t.Errorf("expected exit 126, got %d", events[0].ExitCode)
		}
	})
}

// TestCronTestHandlerPropagatesStderr covers that both streaming callbacks
// reach the client before the done event.
func TestCronTestHandlerPropagatesStderr(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, cron.Filename("foo")), []byte("---\nname: foo\ncron: \"0 9 * * *\"\n---\nx"), 0o644)

	runner := func(_ context.Context, _ *cron.CronJob, opts cron.TestOptions) cron.TestResult {
		opts.OnStderr("oh no")
		return cron.TestResult{ExitCode: 1}
	}

	events := runHandler(t, dir, CronTestRequest{Type: "cron.test", Name: "foo"}, runner)
	if events[0].Type != "stderr" || events[0].Data != "oh no" {
		t.Errorf("expected stderr event first, got %+v", events[0])
	}
	if events[len(events)-1].Type != "done" || events[len(events)-1].ExitCode != 1 {
		t.Errorf("expected done with exit 1, got %+v", events[len(events)-1])
	}
}

