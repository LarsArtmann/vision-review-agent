// vision is a CLI tool for analyzing screenshots and images with AI.
//
// Usage:
//
//	vision -provider openai -model gpt-4o -prompt "Describe this UI" screenshot.png
//	vision -provider openrouter -model openai/gpt-4o -prompt "Find bugs" *.png
//	vision -stream -prompt "Analyze this" screenshot.png
//	vision -json -prompt "Find bugs" screenshot.png | jq '.text'
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"charm.land/fantasy"
	"charm.land/fantasy/providers/anthropic"
	"charm.land/fantasy/providers/google"
	"charm.land/fantasy/providers/openai"
	"charm.land/fantasy/providers/openaicompat"
	"charm.land/fantasy/providers/openrouter"
	"github.com/larsartmann/vision-review-agent/internal/cli"
	"github.com/larsartmann/vision-review-agent/pkg/vision"
)

const (
	defaultTemperature = 0.3
)

// version is the CLI version string. It is a var (not a const) so release
// tooling can override it at build time via -ldflags "-X main.version=...".
// The default reflects an unreleased working tree; tagged builds inject the
// real semver (see flake.nix buildGoModule ldflags).
var version = "0.3.0-dev"

var (
	errEnvVarNotSet    = errors.New("environment variable not set")
	errUnknownProvider = errors.New("unknown provider")
)

func main() {
	cfg, err := parseFlags(flag.CommandLine, os.Args[1:])
	cli.ExitOnError(err, "")

	if cfg.showVersion {
		fmt.Println("vision", version)
		os.Exit(0)
	}

	if len(cfg.args) == 0 {
		flag.Usage()
		os.Exit(1)
	}

	ctx := context.Background()
	provider, err := createProvider(cfg.providerName)
	cli.ExitOnError(err, "")

	model, err := provider.LanguageModel(ctx, cfg.modelID)
	cli.ExitOnError(err, "Error getting model")

	agent, err := vision.NewAgent(buildConfig(model, cfg))
	cli.ExitOnError(err, "Error creating agent")

	images, err := loadImages(cfg.args)
	cli.ExitOnError(err, "")

	runAnalysis(ctx, agent, cfg, images)
}

type config struct {
	providerName string
	modelID      string
	prompt       string
	systemPrompt string
	stream       bool
	structured   bool
	temperature  float64
	maxTokens    int64
	jsonOutput   bool
	timeout      int64
	showVersion  bool
	args         []string // positional image paths
}

// parseFlags parses the CLI flags from args using fs. It does NOT call os.Exit:
// version/usage decisions are returned via cfg.showVersion and cfg.args so the
// caller (and tests) can act on them. Using a *flag.FlagSet lets tests pass a
// fresh, isolated flag set.
func parseFlags(flagSet *flag.FlagSet, args []string) (*config, error) {
	providerName := flagSet.String(
		"provider",
		"openai",
		"Provider: openai, openrouter, anthropic, google, openaicompat",
	)
	modelID := flagSet.String("model", "gpt-4o", "Model ID (e.g., gpt-4o, openai/gpt-4o)")
	prompt := flagSet.String(
		"prompt",
		"Describe what you see in this image.",
		"Analysis prompt",
	)
	systemPrompt := flagSet.String("system", "", "Custom system prompt (optional)")
	stream := flagSet.Bool("stream", false, "Stream the response")
	temperature := flagSet.Float64("temperature", defaultTemperature, "Temperature (0.0-2.0)")
	maxTokens := flagSet.Int64("max-tokens", 0, "Max output tokens (0 = unlimited)")
	jsonOutput := flagSet.Bool("json", false, "Output result as JSON")
	structured := flagSet.Bool("structured", false, "Emit a structured UI review as JSON (built-in schema)")
	timeout := flagSet.Int64("timeout", 0, "Request timeout in seconds (0 = unlimited)")
	showVersion := flagSet.Bool("version", false, "Show version and exit")

	flagSet.Usage = usageFunc(flagSet)

	if err := flagSet.Parse(args); err != nil {
		return nil, fmt.Errorf("parse flags: %w", err)
	}

	return &config{
		providerName: *providerName,
		modelID:      *modelID,
		prompt:       *prompt,
		systemPrompt: *systemPrompt,
		stream:       *stream,
		temperature:  *temperature,
		maxTokens:    *maxTokens,
		jsonOutput:   *jsonOutput,
		structured:   *structured,
		timeout:      *timeout,
		showVersion:  *showVersion,
		args:         flagSet.Args(),
	}, nil
}

