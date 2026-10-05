package catalog

import (
	"charm.land/catwalk/pkg/catwalk"
	id "github.com/larsartmann/go-branded-id"
)

// ProviderID identifies a catwalk provider (an InferenceProvider such as
// "openai" or "gemini"). It is an alias of the catalog's own
// [catwalk.InferenceProvider] type: one concept, one type, so the Service
// surface stays catwalk-typed on both sides of every lookup.
type ProviderID = catwalk.InferenceProvider

// art-dupl:accept each package owns its brand; a shared ModelID would couple pkg, internal/catalog, and the daemon

// ModelBrand marks branded model identifiers.
type ModelBrand struct{}

// Name implements [id.BrandNamer], so a ModelID renders as "Model:<id>" in
// debug and log output.
func (ModelBrand) Name() string { return "Model" }

// ModelID identifies a model in the catwalk catalog. The underlying catalog
// stores model IDs as plain strings; the brand makes provider names, view
// keys, or arbitrary strings unusable where a model ID is required.
type ModelID = id.ID[ModelBrand, string]

// NewModelID brands s as a ModelID.
func NewModelID(s string) ModelID { return id.NewID[ModelBrand](s) }
