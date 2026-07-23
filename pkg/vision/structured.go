package vision

import (
	"context"
	"reflect"

	"charm.land/fantasy"
	"charm.land/fantasy/schema"
	"github.com/larsartmann/vision-review-agent/internal/visionutil"
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
	agent *Agent,
	prompt string,
	images ...*ImageSource,
) (*fantasy.ObjectResult[T], error) {
	validImages, err := validateAnalyzeInput(prompt, images)
	if err != nil {
		return nil, err
	}

	ctx, cancel := agent.withTimeout(ctx)
	defer cancel()

	files := toFileParts(validImages)

	var zero T
	schemaDef := schema.Generate(reflect.TypeOf(zero))

	call := fantasy.ObjectCall{
		Prompt: visionutil.AppendSystemAndPrompt(
			agent.config.SystemPrompt,
			prompt,
			files,
		),
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
		return nil, wrapWithPrompt("vision agent structured generate", prompt, err)
	}

	var typedResult T
	if result.Object != nil {
		if err := visionutil.UnmarshalToType(result.Object, &typedResult); err != nil {
			return nil, wrapWithPrompt("vision agent unmarshal result", prompt, err)
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
