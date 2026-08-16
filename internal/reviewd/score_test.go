package reviewed

import (
	"fmt"
	"os"
	"testing"
)

func writeFile(path string, data []byte) error {
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}

	return nil
}

func TestExtractScore(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		markdown string
		want     int
	}{
		{name: "plain contract line", markdown: "## Summary\nok\n\nScore: 7/10", want: 7},
		{name: "zero", markdown: "Score: 0/10", want: 0},
		{name: "ten", markdown: "Score: 10/10", want: 10},
		{name: "bolded", markdown: "**Score: 8/10**", want: 8},
		{name: "case insensitive", markdown: "score: 4/10", want: 4},
		{name: "spaced", markdown: "Score:  3 / 10", want: 3},
		{name: "indented", markdown: "  Score: 5/10", want: 5},
		{name: "last match wins", markdown: "Quoting \"Score: 2/10\" format.\n\nScore: 6/10", want: 6},
		{name: "no score", markdown: "## Summary\nnothing", want: ScoreUnknown},
		{name: "out of range high", markdown: "Score: 11/10", want: ScoreUnknown},
		{name: "out of range negative", markdown: "Score: -1/10", want: ScoreUnknown},
		{name: "not a slash ten", markdown: "Score: 7 out of 10", want: ScoreUnknown},
		{name: "mid text no line start", markdown: "the Score: 7/10 inside", want: ScoreUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := ExtractScore(tt.markdown); got != tt.want {
				t.Fatalf("ExtractScore(%q) = %d, want %d", tt.markdown, got, tt.want)
			}
		})
	}
}
