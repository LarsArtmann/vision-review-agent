package a2ui

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// messageKinds lists the wire keys of every server-to-client message kind;
// used for dispatch and error reporting.
func messageKinds() []string {
	return []string{
		kindCreateSurface,
		kindUpdateComponents,
		kindUpdateDataModel,
		kindDeleteSurface,
	}
}

// ErrMalformedMessage wraps every UnmarshalMessage envelope failure.
var ErrMalformedMessage = errors.New("a2ui: malformed message")

// Message is one A2UI server-to-client message. Concrete kinds are
// *CreateSurface, *UpdateComponents, *UpdateDataModel, and *DeleteSurface.
// Messages travel to clients as JSON Lines (one JSON object per line, see
// MarshalJSONL).
type Message interface {
	isMessage()

	// Surface identifies the surface the message applies to.
	Surface() string

	// Version reports the protocol version carried on the wire.
	Version() string
}

// CreateSurface signals the client to create a new surface and begin
// rendering it. It is an error to send CreateSurface for a surface that
// already exists without deleting it first.
type CreateSurface struct {
	// SurfaceID uniquely identifies the surface.
	SurfaceID string

	// CatalogID identifies the component catalog the components belong to.
	CatalogID string

	// Theme carries theme parameters (e.g. {"primaryColor": "#FF0000"}).
	// Optional.
	Theme map[string]any

	// SendDataModel makes the client send the full data model back with
	// every action it dispatches. Optional.
	SendDataModel bool

	version string
}

// NewCreateSurface builds a CreateSurface message for the given surface,
// stamped with VersionV091.
func NewCreateSurface(surfaceID, catalogID string) *CreateSurface {
	return &CreateSurface{SurfaceID: surfaceID, CatalogID: catalogID, version: VersionV091}
}

func (m *CreateSurface) isMessage()      {}
func (m *CreateSurface) Surface() string { return m.SurfaceID }
func (m *CreateSurface) Version() string { return m.version }

// MarshalJSON renders the wire shape {"version": ..., "createSurface": {...}}.
func (m *CreateSurface) MarshalJSON() ([]byte, error) {
	payload := struct {
		SurfaceID     string         `json:"surfaceId"`
		CatalogID     string         `json:"catalogId"`
		Theme         map[string]any `json:"theme,omitempty"`
		SendDataModel bool           `json:"sendDataModel,omitempty"`
	}{
		SurfaceID:     m.SurfaceID,
		CatalogID:     m.CatalogID,
		Theme:         m.Theme,
		SendDataModel: m.SendDataModel,
	}

	encoded, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("encode createSurface payload: %w", err)
	}

	return marshalEnvelope(m.version, kindCreateSurface, encoded)
}

// UnmarshalJSON decodes the createSurface wire shape.
func (m *CreateSurface) UnmarshalJSON(data []byte) error {
	var payload struct {
		SurfaceID     string         `json:"surfaceId"`
		CatalogID     string         `json:"catalogId"`
		Theme         map[string]any `json:"theme"`
		SendDataModel bool           `json:"sendDataModel"`
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		return fmt.Errorf("%w: decode createSurface payload: %w", ErrMalformedMessage, err)
	}

	m.SurfaceID = payload.SurfaceID
	m.CatalogID = payload.CatalogID
	m.Theme = payload.Theme
	m.SendDataModel = payload.SendDataModel

	return nil
}

// UpdateComponents replaces the component tree of an existing surface. One
// component in the list must use the ID "root" (see RootID).
type UpdateComponents struct {
	// SurfaceID identifies the surface to update.
	SurfaceID string

	// Components is the full component list for the surface.
	Components []Component

	version string
}

// NewUpdateComponents builds an UpdateComponents message, stamped with
// VersionV091.
func NewUpdateComponents(surfaceID string, components ...Component) *UpdateComponents {
	return &UpdateComponents{
		SurfaceID:  surfaceID,
		Components: components,
		version:    VersionV091,
	}
}

func (m *UpdateComponents) isMessage()      {}
func (m *UpdateComponents) Surface() string { return m.SurfaceID }
func (m *UpdateComponents) Version() string { return m.version }

