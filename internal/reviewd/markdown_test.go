package reviewed

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

var testStamp = time.Date(2026, 8, 16, 20, 31, 0, 0, time.UTC)

func TestRenderViewReview(t *testing.T) {
	t.Parallel()

	viewKey := ViewKey{Page: "Settings", Theme: "dark", Viewport: "desktop"}
	capture := Captured{
		SourcePath: "/home/lars/projects/DiscordSync/internal/web/testdata/visual/Settings--dark--desktop.png",
		BlobPath:   "/data/images/abc.png",
		SHA256:     "0123456789abcdef0123456789abcdef",
		CapturedAt: testStamp.Add(-time.Hour),
	}
	review := Reviewed{
		SHA256:     capture.SHA256,
		Model:      "test-model",
		Markdown:   "## Summary\nFine.",
		Score:      8,
		ReviewedAt: testStamp,
	}

	doc := RenderViewReview("discordsync", viewKey, capture, review)

	assertions := []string{
		generatedNotice,
		"# Settings · dark · desktop",
		"- **Project:** discordsync",
		"- **View:** `Settings--dark--desktop`",
		"- **Score:** 8/10",
		"- **Model:** `test-model`",
		"- **Capture:** `0123456789ab`",
		"## Model review",
		"## Summary\nFine.",
		"[comparisons/](comparisons/)",
	}

	for _, want := range assertions {
		if !strings.Contains(doc, want) {
			t.Fatalf("view review should contain %q:\n%s", want, doc)
		}
	}
}

func TestRenderViewReviewUnknownScore(t *testing.T) {
	t.Parallel()

	viewKey := ViewKey{Page: "P", Theme: "dark", Viewport: "desktop"}
	review := Reviewed{Model: "m", Markdown: "no score", Score: ScoreUnknown, ReviewedAt: testStamp}

	doc := RenderViewReview("proj", viewKey, Captured{SHA256: "abc"}, review)

	if !strings.Contains(doc, "- **Score:** ?") {
		t.Fatalf("unknown score should render as ?:\n%s", doc)
	}
}

func TestRenderComparison(t *testing.T) {
	t.Parallel()

	viewKey := ViewKey{Page: "Settings", Theme: "dark", Viewport: "desktop"}
	compared := Compared{
		BeforeSHA256: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		AfterSHA256:  "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		Model:        "test-model",
		Markdown:     "## What improved\n- spacing",
		ComparedAt:   testStamp,
	}

	doc := RenderComparison("discordsync", viewKey, compared)

	assertions := []string{
		generatedNotice,
		"# Comparison · Settings · dark · desktop",
		"- **Before:** `aaaaaaaaaaaa`",
		"- **After:** `bbbbbbbbbbbb`",
		"## Model comparison",
		"## What improved\n- spacing",
	}

	for _, want := range assertions {
		if !strings.Contains(doc, want) {
			t.Fatalf("comparison should contain %q:\n%s", want, doc)
		}
	}
}

func TestRenderIndexSortsRowsAndShowsTrend(t *testing.T) {
	t.Parallel()

	rows := []IndexRow{
		{
			ViewKey: ViewKey{Page: "Zeta", Theme: "dark", Viewport: "desktop"},
			Score:   5, Previous: 7, UpdatedAt: testStamp,
		},
		{
			ViewKey: ViewKey{Page: "Alpha", Theme: "dark", Viewport: "desktop"},
			Score:   9, Previous: 8, UpdatedAt: testStamp,
		},
		{
			ViewKey: ViewKey{Page: "Mid", Theme: "dark", Viewport: "desktop"},
			Score:   6, Previous: 6, UpdatedAt: testStamp,
		},
		{
			ViewKey: ViewKey{Page: "Unk", Theme: "dark", Viewport: "desktop"},
			Score:   ScoreUnknown, UpdatedAt: testStamp,
		},
	}

	doc := RenderIndex("discordsync", testStamp, rows)

	assertions := []string{
		"# discordsync - UI review index",
		"| [`Alpha--dark--desktop`](views/Alpha--dark--desktop.md) | 9/10 | ▲ +1 |",
		"| [`Mid--dark--desktop`](views/Mid--dark--desktop.md) | 6/10 | ▬ = |",
		"| [`Unk--dark--desktop`](views/Unk--dark--desktop.md) | ? | · |",
		"| [`Zeta--dark--desktop`](views/Zeta--dark--desktop.md) | 5/10 | ▼ -2 |",
	}

	for _, want := range assertions {
		if !strings.Contains(doc, want) {
			t.Fatalf("index should contain %q:\n%s", want, doc)
		}
	}

	alphaAt := strings.Index(doc, "Alpha")

	zetaAt := strings.Index(doc, "Zeta")
	if alphaAt > zetaAt {
		t.Fatal("index rows should be sorted by view key")
	}
}

