package main

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	reviewed "github.com/larsartmann/vision-review-agent/internal/reviewd"
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

func TestRunDoctorNotImplementedYet(t *testing.T) {
	t.Parallel()

	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}

	if code := run([]string{"doctor"}, stdout, stderr); code != exitFailed {
		t.Fatalf("exit = %d, want %d", code, exitFailed)
	}

	if !strings.Contains(stderr.String(), "not implemented yet") {
		t.Fatalf("stderr should say not implemented yet:\n%s", stderr)
	}
}

// writeEventConfig writes a minimal valid daemon config pointing data and
// reviews at temp dirs and returns its path.
func writeEventConfig(t *testing.T, dataDir, reviewsDir string) string {
	t.Helper()

	config := fmt.Sprintf(
		"{\"model\": \"stub\", \"baseUrl\": \"http://127.0.0.1:9/v1\", \"dataDir\": %q, \"reviewsDir\": %q, \"projects\": {\"myapp\": [%q]}}",
		dataDir,
		reviewsDir,
		filepath.Join(t.TempDir(), "*.png"),
	)

	path := filepath.Join(t.TempDir(), "config.json")

	if err := os.WriteFile(path, []byte(config), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	return path
}

// seedJournal records one capture and one review event for the myapp Home
// view so events/replay have something to work with.
func seedJournal(t *testing.T, dataDir string) {
	t.Helper()

	store, err := reviewed.OpenStore(filepath.Join(dataDir, "events.db"), slog.Default())
	if err != nil {
		t.Fatalf("OpenStore: %v", err)
	}

	defer func() {
		if closeErr := store.Close(); closeErr != nil {
			t.Fatalf("close store: %v", closeErr)
		}
	}()

	viewKey, err := reviewed.ParseViewKey("Home--dark--desktop")
	if err != nil {
		t.Fatalf("ParseViewKey: %v", err)
	}

	stamp := time.Date(2026, 8, 16, 20, 0, 0, 0, time.UTC)
	ctx := context.Background()

	if err := store.RecordCapture(ctx, "myapp", viewKey, reviewed.Captured{
		SourcePath: "/shots/Home--dark--desktop.png",
		BlobPath:   "blobs/aa",
		SHA256:     "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		CapturedAt: stamp,
	}); err != nil {
		t.Fatalf("RecordCapture: %v", err)
	}

	if err := store.RecordReview(ctx, "myapp", viewKey, reviewed.Reviewed{
		SHA256:     "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		Model:      "stub",
		Markdown:   "fine",
		Score:      7,
		ReviewedAt: stamp.Add(time.Minute),
	}); err != nil {
		t.Fatalf("RecordReview: %v", err)
	}
}

func TestRunEventsPrintsJournal(t *testing.T) {
	t.Parallel()

	dataDir := t.TempDir()

	seedJournal(t, dataDir)

	configPath := writeEventConfig(t, dataDir, t.TempDir())

	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}

	code := run([]string{"events", "-config", configPath}, stdout, stderr)
	if code != exitOK {
		t.Fatalf("exit = %d, want %d (stderr: %s)", code, exitOK, stderr)
	}

	out := stdout.String()

	for _, want := range []string{
		"view.captured",
		"view.reviewed",
		"myapp:Home--dark--desktop",
		"score=7/10",
		"2 events",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("stdout missing %q:\n%s", want, out)
		}
	}
}

func TestRunEventsFiltersAndLimits(t *testing.T) {
	t.Parallel()

	dataDir := t.TempDir()

	seedJournal(t, dataDir)

	configPath := writeEventConfig(t, dataDir, t.TempDir())

	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}

	code := run([]string{"events", "-config", configPath, "-type", "view.reviewed", "-last", "1"}, stdout, stderr)
	if code != exitOK {
		t.Fatalf("exit = %d, want %d (stderr: %s)", code, exitOK, stderr)
	}

	out := stdout.String()

	if strings.Contains(out, "view.captured") {
		t.Fatalf("filtered output must not contain view.captured:\n%s", out)
	}

	if !strings.Contains(out, "1 events") {
		t.Fatalf("stdout should report 1 event:\n%s", out)
	}
}

func TestRunReplayRebuildsReviewsDir(t *testing.T) {
	t.Parallel()

	dataDir := t.TempDir()
	reviewsDir := t.TempDir()

	seedJournal(t, dataDir)

	configPath := writeEventConfig(t, dataDir, reviewsDir)

	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}

	code := run([]string{"replay", "-config", configPath}, stdout, stderr)
	if code != exitOK {
		t.Fatalf("exit = %d, want %d (stderr: %s)", code, exitOK, stderr)
	}

	if !strings.Contains(stdout.String(), "1 projects, 1 views, 1 reviews, 0 comparisons") {
		t.Fatalf("stdout missing replay summary:\n%s", stdout)
	}

	review, err := os.ReadFile(filepath.Join(reviewsDir, "myapp", "views", "Home--dark--desktop.md"))
	if err != nil {
		t.Fatalf("read replayed review: %v", err)
	}

	if !strings.Contains(string(review), "7/10") {
		t.Fatalf("replayed review missing score:\n%s", review)
	}

	if _, err := os.Stat(filepath.Join(reviewsDir, "myapp", "INDEX.md")); err != nil {
		t.Fatalf("replay must write INDEX.md: %v", err)
	}
}
