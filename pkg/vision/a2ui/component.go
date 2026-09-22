package a2ui

import (
	"encoding/json"
	"fmt"
	"maps"
)

// Wire keys of the four server-to-client message kinds.
const (
	kindCreateSurface    = "createSurface"
	kindUpdateComponents = "updateComponents"
	kindUpdateDataModel  = "updateDataModel"
	kindDeleteSurface    = "deleteSurface"
)

// componentStructuralFields counts the wire keys a component carries besides
// its Props: id, component, accessibility, child, children.
const componentStructuralFields = 5

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
	obj := make(map[string]any, len(c.Props)+componentStructuralFields)
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
	for key, rawValue := range raw {
		var value any
		if err := json.Unmarshal(rawValue, &value); err != nil {
			return fmt.Errorf("component %q property %q: %w", structural.ID, key, err)
		}

		props[key] = value
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
	return map[string]any{propPath: path}
}

// Component kind names from the A2UI basic catalog. Every Kind is a string
// so custom catalogs extend naturally; these cover the kinds this package's
// builders and prompt teach.
const (
	KindText          = "Text"
	KindButton        = "Button"
	KindCard          = "Card"
	KindColumn        = "Column"
	KindRow           = "Row"
	KindList          = "List"
	KindImage         = "Image"
	KindDivider       = "Divider"
	KindIcon          = "Icon"
	KindTabs          = "Tabs"
	KindModal         = "Modal"
	KindCheckBox      = "CheckBox"
	KindChoicePicker  = "ChoicePicker"
	KindTextField     = "TextField"
	KindDateTimeInput = "DateTimeInput"
	KindSlider        = "Slider"
	KindAudioPlayer   = "AudioPlayer"
	KindVideo         = "Video"
)

// Catalog property keys used across the builders and tests.
const (
	propText        = "text"
	propName        = "name"
	propVariant     = "variant"
	propAction      = "action"
	propEvent       = "event"
	propURL         = "url"
	propDescription = "description"
	propLabel       = "label"
	propValue       = "value"
	propOptions     = "options"
	propMin         = "min"
	propMax         = "max"
	propTabs        = "tabs"
	propDirection   = "direction"
	propAlign       = "align"
	propTrigger     = "trigger"
	propContent     = "content"
	propTitle       = "title"
	propChild       = "child"
	propEnableDate  = "enableDate"
	propEnableTime  = "enableTime"
	propPath        = "path"
	propStar        = "*"
)

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

// ChoicePicker selection variants (ChoicePicker.variant) and display
// styles (ChoicePicker.displayStyle).
const (
	ChoiceMultipleSelection = "multipleSelection"
	ChoiceMutuallyExclusive = "mutuallyExclusive"
	ChoiceStyleCheckbox     = "checkbox"
	ChoiceStyleChips        = "chips"
)

// TextField input variants (TextField.variant).
const (
	FieldShortText = "shortText"
	FieldLongText  = "longText"
	FieldNumber    = "number"
	FieldObscured  = "obscured"
)

// List directions (List.direction) and container alignment values
// (Column/Row/List .align).
const (
	DirectionVertical   = "vertical"
	DirectionHorizontal = "horizontal"
	AlignStart          = "start"
	AlignCenter         = "center"
	AlignEnd            = "end"
	AlignStretch        = "stretch"
)

// NewText builds a Text component. variant selects the base text style; pass
// TextBody (or "") for the default.
func NewText(componentID, text, variant string) Component {
	props := map[string]any{propText: text}

	if variant != "" {
		props["variant"] = variant
	}

	return Component{ID: componentID, Kind: KindText, Props: props}
}

// NewColumn builds a Column container laying children out vertically.
func NewColumn(id string, children ...string) Component {
	return Component{ID: id, Kind: KindColumn, Children: StaticChildren(children...)}
}

// NewRow builds a Row container laying children out horizontally.
func NewRow(id string, children ...string) Component {
	return Component{ID: id, Kind: KindRow, Children: StaticChildren(children...)}
}

// NewCard builds a Card wrapping a single child.
func NewCard(id, childID string) Component {
	return Component{ID: id, Kind: KindCard, Child: new(childID)}
}

// NewButton builds a Button whose labeled child is childID and which
// dispatches the server-side event name.
func NewButton(id, childID, event string) Component {
	return Component{
		ID:    id,
		Kind:  KindButton,
		Child: new(childID),
		Props: map[string]any{
			propAction: map[string]any{propEvent: map[string]any{propName: event}},
		},
	}
}

// NewImage builds an Image component showing url. description is optional
// alt text; pass "" to omit.
func NewImage(componentID, url, description string) Component {
	return Component{ID: componentID, Kind: KindImage, Props: urlDescriptionProps(url, description)}
}

// NewDivider builds a horizontal Divider.
func NewDivider(id string) Component {
	return Component{ID: id, Kind: KindDivider}
}

