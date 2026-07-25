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

// newTestProviderErr builds a fantasy.ProviderError for testing.
func newTestProviderErr(statusCode int) *fantasy.ProviderError {
	return &fantasy.ProviderError{
		Title:      fantasy.ErrorTitleForStatusCode(statusCode),
		Message:    "test provider error",
		StatusCode: statusCode,
	}
}

// newTestAgent creates an agent with the given mock model for testify-based tests.
// Unlike setupAgentWithModel, it does not require a gomega fail handler.
func newTestAgent(t *testing.T, model *mockModel) *Agent {
	t.Helper()
	agent, err := NewAgent(Config{Model: model})
	require.NoError(t, err)
	return agent
}

func TestAnalyzeClassifiesModelError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		modelErr  error
		wantKind  apperrors.ErrorKind
		wantRetry bool
	}{
		{
			name:      "rate limited",
			modelErr:  newTestProviderErr(http.StatusTooManyRequests),
			wantKind:  apperrors.KindRateLimited,
			wantRetry: true,
		},
		{
			name:      "authentication",
			modelErr:  newTestProviderErr(http.StatusUnauthorized),
			wantKind:  apperrors.KindAuthentication,
			wantRetry: false,
		},
		{
			name:      "not found",
			modelErr:  newTestProviderErr(http.StatusNotFound),
			wantKind:  apperrors.KindNotFound,
			wantRetry: false,
		},
		{
			name:      "server error",
			modelErr:  newTestProviderErr(http.StatusInternalServerError),
			wantKind:  apperrors.KindServerError,
			wantRetry: true,
		},
		{
			name:      "bad request",
			modelErr:  newTestProviderErr(http.StatusBadRequest),
			wantKind:  apperrors.KindBadRequest,
			wantRetry: false,
		},
		{name: "context cancelled", modelErr: context.Canceled, wantKind: apperrors.KindCancelled, wantRetry: false},
		{
			name:      "deadline exceeded",
			modelErr:  context.DeadlineExceeded,
			wantKind:  apperrors.KindTimeout,
			wantRetry: true,
		},
		{name: "generic error", modelErr: errors.New("boom"), wantKind: apperrors.KindUnknown, wantRetry: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			model := &mockModel{generateErr: tt.modelErr}
			agent := newTestAgent(t, model)

			_, err := agent.Analyze(context.Background(), "test prompt", ImageSrc())
			require.Error(t, err)

			me, ok := errors.AsType[*apperrors.ModelError](err)
			require.True(t, ok, "error must be extractable as *ModelError")
			require.Equal(t, tt.wantKind, me.Kind)
			require.Equal(t, tt.wantRetry, me.IsRetryable())
			require.Equal(t, "test prompt", me.Prompt)
			require.Contains(t, me.Op, "generate")
		})
	}
}

func TestAnalyzeStreamClassifiesModelError(t *testing.T) {
	t.Parallel()

	model := &mockModel{streamErr: newTestProviderErr(http.StatusTooManyRequests)}
	agent := newTestAgent(t, model)

	_, err := agent.AnalyzeStream(context.Background(), "test prompt", nil, ImageSrc())
	require.Error(t, err)

	me, ok := errors.AsType[*apperrors.ModelError](err)
	require.True(t, ok)
	require.Equal(t, apperrors.KindRateLimited, me.Kind)
	require.True(t, me.IsRetryable())
}

func TestAnalyzeStructuredClassifiesModelError(t *testing.T) {
	t.Parallel()

	model := &mockModel{generateObjectErr: newTestProviderErr(http.StatusUnauthorized)}
	agent := newTestAgent(t, model)

	_, err := AnalyzeStructured[testReview](context.Background(), agent, "test prompt", ImageSrc())
	require.Error(t, err)

	me, ok := errors.AsType[*apperrors.ModelError](err)
	require.True(t, ok)
	require.Equal(t, apperrors.KindAuthentication, me.Kind)
	require.False(t, me.IsRetryable())
}

func TestAnalyzeStructuredClassifiesParseError(t *testing.T) {
	t.Parallel()

	model := &mockModel{generateObjectErr: &fantasy.NoObjectGeneratedError{RawText: "garbage"}}
	agent := newTestAgent(t, model)

	_, err := AnalyzeStructured[testReview](context.Background(), agent, "test prompt", ImageSrc())
	require.Error(t, err)

	me, ok := errors.AsType[*apperrors.ModelError](err)
	require.True(t, ok)
	require.Equal(t, apperrors.KindStructuredParse, me.Kind)
}

func TestClassifiedErrorPreservesCauseChain(t *testing.T) {
	t.Parallel()

	providerErr := newTestProviderErr(http.StatusTooManyRequests)
	model := &mockModel{generateErr: providerErr}
	agent := newTestAgent(t, model)

	_, err := agent.Analyze(context.Background(), "test", ImageSrc())
	require.Error(t, err)

	// errors.AsType must find the original ProviderError through the ModelError.
	extracted, ok := errors.AsType[*fantasy.ProviderError](err)
	require.True(t, ok, "ProviderError must be extractable through ModelError")
	require.Equal(t, http.StatusTooManyRequests, extracted.StatusCode)
}

func TestVisionIsRetryable(t *testing.T) {
	t.Parallel()

	model := &mockModel{generateErr: newTestProviderErr(http.StatusTooManyRequests)}
	agent := newTestAgent(t, model)

	_, err := agent.Analyze(context.Background(), "test", ImageSrc())
	require.Error(t, err)
	require.True(t, IsRetryable(err))

	model2 := &mockModel{generateErr: newTestProviderErr(http.StatusUnauthorized)}
	agent2 := newTestAgent(t, model2)

	_, err2 := agent2.Analyze(context.Background(), "test", ImageSrc())
	require.Error(t, err2)
	require.False(t, IsRetryable(err2))
}
