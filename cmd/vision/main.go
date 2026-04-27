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
	"flag"
	"fmt"
	"os"
	"strings"

	"charm.land/fantasy"
	"charm.land/fantasy/providers/openai"
	"charm.land/fantasy/providers/openrouter"
	"github.com/larsartmann/vision-review-agent/vision"
)

const version = "0.1.0"

func main() {
	var (
		providerName = flag.String("provider", "openai", "Provider: openai, openrouter")
		modelID      = flag.String("model", "gpt-4o", "Model ID (e.g., gpt-4o, openai/gpt-4o)")
		prompt       = flag.String("prompt", "Describe what you see in this image.", "Analysis prompt")
		systemPrompt = flag.String("system", "", "Custom system prompt (optional)")
		stream       = flag.Bool("stream", false, "Stream the response")
		temperature  = flag.Float64("temperature", 0.3, "Temperature (0.0-2.0)")
		maxTokens    = flag.Int64("max-tokens", 0, "Max output tokens (0 = unlimited)")
		jsonOutput   = flag.Bool("json", false, "Output result as JSON")
		timeout      = flag.Duration("timeout", 0, "Request timeout (e.g., 30s, 2m)")
		showVersion  = flag.Bool("version", false, "Show version and exit")
	)
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options] <image1.png> [image2.png ...]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Analyze images/screenshots with AI vision models.\n\n")
		fmt.Fprintf(os.Stderr, "Environment variables:\n")
		fmt.Fprintf(os.Stderr, "  OPENAI_API_KEY     - Required for OpenAI provider\n")
		fmt.Fprintf(os.Stderr, "  OPENROUTER_API_KEY - Required for OpenRouter provider\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  %s -prompt \"Find UI bugs\" screenshot.png\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -provider openrouter -model anthropic/claude-3.5-sonnet screenshot.png\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -stream -prompt \"Describe this\" *.png\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -json -prompt \"Find bugs\" screenshot.png | jq '.text'\n", os.Args[0])
	}
	flag.Parse()

	if *showVersion {
		fmt.Println("vision", version)
		os.Exit(0)
	}

	if flag.NArg() == 0 {
		flag.Usage()
		os.Exit(1)
	}

	ctx := context.Background()

	// Setup provider
	provider, err := createProvider(*providerName)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}

	// Get model
	model, err := provider.LanguageModel(ctx, *modelID)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error getting model:", err)
		os.Exit(1)
	}

	// Build agent config
	config := vision.Config{
		Model:           model,
		Temperature:     *temperature,
		MaxOutputTokens: *maxTokens,
	}

	if *systemPrompt != "" {
		config.SystemPrompt = *systemPrompt
	}
	if *timeout > 0 {
		config.RequestTimeout = *timeout
	}

	agent, err := vision.NewAgent(config)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error creating agent:", err)
		os.Exit(1)
	}

	// Load images
	images := make([]*vision.ImageSource, flag.NArg())
	for i := 0; i < flag.NArg(); i++ {
		img, err := vision.LoadImageFromFile(flag.Arg(i))
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading %s: %v\n", flag.Arg(i), err)
			os.Exit(1)
		}
		images[i] = img
	}

	// Analyze
	if *stream {
		fmt.Println("Analyzing (streaming)...")
		result, err := agent.AnalyzeStream(ctx, *prompt, func(text string) error {
			fmt.Print(text)
			return nil
		}, images...)
		if err != nil {
			fmt.Fprintln(os.Stderr, "\nError:", err)
			os.Exit(1)
		}
		if *jsonOutput {
			printJSON(result)
		} else {
			fmt.Printf("\n\nTokens used: %d\n", result.Usage.TotalTokens)
		}
	} else {
		fmt.Println("Analyzing...")
		result, err := agent.Analyze(ctx, *prompt, images...)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error:", err)
			os.Exit(1)
		}
		if *jsonOutput {
			printJSON(result)
		} else {
			fmt.Println("\n--- Analysis ---")
			fmt.Println(result.Text)
			fmt.Printf("\nTokens used: %d\n", result.Usage.TotalTokens)
		}
	}
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
	_ = enc.Encode(output)
}

func createProvider(name string) (fantasy.Provider, error) {
	switch strings.ToLower(name) {
	case "openai":
		apiKey := os.Getenv("OPENAI_API_KEY")
		if apiKey == "" {
			return nil, fmt.Errorf("OPENAI_API_KEY environment variable not set")
		}
		return openai.New(openai.WithAPIKey(apiKey))
	case "openrouter":
		apiKey := os.Getenv("OPENROUTER_API_KEY")
		if apiKey == "" {
			return nil, fmt.Errorf("OPENROUTER_API_KEY environment variable not set")
		}
		return openrouter.New(openrouter.WithAPIKey(apiKey))
	default:
		return nil, fmt.Errorf("unknown provider: %s (supported: openai, openrouter)", name)
	}
}
