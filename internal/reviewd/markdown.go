package reviewed

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

// generatedNotice marks every file visionreviewd writes, so readers (and
// agents) know manual edits are overwritten on the next pass.
const generatedNotice = "<!-- visionreviewd: generated file - manual edits are overwritten on the next pass -->"

// timeFormat is the human display format used in markdown.
const timeFormat = "2006-01-02 15:04"

// agentFooter tells consuming agents what to do with the file.
const agentFooter = "---\n\n*Point an agent (e.g. Crush) at this file and ask it to fix the issues\nabove, then regenerate the screenshots for a fresh review.*"

// shortSHA renders the first 12 hex chars of a digest for display.
const shortSHALength = 12

// RenderViewReview renders the markdown review file for one view.
func RenderViewReview(project string, viewKey ViewKey, capture Captured, review Reviewed) string {
	var doc strings.Builder

	doc.WriteString(generatedNotice)
	doc.WriteString("\n\n# ")
	doc.WriteString(viewKey.Page)
	doc.WriteString(" · ")
	doc.WriteString(viewKey.Theme)
	doc.WriteString(" · ")
	doc.WriteString(viewKey.Viewport)
	doc.WriteString("\n\n")

	fmt.Fprintf(&doc, "- **Project:** %s\n", project)
	fmt.Fprintf(&doc, "- **View:** `%s`\n", viewKey)
	fmt.Fprintf(&doc, "- **Score:** %s\n", FormatScore(review.Score))
	fmt.Fprintf(&doc, "- **Model:** `%s`\n", review.Model)
	fmt.Fprintf(&doc, "- **Reviewed:** %s\n", review.ReviewedAt.UTC().Format(timeFormat))
	capturedAt := capture.CapturedAt.UTC().Format(timeFormat)
	fmt.Fprintf(&doc, "- **Capture:** `%s` (%s)\n", ShortSHA(review.SHA256), capturedAt)
	fmt.Fprintf(&doc, "- **Source:** `%s`\n", capture.SourcePath)
	doc.WriteString("- **Comparisons:** [comparisons/](comparisons/)\n")

	doc.WriteString("\n## Model review\n\n")
	doc.WriteString(strings.TrimSpace(review.Markdown))
	doc.WriteString("\n\n")
	doc.WriteString(agentFooter)
	doc.WriteString("\n")

	return doc.String()
}

// RenderComparison renders the markdown comparison file for a BEFORE/AFTER
// pair of one view.
func RenderComparison(project string, viewKey ViewKey, compared Compared) string {
	var doc strings.Builder

	doc.WriteString(generatedNotice)
	doc.WriteString("\n\n# Comparison · ")
	doc.WriteString(viewKey.Page)
	doc.WriteString(" · ")
	doc.WriteString(viewKey.Theme)
	doc.WriteString(" · ")
	doc.WriteString(viewKey.Viewport)
	doc.WriteString("\n\n")

	fmt.Fprintf(&doc, "- **Project:** %s\n", project)
	fmt.Fprintf(&doc, "- **View:** `%s`\n", viewKey)
	fmt.Fprintf(&doc, "- **Compared:** %s\n", compared.ComparedAt.UTC().Format(timeFormat))
	fmt.Fprintf(&doc, "- **Model:** `%s`\n", compared.Model)
	fmt.Fprintf(&doc, "- **Before:** `%s`\n", ShortSHA(compared.BeforeSHA256))
	fmt.Fprintf(&doc, "- **After:** `%s`\n", ShortSHA(compared.AfterSHA256))

	doc.WriteString("\n## Model comparison\n\n")
	doc.WriteString(strings.TrimSpace(compared.Markdown))
	doc.WriteString("\n")

	return doc.String()
}

// IndexRow is one view line in a project INDEX.md.
type IndexRow struct {
	ViewKey   ViewKey
	Score     int
	Previous  int
	UpdatedAt time.Time
}

// RenderIndex renders a project's INDEX.md with a score table and trend
// arrows. Rows are sorted by view key.
func RenderIndex(project string, generatedAt time.Time, rows []IndexRow) string {
	sorted := make([]IndexRow, len(rows))
	copy(sorted, rows)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].ViewKey.String() < sorted[j].ViewKey.String()
	})

	var doc strings.Builder

	doc.WriteString(generatedNotice)
	doc.WriteString("\n\n# ")
	doc.WriteString(project)
	doc.WriteString(" - UI review index\n\n")
	fmt.Fprintf(&doc, "_Generated %s by visionreviewd._\n\n", generatedAt.UTC().Format(timeFormat))

	doc.WriteString("| View | Score | Trend | Updated |\n")
	doc.WriteString("| ---- | ----- | ----- | ------- |\n")

	for _, row := range sorted {
		fmt.Fprintf(&doc, "| [`%s`](views/%s.md) | %s | %s | %s |\n",
			sanitizeCell(row.ViewKey.String()),
			row.ViewKey,
			sanitizeCell(FormatScore(row.Score)),
			sanitizeCell(Trend(row.Score, row.Previous)),
			row.UpdatedAt.UTC().Format(timeFormat),
		)
	}

	doc.WriteString("\n")

	return doc.String()
}

// FormatScore renders a score for markdown; unknown scores become "?".
func FormatScore(score int) string {
	if score == ScoreUnknown {
		return "?"
	}

	return fmt.Sprintf("%d/10", score)
}

// Trend renders the score delta as an arrow with sign, or a dot when either
// score is unknown.
func Trend(score, previous int) string {
	if score == ScoreUnknown || previous == ScoreUnknown {
		return "·"
	}

	switch delta := score - previous; {
	case delta > 0:
		return fmt.Sprintf("▲ +%d", delta)
	case delta < 0:
		return fmt.Sprintf("▼ %d", delta)
	default:
		return "▬ ="
	}
}

// ShortSHA renders the display prefix of a digest.
func ShortSHA(sha string) string {
	if len(sha) <= shortSHALength {
		return sha
	}

	return sha[:shortSHALength]
}

// sanitizeCell makes a string safe for a markdown table cell.
func sanitizeCell(value string) string {
	return strings.ReplaceAll(value, "|", "\\|")
}
