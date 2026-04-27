package vision

import (
	"errors"
	"testing"
)

func TestDetectImageFormat(t *testing.T) {
	tests := []struct {
		name string
		data []byte
		want string
	}{
		{"png", []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}, "png"},
		{"jpg", []byte{0xFF, 0xD8, 0xFF, 0xE0}, "jpg"},
		{"gif", []byte{0x47, 0x49, 0x46, 0x38}, "gif"},
		{
			"webp",
			[]byte{0x52, 0x49, 0x46, 0x46, 0x00, 0x00, 0x00, 0x00, 0x57, 0x45, 0x42, 0x50},
			"webp",
		},
		{"bmp", []byte{0x42, 0x4D, 0x36, 0x00}, "bmp"},
		{"unknown", []byte{0x00, 0x00, 0x00, 0x00}, ""},
		{"too short", []byte{0x89, 0x50}, ""},
		{"empty", []byte{}, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DetectImageFormat(tt.data)
			if got != tt.want {
				t.Errorf("DetectImageFormat() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestIsValidImage(t *testing.T) {
	if !IsValidImage([]byte{0x89, 0x50, 0x4E, 0x47}) {
		t.Error("expected PNG to be valid")
	}
	if IsValidImage([]byte{0x00, 0x00, 0x00, 0x00}) {
		t.Error("expected unknown to be invalid")
	}
	if IsValidImage([]byte{}) {
		t.Error("expected empty to be invalid")
	}
}

func TestValidateImage(t *testing.T) {
	t.Run("valid png", func(t *testing.T) {
		if err := ValidateImage([]byte{0x89, 0x50, 0x4E, 0x47}); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("invalid data", func(t *testing.T) {
		if err := ValidateImage([]byte{0x00, 0x00, 0x00, 0x00}); !errors.Is(err, ErrInvalidImage) {
			t.Errorf("expected ErrInvalidImage, got %v", err)
		}
	})

	t.Run("empty data", func(t *testing.T) {
		if err := ValidateImage([]byte{}); err == nil {
			t.Error("expected error for empty data")
		}
	})
}
