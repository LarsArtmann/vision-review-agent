package vision

import (
	"context"
	"fmt"
	"time"

	"charm.land/fantasy"
)

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
	config Config
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
	return sa
}

// WithMaxOutputTokens sets the maximum output tokens.
func (sa *ScreenshotAnalyzer) WithMaxOutputTokens(tokens int64) *ScreenshotAnalyzer {
	sa.config.MaxOutputTokens = tokens
	return sa
}

// WithTemperature sets the temperature.
func (sa *ScreenshotAnalyzer) WithTemperature(temp float64) *ScreenshotAnalyzer {
	sa.config.Temperature = temp
	return sa
}

// WithMaxRetries sets the maximum number of retries on transient errors.
func (sa *ScreenshotAnalyzer) WithMaxRetries(retries int) *ScreenshotAnalyzer {
	sa.config.MaxRetries = retries
	return sa
}

// WithRequestTimeout sets a per-request timeout.
func (sa *ScreenshotAnalyzer) WithRequestTimeout(timeout time.Duration) *ScreenshotAnalyzer {
	sa.config.RequestTimeout = timeout
	return sa
}

// agent lazily creates the underlying Agent with current config.
func (sa *ScreenshotAnalyzer) agent() (*Agent, error) {
	return NewAgent(sa.config)
}

// analyzeWithAgent calls the agent's Analyze method with the given prompt and images.
func (sa *ScreenshotAnalyzer) analyzeWithAgent(
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

// AnalyzeScreenshot analyzes a single screenshot with the given prompt.
func (sa *ScreenshotAnalyzer) AnalyzeScreenshot(
	ctx context.Context,
	prompt string,
	screenshotPath string,
) (*AnalyzeResult, error) {
	img, err := LoadImageFromFile(screenshotPath)
	if err != nil {
		return nil, err
	}
	return sa.AnalyzeScreenshotImage(ctx, prompt, img)
}

// AnalyzeScreenshotImage analyzes a single screenshot ImageSource with the given prompt.
func (sa *ScreenshotAnalyzer) AnalyzeScreenshotImage(
	ctx context.Context,
	prompt string,
	img *ImageSource,
) (*AnalyzeResult, error) {
	return sa.analyzeWithAgent(ctx, prompt, img)
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
			return nil, fmt.Errorf("load screenshot %q: %w", path, err)
		}
		images[i] = img
	}
	return sa.AnalyzeScreenshotImages(ctx, prompt, images...)
}

// AnalyzeScreenshotImages analyzes multiple ImageSources with the given prompt.
func (sa *ScreenshotAnalyzer) AnalyzeScreenshotImages(
	ctx context.Context,
	prompt string,
	images ...*ImageSource,
) (*AnalyzeResult, error) {
	return sa.analyzeWithAgent(ctx, prompt, images...)
}
