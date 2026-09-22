package a2ui

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestComponentRoundtrip(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		wire string
		want Component
	}{
		{
			name: "text with variant",
			wire: `{"id":"title","component":"Text","text":"Hello","variant":"h1"}`,
			want: Component{
				ID:    "title",
				Kind:  "Text",
				Props: map[string]any{"text": "Hello", "variant": "h1"},
			},
		},
		{
			name: "column with static children and align",
			wire: `{"id":"root","component":"Column","children":["a","b"],"align":"center"}`,
			want: Component{
				ID:       "root",
				Kind:     "Column",
				Children: StaticChildren("a", "b"),
				Props:    map[string]any{"align": "center"},
			},
		},
		{
			name: "card with single child",
			wire: `{"id":"card","component":"Card","child":"content"}`,
			want: Component{
				ID:    "card",
				Kind:  "Card",
				Child: new("content"),
			},
		},
		{
			name: "dynamic children",
			wire: `{"id":"list","component":"List","children":{"componentId":"row-tpl","path":"/items"}}`,
			want: Component{
				ID:       "list",
				Kind:     "List",
				Children: DynamicChildrenOf("row-tpl", "/items"),
			},
		},
		{
			name: "accessibility labels",
			wire: `{"id":"btn","component":"Button","child":"lbl","accessibility":{"label":"Submit"}}`,
			want: Component{
				ID:            "btn",
				Kind:          "Button",
				Child:         new("lbl"),
				Accessibility: &Accessibility{Label: "Submit"},
			},
		},
		{
			name: "no properties",
			wire: `{"id":"sep","component":"Divider"}`,
			want: Component{ID: "sep", Kind: "Divider"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var decoded Component
			require.NoError(t, json.Unmarshal([]byte(tc.wire), &decoded))
			require.Equal(t, tc.want, decoded)

			encoded, err := json.Marshal(decoded)
			require.NoError(t, err)

			var redecoded Component
			require.NoError(t, json.Unmarshal(encoded, &redecoded))
			require.Equal(t, decoded, redecoded)
		})
	}
}

func TestComponentMarshalWireShape(t *testing.T) {
	t.Parallel()

	encoded, err := json.Marshal(NewButton("submit", "submit-label", "form.submitted"))
	require.NoError(t, err)

	var fields map[string]any
	require.NoError(t, json.Unmarshal(encoded, &fields))

	require.Equal(t, "submit", fields["id"])
	require.Equal(t, "Button", fields["component"])
	require.Equal(t, "submit-label", fields["child"])
	require.Equal(t,
		map[string]any{"event": map[string]any{"name": "form.submitted"}},
		fields["action"],
	)
}

func TestBuilders(t *testing.T) {
	t.Parallel()

	text := NewText("title", "Welcome", TextH1)
	require.Equal(t, "Text", text.Kind)
	require.Equal(t, map[string]any{"text": "Welcome", "variant": "h1"}, text.Props)

	body := NewText("p", "Body copy", "")
	require.NotContains(t, body.Props, "variant")

	column := NewColumn("col", "a", "b")
	require.Equal(t, StaticChildren("a", "b"), column.Children)

	row := NewRow("row", "x")
	require.Equal(t, []string{"x"}, row.Children.Static)

	card := NewCard("card", "inner")
	require.Equal(t, "inner", *card.Child)

	divider := NewDivider("div")
	require.Empty(t, divider.Props)

	icon := NewIcon("star", "star")
	require.Equal(t, map[string]any{"name": "star"}, icon.Props)

	image := NewImage("shot", "https://example.com/a.png", "A screenshot")
	require.Equal(t, map[string]any{"url": "https://example.com/a.png", "description": "A screenshot"}, image.Props)
}

func TestBindLiteral(t *testing.T) {
	t.Parallel()

	require.Equal(t, "Hello", Literal("Hello"))
	require.Equal(t, map[string]any{"path": "/message"}, Bind("/message"))
}