// NewIcon builds an Icon component by catalog icon name.
func NewIcon(id, name string) Component {
	return Component{ID: id, Kind: KindIcon, Props: map[string]any{propName: name}}
}

// NewList builds a List container stacking children along direction
// (DirectionVertical or DirectionHorizontal; "" omits the property).
func NewList(componentID, direction string, children ...string) Component {
	component := Component{ID: componentID, Kind: KindList, Children: StaticChildren(children...)}

	if direction != "" {
		component.Props = map[string]any{propDirection: direction}
	}

	return component
}

// NewModal builds a Modal: interacting with the trigger child opens the
// content child.
func NewModal(componentID, triggerID, contentID string) Component {
	return Component{
		ID:   componentID,
		Kind: KindModal,
		Props: map[string]any{
			propTrigger: triggerID,
			propContent: contentID,
		},
	}
}

// NewTabs builds a Tabs container showing one Tab's child at a time.
func NewTabs(componentID string, tabs ...Tab) Component {
	wireTabs := make([]map[string]any, len(tabs))

	for i, tab := range tabs {
		wireTabs[i] = map[string]any{propTitle: tab.Title, propChild: tab.ChildID}
	}

	return Component{ID: componentID, Kind: KindTabs, Props: map[string]any{propTabs: wireTabs}}
}

// Tab is one tab of a Tabs component: a title and the child component
// rendered while the tab is selected.
type Tab struct {
	Title   string
	ChildID string
}

// NewCheckBox builds a CheckBox labeled label, checked per value.
func NewCheckBox(componentID, label string, value bool) Component {
	return Component{
		ID:   componentID,
		Kind: KindCheckBox,
		Props: map[string]any{
			propLabel: label,
			propValue: value,
		},
	}
}

// ChoicePickerOption is one selectable option of a ChoicePicker: a human
// label and the stable value it carries.
type ChoicePickerOption struct {
	Label string
	Value string
}

// NewChoicePicker builds a ChoicePicker over options. value is the current
// selection: a []string of option values, a Bind path to a string array in
// the data model, or []string{} for nothing selected. variant selects
// ChoiceMultipleSelection (the default) or ChoiceMutuallyExclusive.
func NewChoicePicker(componentID, label string, options []ChoicePickerOption, value any, variant string) Component {
	wireOptions := make([]map[string]any, len(options))

	for i, option := range options {
		wireOptions[i] = map[string]any{propLabel: option.Label, propValue: option.Value}
	}

	props := map[string]any{
		propLabel:   label,
		propOptions: wireOptions,
		propValue:   value,
	}

	if variant != "" {
		props[propVariant] = variant
	}

	return Component{ID: componentID, Kind: KindChoicePicker, Props: props}
}

// NewTextField builds a text input labeled label holding value. variant
// picks FieldShortText (the default), FieldLongText, FieldNumber, or
// FieldObscured; pass "" to omit value or variant.
func NewTextField(componentID, label, value, variant string) Component {
	props := map[string]any{propLabel: label}

	if value != "" {
		props[propValue] = value
	}

	if variant != "" {
		props[propVariant] = variant
	}

	return Component{ID: componentID, Kind: KindTextField, Props: props}
}

// NewDateTimeInput builds a DateTimeInput labeled label, holding value (an
// ISO-8601 timestamp or a Bind path); enableDate and enableTime toggle the
// date and time pickers.
func NewDateTimeInput(componentID, label, value string, enableDate, enableTime bool) Component {
	props := map[string]any{propLabel: label, propValue: value}

	if enableDate {
		props[propEnableDate] = true
	}

	if enableTime {
		props[propEnableTime] = true
	}

	return Component{ID: componentID, Kind: KindDateTimeInput, Props: props}
}

// NewSlider builds a Slider labeled label, bound to value between minValue
// and maxValue.
func NewSlider(componentID, label string, value, minValue, maxValue float64) Component {
	return Component{
		ID:   componentID,
		Kind: KindSlider,
		Props: map[string]any{
			propLabel: label,
			propValue: value,
			propMin:   minValue,
			propMax:   maxValue,
		},
	}
}

// NewAudioPlayer builds an AudioPlayer for url; description is optional
// alt text, pass "" to omit.
func NewAudioPlayer(componentID, url, description string) Component {
	return Component{ID: componentID, Kind: KindAudioPlayer, Props: urlDescriptionProps(url, description)}
}

// urlDescriptionProps builds the shared url-plus-optional-description
// property set of the Image and AudioPlayer builders.
func urlDescriptionProps(url, description string) map[string]any {
	props := map[string]any{propURL: url}

	if description != "" {
		props[propDescription] = description
	}

	return props
}

// NewVideo builds a Video player for url.
func NewVideo(componentID, url string) Component {
	return Component{ID: componentID, Kind: KindVideo, Props: map[string]any{propURL: url}}
}
