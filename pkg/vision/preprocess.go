package vision

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"

	_ "golang.org/x/image/bmp" // register BMP decoder
	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp" // register WebP decoder

	_ "image/gif" // register GIF decoder
)

// resizeJPEGQuality is the quality used when re-encoding resized images as
// JPEG. 85 is a good perceptual/size trade-off for vision model input.
const resizeJPEGQuality = 85

// ResizeImage resizes img so that its longest side is at most maxDimension
// pixels, preserving aspect ratio with high-quality Catmull-Rom interpolation.
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
	if img == nil {
		return nil, ErrEmptyImageData
	}

	if maxDimension <= 0 {
		return nil, fmt.Errorf("resize: maxDimension must be positive, got %d", maxDimension)
	}

	decoded, _, err := decodeImageForResize(img.Data)
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

	var buf bytes.Buffer

	mediaType := img.MediaType

	switch mediaType {
	case MediaTypePNG:
		if err := png.Encode(&buf, resized); err != nil {
			return nil, fmt.Errorf("resize: encode png: %w", err)
		}
	case MediaTypeJPEG:
		if err := jpeg.Encode(&buf, resized, &jpeg.Options{Quality: resizeJPEGQuality}); err != nil {
			return nil, fmt.Errorf("resize: encode jpeg: %w", err)
		}
	default:
		// GIF/WebP/BMP and unknowns re-encode as JPEG for size efficiency.
		if err := jpeg.Encode(&buf, resized, &jpeg.Options{Quality: resizeJPEGQuality}); err != nil {
			return nil, fmt.Errorf("resize: encode jpeg: %w", err)
		}

		mediaType = MediaTypeJPEG
	}

	return NewImageSource(buf.Bytes(), mediaType, img.Filename)
}

// dimensions holds a width/height pair.
type dimensions struct {
	width  int
	height int
}

// scaleDimensions computes the largest aspect-preserving size whose longest
// side is maxDimension, never smaller than 1x1.
func scaleDimensions(width, height, maxDimension int) dimensions {
	longest := max(height, width)

	scale := float64(maxDimension) / float64(longest)

	return dimensions{
		width:  maxInt(1, int(float64(width)*scale)),
		height: maxInt(1, int(float64(height)*scale)),
	}
}

// decodeImageForResize decodes an image, returning the image, its format name,
// and any error. PNG/JPEG/GIF/WebP are registered via blank imports.
func decodeImageForResize(data []byte) (image.Image, string, error) {
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