func TestTrend(t *testing.T) {
	t.Parallel()

	tests := []struct {
		score, previous int
		want            string
	}{
		{score: 8, previous: 6, want: "▲ +2"},
		{score: 3, previous: 9, want: "▼ -6"},
		{score: 5, previous: 5, want: "▬ ="},
		{score: ScoreUnknown, previous: 5, want: "·"},
		{score: 5, previous: ScoreUnknown, want: "·"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			t.Parallel()

			if got := Trend(tt.score, tt.previous); got != tt.want {
				t.Fatalf("Trend(%d, %d) = %q, want %q", tt.score, tt.previous, got, tt.want)
			}
		})
	}
}

func TestWriterPaths(t *testing.T) {
	t.Parallel()

	writer := NewWriter("/reviews")

	viewKey := ViewKey{Page: "Settings", Theme: "dark", Viewport: "desktop"}

	if got, want := writer.ViewReviewPath("discordsync", viewKey),
		filepath.Join("/reviews", "discordsync", "views", "Settings--dark--desktop.md"); got != want {
		t.Fatalf("view path = %q, want %q", got, want)
	}

	wantComparison := filepath.Join(
		"/reviews", "discordsync", "comparisons", "2026-08-16_2031_Settings--dark--desktop.md")
	if got := writer.ComparisonPath("discordsync", viewKey, testStamp); got != wantComparison {
		t.Fatalf("comparison path = %q, want %q", got, wantComparison)
	}

	if got, want := writer.IndexPath("discordsync"),
		filepath.Join("/reviews", "discordsync", "INDEX.md"); got != want {
		t.Fatalf("index path = %q, want %q", got, want)
	}
}

func TestWriterSanitizesProjectName(t *testing.T) {
	t.Parallel()

	writer := NewWriter("/reviews")
	viewKey := ViewKey{Page: "P", Theme: "dark", Viewport: "desktop"}

	got := writer.ViewReviewPath("../../etc/passwd", viewKey)

	if want := filepath.Join("/reviews", "etc-passwd", "views", "P--dark--desktop.md"); got != want {
		t.Fatalf("project name should be sanitized: got %q, want %q", got, want)
	}
}

func TestWriterWritesAtomically(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writer := NewWriter(dir)
	viewKey := ViewKey{Page: "Settings", Theme: "dark", Viewport: "desktop"}

	if err := writer.WriteViewReview("discordsync", viewKey, "# hello"); err != nil {
		t.Fatalf("WriteViewReview: %v", err)
	}

	content, err := os.ReadFile(writer.ViewReviewPath("discordsync", viewKey))
	if err != nil {
		t.Fatalf("read back: %v", err)
	}

	if string(content) != "# hello" {
		t.Fatalf("content = %q", content)
	}

	entries, err := os.ReadDir(filepath.Join(dir, "discordsync", "views"))
	if err != nil {
		t.Fatalf("list views dir: %v", err)
	}

	if len(entries) != 1 {
		t.Fatalf("temp files should be renamed away, found %d entries", len(entries))
	}

	if err := writer.WriteIndex("discordsync", "# index"); err != nil {
		t.Fatalf("WriteIndex: %v", err)
	}

	if err := writer.WriteComparison("discordsync", viewKey, testStamp, "# compare"); err != nil {
		t.Fatalf("WriteComparison: %v", err)
	}

	for _, path := range []string{
		writer.IndexPath("discordsync"),
		writer.ComparisonPath("discordsync", viewKey, testStamp),
	} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected file %s: %v", path, err)
		}
	}
}

func TestShortSHA(t *testing.T) {
	t.Parallel()

	if got := ShortSHA("0123456789abcdef"); got != "0123456789ab" {
		t.Fatalf("ShortSHA = %q", got)
	}

	if got := ShortSHA("short"); got != "short" {
		t.Fatalf("ShortSHA should pass through short values, got %q", got)
	}
}
