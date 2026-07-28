package main

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"charm.land/fantasy"
	"github.com/larsartmann/vision-review-agent/pkg/vision"
	"github.com/stretchr/testify/require"
)

// cliMockModel is a minimal fantasy.LanguageModel for cmd/vision tests. It
// returns canned success responses so the happy paths of runAnalysis,
// runStructured, and print* can be exercised without a live provider.
type cliMockModel struct {
	generateErr error
}

func (m *cliMockModel) Generate(_ context.Context, _ fantasy.Call) (*fantasy.Response, error) {
	if m.generateErr != nil {
		return nil, m.generateErr
	}

	return &fantasy.Response{
		Content:      []fantasy.Content{fantasy.TextContent{Text: "mock analysis"}},
		FinishReason: fantasy.FinishReasonStop,
		Usage:        fantasy.Usage{TotalTokens: 42, InputTokens: 10, OutputTokens: 32},
	}, nil
}

func (m *cliMockModel) Stream(_ context.Context, _ fantasy.Call) (fantasy.StreamResponse, error) {
	return func(yield func(fantasy.StreamPart) bool) {
		_ = yield(fantasy.StreamPart{Type: fantasy.StreamPartTypeTextDelta, Delta: "mock stream"})
		_ = yield(fantasy.StreamPart{
			Type:         fantasy.StreamPartTypeFinish,
			Usage:        fantasy.Usage{TotalTokens: 42, InputTokens: 10, OutputTokens: 32},
			FinishReason: fantasy.FinishReasonStop,
		})
	}, nil
}

func (m *cliMockModel) GenerateObject(
	_ context.Context,
	_ fantasy.ObjectCall,
) (*fantasy.ObjectResponse, error) {
	return &fantasy.ObjectResponse{
		Object:       map[string]any{"layout": "test layout", "score": float64(8)},
		RawText:      `{"layout":"test layout","score":8}`,
		Usage:        fantasy.Usage{TotalTokens: 42, InputTokens: 10, OutputTokens: 32},
		FinishReason: fantasy.FinishReasonStop,
	}, nil
}

func (m *cliMockModel) StreamObject(
	_ context.Context,
	_ fantasy.ObjectCall,
) (fantasy.ObjectStreamResponse, error) {
	return func(yield func(fantasy.ObjectStreamPart) bool) {
		_ = yield(fantasy.ObjectStreamPart{
			Type:   fantasy.ObjectStreamPartTypeObject,
			Object: map[string]any{"layout": "test layout"},
		})
		_ = yield(fantasy.ObjectStreamPart{
			Type:         fantasy.ObjectStreamPartTypeFinish,
			Usage:        fantasy.Usage{TotalTokens: 42},
			FinishReason: fantasy.FinishReasonStop,
		})
	}, nil
}

func (m *cliMockModel) Provider() string { return "mock" }
func (m *cliMockModel) Model() string    { return "mock-model" }

// newTestAgent builds a vision.Agent backed by cliMockModel for run* tests.
func newTestAgent(t *testing.T, model fantasy.LanguageModel) *vision.Agent {
	t.Helper()

	agent, err := vision.NewAgent(vision.Config{Model: model})
	require.NoError(t, err)

	return agent
}

// testImage returns a minimal ImageSource suitable for mock-backed run* tests.
func testImage() *vision.ImageSource {
	img, _ := vision.NewImageSource([]byte("test"), vision.MediaTypePNG, "test.png")

	return img
}

func TestLoadImagesValidFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "img.png")
	require.NoError(t, os.WriteFile(path, []byte("fake-png"), 0o600))

	images, err := loadImages([]string{path})
	require.NoError(t, err)
	require.Len(t, images, 1)
	require.Equal(t, "img.png", images[0].Filename)
}

func TestLoadImagesMultipleFiles(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	a := filepath.Join(dir, "a.png")
	b := filepath.Join(dir, "b.png")

	require.NoError(t, os.WriteFile(a, []byte("x"), 0o600))
	require.NoError(t, os.WriteFile(b, []byte("y"), 0o600))

	images, err := loadImages([]string{a, b})
	require.NoError(t, err)
	require.Len(t, images, 2)
}

func TestLoadImagesMissingFileReturnsError(t *testing.T) {
	t.Parallel()

	_, err := loadImages([]string{"/nonexistent/path/xyz.png"})
	require.Error(t, err)
	require.ErrorContains(t, err, "loading /nonexistent/path/xyz.png")
}

func TestLoadImagesEmptyArgsReturnsEmpty(t *testing.T) {
	t.Parallel()

	images, err := loadImages([]string{})
	require.NoError(t, err)
	require.Empty(t, images)
}

func TestPrintJSON(t *testing.T) {
	t.Parallel()

	result := &vision.AnalyzeResult{
		Text: "hello world",
		Usage: fantasy.Usage{
			InputTokens:  5,
			OutputTokens: 7,
			TotalTokens:  12,
		},
	}

	var buf bytes.Buffer
	printJSON(&buf, result)

	out := buf.String()
	require.Contains(t, out, "hello world")
	require.Contains(t, out, `"totalTokens": 12`)
	require.Contains(t, out, `"inputTokens": 5`)
	require.Contains(t, out, `"outputTokens": 7`)
}

