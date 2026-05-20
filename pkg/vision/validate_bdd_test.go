package vision

import (
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

var _ = ginkgo.Describe("Image Validation", func() {
	ginkgo.Describe("DetectImageFormat", func() {
		ginkgo.It("should detect PNG from magic bytes", func() {
			data := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
			gomega.Expect(DetectImageFormat(data)).To(gomega.Equal(formatPNG))
		})

		ginkgo.It("should detect JPEG from magic bytes", func() {
			data := []byte{0xFF, 0xD8, 0xFF, 0xE0}
			gomega.Expect(DetectImageFormat(data)).To(gomega.Equal(formatJPG))
		})

		ginkgo.It("should detect GIF from magic bytes", func() {
			data := []byte{0x47, 0x49, 0x46, 0x38}
			gomega.Expect(DetectImageFormat(data)).To(gomega.Equal(formatGIF))
		})

		ginkgo.It("should detect WebP from magic bytes", func() {
			data := []byte{0x52, 0x49, 0x46, 0x46, 0x00, 0x00, 0x00, 0x00, 0x57, 0x45, 0x42, 0x50}
			gomega.Expect(DetectImageFormat(data)).To(gomega.Equal(formatWebP))
		})

		ginkgo.It("should detect BMP from magic bytes", func() {
			data := []byte{0x42, 0x4D, 0x36, 0x00}
			gomega.Expect(DetectImageFormat(data)).To(gomega.Equal(formatBMP))
		})

		ginkgo.It("should return empty string for unknown data", func() {
			data := []byte{0x00, 0x00, 0x00, 0x00}
			gomega.Expect(DetectImageFormat(data)).To(gomega.BeEmpty())
		})

		ginkgo.It("should return empty string for data shorter than 4 bytes", func() {
			data := []byte{0x89, 0x50}
			gomega.Expect(DetectImageFormat(data)).To(gomega.BeEmpty())
		})

		ginkgo.It("should return empty string for empty data", func() {
			gomega.Expect(DetectImageFormat([]byte{})).To(gomega.BeEmpty())
		})
	})

	ginkgo.Describe("IsValidImage", func() {
		ginkgo.It("should return true for PNG data", func() {
			gomega.Expect(IsValidImage([]byte{0x89, 0x50, 0x4E, 0x47})).To(gomega.BeTrue())
		})

		ginkgo.It("should return true for JPEG data", func() {
			gomega.Expect(IsValidImage([]byte{0xFF, 0xD8, 0xFF, 0xE0})).To(gomega.BeTrue())
		})

		ginkgo.It("should return false for unknown data", func() {
			gomega.Expect(IsValidImage([]byte{0x00, 0x00, 0x00, 0x00})).To(gomega.BeFalse())
		})

		ginkgo.It("should return false for empty data", func() {
			gomega.Expect(IsValidImage([]byte{})).To(gomega.BeFalse())
		})
	})

	ginkgo.Describe("ValidateImage", func() {
		ginkgo.It("should return nil for valid PNG data", func() {
			gomega.Expect(ValidateImage([]byte{0x89, 0x50, 0x4E, 0x47})).To(gomega.Succeed())
		})

		ginkgo.It("should return ErrInvalidImage for unknown data", func() {
			err := ValidateImage([]byte{0x00, 0x00, 0x00, 0x00})
			gomega.Expect(err).To(gomega.Equal(ErrInvalidImage))
		})

		ginkgo.It("should return error for empty data", func() {
			err := ValidateImage([]byte{})
			gomega.Expect(err).To(gomega.HaveOccurred())
		})
	})

	ginkgo.Describe("NewImageSource", func() {
		ginkgo.It("should create ImageSource with valid data", func() {
			img, err := NewImageSource([]byte("valid"), MediaTypePNG, "test.png")
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			gomega.Expect(img).NotTo(gomega.BeNil())
			gomega.Expect(img.Data).To(gomega.Equal([]byte("valid")))
			gomega.Expect(img.MediaType).To(gomega.Equal(MediaTypePNG))
			gomega.Expect(img.Filename).To(gomega.Equal("test.png"))
		})

		ginkgo.It("should return ErrEmptyImageData for empty data", func() {
			_, err := NewImageSource([]byte{}, MediaTypePNG, "test.png")
			gomega.Expect(err).To(gomega.Equal(ErrEmptyImageData))
		})
	})
})
