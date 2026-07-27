package vision

import (
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
	mu    sync.Mutex
	total fantasy.Usage
	calls int
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
