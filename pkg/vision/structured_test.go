package vision

import (
	"context"
	"errors"
	"testing"

	"charm.land/fantasy"
	apperrors "github.com/larsartmann/vision-review-agent/pkg/errors"
	"github.com/stretchr/testify/require"
)

func TestAnalyzeStructured(t *testing.T) {
	t.Parallel()

	agent, err := NewAgent(Config{Model: testModel()})
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	img := ImageSrc()

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
			wantLayout: testLayout,
			wantTokens: 10,
			wantErr:    false,
		},
		{
			name:        testEmptyPrompt,
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
		// capture range variable
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result, err := AnalyzeStructured[testReview](ctx, agent, tt.prompt, tt.images...)
			if AssertErr(t, tt.wantErr, tt.wantErrType, err) {
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

// TestAnalyzeStructuredStreamUnmarshalFailure verifies that a malformed final
// object in the stream produces a KindStructuredParse error instead of being
// silently swallowed. This is the most behaviorally significant error-system
// change: previously the unmarshal error was discarded with `_ = ...`.
func TestAnalyzeStructuredStreamUnmarshalFailure(t *testing.T) {
	t.Parallel()

	model := &mockModel{
		streamObjectFunc: func(yield func(fantasy.ObjectStreamPart) bool) {
			_ = yield(fantasy.ObjectStreamPart{
				Type: fantasy.ObjectStreamPartTypeFinish,
				Object: map[string]any{
					"score": "not-a-number", // string into int — must fail
				},
				Usage:        fantasy.Usage{TotalTokens: 10},
				FinishReason: fantasy.FinishReasonStop,
			})
		},
	}
	agent := newTestAgent(t, model)

	const prompt = "test prompt"

	_, err := AnalyzeStructuredStream[testReview](
		context.Background(),
		agent,
		prompt,
		func(testReview) {}, // no-op callback
		ImageSrc(),
	)
	require.Error(t, err)

	me, ok := errors.AsType[*apperrors.ModelError](err)
	require.True(t, ok, "error must be extractable as *ModelError")
	require.Equal(t, apperrors.KindStructuredParse, me.Kind)
	require.False(t, me.IsRetryable(), "structured parse errors are not retryable")
	require.Equal(t, prompt, me.Prompt)
	require.Contains(t, me.Op, "stream")
}

// TestAnalyzeStructuredUnmarshalFailure verifies the non-streaming structured
// path also surfaces unmarshal failures as KindStructuredParse errors.
func TestAnalyzeStructuredUnmarshalFailure(t *testing.T) {
	t.Parallel()

	model := &mockModel{
		generateObjectResponse: &fantasy.ObjectResponse{
			Object: map[string]any{
				"score": "not-a-number", // string into int — must fail
			},
			RawText:      `{"score":"not-a-number"}`,
			Usage:        fantasy.Usage{TotalTokens: 10},
			FinishReason: fantasy.FinishReasonStop,
		},
	}
	agent := newTestAgent(t, model)

	const prompt = "test prompt"

	_, err := AnalyzeStructured[testReview](context.Background(), agent, prompt, ImageSrc())
	require.Error(t, err)

	me, ok := errors.AsType[*apperrors.ModelError](err)
	require.True(t, ok, "error must be extractable as *ModelError")
	require.Equal(t, apperrors.KindStructuredParse, me.Kind)
	require.False(t, me.IsRetryable())
	require.Equal(t, prompt, me.Prompt)
	require.Contains(t, me.Op, "unmarshal")
}
