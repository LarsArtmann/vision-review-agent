package reviewed

import (
	"errors"
	"testing"
)

func TestParseViewKey(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    ViewKey
		wantErr error
	}{
		{
			name:  "conforming name",
			input: "Settings--dark--desktop.png",
			want:  ViewKey{Page: "Settings", Theme: "dark", Viewport: "desktop"},
		},
		{
			name:  "mobile light",
			input: "Login--light--mobile.png",
			want:  ViewKey{Page: "Login", Theme: "light", Viewport: "mobile"},
		},
		{
			name:  "page containing separator keeps last two segments",
			input: "Foo--Bar--light--mobile.png",
			want:  ViewKey{Page: "Foo--Bar", Theme: "light", Viewport: "mobile"},
		},
		{
			name:  "no separators falls back",
			input: "dashboard.png",
			want:  ViewKey{Page: "dashboard", Theme: FallbackTheme, Viewport: FallbackViewport},
		},
		{
			name:  "one separator falls back viewport only",
			input: "Settings--dark.png",
			want:  ViewKey{Page: "Settings", Theme: "dark", Viewport: FallbackViewport},
		},
		{
			name:  "uppercase extension still stripped",
			input: "Home--dark--desktop.PNG",
			want:  ViewKey{Page: "Home", Theme: "dark", Viewport: "desktop"},
		},
		{
			name:  "stem without extension",
			input: "Settings--dark--desktop",
			want:  ViewKey{Page: "Settings", Theme: "dark", Viewport: "desktop"},
		},
		{
			name:    "empty filename errors",
			input:   "",
			wantErr: ErrEmptyViewKey,
		},
		{
			name:    "extension only errors",
			input:   ".png",
			wantErr: ErrEmptyViewKey,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseViewKey(tt.input)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("ParseViewKey(%q) error = %v, want %v", tt.input, err, tt.wantErr)
				}

				return
			}

			if err != nil {
				t.Fatalf("ParseViewKey(%q) unexpected error: %v", tt.input, err)
			}

			if got != tt.want {
				t.Fatalf("ParseViewKey(%q) = %+v, want %+v", tt.input, got, tt.want)
			}
		})
	}
}

func TestViewKeyStringRoundtrip(t *testing.T) {
	tests := []string{
		"Settings--dark--desktop.png",
		"Login--light--mobile.png",
		"dashboard.png",
		"Foo--Bar--light--mobile.png",
	}

	for _, input := range tests {
		key, err := ParseViewKey(input)
		if err != nil {
			t.Fatalf("ParseViewKey(%q): %v", input, err)
		}

		again, err := ParseViewKey(key.String())
		if err != nil {
			t.Fatalf("ParseViewKey(%q) roundtrip: %v", key.String(), err)
		}

		if again != key {
			t.Fatalf("roundtrip mismatch: got %+v, want %+v", again, key)
		}
	}
}

func TestViewStreamID(t *testing.T) {
	key := ViewKey{Page: "Settings", Theme: "dark", Viewport: "desktop"}

	streamID, err := ViewStreamID("discordsync", key)
	if err != nil {
		t.Fatalf("ViewStreamID: %v", err)
	}

	if got := streamID.String(); got != "discordsync:Settings--dark--desktop" {
		t.Fatalf("stream ID = %q, want %q", got, "discordsync:Settings--dark--desktop")
	}
}

func TestViewStreamIDRejectsEmptyProject(t *testing.T) {
	key := ViewKey{Page: "Settings", Theme: "dark", Viewport: "desktop"}

	if _, err := ViewStreamID("", key); err == nil {
		t.Fatal("ViewStreamID with empty project should error")
	}
}