func usageFunc(flagSet *flag.FlagSet) func() {
	name := flagSet.Name()

	return func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options] <image1.png> [image2.png ...]\n\n", name)
		fmt.Fprint(os.Stderr, "Analyze images/screenshots with AI vision models.\n\n")
		fmt.Fprintln(os.Stderr, "Environment variables:")
		fmt.Fprintln(os.Stderr, "  OPENAI_API_KEY          - OpenAI provider")
		fmt.Fprintln(os.Stderr, "  OPENROUTER_API_KEY      - OpenRouter provider")
		fmt.Fprintln(os.Stderr, "  ANTHROPIC_API_KEY       - Anthropic provider")
		fmt.Fprintln(os.Stderr, "  GOOGLE_APPLICATION_*    - Google provider (ADC)")
		fmt.Fprintln(os.Stderr, "  OPENAICOMPAT_BASE_URL   - openaicompat provider (required)")
		fmt.Fprint(os.Stderr, "  OPENAICOMPAT_API_KEY    - openaicompat provider (optional)\n\n")
		fmt.Fprintln(os.Stderr, "Options:")
		flagSet.PrintDefaults()
		fmt.Fprintln(os.Stderr, "\nExamples:")
		fmt.Fprintf(os.Stderr, "  %s -prompt \"Find UI bugs\" screenshot.png\n", name)
		fmt.Fprintf(
			os.Stderr,
			"  %s -provider openrouter -model anthropic/claude-3.5-sonnet screenshot.png\n",
			name,
		)
		fmt.Fprintf(os.Stderr, "  %s -stream -prompt \"Describe this\" *.png\n", name)
		fmt.Fprintf(
			os.Stderr,
			"  %s -json -prompt \"Find bugs\" screenshot.png | jq '.text'\n",
			name,
		)
	}
}

func buildConfig(model fantasy.LanguageModel, cfg *config) vision.Config {
	//nolint:exhaustruct // optional fields intentionally use zero-value defaults
	config := vision.Config{
		Model:           model,
		Temperature:     cfg.temperature,
		MaxOutputTokens: cfg.maxTokens,
	}

	if cfg.systemPrompt != "" {
		config.SystemPrompt = cfg.systemPrompt
	}

	if cfg.timeout > 0 {
		config.RequestTimeout = parseTimeout(cfg.timeout)
	}

	return config
}

func parseTimeout(seconds int64) time.Duration {
	return time.Duration(seconds) * time.Second
}

func loadImages(args []string) ([]*vision.ImageSource, error) {
	images := make([]*vision.ImageSource, 0, len(args))
	for _, path := range args {
		img, err := vision.LoadImageFromFile(path)
		if err != nil {
			return nil, fmt.Errorf("loading %s: %w", path, err)
		}

		images = append(images, img)
	}

	return images, nil
}

func runAnalysis(
	ctx context.Context,
	agent *vision.Agent,
	cfg *config,
	images []*vision.ImageSource,
) {
	if cfg.structured {
		runStructured(ctx, agent, cfg, images)

		return
	}

	var (
		result *vision.AnalyzeResult
		err    error
	)

	if cfg.stream {
		fmt.Println("Analyzing (streaming)...")

		result, err = agent.AnalyzeStream(ctx, cfg.prompt, func(text string) error {
			fmt.Print(text)

			return nil
		}, images...)
	} else {
		fmt.Println("Analyzing...")

		result, err = agent.Analyze(ctx, cfg.prompt, images...)
	}

	if err != nil {
		printAnalysisError(err, cfg.stream)
		os.Exit(1)
	}

	if cfg.jsonOutput {
		printJSON(result)
	} else {
		printText(result, cfg.stream)
	}
}

