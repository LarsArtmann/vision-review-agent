package a2ui_test

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"charm.land/fantasy"
	"github.com/larsartmann/vision-review-agent/pkg/vision"
	"github.com/larsartmann/vision-review-agent/pkg/vision/a2ui"
)

// fakeModel is a minimal fantasy.LanguageModel whose GenerateObject returns a
// canned object (the SurfaceSpec the "model" produced) or an error.
type fakeModel struct {
	object         map[string]any
	err            error
	called         int
	promptSeen     fantasy.Prompt
	schemaSeen     bool
	schemaNameSeen string
}

func (m *fakeModel) Generate(_ context.Context, _ fantasy.Call) (*fantasy.Response, error) {
	return &fantasy.Response{
		Content:      []fantasy.Content{fantasy.TextContent{Text: "unused"}},
		FinishReason: fantasy.FinishReasonStop,
	}, nil
}

func (m *fakeModel) Stream(_ context.Context, _ fantasy.Call) (fantasy.StreamResponse, error) {
	return nil, errors.New("a2ui fake: streaming not supported")
}

func (m *fakeModel) GenerateObject(_ context.Context, in fantasy.ObjectCall) (*fantasy.ObjectResponse, error) {
	m.called++
	m.promptSeen = in.Prompt
	m.schemaSeen = in.Schema.Type != ""
	m.schemaNameSeen = in.SchemaName

	if m.err != nil {
		return nil, m.err
	}

	return &fantasy.ObjectResponse{
		Object:       m.object,
		FinishReason: fantasy.FinishReasonStop,
		Usage:        fantasy.Usage{TotalTokens: 42},
	}, nil
}

func (m *fakeModel) StreamObject(
	_ context.Context,
	_ fantasy.ObjectCall,
) (fantasy.ObjectStreamResponse, error) {
	return nil, errors.New("a2ui fake: object streaming not supported")
}

func (m *fakeModel) Provider() string { return "fake" }
func (m *fakeModel) Model() string    { return "fake-vision-model" }

// validSpecObject is a well-formed SurfaceSpec as raw JSON-shaped maps, the
// way a model would return it.
func validSpecObject() map[string]any {
	return map[string]any{
		"surfaceId": "review-dashboard",
		"catalogId": a2ui.DefaultCatalogID,
		"components": []any{
			map[string]any{
				"id":        a2ui.RootID,
				"component": "Column",
				"children":  []any{"title", "open-btn"},
			},
			map[string]any{
				"id":         "title",
				"component":  "Text",
				"properties": map[string]any{"text": "Latest review", "variant": "h1"},
			},
			map[string]any{
				"id":        "open-btn",
				"component": "Button",
				"child":     "open-label",
				"properties": map[string]any{
					"action": map[string]any{"event": map[string]any{"name": "review.opened"}},
				},
			},
			map[string]any{
				"id":         "open-label",
				"component":  "Text",
				"properties": map[string]any{"text": "Open review"},
			},
		},
		"dataModel": map[string]any{"score": 8},
	}
}

// newAgent builds a vision agent over the fake model.
func newAgent(model fantasy.LanguageModel) *vision.Agent {
	agent, err := vision.NewAgent(vision.Config{Model: model})
	if err != nil {
		panic(fmt.Sprintf("a2ui fake: build agent: %v", err))
	}

	return agent
}

// promptText flattens a fantasy prompt into its text parts for assertions.
func promptText(prompt fantasy.Prompt) string {
	var text strings.Builder

	for _, msg := range prompt {
		for _, part := range msg.Content {
			if textPart, ok := part.(fantasy.TextPart); ok {
				text.WriteString(textPart.Text)
				text.WriteByte('\n')
			}
		}
	}

	return text.String()
}

// testImage builds a minimal PNG-header image source for the analysis path.
func testImage() *vision.ImageSource {
	img, err := vision.NewImageSource(
		[]byte{
			0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, // PNG magic
		},
		vision.MediaTypePNG,
		"shot.png",
	)
	if err != nil {
		panic(fmt.Sprintf("a2ui fake: build image: %v", err))
	}

	return img
}
