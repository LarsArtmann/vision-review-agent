package reviewed

import (
	"context"
	"errors"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"charm.land/fantasy"
)

// mockReviewModel is a mock fantasy.LanguageModel for reviewed tests. Generate
// returns the canned markdown (typically containing a score line); set
// generateErr to inject failures. Every call's prompt is captured for
// assertions.
type mockReviewModel struct {
	mu              sync.Mutex
	markdown        string
	generateErr     error
	generateCalls   int
	capturedPrompts []fantasy.Prompt
}

func newMockReviewModel(markdown string) *mockReviewModel {
	return &mockReviewModel{markdown: markdown}
}

func (m *mockReviewModel) Generate(_ context.Context, in fantasy.Call) (*fantasy.Response, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.generateCalls++
	m.capturedPrompts = append(m.capturedPrompts, in.Prompt)

	if m.generateErr != nil {
		return nil, m.generateErr
	}

	return &fantasy.Response{
		Content: []fantasy.Content{
			fantasy.TextContent{Text: m.markdown},
		},
		FinishReason: fantasy.FinishReasonStop,
		Usage:        fantasy.Usage{TotalTokens: 12},
	}, nil
}

func (m *mockReviewModel) Provider() string { return "mock" }

func (m *mockReviewModel) Model() string { return "mock-review-model" }

func (m *mockReviewModel) Stream(_ context.Context, _ fantasy.Call) (fantasy.StreamResponse, error) {
	return nil, errors.New("streaming unused in reviewed tests")
}

func (m *mockReviewModel) GenerateObject(
	_ context.Context,
	_ fantasy.ObjectCall,
) (*fantasy.ObjectResponse, error) {
	return nil, errors.New("object mode unused in reviewed")
}

func (m *mockReviewModel) StreamObject(
	_ context.Context,
	_ fantasy.ObjectCall,
) (fantasy.ObjectStreamResponse, error) {
	return nil, errors.New("object mode unused in reviewed")
}

func (m *mockReviewModel) calls() int {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.generateCalls
}

func (m *mockReviewModel) lastPrompt() fantasy.Prompt {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.capturedPrompts) == 0 {
		return nil
	}

	return m.capturedPrompts[len(m.capturedPrompts)-1]
}

// promptText concatenates all text parts of a prompt.
func promptText(prompt fantasy.Prompt) string {
	texts := make([]string, 0, len(prompt))

	for _, message := range prompt {
		for _, part := range message.Content {
			if text, ok := part.(fantasy.TextPart); ok {
				texts = append(texts, text.Text)
			}
		}
	}

	return strings.Join(texts, "\n")
}

// countFileParts counts image file parts in a prompt.
func countFileParts(prompt fantasy.Prompt) int {
	count := 0

	for _, message := range prompt {
		for _, part := range message.Content {
			if _, ok := part.(fantasy.FilePart); ok {
				count++
			}
		}
	}

	return count
}

// writeTestPNG writes a minimal 1x1 PNG and returns its path.
func writeTestPNG(t *testing.T, name string) string {
	t.Helper()

	png := []byte{
		0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, 0x00, 0x00, 0x00, 0x0D,
		0x49, 0x48, 0x44, 0x52, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
		0x08, 0x06, 0x00, 0x00, 0x00, 0x1F, 0x15, 0xC4, 0x89, 0x00, 0x00, 0x00,
		0x0D, 0x49, 0x44, 0x41, 0x54, 0x78, 0x9C, 0x62, 0x00, 0x01, 0x00, 0x00,
		0x05, 0x00, 0x01, 0x0D, 0x0A, 0x2D, 0xB4, 0x00, 0x00, 0x00, 0x00, 0x49,
		0x45, 0x4E, 0x44, 0xAE, 0x42, 0x60, 0x82,
	}

	path := t.TempDir() + "/" + name + ".png"
	if err := os.WriteFile(path, png, 0o644); err != nil {
		t.Fatalf("write test png: %v", err)
	}

	return path
}

func TestReviewerReviewParsesScoreAndPrompt(t *testing.T) {
	t.Parallel()

	model := newMockReviewModel("## Summary\nLooks fine.\n\nScore: 8/10")

	reviewer, err := NewReviewer(model, "test-model", time.Minute)
	if err != nil {
		t.Fatalf("NewReviewer: %v", err)
	}

	viewKey := ViewKey{Page: "Settings", Theme: "dark", Viewport: "desktop"}

	result, err := reviewer.Review(t.Context(), viewKey, writeTestPNG(t, "Settings--dark--desktop"))
	if err != nil {
		t.Fatalf("Review: %v", err)
	}

	if result.Score != 8 {
		t.Fatalf("score = %d, want 8", result.Score)
	}

	if result.Markdown != "## Summary\nLooks fine.\n\nScore: 8/10" {
		t.Fatalf("markdown = %q", result.Markdown)
	}

	text := promptText(model.lastPrompt())
	if !strings.Contains(text, `"Settings"`) || !strings.Contains(text, "Review this UI screenshot") {
		t.Fatalf("prompt should contain view context and review instruction, got:\n%s", text)
	}

	if countFileParts(model.lastPrompt()) != 1 {
		t.Fatalf("review prompt should carry exactly 1 image")
	}
}

