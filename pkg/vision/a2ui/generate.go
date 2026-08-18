package a2ui

import (
	"context"
	"fmt"
	"maps"

	"charm.land/fantasy"
	"github.com/larsartmann/vision-review-agent/pkg/vision"
)

// GenerateOptions tunes one Generate call. The zero value is valid: it
// recreates the depicted UI as faithfully as possible on surface "main" with
// the basic catalog.
type GenerateOptions struct {
	// Task instructs the model what to build from the images. Empty means
	// faithful reconstruction (see BuildPrompt).
	Task string

	// SurfaceID overrides the surface identifier; defaults to "main".
	SurfaceID string

	// CatalogID overrides the component catalog; defaults to
	// DefaultCatalogID.
	CatalogID string

	// Theme carries theme parameters (e.g. {"primaryColor": "#FF0000"})
	// applied to the createSurface message when the model produced none.
	// A model-emitted theme wins.
	Theme map[string]any

	// DataModel seeds the initial data model. Keys the model also produced
	// win; seed keys only fill gaps.
	DataModel map[string]any
}

// GenerateResult is the outcome of a Generate call: the validated wire
// messages, the spec the model produced, and the model usage.
type GenerateResult struct {
	// Messages is the compiled, validated message sequence (createSurface,
	// updateComponents, and optionally updateDataModel).
	Messages []Message

	// Spec is the SurfaceSpec exactly as the model produced it (after
	// Compile's defaulting of empty surfaceId/catalogId fields).
	Spec SurfaceSpec

	// Usage reports token usage of the generation call.
	Usage fantasy.Usage

	// RawText is the raw model output for diagnostics.
	RawText string
}

// Generate turns images (screenshots, mockups, wireframes) into a complete,
// validated A2UI surface using the agent's vision model. It runs the model
// via vision.AnalyzeStructured[SurfaceSpec], applies GenerateOptions
// defaults to the returned spec, and compiles it with Compile, so the
// returned messages satisfy Validate.
//
// Model failures come back classified (errors.Is against vision.ModelError
// kinds); a structurally invalid model output fails with an error wrapping
// ErrValidation.
func Generate(
	ctx context.Context,
	agent *vision.Agent,
	opts GenerateOptions,
	images ...*vision.ImageSource,
) (*GenerateResult, error) {
	opts.applyDefaults()

	result, err := vision.AnalyzeStructured[SurfaceSpec](ctx, agent, BuildPrompt(opts.Task), images...)
	if err != nil {
		return nil, fmt.Errorf("a2ui generate: %w", err)
	}

	spec := result.Object
	if spec.SurfaceID == "" {
		spec.SurfaceID = opts.SurfaceID
	}

	if spec.CatalogID == "" {
		spec.CatalogID = opts.CatalogID
	}

	unwrapStarProperties(spec.Components)

	if spec.Theme == nil {
		spec.Theme = opts.Theme
	}

	if len(spec.DataModel) > 0 || len(opts.DataModel) > 0 {
		merged := make(map[string]any, len(opts.DataModel)+len(spec.DataModel))
		maps.Copy(merged, opts.DataModel)
		maps.Copy(merged, spec.DataModel)
		spec.DataModel = merged
	}

	messages, err := Compile(spec)
	if err != nil {
		return nil, fmt.Errorf("a2ui generate: %w", err)
	}

	return &GenerateResult{
		Messages: messages,
		Spec:     spec,
		Usage:    result.Usage,
		RawText:  result.RawText,
	}, nil
}

// applyDefaults fills the zero-value option fields.
func (o *GenerateOptions) applyDefaults() {
	if o.SurfaceID == "" {
		o.SurfaceID = defaultSurfaceID
	}

	if o.CatalogID == "" {
		o.CatalogID = DefaultCatalogID
	}
}

// unwrapStarProperties repairs a model quirk observed with caption-tuned
// fine-tunes (e.g. the nsfwcaption Qwen-VL family): properties arrive nested
// under a literal "*" key — {"properties": {"*": {"text": ...}}} — instead of
// at the top of the properties object. "*" is never a real catalog property,
// and the nesting survives schema-ignoring JSON decode, so Generate unwraps
// it before Compile; otherwise the catalog props vanish from the wire output.
func unwrapStarProperties(components []ComponentSpec) {
	for i := range components {
		props := components[i].Properties
		if len(props) != 1 {
			continue
		}

		nested, ok := props[propStar].(map[string]any)
		if !ok {
			continue
		}

		components[i].Properties = nested
	}
}
