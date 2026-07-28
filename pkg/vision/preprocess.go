package vision

import (
	"bytes"
	"fmt"
	"image"
	_ "image/gif" // register GIF decoder
	"image/jpeg"
	"image/png"

	_ "golang.org/x/image/bmp" // register BMP decoder
	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp" // register WebP decoder
)

// defaultJPEGQuality is the quality used when the caller passes zero or an
// out-of-range value. 85 is a good perceptual/size trade-off for vision model
// input.
const defaultJPEGQuality = 85

// effectiveJPEGQuality returns q clamped to the valid JPEG quality range
// [1, 100], defaulting to defaultJPEGQuality when q is zero or out of range.
func effectiveJPEGQuality(q int) int {
	if q < 1 || q > 100 {
		return defaultJPEGQuality
	}

	return q
}

// ResizeImage resizes img so that its longest side is at most maxDimension
// pixels, preserving aspect ratio with high-quality Catmull-Rom interpolation,
// re-encoding JPEG output at the default quality (85). For control over JPEG
// quality, use [ResizeImageWithQuality].
//
// Images already within bounds are returned unchanged (no re-encode, no copy).
// The output format follows the input media type when it is PNG or JPEG;
// other formats (GIF, WebP, BMP) are re-encoded as JPEG to minimize size.
//
// This is useful before [Agent.Analyze] to reduce token usage and stay within
// provider dimension limits:
//
//	img, _ := vision.LoadImageFromFile("huge.png")
//	small, _ := vision.ResizeImage(img, 1568)
//	result, _ := agent.Analyze(ctx, "review this", small)
func ResizeImage(img *ImageSource, maxDimension int) (*ImageSource, error) {
	return ResizeImageWithQuality(img, maxDimension, defaultJPEGQuality)
}

// ResizeImageWithQuality is like [ResizeImage] but lets the caller control the
// JPEG encoding quality (1-100; zero or out-of-range defaults to 85). Quality
// only affects JPEG output; PNG output is lossless and ignores it.
func ResizeImageWithQuality(img *ImageSource, maxDimension, jpegQuality int) (*ImageSource, error) {
	if img == nil {
		return nil, ErrEmptyImageData
	}

	if maxDimension <= 0 {
		return nil, fmt.Errorf("resize: maxDimension must be positive, got %d", maxDimension)
	}

	decoded, _, err := decodeImage(img.Data)
	if err != nil {
		return nil, fmt.Errorf("resize: decode (mediaType=%v): %w", img.MediaType, err)
	}

	bounds := decoded.Bounds()
	width, height := bounds.Dx(), bounds.Dy()

	if width <= maxDimension && height <= maxDimension {
		return img, nil
	}

	scaled := scaleDimensions(width, height, maxDimension)

	resized := image.NewRGBA(image.Rect(0, 0, scaled.width, scaled.height))
	draw.CatmullRom.Scale(resized, resized.Bounds(), decoded, bounds, draw.Src, nil)

	data, mediaType, err := encodeImage(resized, img.MediaType, jpegQuality)
	if err != nil {
		return nil, fmt.Errorf("resize: %w", err)
	}

	return NewImageSource(data, mediaType, img.Filename)
}

// CompressImage re-encodes img to reduce its byte size without resizing.
//
// JPEG, GIF, WebP, and BMP inputs are re-encoded as JPEG at the given quality
// (1-100; zero or out-of-range defaults to 85). PNG input is re-encoded as PNG
// using best compression — the quality is ignored because PNG is lossless, so
// the output format is preserved.
//
// If re-encoding does not shrink the image (e.g. the source is already
// well-compressed at a lower quality), the original is returned unchanged.
//
// Use this to cut token cost before [Agent.Analyze] when an image is already
// the right dimensions but too large in bytes:
//
//	img, _ := vision.LoadImageFromFile("photo.jpg")
//	smaller, _ := vision.CompressImage(img, 60)
//	result, _ := agent.Analyze(ctx, "review this", smaller)
func CompressImage(img *ImageSource, jpegQuality int) (*ImageSource, error) {
	if img == nil {
		return nil, ErrEmptyImageData
	}

	decoded, _, err := decodeImage(img.Data)
	if err != nil {
		return nil, fmt.Errorf("compress: decode (mediaType=%v): %w", img.MediaType, err)
	}

	data, mediaType, err := encodeImage(decoded, img.MediaType, jpegQuality)
	if err != nil {
		return nil, fmt.Errorf("compress: %w", err)
	}

	// Guard: if re-encoding did not shrink the image (e.g. the source was already
	// well-compressed at a similar or lower quality), return the original
	// unchanged. CompressImage's contract is to reduce size; an output that is
	// equal or larger signals no benefit and would only add re-encoding artifacts.
	if len(data) >= len(img.Data) {
		return img, nil
	}

	return NewImageSource(data, mediaType, img.Filename)
}

