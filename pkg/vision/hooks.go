package vision

import "context"

// Hooks defines optional callbacks for observing the agent's analysis lifecycle.
// All callbacks are optional; nil fields are safe and skipped.
//
// Hooks fire synchronously in the calling goroutine. Keep callbacks fast and non-blocking.
//
// Usage:
//
//	agent, _ := vision.NewAgent(vision.Config{
//	    Model: model,
//	    Hooks: vision.Hooks{
//	        OnStart:  func(ctx, prompt, n) { log.Printf("starting analysis of %d images", n) },
//	        OnFinish: func(ctx, result) { log.Printf("done: %d tokens", result.Usage.TotalTokens) },
//	        OnError:  func(ctx, err) { log.Printf("failed: %v", err) },
//	    },
//	})
type Hooks struct {
	// OnStart is called before any analysis begins, after input validation passes.
	OnStart func(ctx context.Context, prompt string, imageCount int)

	// OnFinish is called after a successful analysis, before the result is returned.
	OnFinish func(ctx context.Context, result *AnalyzeResult)

	// OnError is called when an analysis fails with a non-validation error
	// (i.e., an error from the model call itself, not from empty prompt/images).
	OnError func(ctx context.Context, err error)
}

// fireStart invokes OnStart if set.
func (h Hooks) fireStart(ctx context.Context, prompt string, imageCount int) {
	if h.OnStart != nil {
		h.OnStart(ctx, prompt, imageCount)
	}
}

// fireFinish invokes OnFinish if set.
func (h Hooks) fireFinish(ctx context.Context, result *AnalyzeResult) {
	if h.OnFinish != nil {
		h.OnFinish(ctx, result)
	}
}

// fireError invokes OnError if set.
func (h Hooks) fireError(ctx context.Context, err error) {
	if h.OnError != nil {
		h.OnError(ctx, err)
	}
}
