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
	"io"
	"os"
	"strings"
	"time"

	"charm.land/fantasy"
	"charm.land/fantasy/providers/openaicompat"
	"github.com/larsartmann/vision-review-agent/internal/catalog"
	"github.com/larsartmann/vision-review-agent/internal/cli"
	"github.com/larsartmann/vision-review-agent/pkg/vision"
)

const (
	defaultTemperature  = 0.3
	providerGeminiAlias = "gemini"
	syncTimeout         = 5 * time.Second
)

// version is the CLI version string. It is a var (not a const) so release
// tooling can override it at build time via -ldflags "-X main.version=...".
// The default reflects an unreleased working tree; tagged builds inject the
// real semver (see flake.nix buildGoModule ldflags).
var version = "0.5.0"

var (
	// errEnvVarNotSet aliases catalog.ErrAPIKeyNotSet so existing tests and
	// error-handling code continue to match via errors.Is.
	errEnvVarNotSet    = catalog.ErrAPIKeyNotSet
	errUnknownProvider = errors.New("unknown provider")
)

// buildCatalog creates the catalog service. When CATWALK_URL is set, it attempts
// a remote sync with a short timeout, falling back to cache then embedded data.
func buildCatalog(ctx context.Context) *catalog.Service {
	if os.Getenv("CATWALK_URL") == "" {
		return catalog.New()
	}

	syncCtx, cancel := context.WithTimeout(ctx, syncTimeout)
	defer cancel()

	syncer := catalog.NewSync(catalog.DefaultCachePath())

	return catalog.NewWithProviders(syncer.Fetch(syncCtx))
}

func main() {
	cfg, err := parseFlags(flag.CommandLine, os.Args[1:])
	cli.ExitOnError(err, "")

	if cfg.showVersion {
		fmt.Println("vision", version)
		os.Exit(0)
	}

	ctx := context.Background()
	svc := buildCatalog(ctx)

	if cfg.listProviders {
		printProviders(os.Stdout, svc)

		return
	}

	if cfg.listModels {
		printVisionModels(os.Stdout, svc, cfg.providerName)

		return
	}

	if cfg.providerInfo {
		printProviderInfo(os.Stdout, svc, cfg.providerName)

		return
	}

	if len(cfg.args) == 0 {
		flag.Usage()
		os.Exit(1)
	}

	provider, err := createProvider(svc, cfg.providerName)
	cli.ExitOnError(err, "")

	if _, _, ok := svc.FindModel(cfg.modelID); !ok {
		if suggestion := suggestModel(svc, cfg.modelID); suggestion != "" {
			fmt.Fprintf(os.Stderr, "Warning: model %q not in catalog. Did you mean %q?\n", cfg.modelID, suggestion)
		}
	}

	model, err := provider.LanguageModel(ctx, cfg.modelID)
	cli.ExitOnError(err, "Error getting model")

	var modelInfo *vision.ModelInfo

	normalizedProvider := normalizeProviderName(cfg.providerName)
	if m, ok := svc.FindModelInProvider(normalizedProvider, cfg.modelID); ok {
		info := vision.NewModelInfo(*m)
		modelInfo = &info
	}

	agent, err := vision.NewAgent(buildConfig(model, cfg, modelInfo))
	cli.ExitOnError(err, "Error creating agent")

	images, err := loadImages(cfg.args)
	cli.ExitOnError(err, "")

	runAnalysis(ctx, agent, cfg, images, os.Stdout, os.Stderr)
}

