package reviewed

import (
	"regexp"
	"strconv"
	"strings"
)

// scoreLineRegex matches the score contract line, tolerating markdown bolding,
// case differences, and stray whitespace: "Score: 7/10", "**Score: 7/10**",
// "score: 7 / 10".
var scoreLineRegex = regexp.MustCompile(`(?im)^\s*\**\s*score\s*\**\s*:\s*(\d{1,2})\s*/\s*10`)

// ExtractScore pulls the score out of a review markdown body. The LAST match
// wins (the contract puts the score at the end; earlier mentions may quote the
// format). Returns ScoreUnknown when no valid 0-10 score is present.
func ExtractScore(markdown string) int {
	matches := scoreLineRegex.FindAllStringSubmatch(markdown, -1)
	if len(matches) == 0 {
		return ScoreUnknown
	}

	score, err := strconv.Atoi(strings.TrimSpace(matches[len(matches)-1][1]))
	if err != nil || score < 0 || score > 10 {
		return ScoreUnknown
	}

	return score
}

// StripScoreLines removes the score contract lines from a review body after
// extraction: the rendered file carries the score in its header, so keeping
// the model's trailing "Score: N/10" would show it twice.
func StripScoreLines(markdown string) string {
	return strings.TrimSpace(scoreLineRegex.ReplaceAllString(markdown, ""))
}
