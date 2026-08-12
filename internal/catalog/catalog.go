// Package catalog provides a model discovery layer over charm.land/catwalk.
// It abstracts provider/model lookup, vision-capable model filtering, and
// catwalk-to-fantasy provider construction behind a single Service type.
//
// The catalog is embedded-first: by default it uses data compiled into the
// binary via [embedded.GetAll], requiring no network access. When the
// CATWALK_URL environment variable is set, a remote sync with ETag caching
// keeps the catalog current (see sync.go).
package catalog

import (
	"strings"

	"charm.land/catwalk/pkg/catwalk"
	"charm.land/catwalk/pkg/embedded"
)

// Service provides access to the AI model catalog. It is the single entry
// point for provider and model discovery, abstracting whether the catalog data
// comes from the embedded offline catalog or a remote sync.
type Service struct {
	providers []catwalk.Provider
}

// New returns a Service backed by the embedded catalog. This is the default
// constructor for offline use.
func New() *Service {
	return &Service{providers: embedded.GetAll()}
}

// NewWithProviders returns a Service backed by the given provider list.
// This is primarily for testing.
func NewWithProviders(providers []catwalk.Provider) *Service {
	return &Service{providers: providers}
}

// Providers returns all known providers from the catalog.
func (s *Service) Providers() []catwalk.Provider {
	return s.providers
}

// FindProvider looks up a provider by its InferenceProvider ID (e.g., "openai",
// "anthropic", "gemini"). The lookup is case-insensitive. It returns the
// provider and true if found.
func (s *Service) FindProvider(id string) (catwalk.Provider, bool) {
	target := strings.ToLower(id)

	for _, p := range s.providers {
		if strings.ToLower(string(p.ID)) == target {
			return p, true
		}
	}

	return catwalk.Provider{}, false
}

// ModelEntry pairs a model with the provider that hosts it.
type ModelEntry struct {
	Provider catwalk.Provider
	Model    catwalk.Model
}

// FindModel searches across all providers for a model with the given ID.
// The lookup is case-insensitive. It returns the provider, model, and true
// if found.
func (s *Service) FindModel(modelID string) (catwalk.Provider, catwalk.Model, bool) {
	target := strings.ToLower(modelID)

	for _, p := range s.providers {
		for _, m := range p.Models {
			if strings.ToLower(m.ID) == target {
				return p, m, true
			}
		}
	}

	return catwalk.Provider{}, catwalk.Model{}, false
}

// FindModelInProvider searches for a model within a specific provider.
// The lookup is case-insensitive for both provider ID and model ID.
// This is preferable to FindModel when the user has selected a specific
// provider, because model pricing and capabilities may differ across providers.
func (s *Service) FindModelInProvider(providerID, modelID string) (*catwalk.Model, bool) {
	provider, ok := s.FindProvider(providerID)
	if !ok {
		return nil, false
	}

	target := strings.ToLower(modelID)

	for idx := range provider.Models {
		if strings.ToLower(provider.Models[idx].ID) == target {
			return &provider.Models[idx], true
		}
	}

	return nil, false
}

// VisionModels returns all models that support image inputs
// (SupportsImages == true) across all providers.
func (s *Service) VisionModels() []ModelEntry {
	entries := make([]ModelEntry, 0)

	for _, p := range s.providers {
		for _, m := range p.Models {
			if m.SupportsImages {
				entries = append(entries, ModelEntry{Provider: p, Model: m})
			}
		}
	}

	return entries
}
