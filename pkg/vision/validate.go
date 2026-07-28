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

	// minImageHeaderBytes is the minimum data length needed to check any
	// magic-byte signature. DetectImageFormat returns empty below this.
	minImageHeaderBytes = 4
)

// webpSignature returns the RIFF header shared by all RIFF containers.
func webpSignature() []byte { return []byte{0x52, 0x49, 0x46, 0x46} }

// webpMagic returns the WEBP identifier at offset 8 in a WebP file.
func webpMagic() []byte { return []byte{0x57, 0x45, 0x42, 0x50} }

// imageSignature is one entry in the table of supported image format magic bytes.
type imageSignature struct {
	format    string
	signature []byte
}

// imageSignatures returns the magic byte signatures for supported image formats.
func imageSignatures() []imageSignature {
	return []imageSignature{
		{formatPNG, []byte{0x89, 0x50, 0x4E, 0x47}},
		{formatJPG, []byte{0xFF, 0xD8, 0xFF}},
		{formatGIF, []byte{0x47, 0x49, 0x46}},
		{formatWebP, webpSignature()},
		{formatBMP, []byte{0x42, 0x4D}},
	}
}

// DetectImageFormat attempts to detect the image format from magic bytes.
// Returns the format name (e.g., "png", "jpg") or an empty string if unknown.
// For WebP, also verifies the WEBP identifier at offset 8 to reject other
// RIFF containers like WAV and AVI.
func DetectImageFormat(data []byte) string {
	if len(data) < minImageHeaderBytes {
		return ""
	}

	for _, sig := range imageSignatures() {
		if bytes.HasPrefix(data, sig.signature) {
			if sig.format == formatWebP {
				if len(data) < 12 || !bytes.Equal(data[8:12], webpMagic()) {
					return ""
				}
			}

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
