package vision

import (
	"charm.land/catwalk/pkg/catwalk"
)

// ModelInfo holds metadata about a model from the catalog. When attached to a
// Config, its values provide sensible defaults (e.g., DefaultMaxTokens) and
// enable cost tracking (CostPer1MIn/Out). Explicit Config fields always take
// priority over ModelInfo defaults.
//
// Construct from catalog data:
//
//	catwalkModel := provider.Models[0]
//	config := vision.Config{
//	    Model:     languageModel,
//	    ModelInfo: vision.NewModelInfo(catwalkModel),
//	}
type ModelInfo struct {
	// ID is the model identifier (e.g., "gpt-4o").
	ID string

	// Name is the human-readable model name (e.g., "GPT-4o").
	Name string

	// SupportsImages indicates whether the model can process image inputs.
	SupportsImages bool

	// ContextWindow is the maximum number of input tokens the model accepts.
	ContextWindow int64

	// DefaultMaxTokens is the model's recommended default max output tokens.
	// When Config.MaxOutputTokens is 0 and ModelInfo is set, this value is used.
	DefaultMaxTokens int64

	// CostPer1MIn is the cost in USD per 1 million input tokens.
	CostPer1MIn float64

	// CostPer1MOut is the cost in USD per 1 million output tokens.
	CostPer1MOut float64

	// CanReason indicates whether the model supports reasoning/thinking.
	CanReason bool
}

// NewModelInfo creates a ModelInfo from a catwalk.Model, mapping all catalog
// metadata fields. This is the standard constructor when working with the
// catwalk model catalog.
func NewModelInfo(m catwalk.Model) ModelInfo {
	return ModelInfo{
		ID:               m.ID,
		Name:             m.Name,
		SupportsImages:   m.SupportsImages,
		ContextWindow:    m.ContextWindow,
		DefaultMaxTokens: m.DefaultMaxTokens,
		CostPer1MIn:      m.CostPer1MIn,
		CostPer1MOut:     m.CostPer1MOut,
		CanReason:        m.CanReason,
	}
}

// applyModelInfoDefaults fills in zero-valued Config fields from ModelInfo.
// It only sets defaults for fields the caller left unset (zero value);
// explicit Config values always take priority. This is called by NewAgent
// after validation, before building fantasy options.
func (c *Config) applyModelInfoDefaults() {
	if c.ModelInfo == nil {
		return
	}

	if c.MaxOutputTokens == 0 && c.ModelInfo.DefaultMaxTokens > 0 {
		c.MaxOutputTokens = c.ModelInfo.DefaultMaxTokens
	}
}
