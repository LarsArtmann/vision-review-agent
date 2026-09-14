// Package uireview provides the shared structured-output schema used by the
// structured-output examples (examples/structured and examples/structured-stream).
// It exists as an internal package so both examples can reuse a single canonical
// definition of UIReview / Issue instead of duplicating the schema.
package uireview

// UIReview is the structured output type produced by a UI/UX review.
type UIReview struct {
	Layout      string   `json:"layout"      description:"Brief description of the overall layout"`
	Components  []string `json:"components"  description:"List of UI components identified"`
	Issues      []Issue  `json:"issues"      description:"List of issues found"`
	Score       int      `json:"score"       description:"Overall UX score from 1-10"`
	Suggestions []string `json:"suggestions" description:"Actionable improvement suggestions"`
}

// Issue represents a single UI issue found during review.
type Issue struct {
	Severity    string `json:"severity"    description:"Severity: critical, major, minor, or info"`
	Component   string `json:"component"   description:"Which component has the issue"`
	Description string `json:"description" description:"Detailed description of the issue"`
}
