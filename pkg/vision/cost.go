package vision

import (
	"context"
	"fmt"
	"sync"

	"charm.land/fantasy"
)

// CostTracker accumulates token usage across analysis calls. It is safe for
// concurrent use and integrates naturally with [Hooks.OnFinish]:
//
//	tracker := vision.NewCostTracker()
//	agent, _ := vision.NewAgent(vision.Config{
//	    Model: model,
//	    Hooks: vision.Hooks{
//	        OnFinish: func(_ context.Context, r *vision.AnalyzeResult) { tracker.Add(r.Usage) },
//	    },
//	})
//	// ... run analyses ...
//	fmt.Printf("total tokens: %d across %d calls\n", tracker.Total().TotalTokens, tracker.Calls())
type CostTracker struct {
	mu            sync.Mutex
	total         fantasy.Usage
	calls         int
	inPricePer1M  float64
	outPricePer1M float64
}

// NewCostTracker returns a ready-to-use, zero-value CostTracker.
func NewCostTracker() *CostTracker {
	return &CostTracker{}
}

// Add accumulates a single call's usage and increments the call counter.
func (c *CostTracker) Add(usage fantasy.Usage) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.total.InputTokens += usage.InputTokens
	c.total.OutputTokens += usage.OutputTokens
	c.total.TotalTokens += usage.TotalTokens
	c.calls++
}

// AddResult is a convenience for Hooks.OnFinish, accumulating a result's usage.
func (c *CostTracker) AddResult(result *AnalyzeResult) {
	if result == nil {
		return
	}

	c.Add(result.Usage)
}

// Total returns the cumulative token usage across all added calls.
func (c *CostTracker) Total() fantasy.Usage {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.total
}

// Calls returns the number of calls whose usage has been added.
func (c *CostTracker) Calls() int {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.calls
}

// Reset zeroes the accumulated usage and call count.
func (c *CostTracker) Reset() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.total = fantasy.Usage{}
	c.calls = 0
}

// SetPricing configures the per-1M-token costs used by CostUSD. Pass the
// provider's published rates (e.g., 2.50 for $2.50 per 1M input tokens).
// When pricing is not set (both zero), CostUSD returns 0.
func (c *CostTracker) SetPricing(inPer1M, outPer1M float64) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.inPricePer1M = inPer1M
	c.outPricePer1M = outPer1M
}

const tokensPerMillion = 1_000_000

// CostUSD returns the estimated dollar cost based on accumulated token usage
// and the pricing set via SetPricing. Returns 0 when no pricing is configured.
func (c *CostTracker) CostUSD() float64 {
	c.mu.Lock()
	defer c.mu.Unlock()

	inputCost := float64(c.total.InputTokens) / tokensPerMillion * c.inPricePer1M
	outputCost := float64(c.total.OutputTokens) / tokensPerMillion * c.outPricePer1M

	return inputCost + outputCost
}

// NewAgentWithCostTracker creates an Agent whose Hooks.OnFinish automatically
// feeds every analysis result's token usage into the returned CostTracker.
// It composes any caller-supplied Hooks: the cost-tracker callback runs first,
// then the caller's OnFinish (if any) runs, so both see the result.
//
// Usage:
//
//	agent, tracker, _ := vision.NewAgentWithCostTracker(vision.Config{Model: model})
//	agent.Analyze(ctx, "review this", img)
//	fmt.Printf("tokens: %d\n", tracker.Total().TotalTokens)
func NewAgentWithCostTracker(config Config) (*Agent, *CostTracker, error) {
	tracker := NewCostTracker()

	if config.ModelInfo != nil {
		tracker.SetPricing(config.ModelInfo.CostPer1MIn, config.ModelInfo.CostPer1MOut)
	}

	userOnFinish := config.Hooks.OnFinish

	config.Hooks.OnFinish = func(ctx context.Context, result *AnalyzeResult) {
		tracker.AddResult(result)

		if userOnFinish != nil {
			userOnFinish(ctx, result)
		}
	}

	agent, err := NewAgent(config)
	if err != nil {
		return nil, nil, fmt.Errorf("new agent with cost tracker: %w", err)
	}

	return agent, tracker, nil
}
