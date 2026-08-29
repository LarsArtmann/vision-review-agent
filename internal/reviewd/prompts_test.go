package reviewed

import (
	"strings"
	"testing"
)

// sectionOrder asserts headings appear in the prompt in the given order: the
// prompt demands "EXACTLY this markdown structure", so the contract itself
// must be pinned.
func sectionOrder(t *testing.T, prompt string, sections ...string) {
	t.Helper()

	last := -1

	for _, section := range sections {
		index := strings.Index(prompt, section)
		if index < 0 {
			t.Fatalf("prompt missing section %q:\n%s", section, prompt)
		}

		if index <= last {
			t.Fatalf("section %q out of order (index %d, previous %d):\n%s", section, index, last, prompt)
		}

		last = index
	}
}

func TestReviewPromptContract(t *testing.T) {
	t.Parallel()

	prompt := ReviewPrompt(ViewKey{Page: "Settings", Theme: "dark", Viewport: "desktop"})

	if !strings.HasPrefix(prompt, `View: page "Settings", theme "dark", viewport "desktop".`+"\n") {
		t.Fatalf("prompt must open with the view context, got:\n%s", prompt)
	}

	sectionOrder(t, prompt, "## Summary", "## Strengths", "## Issues", "## Recommendations")

	if !strings.Contains(prompt, `exactly "Score: N/10"`) || !strings.Contains(prompt, "integer 0-10") {
		t.Fatalf("prompt must pin the score contract line:\n%s", prompt)
	}

	if strings.Contains(prompt, "BEFORE") || strings.Contains(prompt, "AFTER") {
		t.Fatalf("single-view prompt must not reference BEFORE/AFTER:\n%s", prompt)
	}
}

func TestComparePromptContract(t *testing.T) {
	t.Parallel()

	prompt := ComparePrompt(ViewKey{Page: "Login", Theme: "light", Viewport: "mobile"})

	if !strings.HasPrefix(prompt, `View: page "Login", theme "light", viewport "mobile".`+"\n") {
		t.Fatalf("prompt must open with the view context, got:\n%s", prompt)
	}

	if !strings.Contains(prompt, "Image 1 is the BEFORE version") ||
		!strings.Contains(prompt, "Image 2 is the AFTER version.") {
		t.Fatalf("compare prompt must pin image order, got:\n%s", prompt)
	}

	sectionOrder(t, prompt, "## What improved", "## What got worse", "## Unchanged problems", "## Verdict")

	if !strings.Contains(prompt, "rating the AFTER (second) image") {
		t.Fatalf("compare score contract must rate the AFTER image:\n%s", prompt)
	}

	if strings.Contains(prompt, "## Summary") {
		t.Fatalf("compare prompt must not leak single-review sections:\n%s", prompt)
	}
}
