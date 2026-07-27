package vision

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"testing"

	"charm.land/fantasy"
	"github.com/stretchr/testify/require"
)

// totalImageBytes sums the Data length of every FilePart across a Prompt's
// messages. Used to verify that preprocessing actually resized the images
// before they reached the model.
func totalImageBytes(p fantasy.Prompt) int {
	total := 0
	for _, msg := range p {
		for _, part := range msg.Content {
			if file, ok := fantasy.AsContentType[fantasy.FilePart](part); ok {
				total += len(file.Data)
			}
		}
	}
	return total
}

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

// newJPEG builds a JPEG of the given dimensions with high-frequency content so
// that different quality levels produce measurably different byte sizes.
func newJPEG(t *testing.T, w, h, quality int) *ImageSource {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	// High-frequency pattern: adjacent pixels differ a lot, so JPEG quality
	// has a real effect on output size.
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{
				R: uint8((x * 7) & 0xff),
				G: uint8((y * 13) & 0xff),
				B: uint8(((x + y) * 5) & 0xff),
				A: 255,
			})
		}
	}

	var buf bytes.Buffer
	require.NoError(t, jpeg.Encode(&buf, img, &jpeg.Options{Quality: quality}))

	src, err := NewImageSource(buf.Bytes(), MediaTypeJPEG, "test.jpg")
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

func TestEffectiveJPEGQuality(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		in   int
		want int
	}{
		{"zero defaults", 0, defaultJPEGQuality},
		{"negative defaults", -5, defaultJPEGQuality},
		{"over 100 defaults", 101, defaultJPEGQuality},
		{"one is valid", 1, 1},
		{"hundred is valid", 100, 100},
		{"mid range passes through", 50, 50},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tc.want, effectiveJPEGQuality(tc.in))
		})
	}
}

func TestResizeImageWithQualityLowerQualityIsSmaller(t *testing.T) {
	t.Parallel()

	big := newJPEG(t, 400, 400, 95) // large, detailed source

	high, err := ResizeImageWithQuality(big, 200, 90)
	require.NoError(t, err)
	require.Equal(t, MediaTypeJPEG, high.MediaType)

	low, err := ResizeImageWithQuality(big, 200, 20)
	require.NoError(t, err)
	require.Equal(t, MediaTypeJPEG, low.MediaType)

	require.Less(
		t,
		len(low.Data),
		len(high.Data),
		"quality 20 must produce fewer bytes than quality 90 for the same resize",
	)
}

func TestPreprocessImageJPEGQualityIsWiredThroughConfig(t *testing.T) {
	t.Parallel()

	big := newJPEG(t, 400, 400, 95)

	high, err := PreprocessImage(big, &PreprocessConfig{MaxDimension: 200, JPEGQuality: 90})
	require.NoError(t, err)

	low, err := PreprocessImage(big, &PreprocessConfig{MaxDimension: 200, JPEGQuality: 20})
	require.NoError(t, err)

	require.Less(
		t,
		len(low.Data),
		len(high.Data),
		"PreprocessConfig.JPEGQuality must flow through to the JPEG encoder",
	)
}

func TestCompressImageReducesJPEGSize(t *testing.T) {
	t.Parallel()

	src := newJPEG(t, 200, 200, 100) // already target dims, high quality

	compressed, err := CompressImage(src, 30)
	require.NoError(t, err)
	require.Equal(t, MediaTypeJPEG, compressed.MediaType, "JPEG source must stay JPEG")
	require.True(t, len(compressed.Data) > 0, "compressed image must not be empty")
	require.Less(
		t,
		len(compressed.Data),
		len(src.Data),
		"compressing at quality 30 must shrink a quality-100 JPEG of the same dimensions",
	)
}

func TestCompressImagePreservesPNGFormat(t *testing.T) {
	t.Parallel()

	src := newPNG(t, 200, 200)

	compressed, err := CompressImage(src, 30)
	require.NoError(t, err)
	require.Equal(t, MediaTypePNG, compressed.MediaType, "PNG source must remain PNG (quality ignored)")

	// Output must still be a decodable PNG of the same dimensions.
	decoded, err := png.Decode(bytes.NewReader(compressed.Data))
	require.NoError(t, err)
	require.Equal(t, 200, decoded.Bounds().Dx())
	require.Equal(t, 200, decoded.Bounds().Dy())
}

func TestCompressImageRejectsNilAndBadInput(t *testing.T) {
	t.Parallel()

	_, err := CompressImage(nil, 50)
	require.ErrorIs(t, err, ErrEmptyImageData)

	bad := &ImageSource{Data: []byte("not an image"), MediaType: MediaTypeJPEG, Filename: "fake.jpg"}
	_, err = CompressImage(bad, 50)
	require.Error(t, err)
}

