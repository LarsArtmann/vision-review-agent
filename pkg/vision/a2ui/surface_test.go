package a2ui

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCompileDefaultsAndMessages(t *testing.T) {
	t.Parallel()

	spec := SurfaceSpec{
		Components: []ComponentSpec{
			{ID: RootID, Kind: "Column", Children: []string{"title"}},
			{ID: "title", Kind: "Text", Properties: map[string]any{"text": "Hello"}},
		},
		DataModel: map[string]any{"greeting": "Hello"},
		Theme:     map[string]any{"primaryColor": "#0055FF"},
	}

	messages, err := Compile(spec)
	require.NoError(t, err)
	require.Len(t, messages, 3)

	create, ok := messages[0].(*CreateSurface)
	require.True(t, ok, "first message must be createSurface")
	require.Equal(t, "main", create.SurfaceID)
	require.Equal(t, DefaultCatalogID, create.CatalogID)
	require.Equal(t, map[string]any{"primaryColor": "#0055FF"}, create.Theme)

	update, ok := messages[1].(*UpdateComponents)
	require.True(t, ok, "second message must be updateComponents")
	require.Equal(t, "main", update.SurfaceID)
	require.Len(t, update.Components, 2)
	require.Equal(t, "Column", update.Components[0].Kind)
	require.Equal(t, StaticChildren("title"), update.Components[0].Children)
	require.Equal(t, map[string]any{"text": "Hello"}, update.Components[1].Props)

	data, ok := messages[2].(*UpdateDataModel)
	require.True(t, ok, "third message must be updateDataModel")
	require.Equal(t, map[string]any{"greeting": "Hello"}, data.Value)

	require.NoError(t, Validate(messages))
}

func TestCompileWithoutDataModelOmitsMessage(t *testing.T) {
	t.Parallel()

	messages, err := Compile(SurfaceSpec{
		Components: []ComponentSpec{{ID: RootID, Kind: "Text", Properties: map[string]any{"text": "Hi"}}},
	})
	require.NoError(t, err)
	require.Len(t, messages, 2)

	_, isData := messages[len(messages)-1].(*UpdateDataModel)
	require.False(t, isData, "no updateDataModel without a data model")
}

func TestCompileMapsSingleChild(t *testing.T) {
	t.Parallel()

	messages, err := Compile(SurfaceSpec{
		Components: []ComponentSpec{
			{ID: RootID, Kind: "Card", Child: "inner"},
			{ID: "inner", Kind: "Text", Properties: map[string]any{"text": "Inside"}},
		},
	})
	require.NoError(t, err)

	update, ok := messages[1].(*UpdateComponents)
	require.True(t, ok, "second message must be updateComponents")
	require.NotNil(t, update.Components[0].Child)
	require.Equal(t, "inner", *update.Components[0].Child)
	require.Nil(t, update.Components[0].Children)
}

func TestCompileRejectsInvalidSpecs(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		spec SurfaceSpec
	}{
		{
			name: "no root",
			spec: SurfaceSpec{Components: []ComponentSpec{{ID: "a", Kind: "Text"}}},
		},
		{
			name: "no components",
			spec: SurfaceSpec{},
		},
		{
			name: "dangling reference",
			spec: SurfaceSpec{Components: []ComponentSpec{
				{ID: RootID, Kind: "Column", Children: []string{"ghost"}},
			}},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			messages, err := Compile(tc.spec)
			require.Nil(t, messages)
			require.ErrorIs(t, err, ErrValidation)
		})
	}
}
