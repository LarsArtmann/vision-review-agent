package vision

import (
	"context"
	"errors"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"charm.land/fantasy"
	apperrors "github.com/larsartmann/vision-review-agent/pkg/errors"
	"github.com/stretchr/testify/require"
)

func TestWithRetrySucceedsFirstAttempt(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	cfg := RetryConfig{MaxAttempts: 3, InitialBackoff: time.Millisecond}

	result, err := WithRetry(context.Background(), cfg, func(_ context.Context) (*string, error) {
		calls.Add(1)
		s := "ok"
		return &s, nil
	})

	require.NoError(t, err)
	require.Equal(t, "ok", *result)
	require.Equal(t, int32(1), calls.Load(), "must not retry on success")
}

func TestWithRetryRetriesUntilSuccess(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	cfg := RetryConfig{MaxAttempts: 3, InitialBackoff: time.Millisecond, Jitter: false}

	result, err := WithRetry(context.Background(), cfg, func(_ context.Context) (*int, error) {
		if calls.Add(1) < 3 {
			return nil, &ModelError{Kind: KindServerError, Cause: errors.New("boom")}
		}
		v := 42
		return &v, nil
	})

	require.NoError(t, err)
	require.Equal(t, 42, *result)
	require.Equal(t, int32(3), calls.Load())
}

func TestWithRetryDoesNotRetryNonRetryable(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	cfg := RetryConfig{MaxAttempts: 5, InitialBackoff: time.Millisecond}

	_, err := WithRetry(context.Background(), cfg, func(_ context.Context) (*int, error) {
		calls.Add(1)
		return nil, &ModelError{Kind: KindBadRequest, Cause: errors.New("bad input")}
	})

	require.Error(t, err)
	require.Equal(t, int32(1), calls.Load(), "must not retry non-retryable errors")
}

func TestWithRetryExhaustsAttempts(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	cfg := RetryConfig{MaxAttempts: 3, InitialBackoff: time.Millisecond, Jitter: false}

	_, err := WithRetry(context.Background(), cfg, func(_ context.Context) (*int, error) {
		calls.Add(1)
		return nil, &ModelError{Kind: KindNetwork, Cause: errors.New("dropped")}
	})

	require.Error(t, err)
	require.Equal(t, int32(3), calls.Load(), "must attempt exactly MaxAttempts times")
}

func TestWithRetryRespectsContextCancellation(t *testing.T) {
	t.Parallel()

	cfg := RetryConfig{MaxAttempts: 10, InitialBackoff: 5 * time.Second, Jitter: false}
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	var calls atomic.Int32
	start := time.Now()
	_, err := WithRetry(ctx, cfg, func(_ context.Context) (*int, error) {
		calls.Add(1)
		return nil, &ModelError{Kind: KindNetwork, Cause: errors.New("dropped")}
	})

	elapsed := time.Since(start)
	require.Error(t, err)
	require.ErrorIs(t, err, context.DeadlineExceeded)
	require.Less(t, elapsed, time.Second, "must abort quickly on context cancellation")
}

func TestConfigRetryRetriesTransientAnalyze(t *testing.T) {
	t.Parallel()

	// Model fails twice with retryable server errors, then succeeds on the 3rd.
	model := &mockModel{
		generateErrs: []error{
			newTestProviderErr(http.StatusInternalServerError),
			newTestProviderErr(http.StatusBadGateway),
		},
	}
	// MaxRetries is left at zero (default) on purpose: these tests exercise
	// vision-layer retry, so fantasy's HTTP-layer retry is disabled to keep
	// call counts deterministic and avoid its ~5s backoff per retryable call.
	rc := RetryConfig{MaxAttempts: 3, InitialBackoff: time.Millisecond, Jitter: false}
	agent, err := NewAgent(Config{Model: model, Retry: &rc})
	require.NoError(t, err)

	result, err := agent.Analyze(context.Background(), "prompt", ImageSrc())
	require.NoError(t, err)
	require.Contains(t, result.Text, "mock response")
	require.Equal(
		t,
		int32(3),
		model.generateCalls.Load(),
		"2 transient errors + 1 success = exactly 3 vision-layer calls",
	)
}

func TestConfigRetryExhaustsAndClassifies(t *testing.T) {
	t.Parallel()

	model := &mockModel{generateErr: newTestProviderErr(http.StatusTooManyRequests)}
	rc := RetryConfig{MaxAttempts: 2, InitialBackoff: time.Millisecond, Jitter: false}
	agent, err := NewAgent(Config{Model: model, Retry: &rc})
	require.NoError(t, err)

	_, err = agent.Analyze(context.Background(), "prompt", ImageSrc())
	require.Error(t, err)

	me, ok := errors.AsType[*apperrors.ModelError](err)
	require.True(t, ok, "exhausted retry must still be a classified ModelError")
	require.Equal(t, apperrors.KindRateLimited, me.Kind)
	require.Equal(
		t,
		int32(2),
		model.generateCalls.Load(),
		"MaxAttempts=2 with fantasy retry disabled = exactly 2 vision-layer calls",
	)
}

func TestConfigRetryWiredIntoStructured(t *testing.T) {
	t.Parallel()

	model := &mockModel{generateObjectErr: newTestProviderErr(http.StatusServiceUnavailable)}
	rc := RetryConfig{MaxAttempts: 2, InitialBackoff: time.Millisecond, Jitter: false}
	agent, err := NewAgent(Config{Model: model, Retry: &rc})
	require.NoError(t, err)

	_, err = AnalyzeStructured[testReview](context.Background(), agent, "prompt", ImageSrc())
	require.Error(t, err)
	require.Equal(
		t,
		int32(2),
		model.generateObjectCalls.Load(),
		"MaxAttempts=2 with fantasy retry disabled = exactly 2 vision-layer object calls",
	)
}

