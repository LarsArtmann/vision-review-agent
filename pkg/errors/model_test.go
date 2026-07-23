package apperrors

import (
	"context"
	"errors"
	"io"
	"net/http"
	"testing"

	"charm.land/fantasy"
	"github.com/stretchr/testify/require"
)

// Package-level test sentinels to avoid err113 in table-driven tests.
var (
	errTestUpstream   = errors.New("upstream")
	errTestGeneric    = errors.New("something broke")
	errTestTooMany    = errors.New("too many requests")
	errTestParseFail  = errors.New("parse failure")
	errTestSimple     = errors.New("err")
	errTestBoom       = errors.New("boom")
	errTestNoop       = errors.New("noop")
)

// newProviderErr builds a fantasy.ProviderError with the given status code
// for testing classification.
func newProviderErr(statusCode int) *fantasy.ProviderError {
	return &fantasy.ProviderError{
		Title:      fantasy.ErrorTitleForStatusCode(statusCode),
		Message:    "test provider error",
		Cause:      errTestUpstream,
		StatusCode: statusCode,
	}
}

// newContextTooLargeErr builds a ProviderError flagged as context-too-large.
func newContextTooLargeErr() *fantasy.ProviderError {
	return &fantasy.ProviderError{
		Title:              "context too large",
		Message:            "prompt is too long: 9000 tokens > 8000 maximum",
		StatusCode:         400,
		ContextTooLargeErr: true,
		ContextUsedTokens:  9000,
		ContextMaxTokens:   8000,
	}
}

func TestClassify(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		err       error
		wantKind  ErrorKind
		wantRetry bool
	}{
		{
			name:      "nil returns nil",
			err:       nil,
			wantKind:  "",
			wantRetry: false,
		},
		{
			name:      "context cancelled",
			err:       context.Canceled,
			wantKind:  KindCancelled,
			wantRetry: false,
		},
		{
			name:      "context deadline exceeded",
			err:       context.DeadlineExceeded,
			wantKind:  KindTimeout,
			wantRetry: true,
		},
		{
			name:      "wrapped context cancelled",
			err:       wrapNoop(context.Canceled),
			wantKind:  KindCancelled,
			wantRetry: false,
		},
		{
			name:      "provider 401 unauthorized",
			err:       newProviderErr(http.StatusUnauthorized),
			wantKind:  KindAuthentication,
			wantRetry: false,
		},
		{
			name:      "provider 403 forbidden",
			err:       newProviderErr(http.StatusForbidden),
			wantKind:  KindAuthentication,
			wantRetry: false,
		},
		{
			name:      "provider 404 not found",
			err:       newProviderErr(http.StatusNotFound),
			wantKind:  KindNotFound,
			wantRetry: false,
		},
		{
			name:      "provider 400 bad request",
			err:       newProviderErr(http.StatusBadRequest),
			wantKind:  KindBadRequest,
			wantRetry: false,
		},
		{
			name:      "provider 429 too many requests",
			err:       newProviderErr(http.StatusTooManyRequests),
			wantKind:  KindRateLimited,
			wantRetry: true,
		},
		{
			name:      "provider 408 request timeout",
			err:       newProviderErr(http.StatusRequestTimeout),
			wantKind:  KindTimeout,
			wantRetry: true,
		},
		{
			name:      "provider 500 internal server error",
			err:       newProviderErr(http.StatusInternalServerError),
			wantKind:  KindServerError,
			wantRetry: true,
		},
		{
			name:      "provider 502 bad gateway",
			err:       newProviderErr(http.StatusBadGateway),
			wantKind:  KindServerError,
			wantRetry: true,
		},
		{
			name:      "provider 503 service unavailable",
			err:       newProviderErr(http.StatusServiceUnavailable),
			wantKind:  KindServerError,
			wantRetry: true,
		},
		{
			name:      "provider context too large",
			err:       newContextTooLargeErr(),
			wantKind:  KindContextTooLarge,
			wantRetry: false,
		},
		{
			name: "transport error with no status code is network",
			err: &fantasy.ProviderError{
				Title:   "stream transport error",
				Message: io.ErrUnexpectedEOF.Error(),
				Cause:   io.ErrUnexpectedEOF,
			},
			wantKind:  KindNetwork,
			wantRetry: true,
		},
		{
			name: "no object generated error is structured parse",
			err: &fantasy.NoObjectGeneratedError{
				RawText: "garbage",
			},
			wantKind:  KindStructuredParse,
			wantRetry: false,
		},
		{
			name:      "generic unknown error",
			err:       errTestGeneric,
			wantKind:  KindUnknown,
			wantRetry: false,
		},
		{
			name:      "retry error wrapping provider 429",
			err:       &fantasy.RetryError{Errors: []error{newProviderErr(http.StatusTooManyRequests)}},
			wantKind:  KindRateLimited,
			wantRetry: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := Classify(tt.err)
			if tt.err == nil {
				require.Nil(t, result)

				return
			}

			require.NotNil(t, result)
			require.Equal(t, tt.wantKind, result.Kind, "kind mismatch")
			require.Equal(t, tt.wantRetry, result.IsRetryable(), "retryability mismatch")
			require.Equal(t, tt.err, result.Cause, "cause must be preserved")
		})
	}
}

