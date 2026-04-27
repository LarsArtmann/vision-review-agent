package vision

import (
	"context"
	"errors"
	"testing"
)

type testReview struct {
	Layout string `json:"layout"`
	Score  int    `json:"score"`
}

func TestAnalyzeStructured_Success(t *testing.T) {
	agent, err := NewAgent(Config{Model: &mockModel{}})
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	img := &ImageSource{Data: []byte("test"), MediaType: "image/png"}

	result, err := AnalyzeStructured[testReview](ctx, agent, "analyze this", img)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected result, got nil")
	}
	if result.Object.Layout != "test layout" {
		t.Errorf("expected layout 'test layout', got %q", result.Object.Layout)
	}
	if result.Usage.TotalTokens != 10 {
		t.Errorf("expected 10 tokens, got %d", result.Usage.TotalTokens)
	}
}

func TestAnalyzeStructured_Validation(t *testing.T) {
	agent, err := NewAgent(Config{Model: &mockModel{}})
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	img := &ImageSource{Data: []byte("test"), MediaType: "image/png"}

	t.Run("empty prompt", func(t *testing.T) {
		_, err := AnalyzeStructured[testReview](ctx, agent, "", img)
		if !errors.Is(err, ErrEmptyPrompt) {
			t.Errorf("expected ErrEmptyPrompt, got %v", err)
		}
	})

	t.Run("no images", func(t *testing.T) {
		_, err := AnalyzeStructured[testReview](ctx, agent, "test")
		if !errors.Is(err, ErrNoImages) {
			t.Errorf("expected ErrNoImages, got %v", err)
		}
	})
}
