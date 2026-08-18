// Example: Generate an interactive A2UI surface from a screenshot
//
// A2UI (https://a2ui.org/) is a protocol by which agents send declarative
// component descriptions that clients render natively. This example turns an
// image into a validated A2UI message stream, printed as JSON Lines ready to
// feed any A2UI renderer (paste it into https://a2ui-composer.ag-ui.com/theater
// to see it live).
//
// Usage:
//
//	export OPENAI_API_KEY=your-key
//	go run examples/a2ui/main.go mockup.png
package main

import (
	"fmt"
	"os"

	"github.com/larsartmann/vision-review-agent/internal/cli"
	"github.com/larsartmann/vision-review-agent/pkg/vision/a2ui"
)

func main() {
	ctx, agent := cli.NewAgentFromArgs(2, "You are a UI engineer who recreates interfaces as A2UI components.", 0.2)

	img := cli.LoadImageArg()

	fmt.Println("Generating an A2UI surface from the image...")

	result, err := a2ui.Generate(ctx, agent, a2ui.GenerateOptions{}, img)
	cli.ExitOnError(err, "Error generating A2UI surface")

	wire, err := a2ui.MarshalJSONL(result.Messages)
	cli.ExitOnError(err, "Error encoding messages")

	if _, err := os.Stdout.Write(append(wire, '\n')); err != nil {
		cli.ExitOnError(err, "Error writing messages")
	}

	fmt.Fprintf(os.Stderr, "\n%d messages, %d components, %d tokens\n",
		len(result.Messages), len(result.Spec.Components), result.Usage.TotalTokens)
}
