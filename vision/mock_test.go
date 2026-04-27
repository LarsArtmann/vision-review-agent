package vision

import (
	"context"

	"charm.land/fantasy"
)

// mockModel is a mock implementation of fantasy.LanguageModel for testing.
type mockModel struct{}

func (m *mockModel) Generate(ctx context.Context, call fantasy.Call) (*fantasy.Response, error) {
	return &fantasy.Response{
		Content: []fantasy.Content{
			fantasy.TextContent{Text: "mock response"},
		},
		FinishReason: fantasy.FinishReasonStop,
		Usage:        fantasy.Usage{TotalTokens: 10},
	}, nil
}

func (m *mockModel) Stream(ctx context.Context, call fantasy.Call) (fantasy.StreamResponse, error) {
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

func (m *mockModel) GenerateObject(ctx context.Context, call fantasy.ObjectCall) (*fantasy.ObjectResponse, error) {
	return &fantasy.ObjectResponse{
		Object: map[string]any{
			"layout": "test layout",
		},
		RawText:      `{"layout": "test layout"}`,
		Usage:        fantasy.Usage{TotalTokens: 10},
		FinishReason: fantasy.FinishReasonStop,
	}, nil
}

func (m *mockModel) StreamObject(ctx context.Context, call fantasy.ObjectCall) (fantasy.ObjectStreamResponse, error) {
	return func(yield func(fantasy.ObjectStreamPart) bool) {
		_ = yield(fantasy.ObjectStreamPart{
			Type: fantasy.ObjectStreamPartTypeObject,
			Object: map[string]any{
				"layout": "test layout",
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