// encodeImage encodes src according to mediaType, returning the encoded bytes
// and the effective output media type. PNG stays PNG (lossless); JPEG, GIF,
// WebP, BMP, and unknown formats are re-encoded as JPEG at jpegQuality.
func encodeImage(
	src image.Image,
	mediaType MediaType,
	jpegQuality int,
) ([]byte, MediaType, error) {
	quality := effectiveJPEGQuality(jpegQuality)

	if mediaType == MediaTypePNG {
		var buf bytes.Buffer

		enc := png.Encoder{CompressionLevel: png.BestCompression}
		if err := enc.Encode(&buf, src); err != nil {
			return nil, mediaType, fmt.Errorf("encode png: %w", err)
		}

		return buf.Bytes(), MediaTypePNG, nil
	}

	// JPEG, GIF, WebP, BMP, and unknowns re-encode as JPEG for size efficiency.
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, src, &jpeg.Options{Quality: quality}); err != nil {
		return nil, MediaTypeJPEG, fmt.Errorf("encode jpeg: %w", err)
	}

	return buf.Bytes(), MediaTypeJPEG, nil
}

// dimensions holds a width/height pair.
type dimensions struct {
	width  int
	height int
}

// scaleDimensions computes the largest aspect-preserving size whose longest
// side is maxDimension, never smaller than 1x1.
func scaleDimensions(width, height, maxDimension int) dimensions {
	longest := maxInt(height, width)

	scale := float64(maxDimension) / float64(longest)

	return dimensions{
		width:  maxInt(1, int(float64(width)*scale)),
		height: maxInt(1, int(float64(height)*scale)),
	}
}

// decodeImage decodes an image, returning the image, its format name, and any
// error. PNG/JPEG/GIF/WebP/BMP are registered via blank imports.
func decodeImage(data []byte) (image.Image, string, error) {
	return image.Decode(bytes.NewReader(data))
}

// maxInt returns the larger of two ints. Prefer this over the builtin where the
// arguments are not both literals, to keep call sites explicit.
func maxInt(a, b int) int {
	if a > b {
		return a
	}

	return b
}

// PreprocessConfig controls automatic image preprocessing applied before every
// Analyze* call when set via Config.Preprocess. The zero value disables
// preprocessing (images are sent as-is).
type PreprocessConfig struct {
	// MaxDimension caps the longest side in pixels. Images exceeding this are
	// downsampled with Catmull-Rom interpolation, preserving aspect ratio.
	// Zero means no resize. A common value is 1568 (OpenAI's recommended max).
	MaxDimension int

	// JPEGQuality controls re-encoding quality for JPEG output (1-100). Only
	// used when the image is resized or converted. Zero defaults to 85.
	JPEGQuality int
}

// PreprocessImage applies the given PreprocessConfig to an ImageSource,
// returning a new ImageSource if any transformation occurred, or the original
// if no change was needed. A nil config or zero-value config returns the image
// unchanged. This is the function called automatically by the Agent when
// Config.Preprocess is set.
func PreprocessImage(img *ImageSource, cfg *PreprocessConfig) (*ImageSource, error) {
	if img == nil || cfg == nil {
		return img, nil
	}

	if cfg.MaxDimension <= 0 {
		return img, nil
	}

	return ResizeImageWithQuality(img, cfg.MaxDimension, cfg.JPEGQuality)
}

// preprocessAll applies PreprocessConfig to a slice of images, returning a new
// slice. Images that need no change retain their original pointer.
func preprocessAll(images []*ImageSource, cfg *PreprocessConfig) ([]*ImageSource, error) {
	if cfg == nil || cfg.MaxDimension <= 0 {
		return images, nil
	}

	result := make([]*ImageSource, len(images))
	for i, img := range images {
		if img == nil {
			result[i] = nil

			continue
		}

		processed, err := PreprocessImage(img, cfg)
		if err != nil {
			return nil, fmt.Errorf("preprocess image %d (%q): %w", i, img.Filename, err)
		}

		result[i] = processed
	}

	return result, nil
}
