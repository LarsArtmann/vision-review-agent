package vision

import (
	"context"
	"errors"
	"testing"

	"charm.land/fantasy"
)

// Test string constants.
const (
	testNameMissingFile = "missing file"
	testLayout          = "test layout"
	testEmptyPrompt     = "empty prompt"
)

// ImageSrc returns a test ImageSource with default test values.
// If filename is empty, "test.png" is used.
func ImageSrc(filename ...string) *ImageSource {
	name := "test.png"
	if len(filename) > 0 {
		name = filename[0]
	}
	return &ImageSource{
		Data:      []byte("test"),
		MediaType: MediaTypePNG,
		Filename:  name,
	}
}

// testModel returns a singleton mock model for testing.
func testModel() *mockModel {
	return &mockModel{}
}

// testConfig returns a Config with the test model and temperature set.
func testConfig(temp float64) Config {
	return Config{
		Model:       testModel(),
		Temperature: temp,
	}
}

// AssertErr is a helper for testing error cases in table-driven tests.
// It handles the common pattern of checking wantErr and optional wantErrType.
// Returns true if an expected error was found and test should return early.
func AssertErr(t *testing.T, wantErr bool, wantErrType, err error) bool {
	t.Helper()
	if wantErr {
		if err == nil {
			t.Error("expected error, got nil")
		} else if wantErrType != nil && !errors.Is(err, wantErrType) {
			t.Errorf("expected %v, got %v", wantErrType, err)
		}
		return true
	}
	return false
}

// AssertEq is a helper for asserting that a string value equals the expected value.
// Use for field comparison assertions in table-driven tests.
func AssertEq(t *testing.T, got, want string) {
	t.Helper()
	if got != want {
		t.Errorf("expected %q, got %q", want, got)
	}
}

// AssertError is a helper for checking error types in table-driven tests.
// It returns true if the error matches wantErrType via errors.Is.
func AssertError(t *testing.T, wantErrType, err error) bool {
	t.Helper()
	if !errors.Is(err, wantErrType) {
		t.Errorf("expected %v, got %v", wantErrType, err)
		return true
	}
	return false
}

// AssertGotEq is a helper for asserting got == want in table-driven tests.
// name is a description of what is being tested (e.g. "DetectImageFormat()").
func AssertGotEq(t *testing.T, name string, got, want any) {
	t.Helper()
	if got != want {
		t.Errorf("%s = %v, want %v", name, got, want)
	}
}

// mockModel is a mock implementation of fantasy.LanguageModel for testing.
type mockModel struct{}

func (m *mockModel) Generate(_ context.Context, _ fantasy.Call) (*fantasy.Response, error) {
	return &fantasy.Response{
		Content: []fantasy.Content{
			fantasy.TextContent{Text: "mock response"},
		},
		FinishReason: fantasy.FinishReasonStop,
		Usage:        fantasy.Usage{TotalTokens: 10},
	}, nil
}

func (m *mockModel) Stream(_ context.Context, _ fantasy.Call) (fantasy.StreamResponse, error) {
	return func(yield func(fantasy.StreamPart) bool) {
		_ = yield(fantasy.StreamPart{
			Type: fantasy.StreamPartTypeTextStart,
			ID:   "1",
		})
		_ = yield(fantasy.StreamPart{
			Type:  fantasy.StreamPartTypeTextDelta,
			ID:    "1",
			Delta: "mock response",
		})
		_ = yield(fantasy.StreamPart{
			Type: fantasy.StreamPartTypeTextEnd,
			ID:   "1",
		})
		_ = yield(fantasy.StreamPart{
			Type:         fantasy.StreamPartTypeFinish,
			Usage:        fantasy.Usage{TotalTokens: 10},
			FinishReason: fantasy.FinishReasonStop,
		})
	}, nil
}

func (m *mockModel) GenerateObject(
	_ context.Context,
	_ fantasy.ObjectCall,
) (*fantasy.ObjectResponse, error) {
	return &fantasy.ObjectResponse{
		Object: map[string]any{
			"layout": testLayout,
		},
		RawText:      `{"layout": "` + testLayout + `"}`,
		Usage:        fantasy.Usage{TotalTokens: 10},
		FinishReason: fantasy.FinishReasonStop,
	}, nil
}

func (m *mockModel) StreamObject(
	_ context.Context,
	_ fantasy.ObjectCall,
) (fantasy.ObjectStreamResponse, error) {
	return func(yield func(fantasy.ObjectStreamPart) bool) {
		_ = yield(fantasy.ObjectStreamPart{
			Type: fantasy.ObjectStreamPartTypeObject,
			Object: map[string]any{
				"layout": testLayout,
			},
		})
		_ = yield(fantasy.ObjectStreamPart{
			Type:         fantasy.ObjectStreamPartTypeFinish,
			Usage:        fantasy.Usage{TotalTokens: 10},
			FinishReason: fantasy.FinishReasonStop,
		})
	}, nil
}

func (m *mockModel) Provider() string { return "mock" }
func (m *mockModel) Model() string    { return "mock-model" }
