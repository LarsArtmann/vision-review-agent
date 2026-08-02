package vision

import (
	"context"
	"errors"
	"net/http"
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

// TestAnalyzeStructuredStreamInitialError verifies that when StreamObject
// itself returns an error (before any streaming begins), the error is
// classified and surfaced to the caller. This exercises the
// mockModel.streamObjectErr field, distinct from stream-part errors.
func TestAnalyzeStructuredStreamInitialError(t *testing.T) {
	t.Parallel()

	model := &mockModel{
		streamObjectErr: newTestProviderErr(http.StatusServiceUnavailable),
	}
	agent := newTestAgent(t, model)

	const prompt = "test prompt"

	_, err := AnalyzeStructuredStream[testReview](
		context.Background(),
		agent,
		prompt,
		func(testReview) {},
		ImageSrc(),
	)
	require.Error(t, err)

	me, ok := errors.AsType[*apperrors.ModelError](err)
	require.True(t, ok, "error must be extractable as *ModelError")
	require.Equal(t, apperrors.KindServiceUnavailable, me.Kind)
	require.True(t, me.IsRetryable(), "service unavailable is retryable")
	require.Equal(t, prompt, me.Prompt)
	require.Contains(t, me.Op, "stream")
}

// mockObjectStream builds a fantasy.ObjectStreamResponse from a fixed list of
// parts, yielding them in order. Used by consumeObjectStream unit tests.
func mockObjectStream(parts ...fantasy.ObjectStreamPart) fantasy.ObjectStreamResponse {
	return func(yield func(fantasy.ObjectStreamPart) bool) {
		for _, part := range parts {
			if !yield(part) {
				return
			}
		}
	}
}

// TestConsumeObjectStream_PartialObjectInvokesCallback verifies that an
// ObjectStreamPartTypeObject part unmarshals into T, invokes onObject, and
// stores the result as the running finalObject.
func TestConsumeObjectStream_PartialObjectInvokesCallback(t *testing.T) {
	t.Parallel()

	var callbackCount int

	var lastPartial testReview

	stream := mockObjectStream(
		fantasy.ObjectStreamPart{
			Type: fantasy.ObjectStreamPartTypeObject,
			Object: map[string]any{
				"layout": "partial",
				"score":  42,
			},
		},
	)

	result, err := consumeObjectStream[testReview](
		context.Background(),
		Hooks{},
		stream,
		"test prompt",
		func(partial testReview) {
			callbackCount++
			lastPartial = partial
		},
	)

	require.NoError(t, err)
	require.Equal(t, 1, callbackCount, "callback must fire exactly once")
	require.Equal(t, "partial", lastPartial.Layout)
	require.Equal(t, 42, lastPartial.Score)
	require.Equal(t, "partial", result.finalObject.Layout)
}

// TestConsumeObjectStream_NilCallbackDoesNotPanic verifies that a nil onObject
// is safe — partial objects are silently skipped without panicking.
func TestConsumeObjectStream_NilCallbackDoesNotPanic(t *testing.T) {
	t.Parallel()

	stream := mockObjectStream(
		fantasy.ObjectStreamPart{
			Type: fantasy.ObjectStreamPartTypeObject,
			Object: map[string]any{
				"layout": "ignored",
			},
		},
	)

	result, err := consumeObjectStream[testReview](
		context.Background(),
		Hooks{},
		stream,
		"test prompt",
		nil,
	)

	require.NoError(t, err, "nil callback must not cause errors")
	require.Empty(t, result.finalObject.Layout, "object must not be stored when callback is nil")
}

// TestConsumeObjectStream_TextDeltaAccumulates verifies that TextDelta parts
// are concatenated into rawText in arrival order.
func TestConsumeObjectStream_TextDeltaAccumulates(t *testing.T) {
	t.Parallel()

	stream := mockObjectStream(
		fantasy.ObjectStreamPart{Type: fantasy.ObjectStreamPartTypeTextDelta, Delta: "hello "},
		fantasy.ObjectStreamPart{Type: fantasy.ObjectStreamPartTypeTextDelta, Delta: "world"},
	)

	result, err := consumeObjectStream[testReview](
		context.Background(),
		Hooks{},
		stream,
		"test prompt",
		nil,
	)

	require.NoError(t, err)
	require.Equal(t, "hello world", result.rawText)
}

// TestConsumeObjectStream_ErrorPartClassifies verifies that an Error part is
// classified via classifyModelErr and returned immediately, terminating the stream.
func TestConsumeObjectStream_ErrorPartClassifies(t *testing.T) {
	t.Parallel()

	stream := mockObjectStream(
		fantasy.ObjectStreamPart{
			Type:  fantasy.ObjectStreamPartTypeError,
			Error: newTestProviderErr(http.StatusServiceUnavailable),
		},
	)

	_, err := consumeObjectStream[testReview](
		context.Background(),
		Hooks{},
		stream,
		"test prompt",
		nil,
	)

	require.Error(t, err)

	me, ok := errors.AsType[*apperrors.ModelError](err)
	require.True(t, ok)
	require.Equal(t, apperrors.KindServiceUnavailable, me.Kind)
	require.True(t, me.IsRetryable())
}

// TestConsumeObjectStream_FinishValidObjectStoresMetadata verifies that a
// Finish part with a valid Object stores usage, finish reason, and the
// unmarshalled final object.
func TestConsumeObjectStream_FinishValidObjectStoresMetadata(t *testing.T) {
	t.Parallel()

	stream := mockObjectStream(
		fantasy.ObjectStreamPart{
			Type: fantasy.ObjectStreamPartTypeFinish,
			Object: map[string]any{
				"layout": "final",
				"score":  99,
			},
			Usage:        fantasy.Usage{TotalTokens: 42},
			FinishReason: fantasy.FinishReasonStop,
		},
	)

	result, err := consumeObjectStream[testReview](
		context.Background(),
		Hooks{},
		stream,
		"test prompt",
		nil,
	)

	require.NoError(t, err)
	require.Equal(t, "final", result.finalObject.Layout)
	require.Equal(t, 99, result.finalObject.Score)
	require.Equal(t, int64(42), result.usage.TotalTokens)
	require.Equal(t, fantasy.FinishReasonStop, result.finishReason)
}

// TestConsumeObjectStream_FinishMalformedObjectReturnsParseError verifies that
// a Finish part with a malformed Object returns KindStructuredParse directly
// from consumeObjectStream (not just transitively through AnalyzeStructuredStream).
func TestConsumeObjectStream_FinishMalformedObjectReturnsParseError(t *testing.T) {
	t.Parallel()

	stream := mockObjectStream(
		fantasy.ObjectStreamPart{
			Type: fantasy.ObjectStreamPartTypeFinish,
			Object: map[string]any{
				"score": "not-a-number",
			},
			Usage:        fantasy.Usage{TotalTokens: 10},
			FinishReason: fantasy.FinishReasonStop,
		},
	)

	_, err := consumeObjectStream[testReview](
		context.Background(),
		Hooks{},
		stream,
		"test prompt",
		nil,
	)

	require.Error(t, err)

	me, ok := errors.AsType[*apperrors.ModelError](err)
	require.True(t, ok)
	require.Equal(t, apperrors.KindStructuredParse, me.Kind)
	require.False(t, me.IsRetryable())
}
