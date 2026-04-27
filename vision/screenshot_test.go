package vision

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

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

func TestScreenshotAnalyzer_AnalyzeScreenshot(t *testing.T) {
	tmpDir := t.TempDir()
	screenshotPath := filepath.Join(tmpDir, "screenshot.png")
	if err := os.WriteFile(screenshotPath, []byte("fake"), 0o644); err != nil {
		t.Fatal(err)
	}

	sa := NewScreenshotAnalyzer(&mockModel{})
	ctx := context.Background()

	result, err := sa.AnalyzeScreenshot(ctx, "describe", screenshotPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Text != "mock response" {
		t.Errorf("expected 'mock response', got %q", result.Text)
	}
}

func TestScreenshotAnalyzer_AnalyzeScreenshot_MissingFile(t *testing.T) {
	sa := NewScreenshotAnalyzer(&mockModel{})
	ctx := context.Background()

	_, err := sa.AnalyzeScreenshot(ctx, "describe", "/nonexistent/file.png")
	if err == nil {
		t.Error("expected error for missing file")
	}
}

func TestScreenshotAnalyzer_AnalyzeScreenshotImage(t *testing.T) {
	sa := NewScreenshotAnalyzer(&mockModel{})
	ctx := context.Background()
	img := &ImageSource{Data: []byte("test"), MediaType: "image/png"}

	result, err := sa.AnalyzeScreenshotImage(ctx, "describe", img)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Text != "mock response" {
		t.Errorf("expected 'mock response', got %q", result.Text)
	}
}

func TestScreenshotAnalyzer_AnalyzeScreenshots(t *testing.T) {
	tmpDir := t.TempDir()
	path1 := filepath.Join(tmpDir, "1.png")
	path2 := filepath.Join(tmpDir, "2.png")
	for _, p := range []string{path1, path2} {
		if err := os.WriteFile(p, []byte("fake"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	sa := NewScreenshotAnalyzer(&mockModel{})
	ctx := context.Background()

	result, err := sa.AnalyzeScreenshots(ctx, "compare", path1, path2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Text != "mock response" {
		t.Errorf("expected 'mock response', got %q", result.Text)
	}
}

func TestScreenshotAnalyzer_AnalyzeScreenshots_MissingFile(t *testing.T) {
	tmpDir := t.TempDir()
	path1 := filepath.Join(tmpDir, "1.png")
	if err := os.WriteFile(path1, []byte("fake"), 0o644); err != nil {
		t.Fatal(err)
	}

	sa := NewScreenshotAnalyzer(&mockModel{})
	ctx := context.Background()

	_, err := sa.AnalyzeScreenshots(ctx, "compare", path1, "/nonexistent/2.png")
	if err == nil {
		t.Error("expected error for missing file")
	}
}

func TestScreenshotAnalyzer_AnalyzeScreenshotImages(t *testing.T) {
	sa := NewScreenshotAnalyzer(&mockModel{})
	ctx := context.Background()
	img1 := &ImageSource{Data: []byte("test1"), MediaType: "image/png"}
	img2 := &ImageSource{Data: []byte("test2"), MediaType: "image/png"}

	result, err := sa.AnalyzeScreenshotImages(ctx, "compare", img1, img2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Text != "mock response" {
		t.Errorf("expected 'mock response', got %q", result.Text)
	}
}

func TestScreenshotAnalyzer_AnalyzeScreenshotImage_Validation(t *testing.T) {
	sa := NewScreenshotAnalyzer(&mockModel{})
	ctx := context.Background()

	_, err := sa.AnalyzeScreenshotImage(
		ctx,
		"",
		&ImageSource{Data: []byte("test"), MediaType: "image/png"},
	)
	if !errors.Is(err, ErrEmptyPrompt) {
		t.Errorf("expected ErrEmptyPrompt, got %v", err)
	}
}
