package vision

import (
	"context"
	"reflect"

	"charm.land/fantasy"
	"charm.land/fantasy/schema"
	"github.com/larsartmann/vision-review-agent/internal/visionutil"
	apperrors "github.com/larsartmann/vision-review-agent/pkg/errors"
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
	if agent.config.TopP > 0 {
		call.TopP = &agent.config.TopP
	}
	if agent.config.TopK > 0 {
		call.TopK = &agent.config.TopK
	}
	if agent.config.PresencePenalty != 0 {
		call.PresencePenalty = &agent.config.PresencePenalty
	}
	if agent.config.FrequencyPenalty != 0 {
		call.FrequencyPenalty = &agent.config.FrequencyPenalty
	}

	result, err := agent.config.Model.GenerateObject(ctx, call)
	if err != nil {
		return nil, classifyModelErr("vision agent structured generate", prompt, err)
	}

	var typedResult T
	if result.Object != nil {
		if err := visionutil.UnmarshalToType(result.Object, &typedResult); err != nil {
			return nil, apperrors.Wrap(
				apperrors.KindStructuredParse,
				"vision agent unmarshal result",
				prompt,
				err,
			)
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

// AnalyzeStructuredStream sends images to the agent and streams typed structured
// responses as they arrive. The onObject callback is called for each partial
// object the model emits, allowing real-time UI updates.
//
// The response type T must be a struct with json tags.
//
// Example:
//
//	result, err := vision.AnalyzeStructuredStream[UIReview](
//	    ctx, agent, "Review this UI",
//	    func(partial UIReview) { fmt.Printf("partial: %+v\n", partial) },
//	    img,
//	)
//	fmt.Println(result.Object.Score)
func AnalyzeStructuredStream[T any](
	ctx context.Context,
	agent *Agent,
	prompt string,
	onObject func(partial T),
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
	if agent.config.TopP > 0 {
		call.TopP = &agent.config.TopP
	}
	if agent.config.TopK > 0 {
		call.TopK = &agent.config.TopK
	}
	if agent.config.PresencePenalty != 0 {
		call.PresencePenalty = &agent.config.PresencePenalty
	}
	if agent.config.FrequencyPenalty != 0 {
		call.FrequencyPenalty = &agent.config.FrequencyPenalty
	}

	stream, err := agent.config.Model.StreamObject(ctx, call)
	if err != nil {
		return nil, classifyModelErr("vision agent structured stream", prompt, err)
	}

	var (
		finalObject  T
		rawText      string
		usage        fantasy.Usage
		finishReason fantasy.FinishReason
	)

	for part := range stream {
		switch part.Type {
		case fantasy.ObjectStreamPartTypeObject:
			if onObject != nil && part.Object != nil {
				var partial T
				if unmarshalErr := visionutil.UnmarshalToType(part.Object, &partial); unmarshalErr == nil {
					onObject(partial)
					finalObject = partial
				}
			}
		case fantasy.ObjectStreamPartTypeTextDelta:
			rawText += part.Delta
		case fantasy.ObjectStreamPartTypeError:
			if part.Error != nil {
				return nil, classifyModelErr("vision agent structured stream", prompt, part.Error)
			}
		case fantasy.ObjectStreamPartTypeFinish:
			usage = part.Usage
			finishReason = part.FinishReason
			if part.Object != nil {
				_ = visionutil.UnmarshalToType(part.Object, &finalObject)
			}
		}
	}

	return &fantasy.ObjectResult[T]{
		Object:       finalObject,
		RawText:      rawText,
		Usage:        usage,
		FinishReason: finishReason,
	}, nil
}
