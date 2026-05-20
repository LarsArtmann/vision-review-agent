package vision

import (
	"errors"
	"testing"
)

func TestDetectImageFormat(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		data []byte
		want string
	}{
		{formatPNG, []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}, formatPNG},
		{formatJPG, []byte{0xFF, 0xD8, 0xFF, 0xE0}, formatJPG},
		{formatGIF, []byte{0x47, 0x49, 0x46, 0x38}, formatGIF},
		{
			formatWebP,
			[]byte{0x52, 0x49, 0x46, 0x46, 0x00, 0x00, 0x00, 0x00, 0x57, 0x45, 0x42, 0x50},
			formatWebP,
		},
		{formatBMP, []byte{0x42, 0x4D, 0x36, 0x00}, formatBMP},
		{
			"riff-not-webp",
			[]byte{0x52, 0x49, 0x46, 0x46, 0x00, 0x00, 0x00, 0x00, 0x57, 0x41, 0x56, 0x45},
			"",
		},
		{"riff-too-short", []byte{0x52, 0x49, 0x46, 0x46, 0x00, 0x00, 0x00, 0x00}, ""},
		{"unknown", []byte{0x00, 0x00, 0x00, 0x00}, ""},
		{"too short", []byte{0x89, 0x50}, ""},
		{"empty", []byte{}, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := DetectImageFormat(tt.data)
			AssertGotEq(t, "DetectImageFormat()", got, tt.want)
		})
	}
}

func TestIsValidImage(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		data []byte
		want bool
	}{
		{"png", []byte{0x89, 0x50, 0x4E, 0x47}, true},
		{"jpg", []byte{0xFF, 0xD8, 0xFF, 0xE0}, true},
		{"gif", []byte{0x47, 0x49, 0x46, 0x38}, true},
		{"unknown", []byte{0x00, 0x00, 0x00, 0x00}, false},
		{"empty", []byte{}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := IsValidImage(tt.data)
			AssertGotEq(t, "IsValidImage()", got, tt.want)
		})
	}
}

func TestValidateImage(t *testing.T) {
	t.Run("valid png", func(t *testing.T) {
		t.Parallel()
		if err := ValidateImage([]byte{0x89, 0x50, 0x4E, 0x47}); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("invalid data", func(t *testing.T) {
		t.Parallel()
		if err := ValidateImage([]byte{0x00, 0x00, 0x00, 0x00}); !errors.Is(err, ErrInvalidImage) {
			t.Errorf("expected ErrInvalidImage, got %v", err)
		}
	})

	t.Run("empty data", func(t *testing.T) {
		t.Parallel()
		if err := ValidateImage([]byte{}); err == nil {
			t.Error("expected error for empty data")
		}
	})
}
