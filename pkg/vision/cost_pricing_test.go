package vision

import (
	"testing"

	"charm.land/fantasy"
	"github.com/stretchr/testify/require"
)

func TestCostTrackerSetPricing(t *testing.T) {
	t.Parallel()

	tracker := NewCostTracker()
	tracker.SetPricing(2.50, 10.00)

	tracker.Add(fantasy.Usage{
		InputTokens:  1_000_000,
		OutputTokens: 500_000,
		TotalTokens:  1_500_000,
	})

	cost := tracker.CostUSD()
	// 1M input * $2.50 + 0.5M output * $10.00 = $2.50 + $5.00 = $7.50
	require.InEpsilon(t, 7.50, cost, 1e-9)
}

func TestCostTrackerCostUSDWithoutPricing(t *testing.T) {
	t.Parallel()

	tracker := NewCostTracker()

	tracker.Add(fantasy.Usage{
		InputTokens:  1_000_000,
		OutputTokens: 500_000,
		TotalTokens:  1_500_000,
	})

	require.InDelta(t, 0.0, tracker.CostUSD(), 1e-9, "CostUSD must return 0 when no pricing is set")
}

func TestCostTrackerCostUSDAccumulates(t *testing.T) {
	t.Parallel()

	tracker := NewCostTracker()
	tracker.SetPricing(1.00, 2.00)

	tracker.Add(fantasy.Usage{InputTokens: 500_000, OutputTokens: 250_000})
	tracker.Add(fantasy.Usage{InputTokens: 500_000, OutputTokens: 250_000})

	cost := tracker.CostUSD()
	// 1M input * $1.00 + 0.5M output * $2.00 = $1.00 + $1.00 = $2.00
	require.InEpsilon(t, 2.00, cost, 1e-9)
}

func TestCostTrackerResetClearsPricing(t *testing.T) {
	t.Parallel()

	tracker := NewCostTracker()
	tracker.SetPricing(5.0, 10.0)

	// Reset only clears usage and calls, not pricing
	// (pricing is a configuration, not accumulated state)
	tracker.Reset()

	tracker.Add(fantasy.Usage{InputTokens: 1_000_000, OutputTokens: 0})

	cost := tracker.CostUSD()
	// Pricing should still be active: 1M * $5.00 = $5.00
	require.InEpsilon(t, 5.00, cost, 1e-9)
}

func TestCostTrackerZeroTokens(t *testing.T) {
	t.Parallel()

	tracker := NewCostTracker()
	tracker.SetPricing(2.50, 10.00)

	cost := tracker.CostUSD()
	require.InDelta(t, 0.0, cost, 1e-9)
}

func TestNewAgentWithCostTrackerAutoWiresPricing(t *testing.T) {
	t.Parallel()

	info := &ModelInfo{
		CostPer1MIn:  2.50,
		CostPer1MOut: 10.00,
	}

	agent, tracker, err := NewAgentWithCostTracker(Config{
		Model:     &mockModel{},
		ModelInfo: info,
	})
	require.NoError(t, err)
	require.NotNil(t, agent)

	tracker.Add(fantasy.Usage{
		InputTokens:  1_000_000,
		OutputTokens: 500_000,
	})

	cost := tracker.CostUSD()
	require.InEpsilon(t, 7.50, cost, 1e-9, "pricing must be auto-wired from ModelInfo")
}

func TestNewAgentWithCostTrackerWithoutModelInfo(t *testing.T) {
	t.Parallel()

	agent, tracker, err := NewAgentWithCostTracker(Config{
		Model: &mockModel{},
	})
	require.NoError(t, err)
	require.NotNil(t, agent)

	tracker.Add(fantasy.Usage{
		InputTokens:  1_000_000,
		OutputTokens: 500_000,
	})

	cost := tracker.CostUSD()
	require.InDelta(t, 0.0, cost, 1e-9, "CostUSD must be 0 without ModelInfo pricing")
}
