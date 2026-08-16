// Package vision provides a simple SDK for building AI agents with vision capabilities.
// It wraps charm.land/fantasy to enable screenshot/image analysis with configurable
// system prompts and supports multiple LLM providers through a unified API.
//
// Basic usage:
//
//	provider, _ := openai.New(openai.WithAPIKey(key))
//	model, _ := provider.LanguageModel(ctx, "gpt-4o")
//
//	agent := vision.NewAgent(vision.Config{
//	    SystemPrompt: "Describe what you see in the image.",
//	    Model:        model,
//	})
//
//	img, _ := vision.LoadImageFromFile("screenshot.png")
//	result, _ := agent.Analyze(ctx, "What UI issues do you see?", img)
//	fmt.Println(result.Text)
package vision

import (
	"context"
	"fmt"
	"strings"
	"time"

	"charm.land/fantasy"
)

// Config holds the configuration for a VisionAgent.
type Config struct {
	// SystemPrompt is the system prompt that defines the agent's behavior.
	// Optional; if empty, the model's default behavior is used.
	SystemPrompt string

	// MaxOutputTokens limits the response length. Zero means no limit.
	MaxOutputTokens int64

	// Temperature controls randomness (0.0-2.0). Zero means model default.
	Temperature float64

	// TopP controls nucleus sampling (0.0-1.0). Zero means model default.
	TopP float64

	// TopK limits sampling to the top K tokens. Zero means model default.
	TopK int64

	// PresencePenalty penalizes tokens that have appeared (typically -2.0 to 2.0).
	// Zero means model default.
	PresencePenalty float64

	// FrequencyPenalty penalizes tokens based on frequency (typically -2.0 to 2.0).
	// Zero means model default.
	FrequencyPenalty float64

	// Model is the language model to use. Must support vision.
	// Required.
	Model fantasy.LanguageModel

	// MaxRetries sets the maximum number of retries on transient errors at the
	// provider/HTTP layer (forwarded to fantasy via WithMaxRetries). Zero
	// disables HTTP-layer retries entirely (errors surface immediately). For
	// richer control (backoff, jitter, visibility), see Retry below; the two
	// compose as layered retry if both are set.
	MaxRetries int

	// RequestTimeout sets a per-request timeout. Zero means no timeout.
	RequestTimeout time.Duration

	// Retry, when non-nil, enables vision-layer automatic retry of transient
	// ([IsRetryable]) failures across the non-streaming analysis methods —
	// Analyze, AnalyzeConversation, AnalyzeStructured, and AnalyzeBatch (which
	// retries per image via Analyze). Streaming methods (AnalyzeStream,
	// AnalyzeConversationStream, AnalyzeStructuredStream) do NOT auto-retry,
	// because retrying a partial stream has ambiguous delta semantics; wrap
	// those calls in WithRetry manually if needed.
	//
	// This is distinct from MaxRetries: MaxRetries retries individual HTTP
	// requests inside fantasy; Retry retries the whole model invocation at the
	// vision layer with a configurable RetryConfig (backoff + jitter). For most
	// use cases pick one; set both only when you want layered retry.
	Retry *RetryConfig

	// Hooks defines optional lifecycle callbacks for observability (logging, metrics).
	// All callbacks are optional; nil hooks are safe.
	Hooks Hooks

	// Tools defines callable tools the model may invoke during multi-step
	// analysis (e.g. fetch a guideline, measure contrast ratio). Optional.
	Tools []fantasy.AgentTool

	// ToolChoice controls whether and how the model uses tools (none, auto,
	// or a specific tool). Optional; empty means the provider default.
	ToolChoice fantasy.ToolChoice

	// StopConditions ends the agent loop early. Compose helpers from fantasy
	// such as StepCountIs, HasToolCall, or MaxTokensUsed. Optional.
	StopConditions []fantasy.StopCondition

	// PrepareStep intercepts each agent step, allowing per-step mutation of
	// the model, system prompt, or tools before invocation. Optional.
	PrepareStep fantasy.PrepareStepFunction

	// Headers sets extra HTTP headers sent on every provider request. Optional.
	Headers map[string]string

	// UserAgent overrides the default HTTP User-Agent for provider requests.
	// Optional.
	UserAgent string

	// Preprocess controls automatic image preprocessing (resize, compress)
	// applied to every image before analysis. When nil (the default), images
	// are sent as-is. Set MaxDimension to auto-resize large images:
	//
	//	agent, _ := vision.NewAgent(vision.Config{
	//	    Model: model,
	//	    Preprocess: &vision.PreprocessConfig{MaxDimension: 1568},
	//	})
	Preprocess *PreprocessConfig

	// ModelInfo holds optional catalog metadata about the model. When set,
	// its DefaultMaxTokens is used if MaxOutputTokens is 0, and its pricing
	// fields enable cost tracking via CostTracker. Explicit Config fields
	// always take priority over ModelInfo defaults.
	ModelInfo *ModelInfo
}

