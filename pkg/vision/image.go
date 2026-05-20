package vision

import (
	"fmt"
	"io"
	"mime"
	"os"
	"path/filepath"
)

// MediaType represents a valid image media type.
type MediaType string

// Media type constants for supported image formats.
const (
	MediaTypePNG  MediaType = "image/png"
	MediaTypeJPEG MediaType = "image/jpeg"
	MediaTypeGIF  MediaType = "image/gif"
	MediaTypeWebP MediaType = "image/webp"
)

// ImageSource represents the source of an image for analysis.
type ImageSource struct {
	Data      []byte
	MediaType MediaType
	Filename  string
}

// NewImageSource creates an ImageSource with validation.
// Returns ErrEmptyImageData if data is empty.
func NewImageSource(data []byte, mediaType MediaType, filename string) (*ImageSource, error) {
	if len(data) == 0 {
		return nil, ErrEmptyImageData
	}
	return &ImageSource{
		Data:      data,
		MediaType: mediaType,
		Filename:  filename,
	}, nil
}

// LoadImageFromFile reads an image from the filesystem and returns an ImageSource.
// The media type is detected from the file extension, defaulting to image/png.
func LoadImageFromFile(path string) (*ImageSource, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read image file %q: %w", path, err)
	}

	mt := MediaType(mime.TypeByExtension(filepath.Ext(path)))
	if mt == "" {
		mt = MediaTypePNG
	}

	return NewImageSource(data, mt, filepath.Base(path))
}

// maxImageSize is the maximum allowed image size (50 MB).
const maxImageSize = 50 << 20

// LoadImageFromReader reads image data from an io.Reader.
// Returns ErrImageTooLarge if the data exceeds maxImageSize (50 MB).
func LoadImageFromReader(r io.Reader, mediaType MediaType, filename string) (*ImageSource, error) {
	data, err := io.ReadAll(io.LimitReader(r, maxImageSize+1))
	if err != nil {
		return nil, fmt.Errorf("read image data: %w", err)
	}

	if len(data) > maxImageSize {
		return nil, ErrImageTooLarge
	}

	return NewImageSource(data, mediaType, filename)
}
