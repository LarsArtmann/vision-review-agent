package vision

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadImageFromFile(t *testing.T) {
	// Create a temporary image file
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.png")
	if err := os.WriteFile(tmpFile, []byte("fake png data"), 0o644); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{
			name:    "valid file",
			path:    tmpFile,
			wantErr: false,
		},
		{
			name:    "missing file",
			path:    filepath.Join(tmpDir, "missing.png"),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			img, err := LoadImageFromFile(tt.path)
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
			if img == nil {
				t.Error("expected image, got nil")
				return
			}
			if img.MediaType != "image/png" {
				t.Errorf("expected media type image/png, got %s", img.MediaType)
			}
			if string(img.Data) != "fake png data" {
				t.Error("data mismatch")
			}
		})
	}
}

func TestLoadImageFromFile_MediaTypeDetection(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		ext      string
		wantType string
	}{
		{".png", "image/png"},
		{".jpg", "image/jpeg"},
		{".jpeg", "image/jpeg"},
		{".gif", "image/gif"},
		{".webp", "image/webp"},
		{".unknown", "image/png"}, // fallback
	}

	for _, tt := range tests {
		t.Run(tt.ext, func(t *testing.T) {
			path := filepath.Join(tmpDir, "test"+tt.ext)
			if err := os.WriteFile(path, []byte("data"), 0o644); err != nil {
				t.Fatal(err)
			}
			img, err := LoadImageFromFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if img.MediaType != tt.wantType {
				t.Errorf("expected %s, got %s", tt.wantType, img.MediaType)
			}
		})
	}
}

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

func TestVisionAgent_Analyze_Validation(t *testing.T) {
	agent, err := NewAgent(Config{Model: &mockModel{}})
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	img := &ImageSource{Data: []byte("test"), MediaType: "image/png"}

	tests := []struct {
		name    string
		prompt  string
		images  []*ImageSource
		wantErr error
	}{
		{
			name:    "empty prompt",
			prompt:  "",
			images:  []*ImageSource{img},
			wantErr: ErrEmptyPrompt,
		},
		{
			name:    "no images",
			prompt:  "test",
			images:  nil,
			wantErr: ErrNoImages,
		},
		{
			name:    "empty images",
			prompt:  "test",
			images:  []*ImageSource{},
			wantErr: ErrNoImages,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := agent.Analyze(ctx, tt.prompt, tt.images...)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("expected %v, got %v", tt.wantErr, err)
			}
		})
	}
}

func TestVisionAgent_AnalyzeStream_Validation(t *testing.T) {
	agent, err := NewAgent(Config{Model: &mockModel{}})
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	img := &ImageSource{Data: []byte("test"), MediaType: "image/png"}

	tests := []struct {
		name    string
		prompt  string
		images  []*ImageSource
		wantErr error
	}{
		{
			name:    "empty prompt",
			prompt:  "",
			images:  []*ImageSource{img},
			wantErr: ErrEmptyPrompt,
		},
		{
			name:    "no images",
			prompt:  "test",
			images:  nil,
			wantErr: ErrNoImages,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := agent.AnalyzeStream(ctx, tt.prompt, nil, tt.images...)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("expected %v, got %v", tt.wantErr, err)
			}
		})
	}
}

func TestScreenshotAnalyzer_Builder(t *testing.T) {
	model := &mockModel{}

	sa := NewScreenshotAnalyzer(model).
		WithSystemPrompt("custom prompt").
		WithMaxOutputTokens(500).
		WithTemperature(0.7).
		WithMaxRetries(5).
		WithRequestTimeout(30 * time.Second)

	if sa.config.SystemPrompt != "custom prompt" {
		t.Errorf("expected system prompt 'custom prompt', got %q", sa.config.SystemPrompt)
	}
	if sa.config.MaxOutputTokens != 500 {
		t.Errorf("expected max tokens 500, got %d", sa.config.MaxOutputTokens)
	}
	if sa.config.Temperature != 0.7 {
		t.Errorf("expected temperature 0.7, got %f", sa.config.Temperature)
	}
	if sa.config.MaxRetries != 5 {
		t.Errorf("expected max retries 5, got %d", sa.config.MaxRetries)
	}
	if sa.config.RequestTimeout != 30*time.Second {
		t.Errorf("expected timeout 30s, got %v", sa.config.RequestTimeout)
	}
}

func TestScreenshotAnalyzer_DefaultPrompt(t *testing.T) {
	sa := NewScreenshotAnalyzer(&mockModel{})
	if sa.config.SystemPrompt != DefaultScreenshotPrompt {
		t.Error("expected default screenshot prompt")
	}
}

func TestWithTimeout(t *testing.T) {
	agent, err := NewAgent(Config{
		Model:          &mockModel{},
		RequestTimeout: 100 * time.Millisecond,
	})
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	ctx = agent.withTimeout(ctx)

	// Check that deadline is set
	_, ok := ctx.Deadline()
	if !ok {
		t.Error("expected deadline to be set")
	}
}

func TestWithTimeout_Zero(t *testing.T) {
	agent, err := NewAgent(Config{Model: &mockModel{}})
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	ctx = agent.withTimeout(ctx)

	// Check that deadline is NOT set
	_, ok := ctx.Deadline()
	if ok {
		t.Error("expected no deadline when timeout is zero")
	}
}