// Validate checks the configuration for errors. When a constraint is violated,
// the returned error wraps the corresponding sentinel (so errors.Is still
// matches) and includes the offending value for quick diagnosis:
//
//	vision agent: temperature must be between 0.0 and 2.0: got 3.50
func (c *Config) Validate() error {
	if c.Model == nil {
		return ErrNoModel
	}

	if c.Temperature < 0 || c.Temperature > 2.0 {
		return fmt.Errorf("%w: got %.2f, want [0.0, 2.0]", ErrInvalidTemperature, c.Temperature)
	}

	if c.MaxOutputTokens < 0 {
		return fmt.Errorf("%w: got %d, want >= 0", ErrInvalidMaxTokens, c.MaxOutputTokens)
	}

	if c.TopP < 0 || c.TopP > 1.0 {
		return fmt.Errorf("%w: got %.2f, want [0.0, 1.0]", ErrInvalidTopP, c.TopP)
	}

	if c.TopK < 0 {
		return fmt.Errorf("%w: got %d, want >= 0", ErrInvalidTopK, c.TopK)
	}

	if c.PresencePenalty < -2.0 || c.PresencePenalty > 2.0 {
		return fmt.Errorf("%w: got %.2f, want [-2.0, 2.0]", ErrInvalidPresencePenalty, c.PresencePenalty)
	}

	if c.FrequencyPenalty < -2.0 || c.FrequencyPenalty > 2.0 {
		return fmt.Errorf("%w: got %.2f, want [-2.0, 2.0]", ErrInvalidFrequencyPenalty, c.FrequencyPenalty)
	}

	return nil
}

// Analyzer is the interface for AI agents capable of analyzing images/screenshots.
// Consumers can use this interface to mock the agent in their own tests.
type Analyzer interface {
	Analyze(
		ctx context.Context,
		prompt string,
		images ...*ImageSource,
	) (*AnalyzeResult, error)
	AnalyzeStream(
		ctx context.Context,
		prompt string,
		onText func(text string) error,
		images ...*ImageSource,
	) (*AnalyzeResult, error)
}

// Compile-time check: Agent implements Analyzer.
var _ Analyzer = (*Agent)(nil)

// Agent is an AI agent capable of analyzing images/screenshots.
type Agent struct {
	config Config
	agent  fantasy.Agent
}

// VisionAgent is an alias for Agent for backwards compatibility.
//
// Deprecated: Use Agent instead.
type VisionAgent = Agent

// NewAgent creates a new Agent with the given configuration.
// Returns an error if the configuration is invalid.
func NewAgent(config Config) (*Agent, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	config.applyModelInfoDefaults()

	opts := []fantasy.AgentOption{
		fantasy.WithSystemPrompt(config.SystemPrompt),
	}

	if config.MaxOutputTokens > 0 {
		opts = append(opts, fantasy.WithMaxOutputTokens(config.MaxOutputTokens))
	}

	if config.Temperature != 0 {
		opts = append(opts, fantasy.WithTemperature(config.Temperature))
	}

	// Always forward MaxRetries so that zero explicitly disables fantasy's
	// HTTP-layer retry (its default of 3 attempts adds 5+10+20s of blocking
	// backoff, which is a surprising default for callers who set nothing).
	opts = append(opts, fantasy.WithMaxRetries(config.MaxRetries))

	if config.TopP > 0 {
		opts = append(opts, fantasy.WithTopP(config.TopP))
	}

	if config.TopK > 0 {
		opts = append(opts, fantasy.WithTopK(config.TopK))
	}

	if config.PresencePenalty != 0 {
		opts = append(opts, fantasy.WithPresencePenalty(config.PresencePenalty))
	}

	if config.FrequencyPenalty != 0 {
		opts = append(opts, fantasy.WithFrequencyPenalty(config.FrequencyPenalty))
	}

	if len(config.Tools) > 0 {
		opts = append(opts, fantasy.WithTools(config.Tools...))
	}

	if config.ToolChoice != "" {
		opts = append(opts, fantasy.WithToolChoice(config.ToolChoice))
	}

	if len(config.StopConditions) > 0 {
		opts = append(opts, fantasy.WithStopConditions(config.StopConditions...))
	}

	if config.PrepareStep != nil {
		opts = append(opts, fantasy.WithPrepareStep(config.PrepareStep))
	}

	if len(config.Headers) > 0 {
		opts = append(opts, fantasy.WithHeaders(config.Headers))
	}

	if config.UserAgent != "" {
		opts = append(opts, fantasy.WithUserAgent(config.UserAgent))
	}

	return &Agent{
		config: config,
		agent:  fantasy.NewAgent(config.Model, opts...),
	}, nil
}