func TestReviewerReviewUnknownScore(t *testing.T) {
	t.Parallel()

	model := newMockReviewModel("## Summary\nNo score line at all.")

	reviewer, err := NewReviewer(model, "test-model", time.Minute)
	if err != nil {
		t.Fatalf("NewReviewer: %v", err)
	}

	result, err := reviewer.Review(
		t.Context(),
		ViewKey{Page: "P", Theme: "dark", Viewport: "desktop"},
		writeTestPNG(t, "P--dark--desktop"),
	)
	if err != nil {
		t.Fatalf("Review: %v", err)
	}

	if result.Score != ScoreUnknown {
		t.Fatalf("score = %d, want ScoreUnknown", result.Score)
	}
}

func TestReviewerReviewModelError(t *testing.T) {
	t.Parallel()

	model := newMockReviewModel("")
	model.generateErr = errors.New("model exploded")

	reviewer, err := NewReviewer(model, "test-model", time.Minute)
	if err != nil {
		t.Fatalf("NewReviewer: %v", err)
	}

	_, err = reviewer.Review(
		t.Context(),
		ViewKey{Page: "P", Theme: "dark", Viewport: "desktop"},
		writeTestPNG(t, "P--dark--desktop"),
	)
	if err == nil {
		t.Fatal("model error should surface")
	}
}

func TestReviewerReviewMissingImage(t *testing.T) {
	t.Parallel()

	model := newMockReviewModel("Score: 5/10")

	reviewer, err := NewReviewer(model, "test-model", time.Minute)
	if err != nil {
		t.Fatalf("NewReviewer: %v", err)
	}

	_, err = reviewer.Review(
		t.Context(),
		ViewKey{Page: "P", Theme: "dark", Viewport: "desktop"},
		"/nonexistent/shot.png",
	)
	if err == nil {
		t.Fatal("missing image should error")
	}

	if model.calls() != 0 {
		t.Fatalf("model should not be called on missing image, called %d times", model.calls())
	}
}

func TestReviewerCompareSendsTwoImages(t *testing.T) {
	t.Parallel()

	model := newMockReviewModel("## What improved\n- spacing\n\nScore: 9/10")

	reviewer, err := NewReviewer(model, "test-model", time.Minute)
	if err != nil {
		t.Fatalf("NewReviewer: %v", err)
	}

	viewKey := ViewKey{Page: "P", Theme: "dark", Viewport: "desktop"}

	result, err := reviewer.Compare(t.Context(), viewKey, writeTestPNG(t, "before"), writeTestPNG(t, "after"))
	if err != nil {
		t.Fatalf("Compare: %v", err)
	}

	if result.Score != 9 {
		t.Fatalf("score = %d, want 9", result.Score)
	}

	if got := countFileParts(model.lastPrompt()); got != 2 {
		t.Fatalf("compare prompt should carry 2 images, got %d", got)
	}

	if text := promptText(model.lastPrompt()); !strings.Contains(text, "BEFORE") {
		t.Fatalf("compare prompt should explain before/after order, got:\n%s", text)
	}
}

func TestReviewerCompareMissingAfter(t *testing.T) {
	t.Parallel()

	model := newMockReviewModel("Score: 1/10")

	reviewer, err := NewReviewer(model, "test-model", time.Minute)
	if err != nil {
		t.Fatalf("NewReviewer: %v", err)
	}

	_, err = reviewer.Compare(
		t.Context(),
		ViewKey{Page: "P", Theme: "dark", Viewport: "desktop"},
		writeTestPNG(t, "before"),
		"/nonexistent/after.png",
	)
	if err == nil {
		t.Fatal("missing after image should error")
	}

	if model.calls() != 0 {
		t.Fatalf("model should not be called, called %d times", model.calls())
	}
}

func TestReviewerModelAccessor(t *testing.T) {
	t.Parallel()

	reviewer, err := NewReviewer(newMockReviewModel(""), "some-model", time.Minute)
	if err != nil {
		t.Fatalf("NewReviewer: %v", err)
	}

	if reviewer.Model() != "some-model" {
		t.Fatalf("Model() = %q, want some-model", reviewer.Model())
	}
}
