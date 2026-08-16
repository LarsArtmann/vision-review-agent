package reviewed

import (
	"testing"
	"time"
)

func TestViewStateUpdatedAt(t *testing.T) {
	t.Parallel()

	capturedAt := testStamp
	reviewedAt := testStamp.Add(time.Hour)
	staleReviewAt := testStamp.Add(-time.Hour)

	tests := []struct {
		name  string
		state ViewState
		want  time.Time
	}{
		{
			name:  "never captured is zero",
			state: initialViewState(),
			want:  time.Time{},
		},
		{
			name: "captured but unreviewed falls back to capture time",
			state: ViewState{
				CapturedAt: capturedAt,
				Captures:   1,
			},
			want: capturedAt,
		},
		{
			name: "review of the current capture wins",
			state: ViewState{
				CapturedAt:  capturedAt,
				SHA256:      "sha",
				ReviewedSHA: "sha",
				LastReview:  &Reviewed{ReviewedAt: reviewedAt},
			},
			want: reviewedAt,
		},
		{
			name: "stale review after a newer capture falls back to capture time",
			state: ViewState{
				CapturedAt:  capturedAt,
				SHA256:      "sha",
				ReviewedSHA: "",
				LastReview:  &Reviewed{ReviewedAt: staleReviewAt},
			},
			want: capturedAt,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := tt.state.UpdatedAt(); !got.Equal(tt.want) {
				t.Fatalf("UpdatedAt() = %v, want %v", got, tt.want)
			}
		})
	}
}
