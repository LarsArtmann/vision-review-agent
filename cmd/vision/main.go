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
	"charm.land/fantasy/providers/openai"
	"charm.land/fantasy/providers/openrouter"
	"github.com/larsartmann/vision-review-agent/internal/cli"
	"github.com/larsartmann/vision-review-agent/pkg/vision"
)

const version = "0.1.0"

func main() {
	cfg, err := parseFlags()
	cli.ExitOnError(err, "")

	ctx := context.Background()
	provider, err := createProvider(cfg.providerName)
	cli.ExitOnError(err, "")

	model, err := provider.LanguageModel(ctx, cfg.modelID)
	cli.ExitOnError(err, "Error getting model")

	agent, err := vision.NewAgent(buildConfig(model, cfg))
	cli.ExitOnError(err, "Error creating agent")

	images, err := loadImages()
	cli.ExitOnError(err, "")

	runAnalysis(ctx, agent, cfg, images)
}

type config struct {
	providerName string
	modelID      string
	prompt       string
	systemPrompt string
	stream       bool
	temperature  float64
	maxTokens    int64
	jsonOutput   bool
	timeout      int64
}

func parseFlags() (*config, error) {
	var (
		providerName = flag.String("provider", "openai", "Provider: openai, openrouter")
		modelID      = flag.String("model", "gpt-4o", "Model ID (e.g., gpt-4o, openai/gpt-4o)")
		prompt       = flag.String(
			"prompt",
			"Describe what you see in this image.",
			"Analysis prompt",
		)
		systemPrompt = flag.String("system", "", "Custom system prompt (optional)")
		stream       = flag.Bool("stream", false, "Stream the response")
		temperature  = flag.Float64("temperature", 0.3, "Temperature (0.0-2.0)")
		maxTokens    = flag.Int64("max-tokens", 0, "Max output tokens (0 = unlimited)")
		jsonOutput   = flag.Bool("json", false, "Output result as JSON")
		timeout      = flag.Int64("timeout", 0, "Request timeout in seconds (0 = unlimited)")
		showVersion  = flag.Bool("version", false, "Show version and exit")
	)
	flag.Usage = usageFunc(os.Args[0])
	flag.Parse()

	if *showVersion {
		fmt.Println("vision", version)
		os.Exit(0)
	}

	if flag.NArg() == 0 {
		flag.Usage()
		os.Exit(1)
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
		timeout:      *timeout,
	}, nil
}

func usageFunc(name string) func() {
	return func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options] <image1.png> [image2.png ...]\n\n", name)
		fmt.Fprint(os.Stderr, "Analyze images/screenshots with AI vision models.\n\n")
		fmt.Fprintln(os.Stderr, "Environment variables:")
		fmt.Fprintln(os.Stderr, "  OPENAI_API_KEY     - Required for OpenAI provider")
		fmt.Fprint(os.Stderr, "  OPENROUTER_API_KEY - Required for OpenRouter provider\n\n")
		fmt.Fprintln(os.Stderr, "Options:")
		flag.PrintDefaults()
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

func loadImages() ([]*vision.ImageSource, error) {
	images := make([]*vision.ImageSource, flag.NArg())
	for i := range flag.NArg() {
		img, err := vision.LoadImageFromFile(flag.Arg(i))
		if err != nil {
			return nil, fmt.Errorf("loading %s: %w", flag.Arg(i), err)
		}
		images[i] = img
	}
	return images, nil
}

func runAnalysis(
	ctx context.Context,
	agent *vision.Agent,
	cfg *config,
	images []*vision.ImageSource,
) {
	var analyzeFn func() (*vision.AnalyzeResult, error)

	if cfg.stream {
		fmt.Println("Analyzing (streaming)...")
		analyzeFn = func() (*vision.AnalyzeResult, error) {
			return agent.AnalyzeStream(ctx, cfg.prompt, func(text string) error {
				fmt.Print(text)
				return nil
			}, images...)
		}
	} else {
		fmt.Println("Analyzing...")
		analyzeFn = func() (*vision.AnalyzeResult, error) {
			return agent.Analyze(ctx, cfg.prompt, images...)
		}
	}

	result, err := analyzeFn()
	if err != nil {
		if cfg.stream {
			fmt.Fprintln(os.Stderr, "\nError:", err)
		} else {
			fmt.Fprintln(os.Stderr, "Error:", err)
		}
		os.Exit(1)
	}

	if cfg.jsonOutput {
		printJSON(result)
	} else {
		printText(result, cfg.stream)
	}
}

func printText(result *vision.AnalyzeResult, streamed bool) {
	if !streamed {
		fmt.Println("\n--- Analysis ---")
		fmt.Println(result.Text)
	}
	fmt.Printf("\nTokens used: %d\n", result.Usage.TotalTokens)
}

func printJSON(result *vision.AnalyzeResult) {
	output := struct {
		Text  string `json:"text"`
		Usage struct {
			InputTokens  int64 `json:"input_tokens"`
			OutputTokens int64 `json:"output_tokens"`
			TotalTokens  int64 `json:"total_tokens"`
		} `json:"usage"`
	}{
		Text: result.Text,
		Usage: struct {
			InputTokens  int64 `json:"input_tokens"`
			OutputTokens int64 `json:"output_tokens"`
			TotalTokens  int64 `json:"total_tokens"`
		}{
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

func createProvider(name string) (fantasy.Provider, error) {
	switch strings.ToLower(name) {
	case "openai":
		apiKey := os.Getenv("OPENAI_API_KEY")
		if apiKey == "" {
			return nil, errors.New("OPENAI_API_KEY environment variable not set")
		}
		provider, err := openai.New(openai.WithAPIKey(apiKey))
		if err != nil {
			return nil, fmt.Errorf("create openai provider: %w", err)
		}
		return provider, nil
	case "openrouter":
		apiKey := os.Getenv("OPENROUTER_API_KEY")
		if apiKey == "" {
			return nil, errors.New("OPENROUTER_API_KEY environment variable not set")
		}
		provider, err := openrouter.New(openrouter.WithAPIKey(apiKey))
		if err != nil {
			return nil, fmt.Errorf("create openrouter provider: %w", err)
		}
		return provider, nil
	default:
		return nil, fmt.Errorf("unknown provider: %s (supported: openai, openrouter)", name)
	}
}
