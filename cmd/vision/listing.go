package main

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"charm.land/catwalk/pkg/catwalk"
	"github.com/larsartmann/vision-review-agent/internal/catalog"
)

const (
	tableWidth      = 82
	modelTableWidth = 85
	maxEditDistance = 3
)

// printProviders writes a table of all catalog providers to w.
func printProviders(w io.Writer, svc *catalog.Service) {
	providers := svc.Providers()

	sort.Slice(providers, func(left, right int) bool {
		return strings.ToLower(string(providers[left].ID)) < strings.ToLower(string(providers[right].ID))
	})

	fmt.Fprintf(w, "%-22s %-24s %-16s %6s %8s\n", "ID", "NAME", "TYPE", "MODELS", "VISION")
	fmt.Fprintln(w, strings.Repeat("-", tableWidth))

	for _, provider := range providers {
		visionCount := 0

		for _, model := range provider.Models {
			if model.SupportsImages {
				visionCount++
			}
		}

		fmt.Fprintf(w, "%-22s %-24s %-16s %6d %8d\n",
			provider.ID, provider.Name, provider.Type, len(provider.Models), visionCount)
	}

	fmt.Fprintf(w, "\n%d providers. Use -list-models to see vision-capable models.\n", len(providers))
}

// printVisionModels writes a table of vision-capable models to w. If providerFilter
// is non-empty, only models from the matching provider are shown.
func printVisionModels(w io.Writer, svc *catalog.Service, providerFilter string) {
	entries := svc.VisionModels()

	filtered := entries

	if providerFilter != "" {
		name := strings.ToLower(normalizeProviderName(providerFilter))
		filtered = filtered[:0]

		for _, entry := range entries {
			if strings.ToLower(string(entry.Provider.ID)) == name {
				filtered = append(filtered, entry)
			}
		}
	}

	sort.Slice(filtered, func(left, right int) bool {
		if filtered[left].Provider.ID != filtered[right].Provider.ID {
			return string(filtered[left].Provider.ID) < string(filtered[right].Provider.ID)
		}

		return filtered[left].Model.ID < filtered[right].Model.ID
	})

	fmt.Fprintf(w, "%-34s %-16s %10s %9s %10s\n", "MODEL ID", "PROVIDER", "CONTEXT", "$/1M IN", "$/1M OUT")
	fmt.Fprintln(w, strings.Repeat("-", modelTableWidth))

	for _, entry := range filtered {
		model := entry.Model
		fmt.Fprintf(w, "%-34s %-16s %10d %9.2f %10.2f\n",
			model.ID, entry.Provider.ID, model.ContextWindow, model.CostPer1MIn, model.CostPer1MOut)
	}

	fmt.Fprintf(w, "\n%d vision-capable models", len(filtered))

	if providerFilter != "" {
		fmt.Fprintf(w, " from provider %q", providerFilter)
	}

	fmt.Fprintln(w, ".")
}

// printProviderInfo writes detailed information about a single provider to w.
func printProviderInfo(w io.Writer, svc *catalog.Service, providerName string) {
	name := normalizeProviderName(providerName)

	provider, ok := svc.FindProvider(name)
	if !ok {
		fmt.Fprintf(w, "Provider %q not found. Use -list-providers to see options.\n", providerName)

		return
	}

	fmt.Fprintf(w, "Provider: %s (%s)\n", provider.Name, provider.ID)
	fmt.Fprintf(w, "Type: %s\n", provider.Type)

	if provider.APIKey != "" {
		fmt.Fprintf(w, "API Key: %s\n", provider.APIKey)
	}

	if provider.APIEndpoint != "" {
		fmt.Fprintf(w, "Endpoint: %s\n", provider.APIEndpoint)
	}

	if provider.DefaultLargeModelID != "" {
		fmt.Fprintf(w, "Default model: %s\n", provider.DefaultLargeModelID)
	}

	visionCount := 0

	for _, model := range provider.Models {
		if model.SupportsImages {
			visionCount++
		}
	}

	fmt.Fprintf(w, "\nModels (%d total, %d vision-capable):\n", len(provider.Models), visionCount)

	sortedModels := make([]catwalk.Model, len(provider.Models))
	copy(sortedModels, provider.Models)

	sort.Slice(sortedModels, func(left, right int) bool {
		return sortedModels[left].ID < sortedModels[right].ID
	})

	for _, model := range sortedModels {
		flags := buildModelFlags(model)
		fmt.Fprintf(w, "  %-34s %-28s %-14s $%.2f/$%.2f per 1M\n",
			model.ID, model.Name, flags, model.CostPer1MIn, model.CostPer1MOut)
	}
}

func buildModelFlags(model catwalk.Model) string {
	var parts []string
	if model.SupportsImages {
		parts = append(parts, "vision")
	}

	if model.CanReason {
		parts = append(parts, "reasoning")
	}

	if len(parts) == 0 {
		return "-"
	}

	return strings.Join(parts, ", ")
}

// suggestModel finds the closest catalog model ID to the input using edit
// distance. Returns "" if no model is within 3 edits of the input.
func suggestModel(svc *catalog.Service, input string) string {
	entries := svc.VisionModels()
	target := strings.ToLower(input)

	bestMatch := ""
	bestDist := maxEditDistance + 1

	for _, entry := range entries {
		dist := levenshtein(target, strings.ToLower(entry.Model.ID))
		if dist < bestDist {
			bestDist = dist
			bestMatch = entry.Model.ID
		}
	}

	return bestMatch
}

// levenshtein computes the edit distance between two strings using the
// standard dynamic programming algorithm.
func levenshtein(source, target string) int {
	sourceRunes := []rune(source)
	targetRunes := []rune(target)

	if len(sourceRunes) == 0 {
		return len(targetRunes)
	}

	if len(targetRunes) == 0 {
		return len(sourceRunes)
	}

	previous := make([]int, len(targetRunes)+1)
	current := make([]int, len(targetRunes)+1)

	for col := range previous {
		previous[col] = col
	}

	for row := 1; row <= len(sourceRunes); row++ {
		current[0] = row

		for col := 1; col <= len(targetRunes); col++ {
			cost := 1
			if sourceRunes[row-1] == targetRunes[col-1] {
				cost = 0
			}

			current[col] = minInt(current[col-1]+1, previous[col]+1, previous[col-1]+cost)
		}

		previous, current = current, previous
	}

	return previous[len(targetRunes)]
}

func minInt(first, second, third int) int {
	return min(first, min(second, third))
}
