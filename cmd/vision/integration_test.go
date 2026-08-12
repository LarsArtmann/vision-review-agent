package main

import (
	"testing"

	"charm.land/fantasy"
	"github.com/larsartmann/vision-review-agent/internal/catalog"
	"github.com/larsartmann/vision-review-agent/pkg/vision"
	"github.com/stretchr/testify/require"
)

func TestIntegrationCatalogToAgentFlow(t *testing.T) {
	t.Parallel()

	// 1. Create catalog
	svc := catalog.New()
	require.NotEmpty(t, svc.Providers(), "catalog must have providers")

	// 2. Find a known vision model
	provider, model, ok := svc.FindModel("gpt-4o")
	require.True(t, ok, "gpt-4o must be in catalog")
	require.True(t, model.SupportsImages, "gpt-4o must support images")
	require.Positive(t, model.CostPer1MIn, "gpt-4o must have pricing")

	// 3. Create ModelInfo from catalog data
	info := vision.NewModelInfo(model)
	require.Equal(t, "gpt-4o", info.ID)
	require.Equal(t, "openai", string(provider.ID))

	// Re-get with correct provider context
	catwalkModel, ok := svc.FindModelInProvider("openai", "gpt-4o")
	require.True(t, ok)

	info = vision.NewModelInfo(*catwalkModel)

	// 4. Create agent with ModelInfo
	modelInfo := &info
	agent, err := vision.NewAgent(vision.Config{
		Model:     &cliMockModel{},
		ModelInfo: modelInfo,
	})
	require.NoError(t, err)
	require.NotNil(t, agent)

	// 5. Verify pricing via CostTracker
	_, tracker, err := vision.NewAgentWithCostTracker(vision.Config{
		Model:     &cliMockModel{},
		ModelInfo: modelInfo,
	})
	require.NoError(t, err)

	tracker.Add(fantasy.Usage{
		InputTokens:  1_000_000,
		OutputTokens: 1_000_000,
	})

	cost := tracker.CostUSD()
	expectedCost := info.CostPer1MIn + info.CostPer1MOut
	require.InEpsilon(t, expectedCost, cost, 1e-9,
		"cost must match catalog pricing: $%.2f in + $%.2f out", info.CostPer1MIn, info.CostPer1MOut)
}

func TestIntegrationVisionModelDiscovery(t *testing.T) {
	t.Parallel()

	svc := catalog.New()
	entries := svc.VisionModels()

	require.Greater(t, len(entries), 50, "catalog must have many vision-capable models")

	// Verify all entries truly support images
	for _, e := range entries {
		require.True(t, e.Model.SupportsImages,
			"VisionModels must only return image-capable models")
		require.NotEmpty(t, e.Model.ID)
		require.NotEmpty(t, e.Provider.ID)
	}
}

func TestIntegrationProviderBridgeForKnownTypes(t *testing.T) {
	t.Parallel()

	svc := catalog.New()

	// Test each supported provider type
	tests := []struct {
		providerID string
	}{
		{providerID: "openai"},
		{providerID: "anthropic"},
		{providerID: "gemini"},
		{providerID: "openrouter"},
	}

	for _, tt := range tests {
		t.Run(tt.providerID, func(t *testing.T) {
			t.Parallel()

			provider, ok := svc.FindProvider(tt.providerID)
			require.True(t, ok, "%s must be in catalog", tt.providerID)

			apiKey := "sk-test-key"
			if !catalog.RequiresAPIKey(provider) {
				apiKey = ""
			}

			built, err := catalog.BuildProvider(provider, apiKey, catalog.ResolveBaseURL(provider))
			require.NoError(t, err)
			require.NotNil(t, built)
		})
	}
}

func TestIntegrationModelSuggestionForTypo(t *testing.T) {
	t.Parallel()

	svc := catalog.New()

	// Common typos should find the right model
	tests := []struct {
		typo string
		want string
	}{
		{typo: "gpt-4oo", want: "gpt-4o"},
		{typo: "gpt4o", want: "gpt-4o"},
		{typo: "gpt-4o-mini", want: "gpt-4o-mini"},
	}

	for _, tt := range tests {
		t.Run(tt.typo, func(t *testing.T) {
			t.Parallel()

			result := suggestModel(svc, tt.typo)
			require.Equal(t, tt.want, result, "suggestion for typo %q", tt.typo)
		})
	}
}
