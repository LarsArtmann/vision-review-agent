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
	return withPrepared(agent, ctx, prompt, images, func(prep *preparedRequest) (*fantasy.ObjectResult[T], error) {
		call := buildObjectCall[T](agent, prompt, prep.files)

		result, err := generateObject(prep.ctx, agent, call)
		if err != nil {
			classified := classifyModelErr("vision agent structured generate", prompt, err)
			agent.config.Hooks.fireError(prep.ctx, classified)
			return nil, classified
		}

		var typedResult T
		if result.Object != nil {
			if err := visionutil.UnmarshalToType(result.Object, &typedResult); err != nil {
				parseErr := apperrors.Wrap(
					apperrors.KindStructuredParse,
					"vision agent unmarshal result",
					prompt,
					err,
				)
				agent.config.Hooks.fireError(prep.ctx, parseErr)
				return nil, parseErr
			}
		}

		finalResult := &fantasy.ObjectResult[T]{
			Object:           typedResult,
			RawText:          result.RawText,
			Usage:            result.Usage,
			FinishReason:     result.FinishReason,
			Warnings:         result.Warnings,
			ProviderMetadata: result.ProviderMetadata,
		}
		// Structured methods have no *fantasy.AgentResult, so the synthesized
		// AnalyzeResult carries only Text/Usage; RawResponse is intentionally nil
		// (see AnalyzeResult.RawResponse doc). Hooks must nil-check it.
		agent.config.Hooks.fireFinish(prep.ctx, &AnalyzeResult{
			Text:  result.RawText,
			Usage: result.Usage,
		})
		return finalResult, nil
	})
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
	return withPrepared(agent, ctx, prompt, images, func(prep *preparedRequest) (*fantasy.ObjectResult[T], error) {
		call := buildObjectCall[T](agent, prompt, prep.files)

		stream, err := agent.config.Model.StreamObject(prep.ctx, call)
		if err != nil {
			classified := classifyModelErr("vision agent structured stream", prompt, err)
			agent.config.Hooks.fireError(prep.ctx, classified)
			return nil, classified
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
					classified := classifyModelErr("vision agent structured stream", prompt, part.Error)
					agent.config.Hooks.fireError(prep.ctx, classified)
					return nil, classified
				}
			case fantasy.ObjectStreamPartTypeFinish:
				usage = part.Usage
				finishReason = part.FinishReason
				if part.Object != nil {
					_ = visionutil.UnmarshalToType(part.Object, &finalObject)
				}
			}
		}

		finalResult := &fantasy.ObjectResult[T]{
			Object:       finalObject,
			RawText:      rawText,
			Usage:        usage,
			FinishReason: finishReason,
		}
		// Synthesized AnalyzeResult: RawResponse is nil (see AnalyzeResult.RawResponse doc).
		agent.config.Hooks.fireFinish(prep.ctx, &AnalyzeResult{
			Text:  rawText,
			Usage: usage,
		})
		return finalResult, nil
	})
}

// buildObjectCall constructs a fantasy.ObjectCall for a typed structured
// analysis, embedding the JSON schema for T and forwarding optional model
// parameters. Shared by AnalyzeStructured and AnalyzeStructuredStream.
func buildObjectCall[T any](agent *Agent, prompt string, files []fantasy.FilePart) fantasy.ObjectCall {
	t := reflect.TypeOf(*new(T))
	call := fantasy.ObjectCall{
		Prompt: visionutil.AppendSystemAndPrompt(
			agent.config.SystemPrompt,
			prompt,
			files,
		),
		Schema:            schema.Generate(t),
		SchemaName:        t.Name(),
		SchemaDescription: "Structured analysis result for " + t.Name(),
	}
	p := agent.config.optionalParams()
	call.MaxOutputTokens = p.maxOutputTokens
	call.Temperature = p.temperature
	call.TopP = p.topP
	call.TopK = p.topK
	call.PresencePenalty = p.presencePenalty
	call.FrequencyPenalty = p.frequencyPenalty
	return call
}

// generateObject invokes the model's GenerateObject, applying Config.Retry when
// configured. Package-level because AnalyzeStructured is generic and lives
// outside a method receiver. Classification stays in the caller.
func generateObject(
	ctx context.Context,
	agent *Agent,
	call fantasy.ObjectCall,
) (*fantasy.ObjectResponse, error) {
	if agent.config.Retry == nil {
		return agent.config.Model.GenerateObject(ctx, call)
	}

	return WithRetry(ctx, *agent.config.Retry, func(ctx context.Context) (*fantasy.ObjectResponse, error) {
		return agent.config.Model.GenerateObject(ctx, call)
	})
}
