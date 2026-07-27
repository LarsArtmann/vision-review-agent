package vision

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"testing"

	"github.com/stretchr/testify/require"
)

// newPNG builds an RGBA PNG of the given dimensions for testing.
func newPNG(t *testing.T, w, h int) *ImageSource {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	img.Set(0, 0, color.RGBA{R: 255, A: 255})
	img.Set(w-1, h-1, color.RGBA{G: 255, A: 255})

	var buf bytes.Buffer
	require.NoError(t, png.Encode(&buf, img))

	src, err := NewImageSource(buf.Bytes(), MediaTypePNG, "test.png")
	require.NoError(t, err)
	return src
}

func TestResizeImageShrinksLargeImage(t *testing.T) {
	t.Parallel()

	img := newPNG(t, 800, 600)
	resized, err := ResizeImage(img, 200)

	require.NoError(t, err)
	require.NotNil(t, resized)
	require.Equal(t, MediaTypePNG, resized.MediaType)

	// 800x600 scaled to max 200 -> 200x150 (longest side capped).
	decoded, err := png.Decode(bytes.NewReader(resized.Data))
	require.NoError(t, err)

	bounds := decoded.Bounds()
	require.Equal(t, 200, bounds.Dx(), "width must be capped at maxDimension")
	require.Equal(t, 150, bounds.Dy(), "height must preserve aspect ratio")
}

func TestResizeImageReturnsUnchangedWhenWithinBounds(t *testing.T) {
	t.Parallel()

	img := newPNG(t, 100, 100)
	resized, err := ResizeImage(img, 200)

	require.NoError(t, err)
	require.Same(t, img, resized, "must return the same instance when no resize needed")
}

func TestResizeImageRejectsInvalidMaxDimension(t *testing.T) {
	t.Parallel()

	img := newPNG(t, 50, 50)
	_, err := ResizeImage(img, 0)
	require.Error(t, err)

	_, err = ResizeImage(img, -10)
	require.Error(t, err)
}

func TestResizeImageRejectsNilImage(t *testing.T) {
	t.Parallel()

	_, err := ResizeImage(nil, 100)
	require.ErrorIs(t, err, ErrEmptyImageData)
}

func TestResizeImageRejectsNonImage(t *testing.T) {
	t.Parallel()

	img := &ImageSource{Data: []byte("not an image"), MediaType: MediaTypePNG, Filename: "fake.png"}
	_, err := ResizeImage(img, 100)
	require.Error(t, err)
}

func TestScaleDimensionsPreservesAspectRatio(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name         string
		w, h, max    int
		wantW, wantH int
	}{
		{"landscape", 800, 600, 200, 200, 150},
		{"portrait", 600, 800, 200, 150, 200},
		{"square", 500, 500, 100, 100, 100},
		{"tiny floor", 3, 1, 1, 1, 1},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := scaleDimensions(tc.w, tc.h, tc.max)
			require.Equal(t, tc.wantW, got.width)
			require.Equal(t, tc.wantH, got.height)
		})
	}
}
