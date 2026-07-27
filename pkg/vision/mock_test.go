package vision

import (
	"context"
	"errors"
	"testing"

	"charm.land/fantasy"
	"github.com/onsi/gomega"
)

// Test string constants.
const (
	testNameMissingFile = "missing file"
	testLayout          = "test layout"
	testEmptyPrompt     = "empty prompt"
	testAnalysisText    = "test analysis"
	mockResponseText    = "mock response"
)

// testReview is the shared fixture type used by structured-output tests
// (both table-driven TestAnalyzeStructured and the BDD AnalyzeStructured spec).
type testReview struct {
	Layout string `json:"layout"`
	Score  int    `json:"score"`
}

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

// AssertGotEq is a helper for asserting got == want in table-driven tests.
// name is a description of what is being tested (e.g. "DetectImageFormat()").
func AssertGotEq(t *testing.T, name string, got, want any) {
	t.Helper()
	if got != want {
		t.Errorf("%s = %v, want %v", name, got, want)
	}
}

// mockModel is a mock implementation of fantasy.LanguageModel for testing.
// By default it returns canned success responses. Set the *Err fields to
// inject errors for testing classified error paths.
type mockModel struct {
	generateErr       error
	streamErr         error
	generateObjectErr error
}

func (m *mockModel) Generate(_ context.Context, _ fantasy.Call) (*fantasy.Response, error) {
	if m.generateErr != nil {
		return nil, m.generateErr
	}
	return &fantasy.Response{
		Content: []fantasy.Content{
			fantasy.TextContent{Text: "mock response"},
		},
		FinishReason: fantasy.FinishReasonStop,
		Usage:        fantasy.Usage{TotalTokens: 10},
	}, nil
}

func (m *mockModel) Stream(_ context.Context, _ fantasy.Call) (fantasy.StreamResponse, error) {
	if m.streamErr != nil {
		return nil, m.streamErr
	}
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
	if m.generateObjectErr != nil {
		return nil, m.generateObjectErr
	}
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

// setupAgent creates a fresh context and agent for BDD tests.
func setupAgent() (context.Context, *Agent) {
	ctx := context.Background()
	agent, err := NewAgent(Config{Model: testModel()})
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	return ctx, agent
}

// setupAgentWithModel creates a fresh context and agent using the provided mock.
// Use this when you need to inject errors into the mock.
func setupAgentWithModel(model *mockModel) (context.Context, *Agent) {
	ctx := context.Background()
	agent, err := NewAgent(Config{Model: model})
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	return ctx, agent
}

func (m *mockModel) Provider() string { return "mock" }
func (m *mockModel) Model() string    { return "mock-model" }
