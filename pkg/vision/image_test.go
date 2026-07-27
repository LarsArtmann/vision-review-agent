package vision

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMediaTypeFromExtension(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		path string
		want MediaType
	}{
		{"png lowercase", "shot.png", MediaTypePNG},
		{"png uppercase", "SHOT.PNG", MediaTypePNG},
		{"jpg", "photo.jpg", MediaTypeJPEG},
		{"jpeg", "photo.jpeg", MediaTypeJPEG},
		{"jpeg with path", "/tmp/a/b/img.JPEG", MediaTypeJPEG},
		{"gif", "anim.gif", MediaTypeGIF},
		{"webp", "shot.webp", MediaTypeWebP},
		{"bmp", "shot.bmp", MediaTypeBMP},
		{"bmp uppercase", "SHOT.BMP", MediaTypeBMP},
		// No-extension and empty-string cases are deterministic (mime returns "").
		{"no extension defaults to png", "screenshot", MediaTypePNG},
		{"empty string defaults to png", "", MediaTypePNG},
		// NOTE: unknown extensions (.tiff, .heic, ...) fall through to
		// mime.TypeByExtension, which is system-dependent, so they are
		// intentionally NOT asserted here.,
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tc.want, mediaTypeFromExtension(tc.path))
		})
	}
}
