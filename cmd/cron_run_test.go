package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"net"
	"strings"
	"testing"

	"github.com/ludusrusso/wildgecu/pkg/daemon"
)

// fakeDaemon simulates the daemon side of a cron-test streaming connection.
// It reads the CLI's request, emits the configured sequence of events, then
// closes the server-side connection.
func fakeDaemon(t *testing.T, events []daemon.CronTestEvent) dialFn {
	t.Helper()
	return func() (net.Conn, error) {
		client, server := net.Pipe()
		go func() {
			defer server.Close()
			var req daemon.CronTestRequest
			if err := json.NewDecoder(server).Decode(&req); err != nil {
				return
			}
			enc := json.NewEncoder(server)
			for _, ev := range events {
				if err := enc.Encode(ev); err != nil {
					return
				}
			}
		}()
		return client, nil
	}
}

func TestRunCronTest(t *testing.T) {
	t.Run("streams stdout and exits with job code", func(t *testing.T) {
		dial := fakeDaemon(t, []daemon.CronTestEvent{
			{Type: "stdout", Data: "line one\n"},
			{Type: "stdout", Data: "line two\n"},
			{Type: "done", ExitCode: 0, Duration: "1s"},
		})

		var stdout, stderr bytes.Buffer
		code := runCronTest(&stdout, &stderr, "foo", "", dial)

		if code != 0 {
			t.Errorf("expected 0, got %d", code)
		}
		if stdout.String() != "line one\nline two\n" {
			t.Errorf("stdout = %q", stdout.String())
		}
	})

	t.Run("non-zero job exit is propagated", func(t *testing.T) {
		dial := fakeDaemon(t, []daemon.CronTestEvent{
			{Type: "stderr", Data: "boom\n"},
			{Type: "done", ExitCode: 1},
		})

		var stdout, stderr bytes.Buffer
		code := runCronTest(&stdout, &stderr, "foo", "", dial)
		if code != 1 {
			t.Errorf("expected 1, got %d", code)
		}
		if !strings.Contains(stderr.String(), "boom") {
			t.Errorf("expected boom in stderr, got %q", stderr.String())
		}
	})

	t.Run("daemon unreachable returns 125", func(t *testing.T) {
		dial := func() (net.Conn, error) { return nil, errors.New("no socket") }
		var stdout, stderr bytes.Buffer
		code := runCronTest(&stdout, &stderr, "foo", "", dial)
		if code != exitDaemonUnreachable {
			t.Errorf("expected 125, got %d", code)
		}
		if !strings.Contains(stderr.String(), "daemon unreachable") {
			t.Errorf("expected 'daemon unreachable' in stderr, got %q", stderr.String())
		}
	})

	t.Run("unknown job returns 126 via done event", func(t *testing.T) {
		dial := fakeDaemon(t, []daemon.CronTestEvent{
			{Type: "done", ExitCode: 126, Error: "no job named \"ghost\""},
		})
		var stdout, stderr bytes.Buffer
		code := runCronTest(&stdout, &stderr, "ghost", "", dial)
		if code != 126 {
			t.Errorf("expected 126, got %d", code)
		}
		if !strings.Contains(stderr.String(), "no job named") {
			t.Errorf("expected error in stderr, got %q", stderr.String())
		}
	})

	t.Run("timeout flag is sent in request", func(t *testing.T) {
		var got daemon.CronTestRequest
		dial := func() (net.Conn, error) {
			client, server := net.Pipe()
			go func() {
				defer server.Close()
				_ = json.NewDecoder(server).Decode(&got)
				_ = json.NewEncoder(server).Encode(daemon.CronTestEvent{Type: "done"})
			}()
			return client, nil
		}

		var stdout, stderr bytes.Buffer
		runCronTest(&stdout, &stderr, "foo", "45s", dial)

		if got.Timeout != "45s" {
			t.Errorf("expected timeout=45s in request, got %q", got.Timeout)
		}
		if got.Type != "cron.test" {
			t.Errorf("expected type=cron.test, got %q", got.Type)
		}
		if got.Name != "foo" {
			t.Errorf("expected name=foo, got %q", got.Name)
		}
	})

	t.Run("unexpected EOF before done returns 125", func(t *testing.T) {
		dial := func() (net.Conn, error) {
			client, server := net.Pipe()
			go func() {
				var req daemon.CronTestRequest
				_ = json.NewDecoder(server).Decode(&req)
				server.Close()
			}()
			return client, nil
		}
		var stdout, stderr bytes.Buffer
		code := runCronTest(&stdout, &stderr, "foo", "", dial)
		if code != exitDaemonUnreachable {
			t.Errorf("expected 125 on EOF, got %d", code)
		}
	})
}
