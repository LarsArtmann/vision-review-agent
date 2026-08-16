package reviewed

import (
	"fmt"
	"strings"
)

// ReviewSystemPrompt is the system prompt for single-view reviews. The model
// is caption-tuned (descriptive by training), so the prompt steers it from
// description to critical judgment and pins a strict output contract.
const ReviewSystemPrompt = `You are a meticulous UI review engine. You look at screenshots of user interfaces and judge them like a senior product designer. You only comment on what is actually visible. You never invent elements. You are concrete, specific, and terse.`

// CompareSystemPrompt is the system prompt for A/B comparisons.
const CompareSystemPrompt = `You are a meticulous UI review engine comparing two versions of the same view. You identify what changed, judge whether each change improved or hurt the interface, and only comment on visible differences. You are concrete, specific, and terse.`

// scoreContractLine is the mandatory final line of every model answer.
const scoreContractLine = "Score: N/10"

// ReviewPrompt builds the user prompt for reviewing one screenshot.
func ReviewPrompt(viewKey ViewKey) string {
	var prompt strings.Builder

	prompt.WriteString(viewContext(viewKey))
	prompt.WriteString(`
Review this UI screenshot.

Answer with EXACTLY this markdown structure:

## Summary
One short paragraph describing the visible UI.

## Strengths
- Bullet list of things done well (omit the heading entirely if there are none).

## Issues
- Bullet list of concrete problems: layout breakage, misalignment, low contrast,
  overflow, clipped or truncated text, inconsistent spacing, unreadable labels,
  broken images, missing states.

## Recommendations
- Bullet list of the most impactful concrete fixes, most important first.

`)
	scoreRules(&prompt, false)

	return prompt.String()
}

// ComparePrompt builds the user prompt for a BEFORE/AFTER comparison. The
// first image is the before version, the second the after version.
func ComparePrompt(viewKey ViewKey) string {
	var prompt strings.Builder

	prompt.WriteString(viewContext(viewKey))
	prompt.WriteString(`
Compare two versions of this view. Image 1 is the BEFORE version. Image 2 is the AFTER version.

Answer with EXACTLY this markdown structure:

## What improved
- Bullet list of changes that made the UI better (omit the heading if none).

## What got worse
- Bullet list of changes that made the UI worse (omit the heading if none).

## Unchanged problems
- Bullet list of issues present in both versions (omit the heading if none).

## Verdict
One short paragraph: is the AFTER version better overall, and why?

`)
	scoreRules(&prompt, true)

	return prompt.String()
}

// scoreRules appends the scoring contract to a prompt. ratesAfter switches
// the wording for comparisons, where the score rates the AFTER image.
func scoreRules(prompt *strings.Builder, ratesAfter bool) {
	prompt.WriteString("Rules:\n")

	if ratesAfter {
		prompt.WriteString("- The FINAL line MUST be exactly \"" + scoreContractLine + "\" where N is an integer 0-10\n")
		prompt.WriteString("  rating the AFTER (second) image.\n")
	} else {
		prompt.WriteString("- The FINAL line MUST be exactly \"" + scoreContractLine + "\" where N is an integer 0-10.\n")
	}

	prompt.WriteString("- Be concrete and reference what is actually visible. Do not invent elements.\n")
	prompt.WriteString("- 10 means flawless and ready to ship, 0 means unusable. Use the full range.\n")
}

// viewContext renders the identifying context line(s) for a view.
func viewContext(viewKey ViewKey) string {
	return fmt.Sprintf("View: page %q, theme %q, viewport %q.\n", viewKey.Page, viewKey.Theme, viewKey.Viewport)
}