type config struct {
	providerName  string
	modelID       string
	prompt        string
	systemPrompt  string
	stream        bool
	structured    bool
	temperature   float64
	maxTokens     int64
	jsonOutput    bool
	timeout       int64
	showVersion   bool
	listProviders bool
	listModels    bool
	providerInfo  bool
	args          []string // positional image paths
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
	listProviders := flagSet.Bool("list-providers", false, "List all supported providers and exit")
	listModels := flagSet.Bool("list-models", false, "List vision-capable models and exit")
	providerInfo := flagSet.Bool("provider-info", false, "Show details for -provider and exit")

	flagSet.Usage = usageFunc(flagSet)

	if err := flagSet.Parse(args); err != nil {
		return nil, fmt.Errorf("parse flags: %w", err)
	}

	return &config{
		providerName:  *providerName,
		modelID:       *modelID,
		prompt:        *prompt,
		systemPrompt:  *systemPrompt,
		stream:        *stream,
		temperature:   *temperature,
		maxTokens:     *maxTokens,
		jsonOutput:    *jsonOutput,
		structured:    *structured,
		timeout:       *timeout,
		showVersion:   *showVersion,
		listProviders: *listProviders,
		listModels:    *listModels,
		providerInfo:  *providerInfo,
		args:          flagSet.Args(),
	}, nil
}

