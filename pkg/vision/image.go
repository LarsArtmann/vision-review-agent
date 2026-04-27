package vision

import (
	"fmt"
	"io"
	"mime"
	"os"
	"path/filepath"
)

// ImageSource represents the source of an image for analysis.
type ImageSource struct {
	Data      []byte
	MediaType string
	Filename  string
}

// LoadImageFromFile reads an image from the filesystem and returns an ImageSource.
// The media type is detected from the file extension, defaulting to image/png.
func LoadImageFromFile(path string) (*ImageSource, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read image file %q: %w", path, err)
	}

	mediaType := mime.TypeByExtension(filepath.Ext(path))
	if mediaType == "" {
		mediaType = "image/png"
	}

	return &ImageSource{
		Data:      data,
		MediaType: mediaType,
		Filename:  filepath.Base(path),
	}, nil
}

// LoadImageFromReader reads image data from an io.Reader.
func LoadImageFromReader(r io.Reader, mediaType, filename string) (*ImageSource, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("read image data: %w", err)
	}

	return &ImageSource{
		Data:      data,
		MediaType: mediaType,
		Filename:  filename,
	}, nil
}
