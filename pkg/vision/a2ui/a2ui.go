// Package a2ui provides A2UI (https://a2ui.org/) support for vision agents:
// it turns screenshots and design images into valid, interactive A2UI
// surfaces that render natively in any A2UI client.
//
// A2UI is a protocol by which agents send declarative component descriptions
// (not code) that clients render with their own native widgets. This package
// covers the agent side of the protocol for spec version v0.9.1 (current):
//
//   - Wire types for the four server-to-client message kinds plus a JSON Lines
//     codec, the on-the-wire transport encoding (see MarshalJSONL).
//   - Compile, which converts a simplified LLM-facing SurfaceSpec into a
//     validated message sequence.
//   - Generate, the vision bridge: any vision model from the parent SDK can
//     produce an A2UI surface from one or more images via
//     vision.AnalyzeStructured.
//   - Validate, structural validation of message streams (root component,
//     unique IDs, resolvable child references, acyclicity, message ordering).
//
// Basic usage:
//
//	agent, err := vision.NewAgent(vision.Config{Model: model})
//	img, _ := vision.LoadImageFromFile("mockup.png")
//	result, err := a2ui.Generate(ctx, agent, a2ui.GenerateOptions{}, img)
//	if err != nil {
//		log.Fatal(err)
//	}
//	wire, _ := a2ui.MarshalJSONL(result.Messages)
//	os.Stdout.Write(wire)
//
// The generated messages implement the v0.9.1 message shapes
// (https://a2ui.org/specification/v0_9_1/); components come from the A2UI
// basic catalog. Conformance against the official v0.9.1 JSON schemas is
// machine-checked in every test run (conformance_test.go, with the schemas
// pinned under testdata/official/).
package a2ui

// VersionV09 is the v0.9 protocol version, accepted on input.
const VersionV09 = "v0.9"

// VersionV091 is the A2UI protocol version emitted by this package.
const VersionV091 = "v0.9.1"

// DefaultCatalogID identifies the A2UI basic catalog. Compile and Generate
// fall back to it when no catalog is specified.
const DefaultCatalogID = "https://a2ui.org/specification/v0_9/catalogs/basic/catalog.json"

// RootID is the conventional ID of the root component of a surface. Exactly
// one component in an updateComponents message must use it.
const RootID = "root"