func TestConfigRetryOffByDefault(t *testing.T) {
	t.Parallel()

	model := &mockModel{generateErr: newTestProviderErr(http.StatusInternalServerError)}
	agent, err := NewAgent(Config{Model: model}) // no Retry, no MaxRetries
	require.NoError(t, err)
	require.Nil(t, agent.config.Retry, "Retry must be nil when not configured")

	_, err = agent.Analyze(context.Background(), "prompt", ImageSrc())
	require.Error(t, err)
}

func TestRetryConfigDelayForRespectsCap(t *testing.T) {
	t.Parallel()

	cfg := RetryConfig{
		MaxAttempts:    3,
		InitialBackoff: 10 * time.Millisecond,
		MaxBackoff:     50 * time.Millisecond,
		Multiplier:     10,
		Jitter:         false,
	}

	// 10ms * 10^2 = 1000ms, but capped at 50ms.
	require.Equal(t, 50*time.Millisecond, cfg.delayFor(2))
	require.Equal(t, 10*time.Millisecond, cfg.delayFor(0))
}

func TestCostTrackerAccumulates(t *testing.T) {
	t.Parallel()

	tracker := NewCostTracker()

	tracker.Add(toolsUsage(10, 20))
	tracker.Add(toolsUsage(5, 7))

	total := tracker.Total()
	require.Equal(t, int64(15), total.InputTokens)
	require.Equal(t, int64(27), total.OutputTokens)
	require.Equal(t, 2, tracker.Calls())
}

func TestCostTrackerAddResultHandlesNil(t *testing.T) {
	t.Parallel()

	tracker := NewCostTracker()
	tracker.AddResult(nil)
	tracker.AddResult(&AnalyzeResult{Usage: toolsUsage(1, 2)})

	require.Equal(t, 1, tracker.Calls())
	require.Equal(t, int64(1), tracker.Total().InputTokens)
}

func TestCostTrackerReset(t *testing.T) {
	t.Parallel()

	tracker := NewCostTracker()
	tracker.Add(toolsUsage(10, 20))
	tracker.Reset()

	require.Equal(t, 0, tracker.Calls())
	require.Equal(t, int64(0), tracker.Total().TotalTokens)
}

func TestCostTrackerIntegratesWithHooks(t *testing.T) {
	t.Parallel()

	tracker := NewCostTracker()
	agent, err := NewAgent(Config{
		Model: testModel(),
		Hooks: Hooks{OnFinish: func(_ context.Context, r *AnalyzeResult) { tracker.AddResult(r) }},
	})
	require.NoError(t, err)

	_, err = agent.Analyze(context.Background(), "prompt", ImageSrc())
	require.NoError(t, err)

	require.Equal(t, 1, tracker.Calls())
	require.Greater(t, tracker.Total().TotalTokens, int64(0))
}

func TestNewAgentWithCostTrackerAutoWires(t *testing.T) {
	t.Parallel()

	agent, tracker, err := NewAgentWithCostTracker(Config{Model: testModel()})
	require.NoError(t, err)

	_, err = agent.Analyze(context.Background(), "prompt", ImageSrc())
	require.NoError(t, err)

	require.Equal(t, 1, tracker.Calls(), "tracker must record the call automatically")
	require.Greater(t, tracker.Total().TotalTokens, int64(0))
}

func TestNewAgentWithCostTrackerPreservesUserHooks(t *testing.T) {
	t.Parallel()

	var userHookCalled atomic.Int32
	agent, tracker, err := NewAgentWithCostTracker(Config{
		Model: testModel(),
		Hooks: Hooks{OnFinish: func(_ context.Context, _ *AnalyzeResult) { userHookCalled.Add(1) }},
	})
	require.NoError(t, err)

	_, err = agent.Analyze(context.Background(), "prompt", ImageSrc())
	require.NoError(t, err)

	require.Equal(t, int32(1), userHookCalled.Load(), "user OnFinish must still fire")
	require.Equal(t, 1, tracker.Calls(), "cost tracker must also fire")
}

func toolsUsage(in, out int64) fantasy.Usage {
	return fantasy.Usage{InputTokens: in, OutputTokens: out, TotalTokens: in + out}
}

// TestNewAgentWithCostTrackerStructuredNilRawResponse verifies the documented
// contract for structured methods: OnFinish receives a synthesized
// *AnalyzeResult whose RawResponse is nil (and Text holds raw JSON). The
// CostTracker must still capture token usage from that synthesized result.
func TestNewAgentWithCostTrackerStructuredNilRawResponse(t *testing.T) {
	t.Parallel()

	var seen *AnalyzeResult
	agent, tracker, err := NewAgentWithCostTracker(Config{
		Model: testModel(),
		Hooks: Hooks{OnFinish: func(_ context.Context, r *AnalyzeResult) { seen = r }},
	})
	require.NoError(t, err)

	_, err = AnalyzeStructured[testReview](context.Background(), agent, "prompt", ImageSrc())
	require.NoError(t, err)

	require.NotNil(t, seen, "OnFinish must fire for structured analysis")
	require.Nil(t, seen.RawResponse, "structured methods must synthesize a nil RawResponse")
	require.Equal(t, 1, tracker.Calls(), "cost tracker must still record usage")
	require.Greater(t, tracker.Total().TotalTokens, int64(0))
}
