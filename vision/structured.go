package vision

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"

	"charm.land/fantasy"
	"charm.land/fantasy/schema"
)

// AnalyzeStructured sends images to the agent and returns a typed structured response.
// The response type T must be a struct with json tags.
//
// Example:
//
//	type UIAnalysis struct {
//	    Layout      string   `json:"layout" description:"Overall layout description"`
//	    Issues      []string `json:"issues" description:"List of UI issues found"`
//	    Suggestions []string `json:"suggestions" description:"Improvement suggestions"`
//	}
//
//	result, err := agent.AnalyzeStructured[UIAnalysis](ctx, "Analyze this UI", img)
//	fmt.Println(result.Object.Layout)
func AnalyzeStructured[T any](
	ctx context.Context,
	agent *VisionAgent,
	prompt string,
	images ...*ImageSource,
) (*fantasy.ObjectResult[T], error) {
	if prompt == "" {
		return nil, ErrEmptyPrompt
	}
	validImages := filterValidImages(images)
	if len(validImages) == 0 {
		return nil, ErrNoImages
	}

	ctx, cancel := agent.withTimeout(ctx)
	defer cancel()

	files := toFileParts(validImages)

	var zero T
	schemaDef := schema.Generate(reflect.TypeOf(zero))

	call := fantasy.ObjectCall{
		Prompt:            appendSystemAndPrompt(agent.config.SystemPrompt, prompt, files),
		Schema:            schemaDef,
		SchemaName:        reflect.TypeOf(zero).Name(),
		SchemaDescription: "Structured analysis result for " + reflect.TypeOf(zero).Name(),
	}

	if agent.config.MaxOutputTokens > 0 {
		call.MaxOutputTokens = &agent.config.MaxOutputTokens
	}
	if agent.config.Temperature != 0 {
		call.Temperature = &agent.config.Temperature
	}

	result, err := agent.config.Model.GenerateObject(ctx, call)
	if err != nil {
		return nil, fmt.Errorf("vision agent structured generate: %w", err)
	}

	var typedResult T
	if result.Object != nil {
		if err := unmarshalToType(result.Object, &typedResult); err != nil {
			return nil, fmt.Errorf("vision agent unmarshal result: %w", err)
		}
	}

	return &fantasy.ObjectResult[T]{
		Object:           typedResult,
		RawText:          result.RawText,
		Usage:            result.Usage,
		FinishReason:     result.FinishReason,
		Warnings:         result.Warnings,
		ProviderMetadata: result.ProviderMetadata,
	}, nil
}

// appendSystemAndPrompt builds a prompt with optional system message and files.
func appendSystemAndPrompt(
	systemPrompt, userPrompt string,
	files []fantasy.FilePart,
) fantasy.Prompt {
	var prompt fantasy.Prompt
	if systemPrompt != "" {
		prompt = append(prompt, fantasy.NewSystemMessage(systemPrompt))
	}
	prompt = append(prompt, fantasy.NewUserMessage(userPrompt, files...))
	return prompt
}

// unmarshalToType converts an object to a specific type using JSON round-tripping.
func unmarshalToType(obj, target any) error {
	jsonBytes, err := json.Marshal(obj)
	if err != nil {
		return fmt.Errorf("marshal object: %w", err)
	}
	if err := json.Unmarshal(jsonBytes, target); err != nil {
		return fmt.Errorf("unmarshal into target: %w", err)
	}
	return nil
}
