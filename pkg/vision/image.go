package vision

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// Sentinel errors for image loading operations.
var (
	errImageDownloadFailed = errors.New("image download failed")
)

// MediaType represents a valid image media type.
type MediaType string

// Media type constants for supported image formats.
const (
	MediaTypePNG  MediaType = "image/png"
	MediaTypeJPEG MediaType = "image/jpeg"
	MediaTypeGIF  MediaType = "image/gif"
	MediaTypeWebP MediaType = "image/webp"
	MediaTypeBMP  MediaType = "image/bmp"
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

	mt := mediaTypeFromExtension(path)

	return NewImageSource(data, mt, filepath.Base(path))
}

// maxImageSize is the maximum allowed image size (50 MB).
const maxImageSize = 50 << 20

// LoadImageFromReader reads image data from an io.Reader.
// Returns ErrImageTooLarge if the data exceeds maxImageSize (50 MB).
func LoadImageFromReader(r io.Reader, mediaType MediaType, filename string) (*ImageSource, error) {
	data, err := io.ReadAll(io.LimitReader(r, maxImageSize+1))
	if err != nil {
		return nil, fmt.Errorf(
			"read image data (mediaType=%v, filename=%q): %w",
			mediaType,
			filename,
			err,
		)
	}

	if len(data) > maxImageSize {
		return nil, ErrImageTooLarge
	}

	return NewImageSource(data, mediaType, filename)
}

// LoadImageFromURL downloads an image from a URL and returns an ImageSource.
// It uses [http.DefaultClient]. For a custom client (proxy, timeout, TLS), use
// [LoadImageFromURLWithClient].
// The media type is detected from the Content-Type header, falling back to
// the URL path extension. Returns ErrImageTooLarge if the data exceeds 50 MB
// and ErrInvalidImage if the body is not a recognized image format.
func LoadImageFromURL(ctx context.Context, url string) (*ImageSource, error) {
	return LoadImageFromURLWithClient(ctx, url, http.DefaultClient)
}

// LoadImageFromURLWithClient downloads an image from a URL using the given
// *http.Client, allowing custom transport configuration (proxies, timeouts,
// TLS roots). A nil client falls back to [http.DefaultClient].
func LoadImageFromURLWithClient(
	ctx context.Context,
	url string,
	client *http.Client,
) (*ImageSource, error) {
	if client == nil {
		client = http.DefaultClient
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request for url %q: %w", url, err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("download image from %q: %w", url, err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("%w: %q (HTTP %d)", errImageDownloadFailed, url, resp.StatusCode)
	}

	mediaType := detectMediaTypeFromResponse(resp, url)
	filename := filenameFromURL(url)

	img, err := LoadImageFromReader(resp.Body, mediaType, filename)
	if err != nil {
		return nil, err
	}

	if err := ValidateImage(img.Data); err != nil {
		return nil, fmt.Errorf("download image from %q: %w", url, err)
	}

	return img, nil
}

// LoadImageFromBase64 decodes a base64-encoded image string and returns an ImageSource.
// Accepts both standard and URL-safe base64 encodings, with or without padding.
func LoadImageFromBase64(b64 string, mediaType MediaType, filename string) (*ImageSource, error) {
	data, err := decodeBase64Flex(b64)
	if err != nil {
		return nil, fmt.Errorf("decode base64 image (mediaType=%v, filename=%q): %w", mediaType, filename, err)
	}

	return NewImageSource(data, mediaType, filename)
}

// decodeBase64Flex tries standard, URL-safe, and raw (unpadded) base64 decodings.
func decodeBase64Flex(b64 string) ([]byte, error) {
	if decoded, err := base64.StdEncoding.DecodeString(b64); err == nil {
		return decoded, nil
	}

	if decoded, err := base64.URLEncoding.DecodeString(b64); err == nil {
		return decoded, nil
	}

	decoded, err := base64.RawStdEncoding.DecodeString(strings.TrimRight(b64, "="))
	if err != nil {
		return nil, fmt.Errorf("not valid base64 (standard, url-safe, or raw): %w", err)
	}

	return decoded, nil
}

// detectMediaTypeFromResponse determines the media type from the Content-Type
// header, falling back to the URL path extension, then to PNG.
func detectMediaTypeFromResponse(resp *http.Response, url string) MediaType {
	contentType := resp.Header.Get("Content-Type")
	if contentType != "" {
		mediaType, _, err := mime.ParseMediaType(contentType)
		if err == nil && mediaType != "" {
			return MediaType(mediaType)
		}

		if mediaType := MediaType(strings.TrimSpace(contentType)); mediaType != "" {
			return mediaType
		}
	}

	return mediaTypeFromExtension(url)
}

// mediaTypeFromExtension maps a file path's extension to a MediaType. It first
// checks the project's explicit known-image extension table (deterministic,
// independent of the host's mime.TypeByExtension database), then falls back to
// the system registry, and finally to PNG. The explicit table prevents
// mislabeling on systems where .bmp, .webp, or .gif are not registered.
func mediaTypeFromExtension(path string) MediaType {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".png":
		return MediaTypePNG
	case ".jpg", ".jpeg":
		return MediaTypeJPEG
	case ".gif":
		return MediaTypeGIF
	case ".webp":
		return MediaTypeWebP
	case ".bmp":
		return MediaTypeBMP
	}

	if mt := MediaType(mime.TypeByExtension(filepath.Ext(path))); mt != "" {
		return mt
	}

	return MediaTypePNG
}

func filenameFromURL(url string) string {
	path := url
	if idx := strings.LastIndex(path, "/"); idx >= 0 {
		path = path[idx+1:]
	}

	if path == "" {
		return "image"
	}

	return path
}