// AnalyzeResult contains the result of an image analysis.
type AnalyzeResult struct {
	// Text is the agent's textual analysis/response.
	Text string

	// Usage contains token usage statistics.
	Usage fantasy.Usage

	// RawResponse contains the full response from the model.
	//
	// It is populated by Analyze / AnalyzeStream / AnalyzeConversation /
	// AnalyzeConversationStream. It is nil for the synthesized AnalyzeResult
	// passed to Hooks.OnFinish by AnalyzeStructured and AnalyzeStructuredStream
	// (those methods return a *fantasy.ObjectResult, which has no AgentResult).
	// OnFinish hooks must nil-check this field before dereferencing it.
	RawResponse *fantasy.AgentResult
}

// String returns a human-readable summary of the analysis result.
func (r AnalyzeResult) String() string {
	return fmt.Sprintf("AnalyzeResult{Text: %q, Usage: %s}", r.Text, r.Usage)
}

// Analyze sends one or more images to the agent along with a prompt and returns the analysis.
// Returns ErrEmptyPrompt if prompt is empty, ErrNoImages if no images are provided.
func (va *Agent) Analyze(
	ctx context.Context,
	prompt string,
	images ...*ImageSource,
) (*AnalyzeResult, error) {
	return withPrepared(va, ctx, prompt, images, func(prep *preparedRequest) (*AnalyzeResult, error) {
		call := va.buildAgentCall(prompt, prep.files, nil)

		result, err := va.generate(prep.ctx, call)
		if err != nil {
			classified := classifyModelErr("vision agent generate", prompt, err)
			va.config.Hooks.fireError(prep.ctx, classified)

			return nil, classified
		}

		return va.finishResult(prep.ctx, result.Response.Content.Text(), result), nil
	})
}

// AnalyzeStream sends images to the agent and streams the response.
// The onText callback is called for each chunk of text received.
func (va *Agent) AnalyzeStream(
	ctx context.Context,
	prompt string,
	onText func(text string) error,
	images ...*ImageSource,
) (*AnalyzeResult, error) {
	return withPrepared(va, ctx, prompt, images, func(prep *preparedRequest) (*AnalyzeResult, error) {
		var builder strings.Builder

		streamCall := va.buildAgentStreamCall(prompt, prep.files, nil)
		streamCall.OnTextDelta = func(_, text string) error {
			builder.WriteString(text)

			if onText != nil {
				return onText(text)
			}

			return nil
		}

		result, err := va.agent.Stream(prep.ctx, streamCall)
		if err != nil {
			classified := classifyModelErr("vision agent stream", prompt, err)
			va.config.Hooks.fireError(prep.ctx, classified)

			return nil, classified
		}

		return va.finishResult(prep.ctx, builder.String(), result), nil
	})
}

// AnalyzeConversation analyzes images with the given prompt, incorporating
// conversation history for multi-turn interactions.
// The prompt and images represent the current turn; the conversation provides
// prior context.
//
// After a successful call, persist the turn by calling:
//
//	conv.AddUserMessage(prompt, images...)
//	conv.AddAssistantMessage(result.Text)
func (va *Agent) AnalyzeConversation(
	ctx context.Context,
	conv *Conversation,
	prompt string,
	images ...*ImageSource,
) (*AnalyzeResult, error) {
	return withPrepared(va, ctx, prompt, images, func(prep *preparedRequest) (*AnalyzeResult, error) {
		call := va.buildAgentCall(prompt, prep.files, conv.Messages())

		result, err := va.generate(prep.ctx, call)
		if err != nil {
			classified := classifyModelErr("vision agent conversation generate", prompt, err)
			va.config.Hooks.fireError(prep.ctx, classified)

			return nil, classified
		}

		return va.finishResult(prep.ctx, result.Response.Content.Text(), result), nil
	})
}

