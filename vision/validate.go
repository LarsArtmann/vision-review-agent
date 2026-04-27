package vision

import (
	"bytes"
	"errors"
)

// imageSignatures maps image formats to their magic byte signatures.
var imageSignatures = []struct {
	format    string
	signature []byte
}{
	{"png", []byte{0x89, 0x50, 0x4E, 0x47}},
	{"jpg", []byte{0xFF, 0xD8, 0xFF}},
	{"gif", []byte{0x47, 0x49, 0x46}},
	{"webp", []byte{0x52, 0x49, 0x46, 0x46}}, // RIFF header, WebP uses this
	{"bmp", []byte{0x42, 0x4D}},
}

// ErrInvalidImage is returned when the data does not match any known image format.
var ErrInvalidImage = errors.New("vision agent: data does not appear to be a valid image")

// DetectImageFormat attempts to detect the image format from magic bytes.
// Returns the format name (e.g., "png", "jpg") or an empty string if unknown.
func DetectImageFormat(data []byte) string {
	if len(data) < 4 {
		return ""
	}
	for _, sig := range imageSignatures {
		if bytes.HasPrefix(data, sig.signature) {
			return sig.format
		}
	}
	return ""
}

// IsValidImage checks if the data appears to be a valid image by examining magic bytes.
func IsValidImage(data []byte) bool {
	return DetectImageFormat(data) != ""
}

// ValidateImage checks if the image data has a recognized format.
// Returns ErrInvalidImage if the data does not match any known image signature.
func ValidateImage(data []byte) error {
	if len(data) == 0 {
		return errors.New("vision agent: image data is empty")
	}
	if !IsValidImage(data) {
		return ErrInvalidImage
	}
	return nil
}