func usageFunc(flagSet *flag.FlagSet) func() {
	name := flagSet.Name()

	return func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options] <image1.png> [image2.png ...]\n\n", name)
		fmt.Fprint(os.Stderr, "Analyze images/screenshots with AI vision models.\n\n")
		fmt.Fprintln(os.Stderr, "Environment variables:")
		fmt.Fprintln(os.Stderr, "  OPENAI_API_KEY         - OpenAI provider")
		fmt.Fprintln(os.Stderr, "  ANTHROPIC_API_KEY      - Anthropic provider")
		fmt.Fprintln(os.Stderr, "  GEMINI_API_KEY         - Google Gemini provider")
		fmt.Fprintln(os.Stderr, "  OPENROUTER_API_KEY     - OpenRouter provider")
		fmt.Fprintln(os.Stderr, "  XAI_API_KEY            - xAI provider")
		fmt.Fprintln(
			os.Stderr,
			"  CATWALK_URL           - Remote catalog server (optional, for auto-updating model list)",
		)
		fmt.Fprintln(os.Stderr, "  OPENAICOMPAT_BASE_URL - openaicompat provider (required for local servers)")
		fmt.Fprint(os.Stderr, "  OPENAICOMPAT_API_KEY  - openaicompat provider (optional)\n\n")
		fmt.Fprintln(os.Stderr, "Use -list-providers to see all 40+ supported providers.")
		fmt.Fprintln(os.Stderr, "Use -list-models to see vision-capable models with pricing.")
		fmt.Fprintln(os.Stderr, "Options:")
		flagSet.PrintDefaults()
		fmt.Fprintln(os.Stderr, "\nExamples:")
		fmt.Fprintf(os.Stderr, "  %s -prompt \"Find UI bugs\" screenshot.png\n", name)
		fmt.Fprintf(os.Stderr, "  %s -list-models -provider openai\n", name)
		fmt.Fprintf(
			os.Stderr,
			"  %s -provider openrouter -model openai/gpt-4o screenshot.png\n",
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

func buildConfig(model fantasy.LanguageModel, cfg *config, modelInfo *vision.ModelInfo) vision.Config {
	config := vision.Config{
		Model:           model,
		Temperature:     cfg.temperature,
		MaxOutputTokens: cfg.maxTokens,
		ModelInfo:       modelInfo,
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
	stdout, stderr io.Writer,
) {
	if cfg.structured {
		runStructured(ctx, agent, cfg, images, stdout, stderr)

		return
	}

	var (
		result *vision.AnalyzeResult
		err    error
	)

	if cfg.stream {
		fmt.Fprintln(stdout, "Analyzing (streaming)...")

		result, err = agent.AnalyzeStream(ctx, cfg.prompt, func(text string) error {
			fmt.Fprint(stdout, text)

			return nil
		}, images...)
	} else {
		fmt.Fprintln(stdout, "Analyzing...")

		result, err = agent.Analyze(ctx, cfg.prompt, images...)
	}

	if err != nil {
		printAnalysisError(stderr, err, cfg.stream)
		os.Exit(1)
	}

	if cfg.jsonOutput {
		printJSON(stdout, result)
	} else {
		printText(stdout, result, cfg.stream)
	}
}

// runStructured performs a structured UI review and prints it as JSON.
func runStructured(
	ctx context.Context,
	agent *vision.Agent,
	cfg *config,
	images []*vision.ImageSource,
	stdout, stderr io.Writer,
) {
	fmt.Fprintln(stdout, "Analyzing (structured)...")

	result, err := vision.AnalyzeStructured[uiReview](ctx, agent, cfg.prompt, images...)
	if err != nil {
		printAnalysisError(stderr, err, false)
		os.Exit(1)
	}

	enc := json.NewEncoder(stdout)
	enc.SetIndent("", "  ")

	if err := enc.Encode(result.Object); err != nil {
		fmt.Fprintln(stderr, "Error encoding JSON:", err)
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
func printAnalysisError(w io.Writer, err error, streamed bool) {
	prefix := "Error:"
	if streamed {
		prefix = "\nError:"
	}

	modelErr, ok := errors.AsType[*vision.ModelError](err)
	if !ok {
		fmt.Fprintln(w, prefix, err)

		return
	}

	advice := adviceForKind(modelErr.Kind)
	if advice != "" {
		fmt.Fprintf(w, "%s [%s] %s\n", prefix, modelErr.Kind, advice)
		fmt.Fprintf(w, "  Details: %s\n", modelErr.Cause)

		return
	}

	fmt.Fprintln(w, prefix, err)
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

func printText(w io.Writer, result *vision.AnalyzeResult, streamed bool) {
	if !streamed {
		fmt.Fprintln(w, "\n--- Analysis ---")
		fmt.Fprintln(w, result.Text)
	}

	fmt.Fprintf(w, "\nTokens used: %d\n", result.Usage.TotalTokens)
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

func printJSON(w io.Writer, result *vision.AnalyzeResult) {
	output := jsonOutput{
		Text: result.Text,
		Usage: jsonUsage{
			InputTokens:  result.Usage.InputTokens,
			OutputTokens: result.Usage.OutputTokens,
			TotalTokens:  result.Usage.TotalTokens,
		},
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")

	if err := enc.Encode(output); err != nil {
		fmt.Fprintln(os.Stderr, "Error encoding JSON:", err)
	}
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
	if err != nil {
		return nil, fmt.Errorf("create openai-compat provider: %w", err)
	}

	return provider, nil
}

// normalizeProviderName maps legacy CLI provider names to their catwalk
// InferenceProvider IDs. This preserves backward compatibility for users
// who pass -provider google (which maps to the catwalk "gemini" provider).
func normalizeProviderName(name string) string {
	switch strings.ToLower(name) {
	case "google":
		return providerGeminiAlias
	default:
		return name
	}
}

func createProvider(svc *catalog.Service, name string) (fantasy.Provider, error) {
	name = normalizeProviderName(name)

	provider, ok := svc.FindProvider(name)
	if !ok {
		if strings.EqualFold(name, "openaicompat") {
			return createOpenAICompatProvider()
		}

		return nil, fmt.Errorf(
			"%s (use -list-providers to see supported providers): %w",
			name,
			errUnknownProvider,
		)
	}

	apiKey, err := catalog.ResolveAPIKey(provider)
	if err != nil {
		if catalog.RequiresAPIKey(provider) {
			return nil, fmt.Errorf("provider %s: %w", provider.ID, err)
		}

		apiKey = ""
	}

	baseURL := catalog.ResolveBaseURL(provider)

	fantasyProvider, err := catalog.BuildProvider(provider, apiKey, baseURL)
	if err != nil {
		return nil, fmt.Errorf("build provider %s: %w", provider.ID, err)
	}

	return fantasyProvider, nil
}