// AnalyzeConversationStream streams analysis with conversation history for multi-turn interactions.
// The onText callback is called for each chunk of text received.
func (va *Agent) AnalyzeConversationStream(
	ctx context.Context,
	conv *Conversation,
	prompt string,
	onText func(text string) error,
	images ...*ImageSource,
) (*AnalyzeResult, error) {
	return withPrepared(va, ctx, prompt, images, func(prep *preparedRequest) (*AnalyzeResult, error) {
		var builder strings.Builder

		streamCall := va.buildAgentStreamCall(prompt, prep.files, conv.Messages())
		streamCall.OnTextDelta = func(_, text string) error {
			builder.WriteString(text)

			if onText != nil {
				return onText(text)
			}

			return nil
		}

		result, err := va.agent.Stream(prep.ctx, streamCall)
		if err != nil {
			classified := classifyModelErr("vision agent conversation stream", prompt, err)
			va.config.Hooks.fireError(prep.ctx, classified)

			return nil, classified
		}

		return va.finishResult(prep.ctx, builder.String(), result), nil
	})
}

// preparedRequest bundles the validated inputs and derived context produced
// by prepare, so every analysis method shares the same prologue.
//
//nolint:containedctx // ctx is request-scoped, used immediately by the caller, not stored long-term
type preparedRequest struct {
	ctx    context.Context
	cancel context.CancelFunc
	files  []fantasy.FilePart
}

// prepare runs the shared prologue every analysis method performs: input
// validation, image preprocessing, hook firing, timeout setup, and FilePart
// conversion. The caller must defer prep.cancel().
func (va *Agent) prepare(
	ctx context.Context,
	prompt string,
	images ...*ImageSource,
) (*preparedRequest, error) {
	validImages, err := validateAnalyzeInput(prompt, images)
	if err != nil {
		return nil, err
	}

	validImages, err = va.preprocessImages(validImages)
	if err != nil {
		return nil, err
	}

	va.config.Hooks.fireStart(ctx, prompt, len(validImages))

	ctx, cancel := va.withTimeout(ctx)

	return &preparedRequest{
		ctx:    ctx,
		cancel: cancel,
		files:  toFileParts(validImages),
	}, nil
}

// finishResult builds an AnalyzeResult from a completed model response, fires
// the OnFinish hook, and returns the result. Shared by every non-streaming
// and streaming analysis method that terminates with a *fantasy.AgentResult.
func (va *Agent) finishResult(
	ctx context.Context,
	text string,
	rawResult *fantasy.AgentResult,
) *AnalyzeResult {
	result := &AnalyzeResult{
		Text:        text,
		Usage:       rawResult.TotalUsage,
		RawResponse: rawResult,
	}
	va.config.Hooks.fireFinish(ctx, result)

	return result
}

// withPrepared runs the shared prologue (validate → preprocess → fireStart →
// timeout → toFileParts) via prepare, then invokes fn with the prepared
// request, guaranteeing cancel() is deferred. It is the single owner of the
// "if prepare failed, return early" idiom that every analysis method needs, so
// the idiom appears once instead of six times.
//
// It is a free generic function (not a method) because Go does not permit
// type parameters on methods; AnalyzeStructured[T] / AnalyzeStructuredStream[T]
// instantiate it with *fantasy.ObjectResult[T].
func withPrepared[T any](
	va *Agent,
	ctx context.Context,
	prompt string,
	images []*ImageSource,
	fn func(*preparedRequest) (T, error),
) (T, error) {
	prep, err := va.prepare(ctx, prompt, images...)
	if err != nil {
		var zero T

		return zero, err
	}
	defer prep.cancel()

	return fn(prep)
}

// buildAgentCall constructs a fantasy.AgentCall with model parameters and optional history.
func (va *Agent) buildAgentCall(
	prompt string,
	files []fantasy.FilePart,
	messages []fantasy.Message,
) fantasy.AgentCall {
	call := fantasy.AgentCall{
		Prompt:   prompt,
		Files:    files,
		Messages: messages,
	}
	params := va.config.optionalParams()
	call.MaxOutputTokens = params.maxOutputTokens
	call.Temperature = params.temperature
	call.TopP = params.topP
	call.TopK = params.topK
	call.PresencePenalty = params.presencePenalty
	call.FrequencyPenalty = params.frequencyPenalty

	return call
}