func TestPrintTextNonStreamed(t *testing.T) {
	t.Parallel()

	result := &vision.AnalyzeResult{
		Text:  "the analysis",
		Usage: fantasy.Usage{TotalTokens: 99},
	}

	var buf bytes.Buffer
	printText(&buf, result, false)

	out := buf.String()
	require.Contains(t, out, "--- Analysis ---")
	require.Contains(t, out, "the analysis")
	require.Contains(t, out, "Tokens used: 99")
}

func TestPrintTextStreamedOmitsAnalysisHeader(t *testing.T) {
	t.Parallel()

	result := &vision.AnalyzeResult{
		Text:  "already streamed",
		Usage: fantasy.Usage{TotalTokens: 3},
	}

	var buf bytes.Buffer
	printText(&buf, result, true)

	out := buf.String()
	require.NotContains(t, out, "already streamed", "streamed output must not reprint text")
	require.Contains(t, out, "Tokens used: 3")
}

func TestRunAnalysisText(t *testing.T) {
	t.Parallel()

	agent := newTestAgent(t, &cliMockModel{})
	cfg := &config{prompt: "review this"}
	images := []*vision.ImageSource{testImage()}

	var buf bytes.Buffer
	runAnalysis(context.Background(), agent, cfg, images, &buf, io.Discard)

	out := buf.String()
	require.Contains(t, out, "mock analysis")
	require.Contains(t, out, "Tokens used: 42")
}

func TestRunAnalysisJSON(t *testing.T) {
	t.Parallel()

	agent := newTestAgent(t, &cliMockModel{})
	cfg := &config{prompt: "review this", jsonOutput: true}
	images := []*vision.ImageSource{testImage()}

	var buf bytes.Buffer
	runAnalysis(context.Background(), agent, cfg, images, &buf, io.Discard)

	out := buf.String()
	require.Contains(t, out, `"text": "mock analysis"`)
	require.Contains(t, out, `"totalTokens": 42`)
}

func TestRunAnalysisStream(t *testing.T) {
	t.Parallel()

	agent := newTestAgent(t, &cliMockModel{})
	cfg := &config{prompt: "review this", stream: true}
	images := []*vision.ImageSource{testImage()}

	var buf bytes.Buffer
	runAnalysis(context.Background(), agent, cfg, images, &buf, io.Discard)

	out := buf.String()
	require.Contains(t, out, "mock stream")
}

func TestRunAnalysisStructuredDispatchesToRunStructured(t *testing.T) {
	t.Parallel()

	agent := newTestAgent(t, &cliMockModel{})
	cfg := &config{prompt: "review this", structured: true}
	images := []*vision.ImageSource{testImage()}

	var buf bytes.Buffer
	runAnalysis(context.Background(), agent, cfg, images, &buf, io.Discard)

	out := buf.String()
	require.Contains(t, out, "test layout")
}

func TestRunStructured(t *testing.T) {
	t.Parallel()

	agent := newTestAgent(t, &cliMockModel{})
	cfg := &config{prompt: "review this"}
	images := []*vision.ImageSource{testImage()}

	var buf bytes.Buffer
	runStructured(context.Background(), agent, cfg, images, &buf, io.Discard)

	out := buf.String()
	require.Contains(t, out, "test layout")
}

func TestCreateProviderOpenAICompatWithBaseURL(t *testing.T) {
	t.Setenv("OPENAICOMPAT_BASE_URL", "http://localhost:8080/v1")
	t.Setenv("OPENAICOMPAT_API_KEY", "")

	provider, err := createProvider("openaicompat")
	require.NoError(t, err)
	require.NotNil(t, provider)
}

func TestCreateProviderOpenAICompatMissingBaseURL(t *testing.T) {
	t.Setenv("OPENAICOMPAT_BASE_URL", "")

	_, err := createProvider("openaicompat")
	require.Error(t, err)
	require.ErrorIs(t, err, errEnvVarNotSet)
}

func TestPrintAnalysisErrorClassifiedModelError(t *testing.T) {
	t.Parallel()

	modelErr := &vision.ModelError{
		Kind:       vision.KindRateLimited,
		Op:         "analyze",
		Cause:      &fantasy.ProviderError{Message: "slow down", StatusCode: 429},
		StatusCode: 429,
	}

	var buf bytes.Buffer
	printAnalysisError(&buf, modelErr, false)

	out := buf.String()
	require.Contains(t, out, string(vision.KindRateLimited))
	require.Contains(t, out, "rate-limiting")
}

func TestPrintAnalysisErrorUnclassified(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	printAnalysisError(&buf, errors.New("uncategorized boom"), false)

	out := buf.String()
	require.Contains(t, out, "Error:")
	require.Contains(t, out, "uncategorized boom")
}

func FuzzParseFlags(f *testing.F) {
	f.Add("screenshot.png")
	f.Add("-version")
	f.Add("-prompt hello img.png")
	f.Add("-temperature 0.5 -json shot.png")
	f.Add("-bogus")

	f.Fuzz(func(t *testing.T, input string) {
		args := strings.Fields(input)

		fs := flag.NewFlagSet("vision-fuzz", flag.ContinueOnError)
		fs.SetOutput(io.Discard)

		cfg, err := parseFlags(fs, args)
		if err != nil {
			return // parse errors are expected for arbitrary input
		}

		require.NotNil(t, cfg)
	})
}
