package vision

import id "github.com/larsartmann/go-branded-id"

// art-dupl:accept each package owns its brand; a shared ModelID would couple pkg, internal/catalog, and the daemon

// ModelBrand marks branded model identifiers.
type ModelBrand struct{}

// Name implements [id.BrandNamer], so a ModelID renders as "Model:<id>" in
// debug and log output.
func (ModelBrand) Name() string { return "Model" }

// ModelID identifies a vision-language model (e.g., "gpt-4o"). The brand
// makes plain strings unusable where a model identifier is required.
type ModelID = id.ID[ModelBrand, string]

// NewModelID brands s as a ModelID.
func NewModelID(s string) ModelID { return id.NewID[ModelBrand](s) }
