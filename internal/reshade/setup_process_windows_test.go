//go:build windows

package reshade

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestPlatformSetupProcessRunnerCapturesAndTruncatesOutput(t *testing.T) {
	t.Parallel()

	result, err := (platformSetupProcessRunner{}).RunSetupProcess(
		context.Background(),
		setupProcessRequest{
			ExecutablePath: "cmd.exe",
			Arguments:      []string{"/D", "/S", "/C", "echo 123456789"},
			WorkingDir:     t.TempDir(),
			Timeout:        time.Minute,
			OutputLimit:    5,
		},
	)
	if err != nil {
		t.Fatalf("RunSetupProcess() error = %v", err)
	}
	if result.ExitCode != 0 ||
		result.Stdout != "12345" ||
		!result.StdoutTruncated ||
		result.StderrTruncated {
		t.Fatalf("result = %+v", result)
	}
}

func TestPlatformSetupProcessRunnerTimesOutAndTerminatesJob(t *testing.T) {
	t.Parallel()

	result, err := (platformSetupProcessRunner{}).RunSetupProcess(
		context.Background(),
		setupProcessRequest{
			ExecutablePath: "cmd.exe",
			Arguments:      []string{"/D", "/S", "/C", "ping 127.0.0.1 -n 10 > nul"},
			WorkingDir:     t.TempDir(),
			Timeout:        50 * time.Millisecond,
			OutputLimit:    1024,
		},
	)
	if err == nil || !strings.Contains(err.Error(), "deadline exceeded") {
		t.Fatalf("RunSetupProcess() error = %v", err)
	}
	if !result.TimedOut || result.Cancelled {
		t.Fatalf("result = %+v", result)
	}
}
