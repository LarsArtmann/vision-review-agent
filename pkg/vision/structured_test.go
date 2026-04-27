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

func TestAnalyzeStructured(t *testing.T) {
	t.Parallel()

	agent, err := NewAgent(Config{Model: &mockModel{}})
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	img := &ImageSource{Data: []byte("test"), MediaType: "image/png", Filename: "test.png"}

	tests := []struct {
		name        string
		prompt      string
		images      []*ImageSource
		wantLayout  string
		wantTokens  int64
		wantErr     bool
		wantErrType error
	}{
		{
			name:       "success",
			prompt:     "analyze this",
			images:     []*ImageSource{img},
			wantLayout: "test layout",
			wantTokens: 10,
			wantErr:    false,
		},
		{
			name:        "empty prompt",
			prompt:      "",
			images:      []*ImageSource{img},
			wantErr:     true,
			wantErrType: ErrEmptyPrompt,
		},
		{
			name:        "no images",
			prompt:      "test",
			images:      nil,
			wantErr:     true,
			wantErrType: ErrNoImages,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result, err := AnalyzeStructured[testReview](ctx, agent, tt.prompt, tt.images...)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
					return
				}
				if tt.wantErrType != nil && !errors.Is(err, tt.wantErrType) {
					t.Errorf("expected %v, got %v", tt.wantErrType, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result == nil {
				t.Fatal("expected result, got nil")
			}
			if result.Object.Layout != tt.wantLayout {
				t.Errorf("expected layout %q, got %q", tt.wantLayout, result.Object.Layout)
			}
			if result.Usage.TotalTokens != tt.wantTokens {
				t.Errorf("expected %d tokens, got %d", tt.wantTokens, result.Usage.TotalTokens)
			}
		})
	}
}
