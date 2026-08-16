package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunNoArgsPrintsUsageAndExitsUsage(t *testing.T) {
	t.Parallel()

	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}

	if code := run(nil, stdout, stderr); code != exitUsage {
		t.Fatalf("exit = %d, want %d", code, exitUsage)
	}

	if !strings.Contains(stderr.String(), "Usage: visionreviewd") {
		t.Fatalf("stderr missing usage:\n%s", stderr)
	}
}

func TestRunHelpVariants(t *testing.T) {
	t.Parallel()

	for _, name := range []string{"help", "--help", "-h"} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}

			if code := run([]string{name}, stdout, stderr); code != exitOK {
				t.Fatalf("exit = %d, want %d", code, exitOK)
			}

			if !strings.Contains(stdout.String(), "Commands:") {
				t.Fatalf("stdout missing command list:\n%s", stdout)
			}

			if stderr.Len() != 0 {
				t.Fatalf("help must not write to stderr, got:\n%s", stderr)
			}
		})
	}
}

func TestRunVersion(t *testing.T) {
	t.Parallel()

	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}

	if code := run([]string{"version"}, stdout, stderr); code != exitOK {
		t.Fatalf("exit = %d, want %d", code, exitOK)
	}

	if !strings.HasPrefix(stdout.String(), "visionreviewd ") {
		t.Fatalf("stdout = %q, want version line", stdout.String())
	}
}

func TestRunUnknownCommand(t *testing.T) {
	t.Parallel()

	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}

	if code := run([]string{"explode"}, stdout, stderr); code != exitUsage {
		t.Fatalf("exit = %d, want %d", code, exitUsage)
	}

	if !strings.Contains(stderr.String(), `unknown command "explode"`) {
		t.Fatalf("stderr missing unknown-command error:\n%s", stderr)
	}
}

func TestRunDiscoverPrintsSuggestedConfig(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	shotsDir := filepath.Join(root, "myshop", "screenshots")

	if err := os.MkdirAll(shotsDir, 0o750); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	if err := os.WriteFile(filepath.Join(shotsDir, "Home--dark--desktop.png"), []byte("png"), 0o600); err != nil {
		t.Fatalf("write shot: %v", err)
	}

	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}

	if code := run([]string{"discover", root}, stdout, stderr); code != exitOK {
		t.Fatalf("exit = %d, want %d (stderr: %s)", code, exitOK, stderr)
	}

	if !strings.Contains(stdout.String(), `"myshop"`) {
		t.Fatalf("stdout missing discovered project:\n%s", stdout)
	}
}

func TestRunDiscoverRequiresRoot(t *testing.T) {
	t.Parallel()

	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}

	if code := run([]string{"discover"}, stdout, stderr); code != exitUsage {
		t.Fatalf("exit = %d, want %d", code, exitUsage)
	}
}

func TestRunOnceMissingConfigFails(t *testing.T) {
	t.Parallel()

	missing := filepath.Join(t.TempDir(), "missing.json")

	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}

	code := run([]string{"once", "-config", missing}, stdout, stderr)
	if code != exitFailed {
		t.Fatalf("exit = %d, want %d", code, exitFailed)
	}

	if !strings.Contains(stderr.String(), missing) {
		t.Fatalf("stderr should mention config path:\n%s", stderr)
	}
}

func TestRunDaemonMissingConfigFails(t *testing.T) {
	t.Parallel()

	missing := filepath.Join(t.TempDir(), "missing.json")

	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}

	code := run([]string{"run", "-config", missing}, stdout, stderr)
	if code != exitFailed {
		t.Fatalf("exit = %d, want %d", code, exitFailed)
	}

	if !strings.Contains(stderr.String(), missing) {
		t.Fatalf("stderr should mention config path:\n%s", stderr)
	}
}

func TestRunCompareRequiresProject(t *testing.T) {
	t.Parallel()

	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}

	code := run([]string{"compare", "a.png", "b.png"}, stdout, stderr)
	if code != exitUsage {
		t.Fatalf("exit = %d, want %d", code, exitUsage)
	}

	if !strings.Contains(stderr.String(), "-project is required") {
		t.Fatalf("stderr missing project requirement:\n%s", stderr)
	}
}

func TestRunCompareRequiresTwoPaths(t *testing.T) {
	t.Parallel()

	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}

	code := run([]string{"compare", "-project", "myapp", "a.png"}, stdout, stderr)
	if code != exitUsage {
		t.Fatalf("exit = %d, want %d", code, exitUsage)
	}
}

func TestRunUnimplementedCommandsFailCleanly(t *testing.T) {
	t.Parallel()

	for _, name := range []string{"events", "replay", "doctor"} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}

			if code := run([]string{name}, stdout, stderr); code != exitFailed {
				t.Fatalf("exit = %d, want %d", code, exitFailed)
			}

			if !strings.Contains(stderr.String(), "not implemented yet") {
				t.Fatalf("stderr should say not implemented yet:\n%s", stderr)
			}
		})
	}
}
