package vision

import (
	"context"
	"fmt"
	"time"

	"charm.land/fantasy"
)

// Compile-time check: ScreenshotAnalyzer implements Analyzer.
var _ Analyzer = (*ScreenshotAnalyzer)(nil)

// ScreenshotAnalyzer is a convenience builder for screenshot analysis use cases.
// It provides a fluent API for configuring and running screenshot analysis.
//
// Usage:
//
//	analyzer := vision.NewScreenshotAnalyzer(model).
//	    WithSystemPrompt("Find all accessibility issues").
//	    WithTemperature(0.2)
//
//	result, err := analyzer.AnalyzeScreenshot(ctx, "Describe this UI", "screenshot.png")
type ScreenshotAnalyzer struct {
	config      Config
	cachedAgent *Agent
}

// DefaultScreenshotPrompt is the default system prompt for UI/screenshot analysis.
const DefaultScreenshotPrompt = `You are a precise UI/UX analysis assistant. When given screenshots:
1. Describe what you see clearly and concisely
2. Identify UI components, layout, and design patterns
3. Note any issues, inconsistencies, or areas for improvement
4. Be specific about colors, typography, spacing, and alignment
5. If asked to compare, highlight differences clearly`

// NewScreenshotAnalyzer creates a ScreenshotAnalyzer with a default system prompt
// optimized for UI/screenshot analysis.
func NewScreenshotAnalyzer(model fantasy.LanguageModel) *ScreenshotAnalyzer {
	return &ScreenshotAnalyzer{
		config: Config{
			SystemPrompt: DefaultScreenshotPrompt,
			Model:        model,
		},
	}
}

// WithSystemPrompt sets a custom system prompt.
func (sa *ScreenshotAnalyzer) WithSystemPrompt(prompt string) *ScreenshotAnalyzer {
	sa.config.SystemPrompt = prompt

	return sa.invalidate()
}

// WithMaxOutputTokens sets the maximum output tokens.
func (sa *ScreenshotAnalyzer) WithMaxOutputTokens(tokens int64) *ScreenshotAnalyzer {
	sa.config.MaxOutputTokens = tokens

	return sa.invalidate()
}

// WithTemperature sets the temperature.
func (sa *ScreenshotAnalyzer) WithTemperature(temp float64) *ScreenshotAnalyzer {
	sa.config.Temperature = temp

	return sa.invalidate()
}

// WithTopP sets the top-p (nucleus sampling) parameter.
func (sa *ScreenshotAnalyzer) WithTopP(topP float64) *ScreenshotAnalyzer {
	sa.config.TopP = topP

	return sa.invalidate()
}

// WithTopK sets the top-k sampling parameter.
func (sa *ScreenshotAnalyzer) WithTopK(topK int64) *ScreenshotAnalyzer {
	sa.config.TopK = topK

	return sa.invalidate()
}

// WithPresencePenalty sets the presence penalty.
func (sa *ScreenshotAnalyzer) WithPresencePenalty(penalty float64) *ScreenshotAnalyzer {
	sa.config.PresencePenalty = penalty

	return sa.invalidate()
}

// WithFrequencyPenalty sets the frequency penalty.
func (sa *ScreenshotAnalyzer) WithFrequencyPenalty(penalty float64) *ScreenshotAnalyzer {
	sa.config.FrequencyPenalty = penalty

	return sa.invalidate()
}

// WithMaxRetries sets the maximum number of retries on transient errors.
func (sa *ScreenshotAnalyzer) WithMaxRetries(retries int) *ScreenshotAnalyzer {
	sa.config.MaxRetries = retries

	return sa.invalidate()
}

// WithRequestTimeout sets a per-request timeout.
func (sa *ScreenshotAnalyzer) WithRequestTimeout(timeout time.Duration) *ScreenshotAnalyzer {
	sa.config.RequestTimeout = timeout

	return sa.invalidate()
}

// WithHooks sets lifecycle callbacks (OnStart, OnFinish, OnError) for
// observability such as logging and metrics.
func (sa *ScreenshotAnalyzer) WithHooks(hooks Hooks) *ScreenshotAnalyzer {
	sa.config.Hooks = hooks

	return sa.invalidate()
}

// WithMaxDimension sets automatic preprocessing that resizes images so their
// longest side does not exceed maxDimension pixels before every analysis call.
// Zero disables preprocessing (images sent as-is).
func (sa *ScreenshotAnalyzer) WithMaxDimension(maxDimension int) *ScreenshotAnalyzer {
	if maxDimension > 0 {
		sa.config.Preprocess = &PreprocessConfig{MaxDimension: maxDimension}
	} else {
		sa.config.Preprocess = nil
	}

	return sa.invalidate()
}

