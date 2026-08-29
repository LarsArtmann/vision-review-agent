package reviewed

import (
	"context"
	"fmt"
	"time"

	"charm.land/fantasy"
	"github.com/larsartmann/vision-review-agent/pkg/vision"
)

// ReviewResult is the model's judgment of one or two images.
type ReviewResult struct {
	Markdown string
	Score    int
}

// Reviewer runs model reviews of UI screenshots through the vision SDK. Each
// prompt persona gets its own agent: single-view reviews and BEFORE/AFTER
// comparisons run under their dedicated system prompt.
type Reviewer struct {
	reviewAgent  *vision.Agent
	compareAgent *vision.Agent
	model        string
}

// NewReviewerFromConfig builds a Reviewer from daemon configuration: an
// OpenAI-compatible (llama-server) language model with review prompts and the
// configured per-request timeout.
func NewReviewerFromConfig(ctx context.Context, config Config) (*Reviewer, error) {
	languageModel, err := LanguageModel(ctx, config)
	if err != nil {
		return nil, err
	}

	return NewReviewer(languageModel, config.Model, config.Timeout)
}

// NewReviewer builds a Reviewer over an existing language model. modelID is
// recorded in events; timeout bounds each model request.
func NewReviewer(languageModel fantasy.LanguageModel, modelID string, timeout time.Duration) (*Reviewer, error) {
	reviewAgent, err := newAgent(languageModel, ReviewSystemPrompt, "review", timeout)
	if err != nil {
		return nil, err
	}

	compareAgent, err := newAgent(languageModel, CompareSystemPrompt, "compare", timeout)
	if err != nil {
		return nil, err
	}

	return &Reviewer{reviewAgent: reviewAgent, compareAgent: compareAgent, model: modelID}, nil
}

// newAgent builds a vision.Agent over languageModel under one system prompt.
// label names the persona in error messages.
func newAgent(
	languageModel fantasy.LanguageModel,
	systemPrompt string,
	label string,
	timeout time.Duration,
) (*vision.Agent, error) {
	agent, err := vision.NewAgent(vision.Config{
		Model:          languageModel,
		SystemPrompt:   systemPrompt,
		RequestTimeout: timeout,
	})
	if err != nil {
		return nil, fmt.Errorf("create %s agent: %w", label, err)
	}

	return agent, nil
}

// Model returns the model id recorded in events.
func (r *Reviewer) Model() string {
	return r.model
}

// Review asks the model to review the screenshot at imagePath and returns its
// markdown judgment plus the parsed score.
func (r *Reviewer) Review(ctx context.Context, viewKey ViewKey, imagePath string) (ReviewResult, error) {
	result, err := analyzeImage(ctx, r.reviewAgent, ReviewPrompt(viewKey), imagePath)
	if err != nil {
		return ReviewResult{}, fmt.Errorf("review %s: %w", viewKey, err)
	}

	return result, nil
}

// Compare asks the model to judge a BEFORE/AFTER pair of the same view. The
// score rates the AFTER image.
func (r *Reviewer) Compare(ctx context.Context, viewKey ViewKey, beforePath, afterPath string) (ReviewResult, error) {
	before, err := vision.LoadImageFromFile(beforePath)
	if err != nil {
		return ReviewResult{}, fmt.Errorf("compare %s: load before %s: %w", viewKey, beforePath, err)
	}

	after, err := vision.LoadImageFromFile(afterPath)
	if err != nil {
		return ReviewResult{}, fmt.Errorf("compare %s: load after %s: %w", viewKey, afterPath, err)
	}

	response, err := r.compareAgent.Analyze(ctx, ComparePrompt(viewKey), before, after)
	if err != nil {
		return ReviewResult{}, fmt.Errorf("compare %s: %w", viewKey, err)
	}

	return ReviewResult{
		Markdown: response.Text,
		Score:    ExtractScore(response.Text),
	}, nil
}

// analyzeImage loads the screenshot at imagePath and runs one model analysis
// under the given agent. It is a free function: it needs no Reviewer state.
func analyzeImage(ctx context.Context, agent *vision.Agent, prompt string, imagePath string) (ReviewResult, error) {
	image, err := vision.LoadImageFromFile(imagePath)
	if err != nil {
		return ReviewResult{}, fmt.Errorf("load %s: %w", imagePath, err)
	}

	response, err := agent.Analyze(ctx, prompt, image)
	if err != nil {
		return ReviewResult{}, fmt.Errorf("analyze: %w", err)
	}

	return ReviewResult{
		Markdown: response.Text,
		Score:    ExtractScore(response.Text),
	}, nil
}
