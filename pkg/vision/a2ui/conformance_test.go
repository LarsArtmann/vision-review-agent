package a2ui

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	jschema "github.com/santhosh-tekuri/jsonschema/v6"
	"github.com/stretchr/testify/require"
)

// The conformance tests in this file validate every message this package
// produces against the OFFICIAL A2UI v0.9.1 JSON schemas, pinned under
// testdata/official/ (see that directory's README for provenance). Passing
// them is what backs the package claim of implementing the v0.9.1 message
// shapes; the internal Validate only checks structural rules.
//
// Validator note: kaptinlin/jsonschema (the fantasy indirect) cannot compile
// the official schemas at all — its exact-decoder rejects every non-number
// JSON value decoded into `any`, which breaks the catalog's string `const`
// and `enum` keywords. santhosh-tekuri/jsonschema/v6 is used instead.

// Schema resource URLs. The official files keep upstream `$id` values with
// v0_9 paths even inside the v0_9_1 directory; refs resolve against those.
const (
	officialS2CURL     = "https://a2ui.org/specification/v0_9/server_to_client.json"
	officialCommonURL  = "https://a2ui.org/specification/v0_9/common_types.json"
	officialCatalogURL = "https://a2ui.org/specification/v0_9/catalogs/basic/catalog.json"
	schemaSiblingAlias = "https://a2ui.org/specification/v0_9/catalog.json"
)

// officialS2CSchema compiles the pinned official server_to_client schema
// together with its dependencies (common_types, basic catalog). All
// resources are registered locally, so compilation never touches the
// network.
func officialS2CSchema(t *testing.T) *jschema.Schema {
	t.Helper()

	compiler := jschema.NewCompiler()

	addResource := func(url, file string) {
		t.Helper()

		doc, err := jschema.UnmarshalJSON(bytes.NewReader(readOfficialSchema(t, file)))
		require.NoErrorf(t, err, "decode %s", file)
		require.NoErrorf(t, compiler.AddResource(url, doc), "register %s", url)
	}

	addResource(officialCommonURL, "common_types.json")
	addResource(officialCatalogURL, "catalog.json")

	// server_to_client.json's relative "$ref": "catalog.json#/$defs/..."
	// entries resolve against its own $id to a sibling URL that differs from
	// the catalog's real $id; register the catalog under both.
	addResource(schemaSiblingAlias, "catalog.json")

	addResource(officialS2CURL, "server_to_client.json")

	schema, err := compiler.Compile(officialS2CURL)
	require.NoError(t, err, "compile official server_to_client.json")

	return schema
}

// readOfficialSchema loads one pinned schema document from testdata.
func readOfficialSchema(t *testing.T, name string) []byte {
	t.Helper()

	data, err := os.ReadFile(filepath.Join("testdata", "official", name))
	require.NoError(t, err, "read pinned schema %s", name)

	return data
}

// decodeInstance decodes a wire document into the validator's instance form.
func decodeInstance(t *testing.T, data []byte) any {
	t.Helper()

	var instance any
	require.NoError(t, json.Unmarshal(data, &instance), "decode instance: %s", data)

	return instance
}

// requireSchemaValid asserts the wire-encoded document passes the official
// schema.
func requireSchemaValid(t *testing.T, schema *jschema.Schema, name string, data []byte) {
	t.Helper()

	err := schema.Validate(decodeInstance(t, data))
	require.NoErrorf(t, err, "%s: official schema rejected the message:\n%s", name, data)
}

// requireSchemaInvalid asserts the document FAILS the official schema; a
// rejection that unexpectedly passes means the conformance test is vacuous.
func requireSchemaInvalid(t *testing.T, schema *jschema.Schema, name string, data []byte) {
	t.Helper()

	err := schema.Validate(decodeInstance(t, data))
	require.Errorf(t, err, "%s: official schema unexpectedly accepted the message:\n%s", name, data)
}

// TestOfficialSchemaConformance is the happy path (plan item M5/F13): specs
// built with this package's public API must Compile into messages whose JSONL
// encoding passes the official v0.9.1 server_to_client schema, line by line.
func TestOfficialSchemaConformance(t *testing.T) {
	t.Parallel()

	schema := officialS2CSchema(t)

	spec := SurfaceSpec{
		SurfaceID: "main",
		Components: []ComponentSpec{
			{
				ID:         RootID,
				Kind:       "Column",
				Children:   []string{"header", "content", "footer"},
				Properties: map[string]any{"align": "center"},
			},
			{ID: "header", Kind: "Text", Properties: map[string]any{"text": Bind("/title"), "variant": "h1"}},
			{ID: "content", Kind: "Row", Children: []string{"avatar", "card"}},
			{
				ID:         "avatar",
				Kind:       "Image",
				Properties: map[string]any{"url": "https://example.com/a.png", "variant": "avatar"},
			},
			{ID: "card", Kind: "Card", Child: "card-body"},
			{ID: "card-body", Kind: "Text", Properties: map[string]any{"text": "Bound and validated."}},
			{
				ID: "footer", Kind: "Button", Child: "button-label",
				Properties: map[string]any{
					"variant": "primary",
					"action":  map[string]any{"event": map[string]any{"name": "clicked"}},
				},
				Accessibility: &Accessibility{Label: "Submit"},
			},
			{ID: "button-label", Kind: "Text", Properties: map[string]any{"text": "Click me"}},
		},
		DataModel: map[string]any{"title": "Conformance"},
		Theme:     map[string]any{"primaryColor": "#0055FF"},
	}

	messages, err := Compile(spec)
	require.NoError(t, err)

	encoded, err := MarshalJSONL(messages)
	require.NoError(t, err)

	for lineIndex, line := range strings.Split(string(encoded), "\n") {
		requireSchemaValid(t, schema, fmt.Sprintf("compiled JSONL line %d", lineIndex), []byte(line))
	}
}

