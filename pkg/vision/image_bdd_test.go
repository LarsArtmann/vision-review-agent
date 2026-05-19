package vision

import (
	"os"
	"path/filepath"

	"charm.land/fantasy"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

const testAnalysisText = "test analysis"

var _ = ginkgo.Describe("Image Loading", func() {
	ginkgo.It("should load image from file", func() {
		tmpDir := ginkgo.GinkgoT().TempDir()
		tmpFile := filepath.Join(tmpDir, "test.png")
		gomega.Expect(os.WriteFile(tmpFile, []byte("png data"), 0o644)).To(gomega.Succeed())

		img, err := LoadImageFromFile(tmpFile)

		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		gomega.Expect(img).NotTo(gomega.BeNil())
		gomega.Expect(img.MediaType).To(gomega.Equal(MediaTypePNG))
		gomega.Expect(string(img.Data)).To(gomega.Equal("png data"))
	})

	ginkgo.It("should return error for missing file", func() {
		_, err := LoadImageFromFile("/nonexistent/file.png")

		gomega.Expect(err).To(gomega.HaveOccurred())
	})

	ginkgo.It("should detect correct media types by extension", func() {
		tmpDir := ginkgo.GinkgoT().TempDir()

		testCases := map[string]string{
			".png":  MediaTypePNG,
			".jpg":  MediaTypeJPEG,
			".jpeg": MediaTypeJPEG,
			".gif":  MediaTypeGIF,
			".webp": MediaTypeWebP,
		}

		for ext, expectedType := range testCases {
			path := filepath.Join(tmpDir, "test"+ext)
			gomega.Expect(os.WriteFile(path, []byte("data"), 0o644)).To(gomega.Succeed())

			img, err := LoadImageFromFile(path)

			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			gomega.Expect(img.MediaType).To(gomega.Equal(expectedType), "for extension "+ext)
		}
	})
})

var _ = ginkgo.Describe("AnalyzeResult", func() {
	ginkgo.It("should return human-readable string representation", func() {
		result := &AnalyzeResult{
			Text:  testAnalysisText,
			Usage: fantasy.Usage{TotalTokens: 42},
		}

		str := result.String()

		gomega.Expect(str).To(gomega.ContainSubstring(testAnalysisText))
		gomega.Expect(str).To(gomega.ContainSubstring("42"))
	})
})
