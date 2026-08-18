package a2ui

import (
	"fmt"
)

// defaultSurfaceID is the conventional surface identifier Compile and
// Generate fall back to.
const defaultSurfaceID = "main"

// SurfaceSpec is the LLM-facing inference format for a complete A2UI
// surface: a simplified, schema-friendly shape that models fill in directly
// (mirroring the official SDK's "inference format" concept). Compile turns a
// SurfaceSpec into the canonical wire messages.
//
// The deliberate differences from the wire format are:
//
//   - One object holds the whole surface instead of a message stream.
//   - ComponentSpec nests catalog properties under "properties" instead of
//     inlining them, so the JSON schema derived from this type stays exact.
//   - Children are always a plain array of IDs (dynamic lists are built in
//     Go with DynamicChildrenOf).
//
// Field descriptions double as JSON-schema descriptions for the model.
type SurfaceSpec struct {
	// SurfaceID uniquely identifies the surface; conventionally "main".
	SurfaceID string `description:"unique surface identifier, e.g. main" json:"surfaceId"`

	// CatalogID identifies the component catalog; DefaultCatalogID when empty.
	CatalogID string `description:"component catalog URL" json:"catalogId"`

	// Components is the flat adjacency list; exactly one entry has id "root".
	Components []ComponentSpec `description:"flat component list; exactly one component has id root" json:"components"`

	// DataModel is the initial application state referenced by Bind paths.
	DataModel map[string]any `description:"initial data model; keys are referenced by property bindings" json:"dataModel,omitempty"`

	// Theme carries theme parameters for createSurface, e.g. primaryColor.
	Theme map[string]any `description:"optional theme parameters, e.g. primaryColor" json:"theme,omitempty"`
}

// ComponentSpec is the LLM-facing shape of one component.
type ComponentSpec struct {
	// ID uniquely identifies the component within the surface; "root" marks
	// the root of the tree.
	ID string `description:"unique component id; the tree root uses root" json:"id"`

	// Kind is the component type from the catalog, e.g. "Text".
	Kind string `description:"component type, e.g. Text, Column, Row, Button" json:"component"`

	// Child references the single child of single-child containers.
	Child string `description:"id of the single child component, for Card and Button" json:"child,omitempty"`

	// Children references the children of list containers.
	Children []string `description:"ids of child components, for Column, Row, and List" json:"children,omitempty"`

	// Accessibility carries assistive-technology labels. Optional.
	Accessibility *Accessibility `description:"optional accessibility labels" json:"accessibility,omitempty"`

	// Properties holds catalog-specific properties: text, variant, url, ...
	Properties map[string]any `description:"catalog-specific properties, e.g. text, variant, url, align, action" json:"properties,omitempty"`
}

// Compile converts a SurfaceSpec into its canonical wire messages:
// createSurface, updateComponents, and (when the spec carries a data model)
// updateDataModel. The result is validated with Validate before it is
// returned, so a spec that compiles is a spec a client can render.
func Compile(spec SurfaceSpec) ([]Message, error) {
	if spec.SurfaceID == "" {
		spec.SurfaceID = defaultSurfaceID
	}

	if spec.CatalogID == "" {
		spec.CatalogID = DefaultCatalogID
	}

	messages := []Message{NewCreateSurface(spec.SurfaceID, spec.CatalogID)}

	createMsg, ok := messages[0].(*CreateSurface)
	if ok {
		createMsg.Theme = spec.Theme
	}

	messages = append(messages, NewUpdateComponents(spec.SurfaceID, componentsToWire(spec.Components)...))

	if len(spec.DataModel) > 0 {
		messages = append(messages, NewUpdateDataModel(spec.SurfaceID, "", spec.DataModel))
	}

	if err := Validate(messages); err != nil {
		return nil, fmt.Errorf("compile surface %q: %w", spec.SurfaceID, err)
	}

	return messages, nil
}

// componentsToWire maps every ComponentSpec to its wire Component.
func componentsToWire(specs []ComponentSpec) []Component {
	components := make([]Component, 0, len(specs))

	for _, spec := range specs {
		component := Component{
			ID:            spec.ID,
			Kind:          spec.Kind,
			Accessibility: spec.Accessibility,
			Props:         spec.Properties,
		}

		if spec.Child != "" {
			child := spec.Child
			component.Child = &child
		}

		if len(spec.Children) > 0 {
			component.Children = StaticChildren(spec.Children...)
		}

		components = append(components, component)
	}

	return components
}