// runStructured performs a structured UI review and prints it as JSON.
func runStructured(
	ctx context.Context,
	agent *vision.Agent,
	cfg *config,
	images []*vision.ImageSource,
) {
	fmt.Println("Analyzing (structured)...")

	result, err := vision.AnalyzeStructured[uiReview](ctx, agent, cfg.prompt, images...)
	if err != nil {
		printAnalysisError(err, false)
		os.Exit(1)
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")

	if err := enc.Encode(result.Object); err != nil {
		fmt.Fprintln(os.Stderr, "Error encoding JSON:", err)
	}
}

// uiReview is the built-in structured schema emitted by the -structured flag.
type uiReview struct {
	Layout      string    `description:"Brief description of the overall layout" json:"layout"`
	Components  []string  `description:"List of UI components identified"        json:"components"`
	Issues      []uiIssue `description:"List of issues found"                    json:"issues"`
	Score       int       `description:"Overall UX score from 1-10"              json:"score"`
	Suggestions []string  `description:"Actionable improvement suggestions"      json:"suggestions"`
}

// uiIssue represents a single UI issue in a structured review.
type uiIssue struct {
	Severity    string `description:"Severity: critical, major, minor, or info" json:"severity"`
	Component   string `description:"Which component has the issue"             json:"component"`
	Description string `description:"Detailed description of the issue"         json:"description"`
}

// printAnalysisError prints a user-friendly error message with actionable
// advice when the error is a classified ModelError, or falls back to the raw
// error string for unclassified errors.
func printAnalysisError(err error, streamed bool) {
	prefix := "Error:"
	if streamed {
		prefix = "\nError:"
	}

	modelErr, ok := errors.AsType[*vision.ModelError](err)
	if !ok {
		fmt.Fprintln(os.Stderr, prefix, err)

		return
	}

	advice := adviceForKind(modelErr.Kind)
	if advice != "" {
		fmt.Fprintf(os.Stderr, "%s [%s] %s\n", prefix, modelErr.Kind, advice)
		fmt.Fprintf(os.Stderr, "  Details: %s\n", modelErr.Cause)

		return
	}

	fmt.Fprintln(os.Stderr, prefix, err)
}

// adviceForKind returns a user-actionable hint for a given error kind,
// or an empty string if no specific advice applies.
func adviceForKind(kind vision.ErrorKind) string {
	switch kind {
	case vision.KindRateLimited:
		return "The AI provider is rate-limiting requests. Wait a moment and retry."
	case vision.KindTimeout:
		return "The request timed out. Increase the -timeout flag or use fewer/smaller images."
	case vision.KindServerError:
		return "The AI provider is experiencing issues. Retry later."
	case vision.KindServiceUnavailable:
		return "The AI provider is temporarily unavailable (503). Retry with backoff; consider a different model if persistent."
	case vision.KindNotImplemented:
		return "The provider does not implement this model or feature (501). Use a different model or provider."
	case vision.KindNetwork:
		return "Could not reach the AI provider. Check your internet connection and retry."
	case vision.KindAuthentication:
		return "Authentication failed. Verify your API key environment variable (OPENAI_API_KEY or OPENROUTER_API_KEY)."
	case vision.KindNotFound:
		return "The requested model was not found. Check the -model flag."
	case vision.KindBadRequest:
		return "The provider rejected the request. Check your prompt and image formats."
	case vision.KindContentFilter:
		return "The provider's content policy rejected the request. Modify your prompt or image."
	case vision.KindContextTooLarge:
		return "The input exceeds the model's context window. Use fewer or smaller images."
	case vision.KindCancelled:
		return "The request was cancelled."
	case vision.KindStructuredParse:
		return "The model failed to produce valid structured output. Try simplifying your prompt."
	default:
		return ""
	}
}

func printText(result *vision.AnalyzeResult, streamed bool) {
	if !streamed {
		fmt.Println("\n--- Analysis ---")
		fmt.Println(result.Text)
	}

	fmt.Printf("\nTokens used: %d\n", result.Usage.TotalTokens)
}

// jsonUsage is the shape of the JSON output produced by `-json`.
type jsonUsage struct {
	InputTokens  int64 `json:"inputTokens"`
	OutputTokens int64 `json:"outputTokens"`
	TotalTokens  int64 `json:"totalTokens"`
}

// jsonOutput is the JSON document written when `-json` is set.
type jsonOutput struct {
	Text  string    `json:"text"`
	Usage jsonUsage `json:"usage"`
}

func printJSON(result *vision.AnalyzeResult) {
	output := jsonOutput{
		Text: result.Text,
		Usage: jsonUsage{
			InputTokens:  result.Usage.InputTokens,
			OutputTokens: result.Usage.OutputTokens,
			TotalTokens:  result.Usage.TotalTokens,
		},
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")

	if err := enc.Encode(output); err != nil {
		fmt.Fprintln(os.Stderr, "Error encoding JSON:", err)
	}
}

// newProviderFromEnv reads an API key from the environment and uses the given
// factory to build the provider. It returns a descriptive error if the key is
// missing or the factory fails.
func newProviderFromEnv(
	envVar string,
	factory func(apiKey string) (fantasy.Provider, error),
) (fantasy.Provider, error) {
	apiKey := os.Getenv(envVar)
	if apiKey == "" {
		return nil, fmt.Errorf("%s: %w", envVar, errEnvVarNotSet)
	}

	provider, err := factory(apiKey)
	if err != nil {
		return nil, fmt.Errorf("create provider (env=%s): %w", envVar, err)
	}

	return provider, nil
}

func createOpenAIProvider(apiKey string) (fantasy.Provider, error) {
	provider, err := openai.New(openai.WithAPIKey(apiKey))

	return wrapProvider("openai", provider, err)
}

func createOpenRouterProvider(apiKey string) (fantasy.Provider, error) {
	provider, err := openrouter.New(openrouter.WithAPIKey(apiKey))

	return wrapProvider("openrouter", provider, err)
}

func createAnthropicProvider(apiKey string) (fantasy.Provider, error) {
	provider, err := anthropic.New(anthropic.WithAPIKey(apiKey))

	return wrapProvider("anthropic", provider, err)
}

// createGoogleProvider builds the Google (Gemini) provider using Application
// Default Credentials. Set GOOGLE_APPLICATION_CREDENTIALS or run `gcloud auth
// application-default login` before use.
func createGoogleProvider() (fantasy.Provider, error) {
	provider, err := google.New()

	return wrapProvider("google", provider, err)
}

// createOpenAICompatProvider builds an OpenAI-compatible provider for local
// model servers (Ollama, LM Studio). OPENAICOMPAT_BASE_URL is required;
// OPENAICOMPAT_API_KEY is optional (most local servers ignore it).
func createOpenAICompatProvider() (fantasy.Provider, error) {
	baseURL := os.Getenv("OPENAICOMPAT_BASE_URL")
	if baseURL == "" {
		return nil, fmt.Errorf("OPENAICOMPAT_BASE_URL: %w", errEnvVarNotSet)
	}

	opts := []openaicompat.Option{openaicompat.WithBaseURL(baseURL)}

	if apiKey := os.Getenv("OPENAICOMPAT_API_KEY"); apiKey != "" {
		opts = append(opts, openaicompat.WithAPIKey(apiKey))
	}

	provider, err := openaicompat.New(opts...)

	return wrapProvider("openaicompat", provider, err)
}

// wrapProvider pairs a provider constructor's (Provider, error) result with a
// consistent error message naming the provider that failed.
func wrapProvider(
	name string,
	provider fantasy.Provider,
	err error,
) (fantasy.Provider, error) {
	if err != nil {
		return nil, fmt.Errorf("create %s provider: %w", name, err)
	}

	return provider, nil
}

func createProvider(name string) (fantasy.Provider, error) {
	switch strings.ToLower(name) {
	case "openai":
		return newProviderFromEnv("OPENAI_API_KEY", createOpenAIProvider)
	case "openrouter":
		return newProviderFromEnv("OPENROUTER_API_KEY", createOpenRouterProvider)
	case "anthropic":
		return newProviderFromEnv("ANTHROPIC_API_KEY", createAnthropicProvider)
	case "google":
		return createGoogleProvider()
	case "openaicompat":
		return createOpenAICompatProvider()
	default:
		return nil, fmt.Errorf(
			"%s (supported: openai, openrouter, anthropic, google, openaicompat): %w",
			name,
			errUnknownProvider,
		)
	}
}
