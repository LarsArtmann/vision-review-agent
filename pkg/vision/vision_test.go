package vision

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"charm.land/fantasy"
)

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr error
	}{
		{
			name: "valid config",
			config: Config{
				Model:       &mockModel{},
				Temperature: 0.5,
			},
			wantErr: nil,
		},
		{
			name:    "no model",
			config:  Config{},
			wantErr: ErrNoModel,
		},
		{
			name: "negative temperature",
			config: Config{
				Model:       &mockModel{},
				Temperature: -0.1,
			},
			wantErr: ErrInvalidTemperature,
		},
		{
			name: "temperature too high",
			config: Config{
				Model:       &mockModel{},
				Temperature: 2.1,
			},
			wantErr: ErrInvalidTemperature,
		},
		{
			name: "negative max tokens",
			config: Config{
				Model:           &mockModel{},
				MaxOutputTokens: -1,
			},
			wantErr: ErrInvalidMaxTokens,
		},
		{
			name: "boundary temperature 0",
			config: Config{
				Model:       &mockModel{},
				Temperature: 0,
			},
			wantErr: nil,
		},
		{
			name: "boundary temperature 2",
			config: Config{
				Model:       &mockModel{},
				Temperature: 2.0,
			},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("expected %v, got %v", tt.wantErr, err)
			}
		})
	}
}

func TestNewAgent(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: Config{
				Model:           &mockModel{},
				SystemPrompt:    "test prompt",
				MaxOutputTokens: 100,
				Temperature:     0.5,
				MaxRetries:      3,
			},
			wantErr: false,
		},
		{
			name:    "invalid config",
			config:  Config{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agent, err := NewAgent(tt.config)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if agent == nil {
				t.Error("expected agent, got nil")
			}
			if agent.config.Model == nil {
				t.Error("expected model to be set")
			}
		})
	}
}

func TestVisionAgent_Analyze(t *testing.T) {
	agent, err := NewAgent(Config{Model: &mockModel{}})
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	img := &ImageSource{Data: []byte("test"), MediaType: "image/png"}

	t.Run("success", func(t *testing.T) {
		result, err := agent.Analyze(ctx, "test prompt", img)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result == nil {
			t.Fatal("expected result, got nil")
		}
		if result.Text != "mock response" {
			t.Errorf("expected 'mock response', got %q", result.Text)
		}
		if result.Usage.TotalTokens != 10 {
			t.Errorf("expected 10 tokens, got %d", result.Usage.TotalTokens)
		}
		if result.RawResponse == nil {
			t.Error("expected RawResponse to be set")
		}
	})

	t.Run("multiple images", func(t *testing.T) {
		img2 := &ImageSource{Data: []byte("test2"), MediaType: "image/png"}
		result, err := agent.Analyze(ctx, "compare", img, img2)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Text != "mock response" {
			t.Errorf("expected 'mock response', got %q", result.Text)
		}
	})

	t.Run("empty prompt", func(t *testing.T) {
		_, err := agent.Analyze(ctx, "", img)
		if !errors.Is(err, ErrEmptyPrompt) {
			t.Errorf("expected ErrEmptyPrompt, got %v", err)
		}
	})

	t.Run("no images", func(t *testing.T) {
		_, err := agent.Analyze(ctx, "test", nil)
		if !errors.Is(err, ErrNoImages) {
			t.Errorf("expected ErrNoImages, got %v", err)
		}
	})

	t.Run("empty images", func(t *testing.T) {
		_, err := agent.Analyze(ctx, "test")
		if !errors.Is(err, ErrNoImages) {
			t.Errorf("expected ErrNoImages, got %v", err)
		}
	})
}

func TestVisionAgent_AnalyzeStream(t *testing.T) {
	agent, err := NewAgent(Config{Model: &mockModel{}})
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	img := &ImageSource{Data: []byte("test"), MediaType: "image/png"}

	t.Run("success", func(t *testing.T) {
		var chunks []string
		result, err := agent.AnalyzeStream(ctx, "test prompt", func(text string) error {
			chunks = append(chunks, text)
			return nil
		}, img)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result == nil {
			t.Fatal("expected result, got nil")
		}
		if result.Text != "mock response" {
			t.Errorf("expected 'mock response', got %q", result.Text)
		}
		if len(chunks) == 0 {
			t.Error("expected chunks to be received")
		}
	})

	t.Run("nil callback", func(t *testing.T) {
		result, err := agent.AnalyzeStream(ctx, "test prompt", nil, img)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Text != "mock response" {
			t.Errorf("expected 'mock response', got %q", result.Text)
		}
	})

	t.Run("empty prompt", func(t *testing.T) {
		_, err := agent.AnalyzeStream(ctx, "", nil, img)
		if !errors.Is(err, ErrEmptyPrompt) {
			t.Errorf("expected ErrEmptyPrompt, got %v", err)
		}
	})

	t.Run("no images", func(t *testing.T) {
		_, err := agent.AnalyzeStream(ctx, "test", nil, nil)
		if !errors.Is(err, ErrNoImages) {
			t.Errorf("expected ErrNoImages, got %v", err)
		}
	})
}

func TestAnalyzeResult_String(t *testing.T) {
	tests := []struct {
		name      string
		result    AnalyzeResult
		wantText  string
		wantToken int64
	}{
		{
			name:      "with text and tokens",
			result:    AnalyzeResult{Text: "test analysis", Usage: fantasy.Usage{TotalTokens: 42}},
			wantText:  "test analysis",
			wantToken: 42,
		},
		{
			name:      "empty text",
			result:    AnalyzeResult{Text: "", Usage: fantasy.Usage{TotalTokens: 0}},
			wantText:  "",
			wantToken: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := tt.result.String()
			if !strings.Contains(s, tt.wantText) {
				t.Errorf("String() should contain %q, got: %s", tt.wantText, s)
			}
			if !strings.Contains(s, fmt.Sprintf("%d", tt.wantToken)) {
				t.Errorf("String() should contain token count %d, got: %s", tt.wantToken, s)
			}
		})
	}
}

func TestWithTimeout(t *testing.T) {
	tests := []struct {
		name        string
		timeout     time.Duration
		wantDeadline bool
	}{
		{
			name:         "with timeout",
			timeout:      100 * time.Millisecond,
			wantDeadline: true,
		},
		{
			name:         "zero timeout",
			timeout:      0,
			wantDeadline: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agent, err := NewAgent(Config{
				Model:          &mockModel{},
				RequestTimeout: tt.timeout,
			})
			if err != nil {
				t.Fatal(err)
			}

			ctx := context.Background()
			ctx, cancel := agent.withTimeout(ctx)
			defer cancel()

			_, ok := ctx.Deadline()
			if ok != tt.wantDeadline {
				t.Errorf("expected deadline=%v, got %v", tt.wantDeadline, ok)
			}
		})
	}
}
