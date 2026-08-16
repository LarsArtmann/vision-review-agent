package reviewed

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// fakeContentPart is one part of a multimodal chat message content array.
//
//nolint:tagliatelle // OpenAI wire format uses snake_case
type fakeContentPart struct {
	Type     string `json:"type"`
	Text     string `json:"text"`
	ImageURL *struct {
		URL string `json:"url"`
	} `json:"image_url"`
}

// fakeChatMessage is one message of an OpenAI chat request.
type fakeChatMessage struct {
	Role    string          `json:"role"`
	Content json.RawMessage `json:"content"`
}

// fakeChatRequest is the subset of the OpenAI chat request the fake server
// inspects.
type fakeChatRequest struct {
	Model    string            `json:"model"`
	Messages []fakeChatMessage `json:"messages"`
}

// fakeServerCounts records what the fake endpoint received.
type fakeServerCounts struct {
	mu         sync.Mutex
	requests   int
	imageParts int
	promptText string
}

func (c *fakeServerCounts) observe(imageParts int, promptText string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.requests++
	c.imageParts = imageParts
	c.promptText = promptText
}

func (c *fakeServerCounts) snapshot() (int, int, string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.requests, c.imageParts, c.promptText
}

// fakeMessage is one assistant/user message in the OpenAI wire format.
type fakeMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// fakeChoice is one completion choice in the OpenAI wire format.
//
//nolint:tagliatelle // OpenAI wire format uses snake_case
type fakeChoice struct {
	Index        int         `json:"index"`
	Message      fakeMessage `json:"message"`
	FinishReason string      `json:"finish_reason"`
}

// fakeUsage reports token counts in the OpenAI wire format.
//
//nolint:tagliatelle // OpenAI wire format uses snake_case
type fakeUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// fakeCompletion is the OpenAI chat completion response the fake server
// returns.
type fakeCompletion struct {
	ID      string       `json:"id"`
	Object  string       `json:"object"`
	Created int64        `json:"created"`
	Model   string       `json:"model"`
	Choices []fakeChoice `json:"choices"`
	Usage   fakeUsage    `json:"usage"`
}

// startFakeModelServer runs an OpenAI-compatible chat completions endpoint
// that replies with markdown. It counts requests, image parts, and remembers
// the concatenated text prompt.
func startFakeModelServer(t *testing.T, markdown string) (*httptest.Server, *fakeServerCounts) {
	t.Helper()

	counts := &fakeServerCounts{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodPost || !strings.HasSuffix(req.URL.Path, "/chat/completions") {
			http.Error(
				w,
				fmt.Sprintf("unexpected %s %s", req.Method, req.URL.Path),
				http.StatusNotFound,
			)

			return
		}

		var request fakeChatRequest

		if err := json.NewDecoder(req.Body).Decode(&request); err != nil {
			http.Error(w, "decode request: "+err.Error(), http.StatusBadRequest)

			return
		}

		imageParts, promptText := summarizeFakeRequest(request)
		counts.observe(imageParts, promptText)

		completion := fakeCompletion{
			ID:      "chatcmpl-fake",
			Object:  "chat.completion",
			Created: time.Now().UTC().Unix(),
			Model:   request.Model,
			Choices: []fakeChoice{{
				Index:        0,
				Message:      fakeMessage{Role: "assistant", Content: markdown},
				FinishReason: "stop",
			}},
			Usage: fakeUsage{PromptTokens: 10, CompletionTokens: 5, TotalTokens: 15},
		}

		w.Header().Set("Content-Type", "application/json")

		if err := json.NewEncoder(w).Encode(completion); err != nil {
			http.Error(w, "encode response: "+err.Error(), http.StatusInternalServerError)
		}
	}))
	t.Cleanup(server.Close)

	return server, counts
}

// summarizeFakeRequest counts image_url parts and concatenates text parts
// across all messages.
func summarizeFakeRequest(request fakeChatRequest) (int, string) {
	imageParts := 0

	texts := make([]string, 0, len(request.Messages))

	for _, message := range request.Messages {
		var parts []fakeContentPart

		if err := json.Unmarshal(message.Content, &parts); err != nil {
			var plain string
			if stringErr := json.Unmarshal(message.Content, &plain); stringErr == nil {
				texts = append(texts, plain)
			}

			continue
		}

		for _, part := range parts {
			switch {
			case part.ImageURL != nil:
				imageParts++
			case part.Text != "":
				texts = append(texts, part.Text)
			}
		}
	}

	return imageParts, strings.Join(texts, "\n")
}