// MarshalJSON renders the wire shape {"version": ..., "updateComponents": {...}}.
func (m *UpdateComponents) MarshalJSON() ([]byte, error) {
	encoded, err := json.Marshal(struct {
		SurfaceID  string      `json:"surfaceId"`
		Components []Component `json:"components"`
	}{SurfaceID: m.SurfaceID, Components: m.Components})
	if err != nil {
		return nil, fmt.Errorf("encode updateComponents payload: %w", err)
	}

	return marshalEnvelope(m.version, kindUpdateComponents, encoded)
}

// UnmarshalJSON decodes the updateComponents wire shape.
func (m *UpdateComponents) UnmarshalJSON(data []byte) error {
	var payload struct {
		SurfaceID  string      `json:"surfaceId"`
		Components []Component `json:"components"`
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		return fmt.Errorf("%w: decode updateComponents payload: %w", ErrMalformedMessage, err)
	}

	m.SurfaceID = payload.SurfaceID
	m.Components = payload.Components

	return nil
}

// UpdateDataModel updates application state for an existing surface. With
// Path empty (or "/") it replaces the entire data model; otherwise it
// replaces (Remove=false) or deletes (Remove=true) the value at Path.
type UpdateDataModel struct {
	// SurfaceID identifies the surface whose data model changes.
	SurfaceID string

	// Path is a JSON Pointer within the data model (e.g. "/user/name").
	// Empty means the whole data model.
	Path string

	// Value is the data to write. Only meaningful when Remove is false.
	Value any

	// Remove deletes the key at Path instead of writing Value.
	Remove bool

	version string
}

// NewUpdateDataModel builds an UpdateDataModel message writing value at path
// ("" for the whole model), stamped with VersionV091.
func NewUpdateDataModel(surfaceID, path string, value any) *UpdateDataModel {
	return &UpdateDataModel{SurfaceID: surfaceID, Path: path, Value: value, version: VersionV091}
}

// NewRemoveDataModelEntry builds an UpdateDataModel message deleting the key
// at path, stamped with VersionV091.
func NewRemoveDataModelEntry(surfaceID, path string) *UpdateDataModel {
	return &UpdateDataModel{SurfaceID: surfaceID, Path: path, Remove: true, version: VersionV091}
}

func (m *UpdateDataModel) isMessage()      {}
func (m *UpdateDataModel) Surface() string { return m.SurfaceID }
func (m *UpdateDataModel) Version() string { return m.version }

// MarshalJSON renders the wire shape {"version": ..., "updateDataModel": {...}}.
func (m *UpdateDataModel) MarshalJSON() ([]byte, error) {
	payload := map[string]any{"surfaceId": m.SurfaceID}

	switch {
	case m.Remove:
		payload["path"] = m.Path
	case m.Path != "":
		payload["path"] = m.Path
		payload["value"] = m.Value
	default:
		payload["value"] = m.Value
	}

	encoded, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("encode updateDataModel payload: %w", err)
	}

	return marshalEnvelope(m.version, kindUpdateDataModel, encoded)
}

// UnmarshalJSON decodes the updateDataModel wire shape.
func (m *UpdateDataModel) UnmarshalJSON(data []byte) error {
	var payload struct {
		SurfaceID string `json:"surfaceId"`
		Path      string `json:"path"`
		Value     any    `json:"value"`
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		return fmt.Errorf("%w: decode updateDataModel payload: %w", ErrMalformedMessage, err)
	}

	m.SurfaceID = payload.SurfaceID
	m.Path = payload.Path
	m.Value = payload.Value
	m.Remove = payload.Value == nil && payload.Path != ""

	return nil
}

// DeleteSurface removes an existing surface.
type DeleteSurface struct {
	// SurfaceID identifies the surface to delete.
	SurfaceID string

	version string
}

// NewDeleteSurface builds a DeleteSurface message, stamped with VersionV091.
func NewDeleteSurface(surfaceID string) *DeleteSurface {
	return &DeleteSurface{SurfaceID: surfaceID, version: VersionV091}
}

func (m *DeleteSurface) isMessage()      {}
func (m *DeleteSurface) Surface() string { return m.SurfaceID }
func (m *DeleteSurface) Version() string { return m.version }

// MarshalJSON renders the wire shape {"version": ..., "deleteSurface": {...}}.
func (m *DeleteSurface) MarshalJSON() ([]byte, error) {
	encoded, err := json.Marshal(struct {
		SurfaceID string `json:"surfaceId"`
	}{SurfaceID: m.SurfaceID})
	if err != nil {
		return nil, fmt.Errorf("encode deleteSurface payload: %w", err)
	}

	return marshalEnvelope(m.version, kindDeleteSurface, encoded)
}

