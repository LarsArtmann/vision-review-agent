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

	// Model is the language model to use. Must support vision.
	// Required.
	Model fantasy.LanguageModel

	// MaxRetries sets the maximum number of retries on transient errors.
	// Zero means use fantasy's default.
	MaxRetries int

	// RequestTimeout sets a per-request timeout. Zero means no timeout.
	RequestTimeout time.Duration
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
	return nil
}

// VisionAgent is an AI agent capable of analyzing images/screenshots.
type VisionAgent struct {
	config Config
	agent  fantasy.Agent
}

// NewAgent creates a new VisionAgent with the given configuration.
// Returns an error if the configuration is invalid.
func NewAgent(config Config) (*VisionAgent, error) {
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

	return &VisionAgent{
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
func (va *VisionAgent) Analyze(
	ctx context.Context,
	prompt string,
	images ...*ImageSource,
) (*AnalyzeResult, error) {
	if prompt == "" {
		return nil, ErrEmptyPrompt
	}
	validImages := filterValidImages(images)
	if len(validImages) == 0 {
		return nil, ErrNoImages
	}

	ctx, cancel := va.withTimeout(ctx)
	defer cancel()

	files := toFileParts(validImages)

	call := fantasy.AgentCall{
		Prompt: prompt,
		Files:  files,
	}

	result, err := va.agent.Generate(ctx, call)
	if err != nil {
		return nil, fmt.Errorf("vision agent generate: %w", err)
	}

	return &AnalyzeResult{
		Text:        result.Response.Content.Text(),
		Usage:       result.TotalUsage,
		RawResponse: result,
	}, nil
}

// AnalyzeStream sends images to the agent and streams the response.
// The onText callback is called for each chunk of text received.
func (va *VisionAgent) AnalyzeStream(
	ctx context.Context,
	prompt string,
	onText func(text string) error,
	images ...*ImageSource,
) (*AnalyzeResult, error) {
	if prompt == "" {
		return nil, ErrEmptyPrompt
	}
	validImages := filterValidImages(images)
	if len(validImages) == 0 {
		return nil, ErrNoImages
	}

	ctx, cancel := va.withTimeout(ctx)
	defer cancel()

	files := toFileParts(validImages)

	var fullText string

	streamCall := fantasy.AgentStreamCall{
		Prompt: prompt,
		Files:  files,
		OnTextDelta: func(id, text string) error {
			fullText += text
			if onText != nil {
				return onText(text)
			}
			return nil
		},
	}

	result, err := va.agent.Stream(ctx, streamCall)
	if err != nil {
		return nil, fmt.Errorf("vision agent stream: %w", err)
	}

	return &AnalyzeResult{
		Text:        fullText,
		Usage:       result.TotalUsage,
		RawResponse: result,
	}, nil
}

// withTimeout applies the configured request timeout if set.
// Returns the context and a cancel function that the caller must call.
func (va *VisionAgent) withTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	if va.config.RequestTimeout > 0 {
		return context.WithTimeout(ctx, va.config.RequestTimeout)
	}
	return ctx, func() {}
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
			Data:      img.Data,
			MediaType: img.MediaType,
			Filename:  img.Filename,
		}
	}
	return files
}
