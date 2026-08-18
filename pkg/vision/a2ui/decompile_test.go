package a2ui

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDecompileRoundTrip(t *testing.T) {
	t.Parallel()

	spec := SurfaceSpec{
		SurfaceID: "editor",
		CatalogID: DefaultCatalogID,
		Components: []ComponentSpec{
			{ID: RootID, Kind: "Column", Children: []string{"title", "save"}},
			{
				ID: "title", Kind: "Text", Properties: map[string]any{"text": "Edit", "variant": "h1"},
				Accessibility: &Accessibility{Label: "Page title"},
			},
			{
				ID: "save", Kind: "Button", Child: "save-label",
				Properties: map[string]any{"action": map[string]any{"event": map[string]any{"name": "saved"}}},
			},
			{ID: "save-label", Kind: "Text", Properties: map[string]any{"text": "Save"}},
		},
		DataModel: map[string]any{"count": 3.0},
		Theme:     map[string]any{"primaryColor": "#0055FF"},
	}

	messages, err := Compile(spec)
	require.NoError(t, err)

	roundTripped, err := Decompile(messages)
	require.NoError(t, err)
	require.Equal(t, spec, roundTripped)
}

func TestDecompileMinimalSpec(t *testing.T) {
	t.Parallel()

	spec := SurfaceSpec{
		Components: []ComponentSpec{
			{ID: RootID, Kind: "Text", Properties: map[string]any{"text": "Solo"}},
		},
	}

	messages, err := Compile(spec)
	require.NoError(t, err)

	// Compile defaults the ids; Decompile returns the defaulted spec.
	roundTripped, err := Decompile(messages)
	require.NoError(t, err)
	require.Equal(t, defaultSurfaceID, roundTripped.SurfaceID)
	require.Equal(t, DefaultCatalogID, roundTripped.CatalogID)
	require.Len(t, roundTripped.Components, 1)
}

func TestDecompileFoldsDataModelWrites(t *testing.T) {
	t.Parallel()

	messages := []Message{
		NewCreateSurface("main", DefaultCatalogID),
		NewUpdateComponents("main", NewText(RootID, "List", TextBody)),
		NewUpdateDataModel("main", "", map[string]any{"user": map[string]any{"name": "Ada"}}),
		NewUpdateDataModel("main", "/user/email", "ada@example.com"),
		NewUpdateDataModel("main", "/user/name", "Ada Lovelace"),
		NewRemoveDataModelEntry("main", "/user/email"),
	}

	spec, err := Decompile(messages)
	require.NoError(t, err)
	require.Equal(t, map[string]any{
		"user": map[string]any{"name": "Ada Lovelace"},
	}, spec.DataModel)
}

func TestDecompileKeepsOrphanComponents(t *testing.T) {
	t.Parallel()

	messages := []Message{
		NewCreateSurface("main", DefaultCatalogID),
		NewUpdateComponents("main",
			NewColumn(RootID, "title"),
			NewText("title", "Visible", TextBody),
			NewText("orphan", "Unreferenced but legal", TextBody),
		),
	}

	spec, err := Decompile(messages)
	require.NoError(t, err)
	require.Len(t, spec.Components, 3)

	ids := []string{spec.Components[0].ID, spec.Components[1].ID, spec.Components[2].ID}
	require.Equal(t, []string{RootID, "title", "orphan"}, ids)
}

func TestDecompileRejects(t *testing.T) {
	t.Parallel()

	baseComponents := NewUpdateComponents("main", NewText(RootID, "x", TextBody))

	cases := []struct {
		name     string
		messages []Message
	}{
		{
			name:     "empty stream",
			messages: nil,
		},
		{
			name: "update before create",
			messages: []Message{
				baseComponents,
			},
		},
		{
			name: "double create",
			messages: []Message{
				NewCreateSurface("main", DefaultCatalogID),
				NewCreateSurface("main", DefaultCatalogID),
			},
		},
		{
			name: "delete surface",
			messages: []Message{
				NewCreateSurface("main", DefaultCatalogID),
				baseComponents,
				NewDeleteSurface("main"),
			},
		},
		{
			name: "dynamic child list",
			messages: []Message{
				NewCreateSurface("main", DefaultCatalogID),
				NewUpdateComponents("main", Component{
					ID: RootID, Kind: "Row", Children: DynamicChildrenOf("item", "/items"),
				}),
			},
		},
		{
			name: "sendDataModel flag",
			messages: func() []Message {
				create := NewCreateSurface("main", DefaultCatalogID)
				create.SendDataModel = true

				return []Message{create, baseComponents}
			}(),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			spec, err := Decompile(tc.messages)
			require.ErrorIs(t, err, ErrDecompile)
			require.Empty(t, spec.Components)
		})
	}
}

// TestCompileDecompileEditLoop proves the edit workflow: compile, decompile,
// mutate the spec, recompile.
func TestCompileDecompileEditLoop(t *testing.T) {
	t.Parallel()

	messages, err := Compile(SurfaceSpec{
		Components: []ComponentSpec{
			{ID: RootID, Kind: "Column", Children: []string{"title"}},
			{ID: "title", Kind: "Text", Properties: map[string]any{"text": "v1"}},
		},
	})
	require.NoError(t, err)

	spec, err := Decompile(messages)
	require.NoError(t, err)

	spec.Components[1].Properties[propText] = "v2"

	recompiled, err := Compile(spec)
	require.NoError(t, err)

	encoded, err := MarshalJSONL(recompiled)
	require.NoError(t, err)
	require.Contains(t, string(encoded), `"text":"v2"`)
}
