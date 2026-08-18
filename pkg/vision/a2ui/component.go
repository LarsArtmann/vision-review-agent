package a2ui

import (
	"encoding/json"
	"fmt"
	"maps"
)

// Component is a single node in an A2UI surface's adjacency list. Components
// reference children by ID (flat list, not a nested tree), which keeps the
// structure easy for models to emit incrementally and for clients to patch.
//
// Kind names the component type in the surface's catalog (for the basic
// catalog: Text, Button, Column, Row, Card, Image, ...). Props holds the
// catalog-specific properties (e.g. text, variant, align); values may be
// literals or data bindings produced with Bind.
//
// Exactly one of Child or Children is set for container components; Validate
// rejects ambiguity.
//
//nolint:recvcheck // MarshalJSON takes a value, UnmarshalJSON a pointer (standard JSON pattern).
type Component struct {
	// ID uniquely identifies the component within its surface. The root
	// component uses the conventional ID "root".
	ID string

	// Kind is the component type name from the catalog, e.g. "Text".
	Kind string

	// Accessibility carries assistive-technology labels. Optional.
	Accessibility *Accessibility

	// Child references the single child component of single-child containers
	// (Card, Button, Modal triggers, ...). Optional.
	Child *string

	// Children references the children of list containers (Column, Row,
	// List). Optional.
	Children *ChildList

	// Props holds every catalog-specific property (text, variant, url, ...).
	// Values are literals (string, float64, bool, ...) or bindings from Bind.
	Props map[string]any
}

// MarshalJSON renders the component in the flat wire shape: id, component,
// accessibility, child, children, then all Props merged in.
func (c Component) MarshalJSON() ([]byte, error) {
	obj := make(map[string]any, len(c.Props)+5)
	maps.Copy(obj, c.Props)
	obj["id"] = c.ID
	obj["component"] = c.Kind

	if c.Accessibility != nil {
		obj["accessibility"] = c.Accessibility
	}

	if c.Child != nil {
		obj["child"] = *c.Child
	}

	if c.Children != nil {
		obj["children"] = c.Children
	}

	encoded, err := json.Marshal(obj)
	if err != nil {
		return nil, fmt.Errorf("encode component %q: %w", c.ID, err)
	}

	return encoded, nil
}

// UnmarshalJSON decodes the flat wire shape, splitting the structural fields
// (id, component, accessibility, child, children) from catalog-specific Props.
func (c *Component) UnmarshalJSON(data []byte) error {
	var structural struct {
		ID            string         `json:"id"`
		Kind          string         `json:"component"`
		Accessibility *Accessibility `json:"accessibility"`
		Child         *string        `json:"child"`
		Children      *ChildList     `json:"children"`
	}
	if err := json.Unmarshal(data, &structural); err != nil {
		return fmt.Errorf("decode component: %w", err)
	}

	c.ID = structural.ID
	c.Kind = structural.Kind
	c.Accessibility = structural.Accessibility
	c.Child = structural.Child
	c.Children = structural.Children

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("decode component %q fields: %w", structural.ID, err)
	}

	delete(raw, "id")
	delete(raw, "component")
	delete(raw, "accessibility")
	delete(raw, "child")
	delete(raw, "children")

	props := make(map[string]any, len(raw))
	for k, v := range raw {
		var value any
		if err := json.Unmarshal(v, &value); err != nil {
			return fmt.Errorf("component %q property %q: %w", structural.ID, k, err)
		}

		props[k] = value
	}

	if len(props) > 0 {
		c.Props = props
	}

	return nil
}

// ChildList is the set of children of a list container. Exactly one shape is
// active: a static list of component IDs, or a dynamic template that
// instantiates one component per data-model list entry. The zero ChildList is
// invalid; use StaticChildren or DynamicChildrenOf.
//
//nolint:recvcheck // MarshalJSON takes a value, UnmarshalJSON a pointer (standard JSON pattern).
type ChildList struct {
	// Static lists child component IDs explicitly.
	Static []string

	// Dynamic describes a template-generated list.
	Dynamic *DynamicChildren
}

// DynamicChildren generates a dynamic list: the client instantiates
// ComponentID once per entry of the data-model list at Path.
type DynamicChildren struct {
	ComponentID string `json:"componentId"`
	Path        string `json:"path"`
}

