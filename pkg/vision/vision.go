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

	// MaxRetries sets the maximum number of retries on transient errors.
	// Zero means use fantasy's default.
	MaxRetries int

	// RequestTimeout sets a per-request timeout. Zero means no timeout.
	RequestTimeout time.Duration

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
}

// Validate checks the configuration for errors.
func (c Config) Validate() error {
	if c.Model == nil {
		return ErrNoModel
	}
	if c.Temperature < 0 || c.Temperature > 2.0 {
		return ErrInvalidTemperature
	}
	if c.MaxOutputTokens < 0 {
		return ErrInvalidMaxTokens
	}
	if c.TopP < 0 || c.TopP > 1.0 {
		return ErrInvalidTopP
	}
	if c.TopK < 0 {
		return ErrInvalidTopK
	}
	if c.PresencePenalty < -2.0 || c.PresencePenalty > 2.0 {
		return ErrInvalidPresencePenalty
	}
	if c.FrequencyPenalty < -2.0 || c.FrequencyPenalty > 2.0 {
		return ErrInvalidFrequencyPenalty
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
type VisionAgent = Agent //nolint:revive

// NewAgent creates a new Agent with the given configuration.
// Returns an error if the configuration is invalid.
func NewAgent(config Config) (*Agent, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	opts := []fantasy.AgentOption{
		fantasy.WithSystemPrompt(config.SystemPrompt),
	}

	if config.MaxOutputTokens > 0 {
		opts = append(opts, fantasy.WithMaxOutputTokens(config.MaxOutputTokens))
	}

	if config.Temperature != 0 {
		opts = append(opts, fantasy.WithTemperature(config.Temperature))
	}

	if config.MaxRetries > 0 {
		opts = append(opts, fantasy.WithMaxRetries(config.MaxRetries))
	}

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
	validImages, err := validateAnalyzeInput(prompt, images)
	if err != nil {
		return nil, err
	}

	va.config.Hooks.fireStart(ctx, prompt, len(validImages))

	ctx, cancel := va.withTimeout(ctx)
	defer cancel()

	files := toFileParts(validImages)

	call := va.buildAgentCall(prompt, files, nil)

	result, err := va.agent.Generate(ctx, call)
	if err != nil {
		classified := classifyModelErr("vision agent generate", prompt, err)
		va.config.Hooks.fireError(ctx, classified)
		return nil, classified
	}

	analysisResult := &AnalyzeResult{
		Text:        result.Response.Content.Text(),
		Usage:       result.TotalUsage,
		RawResponse: result,
	}
	va.config.Hooks.fireFinish(ctx, analysisResult)
	return analysisResult, nil
}

// AnalyzeStream sends images to the agent and streams the response.
// The onText callback is called for each chunk of text received.
func (va *Agent) AnalyzeStream(
	ctx context.Context,
	prompt string,
	onText func(text string) error,
	images ...*ImageSource,
) (*AnalyzeResult, error) {
	validImages, err := validateAnalyzeInput(prompt, images)
	if err != nil {
		return nil, err
	}

	va.config.Hooks.fireStart(ctx, prompt, len(validImages))

	ctx, cancel := va.withTimeout(ctx)
	defer cancel()

	files := toFileParts(validImages)

	var builder strings.Builder

	streamCall := va.buildAgentStreamCall(prompt, files, nil)
	streamCall.OnTextDelta = func(_, text string) error {
		builder.WriteString(text)
		if onText != nil {
			return onText(text)
		}
		return nil
	}

	result, err := va.agent.Stream(ctx, streamCall)
	if err != nil {
		classified := classifyModelErr("vision agent stream", prompt, err)
		va.config.Hooks.fireError(ctx, classified)
		return nil, classified
	}

	analysisResult := &AnalyzeResult{
		Text:        builder.String(),
		Usage:       result.TotalUsage,
		RawResponse: result,
	}
	va.config.Hooks.fireFinish(ctx, analysisResult)
	return analysisResult, nil
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
	validImages, err := validateAnalyzeInput(prompt, images)
	if err != nil {
		return nil, err
	}

	va.config.Hooks.fireStart(ctx, prompt, len(validImages))

	ctx, cancel := va.withTimeout(ctx)
	defer cancel()

	call := va.buildAgentCall(prompt, toFileParts(validImages), conv.Messages())

	result, err := va.agent.Generate(ctx, call)
	if err != nil {
		classified := classifyModelErr("vision agent conversation generate", prompt, err)
		va.config.Hooks.fireError(ctx, classified)
		return nil, classified
	}

	analysisResult := &AnalyzeResult{
		Text:        result.Response.Content.Text(),
		Usage:       result.TotalUsage,
		RawResponse: result,
	}
	va.config.Hooks.fireFinish(ctx, analysisResult)
	return analysisResult, nil
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
	validImages, err := validateAnalyzeInput(prompt, images)
	if err != nil {
		return nil, err
	}

	va.config.Hooks.fireStart(ctx, prompt, len(validImages))

	ctx, cancel := va.withTimeout(ctx)
	defer cancel()

	var builder strings.Builder

	streamCall := va.buildAgentStreamCall(prompt, toFileParts(validImages), conv.Messages())
	streamCall.OnTextDelta = func(_, text string) error {
		builder.WriteString(text)
		if onText != nil {
			return onText(text)
		}
		return nil
	}

	result, err := va.agent.Stream(ctx, streamCall)
	if err != nil {
		classified := classifyModelErr("vision agent conversation stream", prompt, err)
		va.config.Hooks.fireError(ctx, classified)
		return nil, classified
	}

	analysisResult := &AnalyzeResult{
		Text:        builder.String(),
		Usage:       result.TotalUsage,
		RawResponse: result,
	}
	va.config.Hooks.fireFinish(ctx, analysisResult)
	return analysisResult, nil
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
	va.applyModelParamsAgentCall(&call)
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
	va.applyModelParamsStreamCall(&call)
	return call
}

// applyModelParamsAgentCall sets optional model parameters on an AgentCall.
// Zero-valued config fields are skipped, leaving the destination pointer nil
// so the provider uses its own default. Fields are assigned directly to avoid
// the fragile positional double-pointer passing of the former helper.
func (va *Agent) applyModelParamsAgentCall(call *fantasy.AgentCall) {
	if va.config.MaxOutputTokens > 0 {
		call.MaxOutputTokens = &va.config.MaxOutputTokens
	}
	if va.config.Temperature != 0 {
		call.Temperature = &va.config.Temperature
	}
	if va.config.TopP > 0 {
		call.TopP = &va.config.TopP
	}
	if va.config.TopK > 0 {
		call.TopK = &va.config.TopK
	}
	if va.config.PresencePenalty != 0 {
		call.PresencePenalty = &va.config.PresencePenalty
	}
	if va.config.FrequencyPenalty != 0 {
		call.FrequencyPenalty = &va.config.FrequencyPenalty
	}
}

// applyModelParamsStreamCall sets optional model parameters on an AgentStreamCall.
func (va *Agent) applyModelParamsStreamCall(call *fantasy.AgentStreamCall) {
	if va.config.MaxOutputTokens > 0 {
		call.MaxOutputTokens = &va.config.MaxOutputTokens
	}
	if va.config.Temperature != 0 {
		call.Temperature = &va.config.Temperature
	}
	if va.config.TopP > 0 {
		call.TopP = &va.config.TopP
	}
	if va.config.TopK > 0 {
		call.TopK = &va.config.TopK
	}
	if va.config.PresencePenalty != 0 {
		call.PresencePenalty = &va.config.PresencePenalty
	}
	if va.config.FrequencyPenalty != 0 {
		call.FrequencyPenalty = &va.config.FrequencyPenalty
	}
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