// TestOfficialSchemaEachMessageKind marshals one of each message kind,
// including a remove and a dynamic child list, and validates each against
// the official schema.
func TestOfficialSchemaEachMessageKind(t *testing.T) {
	t.Parallel()

	schema := officialS2CSchema(t)

	row := Component{
		ID: RootID, Kind: "Row",
		Children: DynamicChildrenOf("item", "/items"),
	}
	itemTemplate := Component{
		ID: "item", Kind: "Text",
		Props: map[string]any{"text": Bind("/name")},
	}

	messages := []Message{
		NewCreateSurface("main", DefaultCatalogID),
		NewUpdateComponents("main", row, itemTemplate),
		NewUpdateDataModel("main", "/items", []any{map[string]any{"name": "a"}}),
		NewRemoveDataModelEntry("main", "/items"),
		NewDeleteSurface("main"),
	}

	for _, msg := range messages {
		encoded, err := json.Marshal(msg)
		require.NoErrorf(t, err, "marshal %T", msg)

		requireSchemaValid(t, schema, fmt.Sprintf("wire form of %T", msg), encoded)
	}
}

// TestOfficialSchemaRoundTrip pins the decoder to the same contract: every
// message this package emits must also survive UnmarshalMessage unchanged.
func TestOfficialSchemaRoundTrip(t *testing.T) {
	t.Parallel()

	spec := SurfaceSpec{
		Components: []ComponentSpec{
			{ID: RootID, Kind: "Column", Children: []string{"t"}},
			{ID: "t", Kind: "Text", Properties: map[string]any{"text": "Round trip"}},
		},
	}

	messages, err := Compile(spec)
	require.NoError(t, err)

	encoded, err := MarshalJSONL(messages)
	require.NoError(t, err)

	lines := strings.Split(string(encoded), "\n")

	for i, line := range lines {
		decoded, err := UnmarshalMessage([]byte(line))
		require.NoErrorf(t, err, "line %d: %s", i, line)
		require.Equal(t, messages[i], decoded)
	}
}

// TestOfficialExampleValidates is the positive control for the validator
// itself: the official interactive-button example must pass. If pinned
// schemas or the compiler were wired wrong, the conformance tests above
// could pass vacuously.
func TestOfficialExampleValidates(t *testing.T) {
	t.Parallel()

	schema := officialS2CSchema(t)

	var sample struct {
		Messages []json.RawMessage `json:"messages"`
	}
	require.NoError(t, json.Unmarshal(readOfficialSchema(t, "example_interactive-button.json"), &sample))
	require.NotEmpty(t, sample.Messages)

	for i, message := range sample.Messages {
		requireSchemaValid(t, schema, fmt.Sprintf("official example message %d", i), message)
	}
}