func TestClassifyPreservesCauseChain(t *testing.T) {
	t.Parallel()

	providerErr := newProviderErr(http.StatusTooManyRequests)
	classified := Classify(providerErr)

	require.NotNil(t, classified)
	require.Equal(t, KindRateLimited, classified.Kind)

	// errors.AsType must still find the original ProviderError through the
	// ModelError wrapper.
	extracted, ok := errors.AsType[*fantasy.ProviderError](classified)
	require.True(t, ok, "ProviderError must be extractable through ModelError")
	require.Equal(t, http.StatusTooManyRequests, extracted.StatusCode)
}

func TestClassifyContextSentinelsStillMatch(t *testing.T) {
	t.Parallel()

	cancelled := Classify(context.Canceled)
	require.Equal(t, KindCancelled, cancelled.Kind)

	// errors.Is must still match the stdlib sentinel through the wrapper.
	require.ErrorIs(t, cancelled, context.Canceled) //nolint:legacyerrors // stdlib sentinel match

	deadline := Classify(context.DeadlineExceeded)
	require.Equal(t, KindTimeout, deadline.Kind)

	require.ErrorIs(t, deadline, context.DeadlineExceeded) //nolint:legacyerrors // stdlib sentinel match
}

func TestModelErrorError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		modelErr *ModelError
		wantSub  string
	}{
		{
			name: "with op and cause",
			modelErr: &ModelError{
				Kind:   KindRateLimited,
				Op:     "analyze",
				Prompt: "Find bugs",
				Cause:  errTestTooMany,
			},
			wantSub: `analyze failed [rate_limited] (prompt="Find bugs"): too many requests`,
		},
		{
			name: "without op",
			modelErr: &ModelError{
				Kind:   KindTimeout,
				Prompt: "test",
				Cause:  context.DeadlineExceeded,
			},
			wantSub: `[timeout] (prompt="test")`,
		},
		{
			name: "truncates long prompt",
			modelErr: &ModelError{
				Kind:   KindUnknown,
				Op:     "stream",
				Prompt: string(make([]byte, 200)),
				Cause:  errTestSimple,
			},
			wantSub: "...",
		},
		{
			name: "nil cause renders empty",
			modelErr: &ModelError{
				Kind: KindUnknown,
				Op:   "analyze",
			},
			wantSub: `analyze failed [unknown] (prompt=""): `,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			require.Contains(t, tt.modelErr.Error(), tt.wantSub)
		})
	}
}

func TestIsRetryableFunction(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "classified rate limited", err: Classify(newProviderErr(http.StatusTooManyRequests)), want: true},
		{name: "classified authentication", err: Classify(newProviderErr(http.StatusUnauthorized)), want: false},
		{name: "raw provider error retryable", err: newProviderErr(http.StatusTooManyRequests), want: true},
		{name: "raw provider error not retryable", err: newProviderErr(http.StatusBadRequest), want: false},
		{name: "generic error", err: errTestBoom, want: false},
		{name: "wrapped model error", err: wrapNoop(Classify(newProviderErr(500))), want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			require.Equal(t, tt.want, IsRetryable(tt.err))
		})
	}
}

func TestWrap(t *testing.T) {
	t.Parallel()

	modelErr := Wrap(KindStructuredParse, "structured generate", "analyze this", errTestParseFail)

	require.Equal(t, KindStructuredParse, modelErr.Kind)
	require.Equal(t, "structured generate", modelErr.Op)
	require.Equal(t, "analyze this", modelErr.Prompt)
	require.Equal(t, errTestParseFail, modelErr.Cause)
	require.False(t, modelErr.IsRetryable())
}

// wrapNoop wraps an error in a trivial function for testing that
// classification traverses the error chain. The wrapper itself adds no
// additional error value so the original sentinel/type is still matched.
func wrapNoop(err error) error {
	return err
}