// invalidate marks the cached agent as stale so the next analysis call
// rebuilds it from the current config. Every With* builder calls this
// after mutating config fields.
func (sa *ScreenshotAnalyzer) invalidate() *ScreenshotAnalyzer {
	sa.cachedAgent = nil

	return sa
}

// agent returns the underlying cached Agent, initializing it on first call.
func (sa *ScreenshotAnalyzer) agent() (*Agent, error) {
	if sa.cachedAgent != nil {
		return sa.cachedAgent, nil
	}

	agent, err := NewAgent(sa.config)
	if err != nil {
		return nil, fmt.Errorf("screenshot analyzer: build agent: %w", err)
	}

	sa.cachedAgent = agent

	return agent, nil
}

// Analyze analyzes images with the given prompt, satisfying the Analyzer interface.
func (sa *ScreenshotAnalyzer) Analyze(
	ctx context.Context,
	prompt string,
	images ...*ImageSource,
) (*AnalyzeResult, error) {
	return sa.AnalyzeScreenshotImages(ctx, prompt, images...)
}

// AnalyzeStream streams analysis of images with the given prompt, satisfying the Analyzer interface.
func (sa *ScreenshotAnalyzer) AnalyzeStream(
	ctx context.Context,
	prompt string,
	onText func(text string) error,
	images ...*ImageSource,
) (*AnalyzeResult, error) {
	a, err := sa.agent()
	if err != nil {
		return nil, err
	}

	return a.AnalyzeStream(ctx, prompt, onText, images...)
}

// AnalyzeScreenshotImage analyzes a single screenshot ImageSource with the given prompt.
func (sa *ScreenshotAnalyzer) AnalyzeScreenshotImage(
	ctx context.Context,
	prompt string,
	img *ImageSource,
) (*AnalyzeResult, error) {
	return sa.AnalyzeScreenshotImages(ctx, prompt, img)
}

// AnalyzeScreenshot analyzes a single screenshot with the given prompt.
func (sa *ScreenshotAnalyzer) AnalyzeScreenshot(
	ctx context.Context,
	prompt string,
	screenshotPath string,
) (*AnalyzeResult, error) {
	img, err := LoadImageFromFile(screenshotPath)
	if err != nil {
		return nil, fmt.Errorf(
			"AnalyzeScreenshot: screenshotPath=%q, prompt=%q: %w",
			screenshotPath,
			prompt,
			err,
		)
	}

	return sa.AnalyzeScreenshotImage(ctx, prompt, img)
}

// AnalyzeScreenshotImages analyzes multiple ImageSources with the given prompt.
func (sa *ScreenshotAnalyzer) AnalyzeScreenshotImages(
	ctx context.Context,
	prompt string,
	images ...*ImageSource,
) (*AnalyzeResult, error) {
	agent, err := sa.agent()
	if err != nil {
		return nil, err
	}

	return agent.Analyze(ctx, prompt, images...)
}

// AnalyzeScreenshots analyzes multiple screenshots with the given prompt.
func (sa *ScreenshotAnalyzer) AnalyzeScreenshots(
	ctx context.Context,
	prompt string,
	screenshotPaths ...string,
) (*AnalyzeResult, error) {
	images := make([]*ImageSource, len(screenshotPaths))
	for i, path := range screenshotPaths {
		img, err := LoadImageFromFile(path)
		if err != nil {
			return nil, fmt.Errorf("AnalyzeScreenshots: path=%q, prompt=%q: %w", path, prompt, err)
		}

		images[i] = img
	}

	return sa.AnalyzeScreenshotImages(ctx, prompt, images...)
}

// AnalyzeConversation analyzes images with conversation history.
func (sa *ScreenshotAnalyzer) AnalyzeConversation(
	ctx context.Context,
	conv *Conversation,
	prompt string,
	images ...*ImageSource,
) (*AnalyzeResult, error) {
	a, err := sa.agent()
	if err != nil {
		return nil, err
	}

	return a.AnalyzeConversation(ctx, conv, prompt, images...)
}

// AnalyzeConversationStream streams analysis with conversation history.
func (sa *ScreenshotAnalyzer) AnalyzeConversationStream(
	ctx context.Context,
	conv *Conversation,
	prompt string,
	onText func(text string) error,
	images ...*ImageSource,
) (*AnalyzeResult, error) {
	a, err := sa.agent()
	if err != nil {
		return nil, err
	}

	return a.AnalyzeConversationStream(ctx, conv, prompt, onText, images...)
}
