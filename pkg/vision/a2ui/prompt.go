package a2ui

import (
	"fmt"
	"strings"
)

// DefaultTask is the generation instruction used when GenerateOptions.Task
// is empty: faithful reconstruction of the depicted UI.
const DefaultTask = "Recreate the interface shown in the image(s) as faithfully as possible: " +
	"same content, hierarchy, grouping, and reading order."

// catalogSignatures lists the A2UI basic catalog's components with their key
// properties, mirroring the official catalog's LLM instructions. Values in
// square brackets are enums; * marks required properties.
func catalogSignatures() []string {
	return []string{
		"Text: text* (string), variant [h1 h2 h3 h4 h5 caption body]",
		"Heading shortcuts: use Text with variant h1..h5 instead of inventing heading components",
		"Button: child* (id of a Text label or Icon), action* (e.g. {\"event\": {\"name\": \"...\"}}), variant [default primary borderless]",
		"Icon: name* (catalog icon name)",
		"Image: url*, description (alt text), fit [contain cover fill none scaleDown], variant [icon avatar smallFeature mediumFeature largeFeature header]",
		"Divider: axis [horizontal vertical]",
		"Column: children* (vertical stack), align [start center end stretch], justify [start center end spaceBetween spaceAround spaceEvenly stretch]",
		"Row: children* (horizontal stack), align [start center end stretch], justify [center end spaceAround spaceBetween spaceEvenly start stretch]",
		"List: children* (direction-aware stack), direction [vertical horizontal], align [start center end stretch]",
		"Card: child* (single child to elevate)",
		"Modal: content*, trigger* (child id that opens the modal)",
		"CheckBox: label*, value (boolean)",
		"ChoicePicker: label*, options* (array of strings), value, variant [multipleSelection mutuallyExclusive], displayStyle [checkbox chips], filterable (boolean)",
		"TextField: label*, value, variant [shortText longText number obscured], validationRegexp",
		"DateTimeInput: label*, value, min, max, enableDate (boolean), enableTime (boolean)",
		"Slider: label*, value (number), min (number), max (number)",
		"AudioPlayer: url*, description",
		"Video: url*",
	}
}

// BuildPrompt assembles the generation instruction for the model: the
// adjacency-list contract, the basic catalog component signatures, the
// dynamic-value shapes, and the task. The output shape itself (SurfaceSpec)
// is enforced by the structured-output schema; the prompt explains how to
// fill it well.
func BuildPrompt(task string) string {
	if task == "" {
		task = DefaultTask
	}

	var out strings.Builder

	out.WriteString("You are a UI engineer producing an A2UI surface (protocol v0.9.1, basic catalog).\n\n")

	out.WriteString("## Task\n")
	out.WriteString(task)
	out.WriteString("\n\n")

	out.WriteString("## Structure model\n")
	out.WriteString("Components form a FLAT adjacency list, not a nested tree:\n")
	out.WriteString("- Every component has a unique \"id\" and a \"component\" type name.\n")
	out.WriteString("- Exactly one component has id \"root\"; it is the entry point of the UI tree.\n")
	out.WriteString(
		"- Containers reference children BY ID: \"children\": [\"id1\", \"id2\"] for Column/Row/List, \"child\": \"id\" for Card/Button/Modal.\n",
	)
	out.WriteString(
		"- Put catalog-specific properties (text, variant, url, action, ...) in the \"properties\" object.\n",
	)
	out.WriteString("- Use short, descriptive, unique ids (header, title, submit-button, form-column).\n\n")

	out.WriteString("## Dynamic values\n")
	out.WriteString("Property values are either literals (plain strings/numbers/booleans) or bindings ")
	out.WriteString("{\"path\": \"/json/pointer\"} resolving against the data model. ")
	out.WriteString("Prefer literals; bind only values that change (e.g. list data). ")
	out.WriteString("Button actions dispatch server events: {\"event\": {\"name\": \"event-name\"}}.\n\n")

	out.WriteString("## Basic catalog components\n")

	signatures := catalogSignatures()
	for _, signature := range signatures {
		fmt.Fprintf(&out, "- %s\n", signature)
	}

	out.WriteString("\n## Fidelity rules\n")
	out.WriteString("- Match the visual hierarchy: containers reflect grouping, order reflects reading order.\n")
	out.WriteString("- Prefer the simplest component that fits; do not nest containers without visual cause.\n")
	out.WriteString("- Every referenced id must exist in the same components list; never reference across surfaces.\n")
	out.WriteString("- Populate accessibility labels for interactive components when the image conveys purpose.\n")

	return out.String()
}