// fakeModelConfig returns daemon config pointing at the fake endpoint.
func fakeModelConfig(serverURL string) Config {
	return Config{
		Model:   "fake-vision-model",
		BaseURL: serverURL + "/v1",
		APIKey:  "test-key",
		Timeout: 10 * time.Second,
	}
}

func TestReviewerReviewOverHTTP(t *testing.T) {
	t.Parallel()

	server, counts := startFakeModelServer(
		t,
		"## Review\nSpacing is consistent.\n\n**Score: 7/10**",
	)

	reviewer, err := NewReviewerFromConfig(t.Context(), fakeModelConfig(server.URL))
	if err != nil {
		t.Fatalf("NewReviewerFromConfig: %v", err)
	}

	shotPath := filepath.Join(t.TempDir(), "Home--dark--desktop.png")

	if err := os.WriteFile(shotPath, scanTestPNG, 0o644); err != nil {
		t.Fatalf("write shot: %v", err)
	}

	result, err := reviewer.Review(
		t.Context(),
		ViewKey{Page: "Home", Theme: "dark", Viewport: "desktop"},
		shotPath,
	)
	if err != nil {
		t.Fatalf("Review over HTTP: %v", err)
	}

	if result.Score != 7 {
		t.Fatalf("score = %d, want 7", result.Score)
	}

	if !strings.Contains(result.Markdown, "Spacing is consistent") {
		t.Fatalf("markdown missing fake server output:\n%s", result.Markdown)
	}

	requests, imageParts, promptText := counts.snapshot()
	if requests != 1 {
		t.Fatalf("requests = %d, want 1", requests)
	}

	if imageParts != 1 {
		t.Fatalf("image parts = %d, want 1", imageParts)
	}

	if !strings.Contains(promptText, "Home") {
		t.Fatalf("prompt should mention the view page, got:\n%s", promptText)
	}
}

func TestReviewerCompareOverHTTP(t *testing.T) {
	t.Parallel()

	server, counts := startFakeModelServer(t, "## Comparison\nBetter alignment.\n\n**Score: 9/10**")

	reviewer, err := NewReviewerFromConfig(t.Context(), fakeModelConfig(server.URL))
	if err != nil {
		t.Fatalf("NewReviewerFromConfig: %v", err)
	}

	dir := t.TempDir()
	beforePath := writeComparePNG(t, dir, "before", scanTestPNG)
	afterPath := writeComparePNG(t, dir, "Home--dark--desktop", changedScanPNG())

	result, err := reviewer.Compare(
		t.Context(),
		ViewKey{Page: "Home", Theme: "dark", Viewport: "desktop"},
		beforePath,
		afterPath,
	)
	if err != nil {
		t.Fatalf("Compare over HTTP: %v", err)
	}

	if result.Score != 9 {
		t.Fatalf("score = %d, want 9", result.Score)
	}

	if !strings.Contains(result.Markdown, "Better alignment") {
		t.Fatalf("markdown missing fake server output:\n%s", result.Markdown)
	}

	requests, imageParts, _ := counts.snapshot()
	if requests != 1 {
		t.Fatalf("requests = %d, want 1", requests)
	}

	if imageParts != 2 {
		t.Fatalf("image parts = %d, want 2 (before and after)", imageParts)
	}
}

func TestReviewerOverHTTPWrapsUnreachableEndpoint(t *testing.T) {
	t.Parallel()

	config := Config{
		Model:   "fake-vision-model",
		BaseURL: "http://127.0.0.1:1/v1",
		Timeout: 2 * time.Second,
	}

	reviewer, err := NewReviewerFromConfig(context.Background(), config)
	if err != nil {
		t.Fatalf("NewReviewerFromConfig: %v", err)
	}

	shotPath := filepath.Join(t.TempDir(), "Home--dark--desktop.png")

	if err := os.WriteFile(shotPath, scanTestPNG, 0o644); err != nil {
		t.Fatalf("write shot: %v", err)
	}

	viewKey := ViewKey{Page: "Home", Theme: "dark", Viewport: "desktop"}

	if _, err := reviewer.Review(context.Background(), viewKey, shotPath); err == nil {
		t.Fatal("Review against an unreachable endpoint must fail")
	}
}