// UnmarshalJSON decodes the deleteSurface wire shape.
func (m *DeleteSurface) UnmarshalJSON(data []byte) error {
	var payload struct {
		SurfaceID string `json:"surfaceId"`
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		return fmt.Errorf("%w: decode deleteSurface payload: %w", ErrMalformedMessage, err)
	}

	m.SurfaceID = payload.SurfaceID

	return nil
}

// marshalEnvelope wraps an encoded payload in {"version": ..., kind: ...}.
func marshalEnvelope(version, kind string, payload []byte) ([]byte, error) {
	var envelope bytes.Buffer

	envelope.WriteString(`{"version":`)

	encodedVersion, err := json.Marshal(version)
	if err != nil {
		return nil, fmt.Errorf("encode version: %w", err)
	}

	envelope.Write(encodedVersion)
	envelope.WriteString(`,"`)
	envelope.WriteString(kind)
	envelope.WriteString(`":`)
	envelope.Write(payload)
	envelope.WriteString("}")

	return envelope.Bytes(), nil
}

// UnmarshalMessage decodes a single A2UI message from its wire JSON. The
// concrete type is one of *CreateSurface, *UpdateComponents,
// *UpdateDataModel, or *DeleteSurface; the envelope version is copied into
// the message.
func UnmarshalMessage(data []byte) (Message, error) {
	var envelope struct {
		Version string `json:"version"`
	}
	if err := json.Unmarshal(data, &envelope); err != nil {
		return nil, fmt.Errorf("%w: decode envelope: %w", ErrMalformedMessage, err)
	}

	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return nil, fmt.Errorf("%w: decode fields: %w", ErrMalformedMessage, err)
	}

	delete(fields, "version")

	kinds := messageKinds()

	var present []string

	for _, kind := range kinds {
		if _, ok := fields[kind]; ok {
			present = append(present, kind)
		}
	}

	if len(present) != 1 {
		return nil, fmt.Errorf(
			"%w: exactly one kind key required (%s), got %d: %v",
			ErrMalformedMessage, strings.Join(kinds, ", "), len(present), present,
		)
	}

	return decodeKind(present[0], fields[present[0]], envelope.Version)
}

// decodeKind decodes one message payload and stamps the envelope version.
func decodeKind(kind string, payload json.RawMessage, version string) (Message, error) {
	var (
		msg Message
		err error
	)

	switch kind {
	case kindCreateSurface:
		decoded := &CreateSurface{}
		err = decoded.UnmarshalJSON(payload)
		decoded.version = version
		msg = decoded
	case kindUpdateComponents:
		decoded := &UpdateComponents{}
		err = decoded.UnmarshalJSON(payload)
		decoded.version = version
		msg = decoded
	case kindUpdateDataModel:
		decoded := &UpdateDataModel{}
		err = decoded.UnmarshalJSON(payload)
		decoded.version = version
		msg = decoded
	case kindDeleteSurface:
		decoded := &DeleteSurface{}
		err = decoded.UnmarshalJSON(payload)
		decoded.version = version
		msg = decoded
	}

	if err != nil {
		return nil, fmt.Errorf("%w: decode %s payload: %w", ErrMalformedMessage, kind, err)
	}

	return msg, nil
}

// MarshalJSONL encodes messages as JSON Lines: one compact JSON object per
// line, the wire transport encoding of A2UI.
func MarshalJSONL(messages []Message) ([]byte, error) {
	var out bytes.Buffer

	for i, msg := range messages {
		encoded, err := json.Marshal(msg)
		if err != nil {
			return nil, fmt.Errorf("encode message %d (%T): %w", i, msg, err)
		}

		if i > 0 {
			out.WriteByte('\n')
		}

		out.Write(encoded)
	}

	return out.Bytes(), nil
}

// UnmarshalJSONL decodes JSON Lines into messages. Blank lines are skipped.
func UnmarshalJSONL(data []byte) ([]Message, error) {
	var messages []Message

	for i, line := range strings.Split(string(data), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		msg, err := UnmarshalMessage([]byte(trimmed))
		if err != nil {
			return nil, fmt.Errorf("line %d: %w", i+1, err)
		}

		messages = append(messages, msg)
	}

	return messages, nil
}