// StaticChildren builds a ChildList from explicit component IDs.
func StaticChildren(ids ...string) *ChildList {
	return &ChildList{Static: ids}
}

// DynamicChildrenOf builds a ChildList that instantiates componentID once per
// entry of the data-model list at path.
func DynamicChildrenOf(componentID, path string) *ChildList {
	return &ChildList{Dynamic: &DynamicChildren{ComponentID: componentID, Path: path}}
}

// MarshalJSON renders the active shape: a JSON array for static children, an
// object for dynamic children.
func (l ChildList) MarshalJSON() ([]byte, error) {
	if l.Dynamic != nil {
		encoded, err := json.Marshal(l.Dynamic)
		if err != nil {
			return nil, fmt.Errorf("encode dynamic children: %w", err)
		}

		return encoded, nil
	}

	encoded, err := json.Marshal(l.Static)
	if err != nil {
		return nil, fmt.Errorf("encode static children: %w", err)
	}

	return encoded, nil
}

// UnmarshalJSON accepts either wire shape. A null decodes to an empty
// ChildList, which Validate rejects.
func (l *ChildList) UnmarshalJSON(data []byte) error {
	var static []string
	if err := json.Unmarshal(data, &static); err == nil {
		l.Static = static
		l.Dynamic = nil

		return nil
	}

	var dynamic DynamicChildren
	if err := json.Unmarshal(data, &dynamic); err != nil {
		return fmt.Errorf("child list: expected array of IDs or {componentId, path}: %w", err)
	}

	l.Static = nil
	l.Dynamic = &dynamic

	return nil
}

// Accessibility carries labels consumed by assistive technologies. Label and
// Description are DynamicStrings: a literal string or a Bind binding.
type Accessibility struct {
	Label       any `json:"label,omitempty"`
	Description any `json:"description,omitempty"`
}

// Literal wraps a literal value for a dynamic property slot. It is the
// identity function, provided so call sites read declaratively next to Bind.
func Literal(value string) any {
	return value
}

// Bind builds a data binding: the property resolves to the JSON Pointer path
// in the surface's data model at render time.
func Bind(path string) any {
	return map[string]any{"path": path}
}

// Text style variants (Text.variant).
const (
	TextH1      = "h1"
	TextH2      = "h2"
	TextH3      = "h3"
	TextH4      = "h4"
	TextH5      = "h5"
	TextCaption = "caption"
	TextBody    = "body"
)

// NewText builds a Text component. variant selects the base text style; pass
// TextBody (or "") for the default.
func NewText(id, text, variant string) Component {
	props := map[string]any{"text": text}

	if variant != "" {
		props["variant"] = variant
	}

	return Component{ID: id, Kind: "Text", Props: props}
}

// NewColumn builds a Column container laying children out vertically.
func NewColumn(id string, children ...string) Component {
	return Component{ID: id, Kind: "Column", Children: StaticChildren(children...)}
}

// NewRow builds a Row container laying children out horizontally.
func NewRow(id string, children ...string) Component {
	return Component{ID: id, Kind: "Row", Children: StaticChildren(children...)}
}

// NewCard builds a Card wrapping a single child.
func NewCard(id, childID string) Component {
	return Component{ID: id, Kind: "Card", Child: &childID}
}

// NewButton builds a Button whose labeled child is childID and which
// dispatches the server-side event name.
func NewButton(id, childID, event string) Component {
	return Component{
		ID:    id,
		Kind:  "Button",
		Child: &childID,
		Props: map[string]any{
			"action": map[string]any{"event": map[string]any{"name": event}},
		},
	}
}

// NewImage builds an Image component showing url. description is optional
// alt text; pass "" to omit.
func NewImage(id, url, description string) Component {
	props := map[string]any{"url": url}

	if description != "" {
		props["description"] = description
	}

	return Component{ID: id, Kind: "Image", Props: props}
}

// NewDivider builds a horizontal Divider.
func NewDivider(id string) Component {
	return Component{ID: id, Kind: "Divider"}
}

// NewIcon builds an Icon component by catalog icon name.
func NewIcon(id, name string) Component {
	return Component{ID: id, Kind: "Icon", Props: map[string]any{"name": name}}
}
