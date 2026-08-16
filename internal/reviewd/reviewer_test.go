package reviewed

import (
	"context"
	"sync"
	"testing"
	"time"

	"charm.land/fantasy"
)

// mockReviewModel is a mock fantasy.LanguageModel for reviewd tests. It
// returns canned markdown (typically containing a score line) from Generate.
// Set generateErr to inject failures. Set streamMarkdown to serve Stream.
type mockReviewModel struct {
	mu               sync.Mutex
	markdown         string
	generateErr      error
	generateCalls    int
	capturedPrompts  []fantasy.Prompt
	capturedCallsLen []int // number of parts in each Generate call's Prompt
}

func newMockReviewModel(markdown string) *mockReviewModel {
	return &mockReviewModel{markdown: markdown}
}

func (m *mockReviewModel) Generate(_ context.Context, in fantasy.Call) (*fantasy.Response, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.generateCalls++
	m.capturedPrompts = append(m.capturedPrompts, in.Prompt)
	m.capturedCallsLen = append(m.capturedCallsLen, len(in.Prompt.Parts()))

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

func (m *mockReviewModel) Stream(_ context.Context, _ fantasy.Call) (fantasy.StreamResponse, error) {
	return nil, fantasy.ProviderError{Code: "not_implemented", Message: "streaming unused in reviewd tests"}
}

func (m *mockReviewModel) GenerateObject(
	_ context.Context,
	_ fantasy.ObjectCall,
) (*fantasy.ObjectResponse, error) {
	return nil, fantasy.ProviderError{Code: "not_implemented", Message: "object mode unused in reviewd"}
}

func (m *mockReviewModel) StreamObject(
	_ context.Context,
	_ fantasy.ObjectCall,
) (fantasy.ObjectStreamResponse, error) {
	return nil, fantasy.ProviderError{Code: "not_implemented", Message: "object mode unused in reviewd"}
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
		return fantasy.Prompt{}
	}

	return m.capturedPrompts[len(m.capturedPrompts)-1]
}

// writeTestPNG writes a minimal valid PNG file and returns its path.
func writeTestPNG(t *testing.T, content string) string {
	t.Helper()

	// 1x1 transparent PNG.
	png := []byte{
		0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, 0x00, 0x00, 0x00, 0x0D,
		0x49, 0x48, 0x44, 0x52, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
		0x08, 0x06, 0x00, 0x00, 0x00, 0x1F, 0x15, 0xC4, 0x89, 0x00, 0x00, 0x00,
		0x0D, 0x49, 0x44, 0x41, 0x54, 0x78, 0x9C, 0x62, 0x00, 0x01, 0x00, 0x00,
		0x05, 0x00, 0x01, 0x0D, 0x0A, 0x2D, 0xB4, 0x00, 0x00, 0x00, 0x00, 0x49,
		0x45, 0x4E, 0x44, 0xAE, 0x42, 0x60, 0x82,
	}

	path := t.TempDir() + "/" + content + ".png"
	if err := writeFile(path, png); err != nil {
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

	imagePath := writeTestPNG(t, "Settings--dark--desktop")
	viewKey := ViewKey{Page: "Settings", Theme: "dark", Viewport: "desktop"}

	result, err := reviewer.Review(t.Context(), viewKey, imagePath)
	if err != nil {
		t.Fatalf("Review: %v", err)
	}

	if result.Score != 8 {
		t.Fatalf("score = %d, want 8", result.Score)
	}

	if result.Markdown != "## Summary\nLooks fine.\n\nScore: 8/10" {
		t.Fatalf("markdown = %q", result.Markdown)
	}

	prompt := model.lastPrompt()
	foundContext := false
	for _, part := range prompt.Parts() {
		if text, ok := part.(fantasy.TextPart); ok {
			if contains(text.Text, `"Settings"`) && contains(text.Text, "Review this UI screenshot") {
				foundContext = true
			}
		}
	}

	if !foundContext {
		t.Fatalf("prompt should contain view context and review instruction, got %+v", prompt)
	}
}

func TestReviewerReviewUnknownScore(t *testing.T) {
	t.Parallel()

	model := newMockReviewModel("## Summary\nNo score line at all.")
	reviewer, err := NewReviewer(model, "test-model", time.Minute)
	if err != nil {
		t.Fatalf("NewReviewer: %v", err)
	}

	result, err := reviewer.Review(t.Context(), ViewKey{Page: "P", Theme: "dark", Viewport: "desktop"}, writeTestPNG(t, "P--dark--desktop"))
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
	model.generateErr = fantasy.ProviderError{Code: "500", Message: "boom"}
	reviewer, err := NewReviewer(model, "test-model", time.Minute)
	if err != nil {
		t.Fatalf("NewReviewer: %v", err)
	}

	_, err = reviewer.Review(t.Context(), ViewKey{Page: "P", Theme: "dark", Viewport: "desktop"}, writeTestPNG(t, "P--dark--desktop"))
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

	_, err = reviewer.Review(t.Context(), ViewKey{Page: "P", Theme: "dark", Viewport: "desktop"}, "/nonexistent/shot.png")
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

	before := writeTestPNG(t, "before")
	after := writeTestPNG(t, "after")

	result, err := reviewer.Compare(t.Context(), ViewKey{Page: "P", Theme: "dark", Viewport: "desktop"}, before, after)
	if err != nil {
		t.Fatalf("Compare: %v", err)
	}

	if result.Score != 9 {
		t.Fatalf("score = %d, want 9", result.Score)
	}

	if len(model.lastPrompt().Parts()) < 3 {
		t.Fatalf("compare prompt should carry instruction + 2 images, got %d parts", len(model.lastPrompt().Parts()))
	}
}

func TestReviewerCompareMissingAfter(t *testing.T) {
	t.Parallel()

	model := newMockReviewModel("Score: 1/10")
	reviewer, err := NewReviewer(model, "test-model", time.Minute)
	if err != nil {
		t.Fatalf("NewReviewer: %v", err)
	}

	_, err = reviewer.Compare(t.Context(), ViewKey{Page: "P", Theme: "dark", Viewport: "desktop"}, writeTestPNG(t, "before"), "/nonexistent/after.png")
	if err == nil {
		t.Fatal("missing after image should error")
	}

	if model.calls() != 0 {
		t.Fatalf("model should not be called, called %d times", model.calls())
	}
}

func contains(haystack, needle string) bool {
	return len(haystack) >= len(needle) && (haystack == needle || indexOf(haystack, needle) >= 0)
}

func indexOf(haystack, needle string) int {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return i
		}
	}

	return -1
}