// TestOfficialSchemaRejects is the failure-path table (plan item M5/F14):
// documents the official schema rejects must be rejected here too, and
// where the package has its own guard (UnmarshalMessage) the two must
// agree. Divergences the package tolerates on purpose (version enum,
// required-field leniency) carry decoderRejects=false and are documented
// as M8 decisions.
func TestOfficialSchemaRejects(t *testing.T) {
	t.Parallel()

	schema := officialS2CSchema(t)

	cases := []struct {
		name string
		json string
		// decoderRejects: UnmarshalMessage must also reject this document.
		decoderRejects bool
	}{
		{
			name:           "two kind keys in one envelope",
			json:           `{"version":"v0.9.1","createSurface":{"surfaceId":"s","catalogId":"c"},"deleteSurface":{"surfaceId":"s"}}`,
			decoderRejects: true,
		},
		{
			name:           "no kind key",
			json:           `{"version":"v0.9.1","surfaceId":"s"}`,
			decoderRejects: true,
		},
		{
			name:           "unknown kind key",
			json:           `{"version":"v0.9.1","spinSurface":{"surfaceId":"s"}}`,
			decoderRejects: true,
		},
		{
			name: "unknown version (schema enum is v0.9/v0.9.1; decoder accepts any)",
			json: `{"version":"v0.8","createSurface":{"surfaceId":"s","catalogId":"c"}}`,
		},
		{
			name: "createSurface missing catalogId (decoder leniency, M8)",
			json: `{"version":"v0.9.1","createSurface":{"surfaceId":"s"}}`,
		},
		{
			name: "updateComponents component outside catalog",
			json: `{"version":"v0.9.1","updateComponents":{"surfaceId":"s","components":[{"id":"root","component":"Marquee","text":"x"}]}}`,
		},
		{
			name: "text variant outside enum",
			json: `{"version":"v0.9.1","updateComponents":{"surfaceId":"s","components":[{"id":"root","component":"Text","text":"x","variant":"blinking"}]}}`,
		},
		{
			name: "text missing required text property",
			json: `{"version":"v0.9.1","updateComponents":{"surfaceId":"s","components":[{"id":"root","component":"Text","variant":"body"}]}}`,
		},
		{
			name: "button missing required action",
			json: `{"version":"v0.9.1","updateComponents":{"surfaceId":"s","components":[{"id":"root","component":"Button","child":"label"},{"id":"label","component":"Text","text":"Go"}]}}`,
		},
		{
			name: "unknown component property",
			json: `{"version":"v0.9.1","updateComponents":{"surfaceId":"s","components":[{"id":"root","component":"Text","text":"x","sparkle":true}]}}`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			requireSchemaInvalid(t, schema, tc.name, []byte(tc.json))

			if tc.decoderRejects {
				_, err := UnmarshalMessage([]byte(tc.json))
				require.Errorf(t, err, "UnmarshalMessage accepted a document the official schema rejects: %s", tc.json)
			}
		})
	}
}

// TestAllBuildersConformToOfficialSchema compiles a surface that uses every
// builder this package exposes and validates the wire output against the
// official schema: a builder that emits a schema-invalid component fails
// here, not in a downstream client.
func TestAllBuildersConformToOfficialSchema(t *testing.T) {
	t.Parallel()

	schema := officialS2CSchema(t)

	components := []Component{
		NewColumn(
			RootID,
			"greeting",
			"divider",
			"picker",
			"media",
			"inputs",
			"tabs-host",
			"modal-host",
			"list-host",
			"icon",
		),
		NewText("greeting", "Settings", TextH2),
		NewDivider("divider"),
		NewRow("picker", "check", "choice"),
		NewCheckBox("check", "Subscribe", true),
		NewChoicePicker("choice", "Plan", []ChoicePickerOption{
			{Label: "Hobby", Value: "hobby"}, {Label: "Pro", Value: "pro"},
		}, Bind("/plan"), ChoiceMutuallyExclusive),
		NewRow("media", "audio", "video", "picture"),
		NewAudioPlayer("audio", "https://example.com/a.mp3", "Episode 1"),
		NewVideo("video", "https://example.com/v.mp4"),
		NewImage("picture", "https://example.com/a.png", "A picture"),
		NewColumn("inputs", "field", "datetime", "slider"),
		NewTextField("field", "Email", "a@b.c", FieldShortText),
		NewDateTimeInput("datetime", "Starts", "2026-08-19T10:00", true, true),
		NewSlider("slider", "Volume", 7, 0, 11),
		NewTabs(
			"tabs-host",
			Tab{Title: "General", ChildID: "tab-general"},
			Tab{Title: "Advanced", ChildID: "tab-advanced"},
		),
		NewColumn("tab-general", "tg-text"),
		NewText("tg-text", "General settings", TextBody),
		NewColumn("tab-advanced", "ta-text"),
		NewText("ta-text", "Advanced settings", TextBody),
		NewModal("modal-host", "open-btn", "sheet-card"),
		NewButton("open-btn", "open-label", "sheet.opened"),
		NewText("open-label", "Open sheet", TextBody),
		NewCard("sheet-card", "sheet"),
		NewColumn("sheet", "sheet-text"),
		NewText("sheet-text", "In the sheet", TextBody),
		NewList("list-host", DirectionVertical, "row-item"),
		NewText("row-item", "A list entry", TextBody),
		NewIcon("icon", "star"),
	}

	messages := []Message{
		NewCreateSurface(defaultSurfaceID, DefaultCatalogID),
		NewUpdateComponents(defaultSurfaceID, components...),
	}

	// Structural validation too: builder output must compose into a valid
	// surface, not just valid individual components.
	require.NoError(t, Validate(messages))

	encoded, err := MarshalJSONL(messages)
	require.NoError(t, err)

	for lineIndex, line := range strings.Split(string(encoded), "\n") {
		requireSchemaValid(t, schema, fmt.Sprintf("builder surface JSONL line %d", lineIndex), []byte(line))
	}

	// Every basic-catalog kind must appear in the surface: 18 kinds, 18 Kind
	// constants, all exercised.
	kinds := make(map[string]bool)
	for _, component := range components {
		kinds[component.Kind] = true
	}

	for kind := range basicCatalogKinds(t) {
		require.Truef(t, kinds[kind], "kind %s missing from the builder surface", kind)
	}
}