// newBMP builds a minimal valid 24-bit BMP of the given dimensions for testing
// the BMP decode path. Go has no BMP encoder, so the bytes are assembled by
// hand (file header + DIB header + bottom-up padded pixel rows).
func newBMP(t *testing.T, w, h int) *ImageSource {
	t.Helper()

	const (
		fileHeaderSize = 14
		dibHeaderSize  = 40
		bitsPerPixel   = 24
	)
	rowSize := (w*bitsPerPixel + 7) / 8
	rowSize = (rowSize + 3) &^ 3 // rows are padded to a multiple of 4 bytes
	pixelDataSize := rowSize * h
	pixelOffset := fileHeaderSize + dibHeaderSize
	fileSize := pixelOffset + pixelDataSize

	buf := bytes.Buffer{}
	// BITMAPFILEHEADER
	buf.WriteString("BM")
	write32(&buf, fileSize)
	write32(&buf, 0) // reserved
	write32(&buf, pixelOffset)
	// BITMAPINFOHEADER
	write32(&buf, dibHeaderSize)
	write32(&buf, w)
	write32(&buf, h)
	write16(&buf, 1) // planes
	write16(&buf, bitsPerPixel)
	write32(&buf, 0) // compression (BI_RGB)
	write32(&buf, pixelDataSize)
	write32(&buf, 2835) // x ppm (72 DPI)
	write32(&buf, 2835) // y ppm
	write32(&buf, 0)    // colors used
	write32(&buf, 0)    // important colors

	// Pixel rows, bottom-up, blue/green/red order, padded to rowSize.
	row := make([]byte, rowSize)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			off := x * 3
			row[off+0] = byte((x * 30) & 0xff) // B
			row[off+1] = byte((y * 30) & 0xff) // G
			row[off+2] = byte(((x + y) * 15) & 0xff) // R
		}
		_, _ = buf.Write(row)
	}

	src, err := NewImageSource(buf.Bytes(), MediaTypeBMP, "test.bmp")
	require.NoError(t, err)
	return src
}

func write16(buf *bytes.Buffer, v int) {
	buf.Write([]byte{byte(v), byte(v >> 8)})
}

func write32(buf *bytes.Buffer, v int) {
	buf.Write([]byte{byte(v), byte(v >> 8), byte(v >> 16), byte(v >> 24)})
}

func TestResizeImageDecodesAndResizesBMP(t *testing.T) {
	t.Parallel()

	big := newBMP(t, 300, 200) // exceeds maxDimension

	resized, err := ResizeImage(big, 100)
	require.NoError(t, err)

	// BMP is re-encoded as JPEG (non-PNG/JPEG input → JPEG).
	require.Equal(t, MediaTypeJPEG, resized.MediaType)
	require.True(t, len(resized.Data) > 0, "resized BMP must produce non-empty output")

	// The output must decode to a 100-wide (longest side capped) image.
	config, _, err := image.DecodeConfig(bytes.NewReader(resized.Data))
	require.NoError(t, err)
	require.Equal(t, 100, config.Width, "longest side must be capped at maxDimension")
}

func TestPreprocessImagePassthroughNilAndZeroConfig(t *testing.T) {
	t.Parallel()

	img := newPNG(t, 50, 50)

	// nil config → unchanged (same pointer).
	out, err := PreprocessImage(img, nil)
	require.NoError(t, err)
	require.Same(t, img, out)

	// zero-value config (MaxDimension 0) → unchanged (same pointer).
	out, err = PreprocessImage(img, &PreprocessConfig{})
	require.NoError(t, err)
	require.Same(t, img, out)

	// nil image → nil image, no error.
	out, err = PreprocessImage(nil, &PreprocessConfig{MaxDimension: 100})
	require.NoError(t, err)
	require.Nil(t, out)

	// image already within bounds → unchanged (same pointer).
	small := newPNG(t, 40, 40)
	out, err = PreprocessImage(small, &PreprocessConfig{MaxDimension: 100})
	require.NoError(t, err)
	require.Same(t, small, out)
}

func TestConfigPreprocessAppliedInAnalyze(t *testing.T) {
	t.Parallel()

	big := newPNG(t, 800, 600) // ~1.4 MB raw PNG
	model := &mockModel{capture: true}
	agent := newTestAgent(t, model)
	agent.config.Preprocess = &PreprocessConfig{MaxDimension: 100}

	_, err := agent.Analyze(context.Background(), "describe", big)
	require.NoError(t, err)

	sent := totalImageBytes(model.capturedPrompt)
	require.Less(
		t,
		sent,
		len(big.Data),
		"Config.Preprocess must resize the image before it reaches the model",
	)
}

func TestConfigPreprocessAppliedInAnalyzeStructured(t *testing.T) {
	t.Parallel()

	big := newPNG(t, 800, 600) // exceeds the cap
	model := &mockModel{capture: true}
	agent := newTestAgent(t, model)
	agent.config.Preprocess = &PreprocessConfig{MaxDimension: 100}

	_, err := AnalyzeStructured[testReview](context.Background(), agent, "review", big)
	require.NoError(t, err)

	sent := totalImageBytes(model.capturedPrompt)
	require.Less(
		t,
		sent,
		len(big.Data),
		"Config.Preprocess must resize the image before structured generate too",
	)
}
