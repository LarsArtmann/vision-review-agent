// Package uireview provides the shared structured-output schema used by the
// structured-output examples (examples/structured and examples/structured-stream).
// It exists as an internal package so both examples can reuse a single canonical
// definition of UIReview / Issue instead of duplicating the schema.
package uireview

// UIReview is the structured output type produced by a UI/UX review.
type UIReview struct {
	Layout      string   `description:"Brief description of the overall layout" json:"layout"`
	Components  []string `description:"List of UI components identified"        json:"components"`
	Issues      []Issue  `description:"List of issues found"                    json:"issues"`
	Score       int      `description:"Overall UX score from 1-10"              json:"score"`
	Suggestions []string `description:"Actionable improvement suggestions"      json:"suggestions"`
}

// Issue represents a single UI issue found during review.
type Issue struct {
	Severity    string `description:"Severity: critical, major, minor, or info" json:"severity"`
	Component   string `description:"Which component has the issue"             json:"component"`
	Description string `description:"Detailed description of the issue"         json:"description"`
}
