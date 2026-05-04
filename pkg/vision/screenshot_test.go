package vision

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

const mockResponseText = "mock response"

func TestScreenshotAnalyzer_Builder(t *testing.T) {
	t.Parallel()
	model := testModel()

	tests := []struct {
		name        string
		setup       func() *ScreenshotAnalyzer
		wantPrompt  string
		wantTokens  int64
		wantTemp    float64
		wantRetries int
		wantTimeout time.Duration
	}{
		{
			name: "default",
			setup: func() *ScreenshotAnalyzer {
				return NewScreenshotAnalyzer(model)
			},
			wantPrompt:  DefaultScreenshotPrompt,
			wantTokens:  0,
			wantTemp:    0,
			wantRetries: 0,
			wantTimeout: 0,
		},
		{
			name: "all options",
			setup: func() *ScreenshotAnalyzer {
				return NewScreenshotAnalyzer(model).
					WithSystemPrompt("custom prompt").
					WithMaxOutputTokens(500).
					WithTemperature(0.7).
					WithMaxRetries(5).
					WithRequestTimeout(30 * time.Second)
			},
			wantPrompt:  "custom prompt",
			wantTokens:  500,
			wantTemp:    0.7,
			wantRetries: 5,
			wantTimeout: 30 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			sa := tt.setup()
			if sa.config.SystemPrompt != tt.wantPrompt {
				t.Errorf("expected system prompt %q, got %q", tt.wantPrompt, sa.config.SystemPrompt)
			}
			if sa.config.MaxOutputTokens != tt.wantTokens {
				t.Errorf("expected max tokens %d, got %d", tt.wantTokens, sa.config.MaxOutputTokens)
			}
			if sa.config.Temperature != tt.wantTemp {
				t.Errorf("expected temperature %f, got %f", tt.wantTemp, sa.config.Temperature)
			}
			if sa.config.MaxRetries != tt.wantRetries {
				t.Errorf("expected max retries %d, got %d", tt.wantRetries, sa.config.MaxRetries)
			}
			if sa.config.RequestTimeout != tt.wantTimeout {
				t.Errorf("expected timeout %v, got %v", tt.wantTimeout, sa.config.RequestTimeout)
			}
		})
	}
}

func TestScreenshotAnalyzer_DefaultPrompt(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		want string
	}{
		{
			name: "default prompt",
			want: DefaultScreenshotPrompt,
		},
	}

	for _, tt := range tests {
		// capture range variable
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			sa := NewScreenshotAnalyzer(testModel())
			if sa.config.SystemPrompt != tt.want {
				t.Errorf("expected %q, got %q", tt.want, sa.config.SystemPrompt)
			}
		})
	}
}

func TestScreenshotAnalyzer_AnalyzeScreenshot(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	screenshotPath := filepath.Join(tmpDir, "screenshot.png")
	if err := os.WriteFile(screenshotPath, []byte("fake"), 0o644); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name     string
		path     string
		wantErr  bool
		wantText string
	}{
		{
			name:     "valid file",
			path:     screenshotPath,
			wantErr:  false,
			wantText: mockResponseText,
		},
		{
			name:    "missing file",
			path:    "/nonexistent/file.png",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		// capture range variable
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			sa := NewScreenshotAnalyzer(testModel())
			ctx := context.Background()

			result, err := sa.AnalyzeScreenshot(ctx, "describe", tt.path)
			if AssertErr(t, tt.wantErr, nil, err) {
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			AssertEq(t, result.Text, tt.wantText)
		})
	}
}

func TestScreenshotAnalyzer_AnalyzeScreenshotImage(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		img         *ImageSource
		prompt      string
		wantText    string
		wantErr     bool
		wantErrType error
	}{
		{
			name:     "valid image",
			img:      ImageSrc(),
			prompt:   "describe",
			wantText: mockResponseText,
			wantErr:  false,
		},
		{
			name:        "empty prompt",
			img:         ImageSrc(),
			prompt:      "",
			wantErr:     true,
			wantErrType: ErrEmptyPrompt,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			sa := NewScreenshotAnalyzer(testModel())
			ctx := context.Background()

			result, err := sa.AnalyzeScreenshotImage(ctx, tt.prompt, tt.img)
			if AssertErr(t, tt.wantErr, tt.wantErrType, err) {
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			AssertEq(t, result.Text, tt.wantText)
		})
	}
}

func TestScreenshotAnalyzer_AnalyzeScreenshots(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	path1 := filepath.Join(tmpDir, "1.png")
	path2 := filepath.Join(tmpDir, "2.png")
	for _, p := range []string{path1, path2} {
		if err := os.WriteFile(p, []byte("fake"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	tests := []struct {
		name     string
		paths    []string
		wantErr  bool
		wantText string
	}{
		{
			name:     "valid files",
			paths:    []string{path1, path2},
			wantErr:  false,
			wantText: mockResponseText,
		},
		{
			name:    "missing file",
			paths:   []string{path1, "/nonexistent/2.png"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			sa := NewScreenshotAnalyzer(testModel())
			ctx := context.Background()

			result, err := sa.AnalyzeScreenshots(ctx, "compare", tt.paths...)
			if AssertErr(t, tt.wantErr, nil, err) {
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			AssertEq(t, result.Text, tt.wantText)
		})
	}
}

func TestScreenshotAnalyzer_AnalyzeScreenshotImages(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		images   []*ImageSource
		wantText string
		wantErr  bool
	}{
		{
			name: "multiple images",
			images: []*ImageSource{
				{Data: []byte("test1"), MediaType: "image/png", Filename: "test1.png"},
				{Data: []byte("test2"), MediaType: "image/png", Filename: "test2.png"},
			},
			wantText: mockResponseText,
			wantErr:  false,
		},
		{
			name: "empty prompt",
			images: []*ImageSource{
				{Data: []byte("test"), MediaType: "image/png", Filename: "test.png"},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			sa := NewScreenshotAnalyzer(testModel())
			ctx := context.Background()

			prompt := "compare"
			if tt.wantErr {
				prompt = ""
			}

			result, err := sa.AnalyzeScreenshotImages(ctx, prompt, tt.images...)
			if AssertErr(t, tt.wantErr, nil, err) {
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			AssertEq(t, result.Text, tt.wantText)
		})
	}
}
