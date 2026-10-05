package reviewed

import (
	"encoding/json"
	"testing"
	"time"
)

func TestPayloadJSONRoundtrip(t *testing.T) {
	t.Parallel()

	stamp := time.Date(2026, 8, 16, 20, 0, 0, 0, time.UTC)

	tests := []struct {
		name  string
		value any
	}{
		{
			name:  "captured",
			value: &Captured{SourcePath: "/a/b.png", BlobPath: "/x/1.png", SHA256: "abc", CapturedAt: stamp},
		},
		{
			name:  "reviewed",
			value: &Reviewed{SHA256: "abc", Model: "m", Markdown: "# hi", Score: 7, ReviewedAt: stamp},
		},
		{
			name:  "reviewed unknown score",
			value: &Reviewed{SHA256: "abc", Model: "m", Markdown: "", Score: ScoreUnknown, ReviewedAt: stamp},
		},
		{
			name: "compared",
			value: &Compared{
				BeforeSHA256: "a", BeforeBlobPath: "/x/a.png",
				AfterSHA256: "b", AfterBlobPath: "/x/b.png",
				Model: "m", Markdown: "diff", ComparedAt: stamp,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			encoded, err := json.Marshal(tt.value)
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}

			if err := json.Unmarshal(encoded, tt.value); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}

			reencoded, err := json.Marshal(tt.value)
			if err != nil {
				t.Fatalf("remarshal: %v", err)
			}

			if string(reencoded) != string(encoded) {
				t.Fatalf("roundtrip mismatch:\n got %s\nwant %s", reencoded, encoded)
			}
		})
	}
}

func TestEventTypeConstants(t *testing.T) {
	t.Parallel()

	if EventViewCaptured != "view.captured" {
		t.Fatalf("EventViewCaptured = %q", EventViewCaptured)
	}

	if EventViewReviewed != "view.reviewed" {
		t.Fatalf("EventViewReviewed = %q", EventViewReviewed)
	}

	if EventViewCompared != "view.compared" {
		t.Fatalf("EventViewCompared = %q", EventViewCompared)
	}
}
