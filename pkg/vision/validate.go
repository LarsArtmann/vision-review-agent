package vision

import (
	"bytes"
)

// Image format constants for magic byte signatures.
const (
	formatPNG  = "png"
	formatJPG  = "jpg"
	formatGIF  = "gif"
	formatWebP = "webp"
	formatBMP  = "bmp"
)

// imageSignatures returns the magic byte signatures for supported image formats.
func imageSignatures() []struct {
	format    string
	signature []byte
} {
	return []struct {
		format    string
		signature []byte
	}{
		{formatPNG, []byte{0x89, 0x50, 0x4E, 0x47}},
		{formatJPG, []byte{0xFF, 0xD8, 0xFF}},
		{formatGIF, []byte{0x47, 0x49, 0x46}},
		{formatWebP, []byte{0x52, 0x49, 0x46, 0x46}}, // RIFF header, WebP uses this
		{formatBMP, []byte{0x42, 0x4D}},
	}
}

// DetectImageFormat attempts to detect the image format from magic bytes.
// Returns the format name (e.g., "png", "jpg") or an empty string if unknown.
func DetectImageFormat(data []byte) string {
	if len(data) < 4 {
		return ""
	}
	for _, sig := range imageSignatures() {
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
		return ErrEmptyImageData
	}
	if !IsValidImage(data) {
		return ErrInvalidImage
	}
	return nil
}
