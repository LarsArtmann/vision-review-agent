package vision

import (
	"testing"

	"charm.land/catwalk/pkg/catwalk"
	"github.com/stretchr/testify/require"
)

func TestNewModelInfoMapsAllFields(t *testing.T) {
	t.Parallel()

	source := catwalk.Model{
		ID:               "gpt-4o",
		Name:             "GPT-4o",
		CostPer1MIn:      2.50,
		CostPer1MOut:     10.00,
		ContextWindow:    128000,
		DefaultMaxTokens: 16384,
		CanReason:        false,
		SupportsImages:   true,
	}

	info := NewModelInfo(source)

	require.Equal(t, "gpt-4o", info.ID)
	require.Equal(t, "GPT-4o", info.Name)
	require.InEpsilon(t, 2.50, info.CostPer1MIn, 1e-9)
	require.InEpsilon(t, 10.00, info.CostPer1MOut, 1e-9)
	require.Equal(t, int64(128000), info.ContextWindow)
	require.Equal(t, int64(16384), info.DefaultMaxTokens)
	require.False(t, info.CanReason)
	require.True(t, info.SupportsImages)
}

func TestNewModelInfoWithReasoning(t *testing.T) {
	t.Parallel()

	source := catwalk.Model{
		ID:        "o3",
		Name:      "o3",
		CanReason: true,
	}

	info := NewModelInfo(source)
	require.True(t, info.CanReason)
}

func TestApplyModelInfoDefaultsSetsMaxTokens(t *testing.T) {
	t.Parallel()

	cfg := Config{
		ModelInfo: &ModelInfo{
			DefaultMaxTokens: 4096,
		},
	}

	cfg.applyModelInfoDefaults()
	require.Equal(t, int64(4096), cfg.MaxOutputTokens,
		"MaxOutputTokens should be set from ModelInfo.DefaultMaxTokens when zero")
}

func TestApplyModelInfoDefaultsRespectsExplicitMaxTokens(t *testing.T) {
	t.Parallel()

	cfg := Config{
		MaxOutputTokens: 8192,
		ModelInfo: &ModelInfo{
			DefaultMaxTokens: 4096,
		},
	}

	cfg.applyModelInfoDefaults()
	require.Equal(t, int64(8192), cfg.MaxOutputTokens,
		"explicit MaxOutputTokens must not be overridden by ModelInfo")
}

func TestApplyModelInfoDefaultsNilModelInfo(t *testing.T) {
	t.Parallel()

	cfg := Config{MaxOutputTokens: 0}
	cfg.applyModelInfoDefaults()
	require.Equal(t, int64(0), cfg.MaxOutputTokens,
		"nil ModelInfo must not change anything")
}

func TestApplyModelInfoDefaultsZeroDefaultMaxTokens(t *testing.T) {
	t.Parallel()

	cfg := Config{
		ModelInfo: &ModelInfo{
			DefaultMaxTokens: 0,
		},
	}

	cfg.applyModelInfoDefaults()
	require.Equal(t, int64(0), cfg.MaxOutputTokens,
		"zero DefaultMaxTokens must not set MaxOutputTokens")
}

func TestNewAgentAppliesModelInfoDefaults(t *testing.T) {
	t.Parallel()

	agent, err := NewAgent(Config{
		Model: &mockModel{},
		ModelInfo: &ModelInfo{
			DefaultMaxTokens: 4096,
		},
	})
	require.NoError(t, err)
	require.NotNil(t, agent)

	require.Equal(t, int64(4096), agent.config.MaxOutputTokens,
		"NewAgent must apply ModelInfo defaults")
}
