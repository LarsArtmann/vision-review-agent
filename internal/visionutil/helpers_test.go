package visionutil

import (
	"testing"
)

type testReview struct {
	Layout string `json:"layout"`
	Score  int    `json:"score"`
}

func TestUnmarshalToType(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		obj     any
		wantErr bool
	}{
		{
			name:    "valid object",
			obj:     map[string]any{"layout": "test", "score": 5},
			wantErr: false,
		},
		{
			name:    "nil object",
			obj:     nil,
			wantErr: false,
		},
	}

	for _, tc := range cases {
		// capture range variable
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var result testReview

			err := UnmarshalToType(tc.obj, &result)
			if tc.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}

				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)

				return
			}

			if tc.name == "valid object" {
				if result.Layout != "test" {
					t.Errorf("expected layout 'test', got %q", result.Layout)
				}

				if result.Score != 5 {
					t.Errorf("expected score 5, got %d", result.Score)
				}
			}
		})
	}
}

func TestAppendSystemAndPrompt(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		systemPrompt string
		userPrompt   string
		wantLen      int
	}{
		{
			name:         "without system prompt",
			systemPrompt: "",
			userPrompt:   "hello",
			wantLen:      1,
		},
		{
			name:         "with system prompt",
			systemPrompt: "system",
			userPrompt:   "hello",
			wantLen:      2,
		},
	}

	for _, tt := range tests {
		// capture range variable
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			prompt := AppendSystemAndPrompt(tt.systemPrompt, tt.userPrompt, nil)
			if len(prompt) != tt.wantLen {
				t.Errorf("expected prompt length %d, got %d", tt.wantLen, len(prompt))
			}
		})
	}
}