// buildAgentStreamCall constructs a fantasy.AgentStreamCall with model parameters and optional history.
func (va *Agent) buildAgentStreamCall(
	prompt string,
	files []fantasy.FilePart,
	messages []fantasy.Message,
) fantasy.AgentStreamCall {
	call := fantasy.AgentStreamCall{
		Prompt:   prompt,
		Files:    files,
		Messages: messages,
	}
	params := va.config.optionalParams()
	call.MaxOutputTokens = params.maxOutputTokens
	call.Temperature = params.temperature
	call.TopP = params.topP
	call.TopK = params.topK
	call.PresencePenalty = params.presencePenalty
	call.FrequencyPenalty = params.frequencyPenalty

	return call
}

// optionalModelParams holds the optional model-generation parameters as
// pointers computed once from Config. A nil field means "use the provider's
// default". Centralising the zero-value checks here keeps the four call sites
// (AgentCall, AgentStreamCall, and the two ObjectCall builds in structured.go)
// from drifting out of sync.
type optionalModelParams struct {
	maxOutputTokens  *int64
	temperature      *float64
	topP             *float64
	topK             *int64
	presencePenalty  *float64
	frequencyPenalty *float64
}

// optionalParams returns the optional model parameters as pointers into c.
// Fields left at their zero value yield nil pointers so the provider applies
// its own default. The receiver is *Config so the returned pointers reference
// stable storage (the agent's config), matching the prior direct &field usage.
func (c *Config) optionalParams() optionalModelParams {
	var params optionalModelParams
	if c.MaxOutputTokens > 0 {
		params.maxOutputTokens = &c.MaxOutputTokens
	}

	if c.Temperature != 0 {
		params.temperature = &c.Temperature
	}

	if c.TopP > 0 {
		params.topP = &c.TopP
	}

	if c.TopK > 0 {
		params.topK = &c.TopK
	}

	if c.PresencePenalty != 0 {
		params.presencePenalty = &c.PresencePenalty
	}

	if c.FrequencyPenalty != 0 {
		params.frequencyPenalty = &c.FrequencyPenalty
	}

	return params
}

// generate invokes the agent's Generate, applying Config.Retry when configured.
// When Retry is nil it is a plain pass-through; otherwise the call is wrapped in
// WithRetry so transient (IsRetryable) failures are retried with backoff.
// Classification and hook firing stay in the caller so they happen once per
// logical request, not per attempt.
func (va *Agent) generate(ctx context.Context, call fantasy.AgentCall) (*fantasy.AgentResult, error) {
	if va.config.Retry == nil {
		return va.agent.Generate(ctx, call)
	}

	return WithRetry(ctx, *va.config.Retry, func(ctx context.Context) (*fantasy.AgentResult, error) {
		return va.agent.Generate(ctx, call)
	})
}

// withTimeout applies the configured request timeout if set.
// Returns the context and a cancel function that the caller must call.
func (va *Agent) withTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	if va.config.RequestTimeout > 0 {
		return context.WithTimeout(ctx, va.config.RequestTimeout)
	}

	return ctx, func() {}
}

// requireImages filters nil images and returns ErrNoImages if the result is empty.
// It is the shared guard at the entry of every public analysis method.
func requireImages(images []*ImageSource) ([]*ImageSource, error) {
	valid := filterValidImages(images)
	if len(valid) == 0 {
		return nil, ErrNoImages
	}

	return valid, nil
}

// validateAnalyzeInput enforces the two preconditions every public analysis
// method must satisfy: a non-empty prompt and at least one non-nil image.
// Centralising the check prevents the entry guards from drifting apart.
func validateAnalyzeInput(prompt string, images []*ImageSource) ([]*ImageSource, error) {
	if prompt == "" {
		return nil, ErrEmptyPrompt
	}

	return requireImages(images)
}

// preprocessImages applies Config.Preprocess to the validated image slice.
// When no preprocessing is configured, images pass through unchanged.
func (va *Agent) preprocessImages(images []*ImageSource) ([]*ImageSource, error) {
	return preprocessAll(images, va.config.Preprocess)
}

// filterValidImages removes nil images from the slice.
func filterValidImages(images []*ImageSource) []*ImageSource {
	var valid []*ImageSource

	for _, img := range images {
		if img != nil {
			valid = append(valid, img)
		}
	}

	return valid
}

// toFileParts converts ImageSources to fantasy FileParts.
func toFileParts(images []*ImageSource) []fantasy.FilePart {
	files := make([]fantasy.FilePart, len(images))
	for i, img := range images {
		files[i] = fantasy.FilePart{
			Data:            img.Data,
			MediaType:       string(img.MediaType),
			Filename:        img.Filename,
			ProviderOptions: nil,
		}
	}

	return files
}
