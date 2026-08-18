package a2ui

import (
	"fmt"
	"strconv"
	"strings"
	"testing"
)

// benchSurface is a mid-sized realistic surface: a settings page with a
// title and twenty labeled rows, each with a button.
func benchSurface() SurfaceSpec {
	children := make([]string, 0, 21)
	children = append(children, "title")

	components := make([]ComponentSpec, 0, 82)
	components = append(components,
		ComponentSpec{ID: RootID, Kind: "Column", Children: children},
		ComponentSpec{ID: "title", Kind: "Text", Properties: map[string]any{"text": "Settings", "variant": "h1"}},
	)

	for i := range 20 {
		row := fmt.Sprintf("row-%d", i)
		children = append(children, row)

		components = append(
			components,
			ComponentSpec{
				ID: row, Kind: "Row",
				Children: []string{row + "-label", row + "-button"},
			},
			ComponentSpec{
				ID:         row + "-label",
				Kind:       "Text",
				Properties: map[string]any{"text": "Row " + strconv.Itoa(i)},
			},
			ComponentSpec{
				ID: row + "-button", Kind: "Button", Child: row + "-button-label",
				Properties: map[string]any{"action": map[string]any{"event": map[string]any{"name": "clicked"}}},
			},
			ComponentSpec{ID: row + "-button-label", Kind: "Text", Properties: map[string]any{"text": "Go"}},
		)
	}

	return SurfaceSpec{Components: components, DataModel: map[string]any{"rows": 20}}
}

func BenchmarkMarshalJSONL(b *testing.B) {
	messages, err := Compile(benchSurface())
	if err != nil {
		b.Fatal(err)
	}

	b.ReportAllocs()
	b.ResetTimer()

	for range b.N {
		if _, err := MarshalJSONL(messages); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkValidate(b *testing.B) {
	messages, err := Compile(benchSurface())
	if err != nil {
		b.Fatal(err)
	}

	b.ReportAllocs()
	b.ResetTimer()

	for range b.N {
		if err := Validate(messages); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkUnmarshalMessage(b *testing.B) {
	messages, err := Compile(benchSurface())
	if err != nil {
		b.Fatal(err)
	}

	encoded, err := MarshalJSONL(messages)
	if err != nil {
		b.Fatal(err)
	}

	lines := strings.Split(string(encoded), "\n")

	b.ReportAllocs()
	b.ResetTimer()

	for range b.N {
		for _, line := range lines {
			if _, err := UnmarshalMessage([]byte(line)); err != nil {
				b.Fatal(err)
			}
		}
	}
}

func BenchmarkDecompile(b *testing.B) {
	messages, err := Compile(benchSurface())
	if err != nil {
		b.Fatal(err)
	}

	b.ReportAllocs()
	b.ResetTimer()

	for range b.N {
		if _, err := Decompile(messages); err != nil {
			b.Fatal(err)
		}
	}
}

func ExampleCompile() {
	spec := SurfaceSpec{
		Components: []ComponentSpec{
			{ID: RootID, Kind: "Column", Children: []string{"title"}},
			{ID: "title", Kind: "Text", Properties: map[string]any{"text": "Hello, A2UI"}},
		},
	}

	messages, err := Compile(spec)
	if err != nil {
		panic(err)
	}

	encoded, err := MarshalJSONL(messages)
	if err != nil {
		panic(err)
	}

	fmt.Println(string(encoded))
	// Output:
	// {"version":"v0.9.1","createSurface":{"surfaceId":"main","catalogId":"https://a2ui.org/specification/v0_9/catalogs/basic/catalog.json"}}
	// {"version":"v0.9.1","updateComponents":{"surfaceId":"main","components":[{"children":["title"],"component":"Column","id":"root"},{"component":"Text","id":"title","text":"Hello, A2UI"}]}}
}

func ExampleDecompile() {
	spec := SurfaceSpec{
		Components: []ComponentSpec{
			{ID: RootID, Kind: "Column", Children: []string{"title"}},
			{ID: "title", Kind: "Text", Properties: map[string]any{"text": "Hello, A2UI"}},
		},
	}

	messages, err := Compile(spec)
	if err != nil {
		panic(err)
	}

	roundTripped, err := Decompile(messages)
	if err != nil {
		panic(err)
	}

	fmt.Println(roundTripped.Components[1].Properties["text"])
	// Output: Hello, A2UI
}
