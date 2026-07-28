package vision

import (
	"context"
	"os"
	"path/filepath"
	"time"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

var _ = ginkgo.Describe("ScreenshotAnalyzer", func() {
	var (
		model *mockModel
		ctx   context.Context
	)

	ginkgo.BeforeEach(func() {
		ctx = context.Background()
		model = testModel()
	})

	ginkgo.Describe("Builder Pattern", func() {
		ginkgo.It("should create analyzer with default system prompt", func() {
			sa := NewScreenshotAnalyzer(model)
			gomega.Expect(sa).NotTo(gomega.BeNil())
			gomega.Expect(sa.config.SystemPrompt).To(gomega.Equal(DefaultScreenshotPrompt))
		})

		ginkgo.It("should allow setting custom system prompt", func() {
			sa := NewScreenshotAnalyzer(model).WithSystemPrompt("custom prompt")
			gomega.Expect(sa.config.SystemPrompt).To(gomega.Equal("custom prompt"))
		})

		ginkgo.It("should allow setting temperature", func() {
			sa := NewScreenshotAnalyzer(model).WithTemperature(0.7)
			gomega.Expect(sa.config.Temperature).To(gomega.Equal(0.7))
		})

		ginkgo.It("should allow setting max output tokens", func() {
			sa := NewScreenshotAnalyzer(model).WithMaxOutputTokens(500)
			gomega.Expect(sa.config.MaxOutputTokens).To(gomega.Equal(int64(500)))
		})

		ginkgo.It("should allow setting max retries", func() {
			sa := NewScreenshotAnalyzer(model).WithMaxRetries(3)
			gomega.Expect(sa.config.MaxRetries).To(gomega.Equal(3))
		})

		ginkgo.It("should allow setting request timeout", func() {
			sa := NewScreenshotAnalyzer(model).WithRequestTimeout(30 * time.Second)
			gomega.Expect(sa.config.RequestTimeout).To(gomega.Equal(30 * time.Second))
		})

		ginkgo.It("should allow setting hooks and fire them via delegation", func() {
			var fired int

			sa := NewScreenshotAnalyzer(model).WithHooks(Hooks{
				OnStart: func(context.Context, string, int) { fired++ },
			})
			gomega.Expect(sa.config.Hooks.OnStart).NotTo(gomega.BeNil())

			_, err := sa.AnalyzeScreenshotImages(ctx, "describe", ImageSrc())
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			gomega.Expect(fired).To(gomega.Equal(1), "OnStart must fire via delegation")
		})

		ginkgo.It("should support fluent chaining", func() {
			sa := NewScreenshotAnalyzer(model).
				WithSystemPrompt("expert").
				WithTemperature(0.5).
				WithMaxOutputTokens(1000).
				WithMaxRetries(2).
				WithRequestTimeout(60 * time.Second)

			gomega.Expect(sa.config.SystemPrompt).To(gomega.Equal("expert"))
			gomega.Expect(sa.config.Temperature).To(gomega.Equal(0.5))
			gomega.Expect(sa.config.MaxOutputTokens).To(gomega.Equal(int64(1000)))
			gomega.Expect(sa.config.MaxRetries).To(gomega.Equal(2))
			gomega.Expect(sa.config.RequestTimeout).To(gomega.Equal(60 * time.Second))
		})
	})

	ginkgo.Describe("Screenshot Analysis", func() {
		ginkgo.It("should analyze screenshot from file path", func() {
			tmpDir := ginkgo.GinkgoT().TempDir()
			screenshotPath := filepath.Join(tmpDir, "screenshot.png")
			gomega.Expect(os.WriteFile(screenshotPath, []byte("fake"), 0o644)).To(gomega.Succeed())

			sa := NewScreenshotAnalyzer(model)
			result, err := sa.AnalyzeScreenshot(ctx, "describe", screenshotPath)

			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			gomega.Expect(result).NotTo(gomega.BeNil())
			gomega.Expect(result.Text).To(gomega.Equal(mockResponseText))
		})

		ginkgo.It("should return error for missing file", func() {
			sa := NewScreenshotAnalyzer(model)
			_, err := sa.AnalyzeScreenshot(ctx, "describe", "/nonexistent/file.png")

			gomega.Expect(err).To(gomega.HaveOccurred())
		})

		ginkgo.It("should analyze screenshot from ImageSource", func() {
			sa := NewScreenshotAnalyzer(model)
			img := ImageSrc()

			result, err := sa.AnalyzeScreenshotImage(ctx, "describe", img)

			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			gomega.Expect(result).NotTo(gomega.BeNil())
		})

		ginkgo.It("should analyze multiple screenshots", func() {
			tmpDir := ginkgo.GinkgoT().TempDir()
			path1 := filepath.Join(tmpDir, "1.png")
			path2 := filepath.Join(tmpDir, "2.png")

			gomega.Expect(os.WriteFile(path1, []byte("fake"), 0o644)).To(gomega.Succeed())
			gomega.Expect(os.WriteFile(path2, []byte("fake"), 0o644)).To(gomega.Succeed())

			sa := NewScreenshotAnalyzer(model)
			result, err := sa.AnalyzeScreenshots(ctx, "compare", path1, path2)

			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			gomega.Expect(result).NotTo(gomega.BeNil())
		})

		ginkgo.It("should return error when one file is missing", func() {
			tmpDir := ginkgo.GinkgoT().TempDir()
			path1 := filepath.Join(tmpDir, "1.png")
			gomega.Expect(os.WriteFile(path1, []byte("fake"), 0o644)).To(gomega.Succeed())

			sa := NewScreenshotAnalyzer(model)
			_, err := sa.AnalyzeScreenshots(ctx, "compare", path1, "/nonexistent/2.png")

			gomega.Expect(err).To(gomega.HaveOccurred())
		})

		ginkgo.It("should analyze multiple ImageSources", func() {
			sa := NewScreenshotAnalyzer(model)
			img1 := ImageSrc("test1.png")
			img2 := ImageSrc("test2.png")

			result, err := sa.AnalyzeScreenshotImages(ctx, "compare", img1, img2)

			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			gomega.Expect(result).NotTo(gomega.BeNil())
		})

		ginkgo.It("should return error for empty prompt", func() {
			sa := NewScreenshotAnalyzer(model)
			img := ImageSrc()

			_, err := sa.AnalyzeScreenshotImage(ctx, "", img)

			gomega.Expect(err).To(gomega.Equal(ErrEmptyPrompt))
		})
	})

	ginkgo.Describe("Conversation Delegation", func() {
		ginkgo.It("should delegate AnalyzeConversation to the underlying agent", func() {
			sa := NewScreenshotAnalyzer(model)
			conv := NewConversation()
			conv.AddUserMessage("previous turn", ImageSrc())

			result, err := sa.AnalyzeConversation(ctx, conv, "follow up", ImageSrc())

			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			gomega.Expect(result).NotTo(gomega.BeNil())
			gomega.Expect(result.Text).To(gomega.Equal(mockResponseText))
		})

		ginkgo.It("should delegate AnalyzeConversationStream to the underlying agent", func() {
			sa := NewScreenshotAnalyzer(model)
			conv := NewConversation()
			conv.AddUserMessage("previous turn", ImageSrc())

			var chunks []string

			result, err := sa.AnalyzeConversationStream(
				ctx,
				conv,
				"follow up",
				func(text string) error {
					chunks = append(chunks, text)

					return nil
				},
				ImageSrc(),
			)

			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			gomega.Expect(result).NotTo(gomega.BeNil())
			gomega.Expect(chunks).NotTo(gomega.BeEmpty())
		})

		ginkgo.It("should return validation errors from AnalyzeConversation", func() {
			sa := NewScreenshotAnalyzer(model)
			conv := NewConversation()

			_, err := sa.AnalyzeConversation(ctx, conv, "", ImageSrc())
			gomega.Expect(err).To(gomega.Equal(ErrEmptyPrompt))
		})
	})
})
