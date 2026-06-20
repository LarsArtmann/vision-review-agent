package vision

import (
	"context"
	"time"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

var _ = ginkgo.Describe("Vision Agent", func() {
	ginkgo.Describe("Configuration Validation", func() {
		ginkgo.It("should create an agent with valid configuration", func() {
			_, agent := setupAgent()
			gomega.Expect(agent).NotTo(gomega.BeNil())
			gomega.Expect(agent.config.Model).NotTo(gomega.BeNil())
		})

		ginkgo.It("should reject configuration without a model", func() {
			_, err := NewAgent(Config{})
			gomega.Expect(err).To(gomega.Equal(ErrNoModel))
		})

		ginkgo.It("should reject negative temperature", func() {
			_, err := NewAgent(Config{
				Model:       testModel(),
				Temperature: -0.1,
			})
			gomega.Expect(err).To(gomega.Equal(ErrInvalidTemperature))
		})

		ginkgo.It("should reject temperature above 2.0", func() {
			_, err := NewAgent(Config{
				Model:       testModel(),
				Temperature: 2.1,
			})
			gomega.Expect(err).To(gomega.Equal(ErrInvalidTemperature))
		})

		ginkgo.DescribeTable(
			"should accept temperature at valid boundaries",
			func(temp float64) {
				a, err := NewAgent(Config{
					Model:       testModel(),
					Temperature: temp,
				})
				gomega.Expect(err).NotTo(gomega.HaveOccurred())
				gomega.Expect(a).NotTo(gomega.BeNil())
			},
			ginkgo.Entry("lower boundary 0", float64(0)),
			ginkgo.Entry("upper boundary 2.0", float64(2.0)),
		)

		ginkgo.It("should reject negative max output tokens", func() {
			_, err := NewAgent(Config{
				Model:           testModel(),
				MaxOutputTokens: -1,
			})
			gomega.Expect(err).To(gomega.Equal(ErrInvalidMaxTokens))
		})
	})

	ginkgo.Describe("Image Analysis", func() {
		ginkgo.It("should analyze a single image and return text response", func() {
			ctx, agent := setupAgent()
			img := ImageSrc()
			result, err := agent.Analyze(ctx, "describe this", img)

			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			gomega.Expect(result).NotTo(gomega.BeNil())
			gomega.Expect(result.Text).To(gomega.Equal(mockResponseText))
			gomega.Expect(result.Usage.TotalTokens).To(gomega.Equal(int64(10)))
		})

		ginkgo.It("should analyze multiple images in a single request", func() {
			ctx, agent := setupAgent()
			img1 := ImageSrc("first.png")
			img2 := ImageSrc("second.png")
			result, err := agent.Analyze(ctx, "compare these", img1, img2)

			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			gomega.Expect(result).NotTo(gomega.BeNil())
			gomega.Expect(result.Text).To(gomega.Equal(mockResponseText))
		})

		ginkgo.It("should filter out nil images from the input", func() {
			ctx, agent := setupAgent()
			img := ImageSrc()
			result, err := agent.Analyze(ctx, "describe", nil, img, nil)

			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			gomega.Expect(result).NotTo(gomega.BeNil())
		})

		ginkgo.It("should return error for empty prompt", func() {
			ctx, agent := setupAgent()
			img := ImageSrc()
			_, err := agent.Analyze(ctx, "", img)

			gomega.Expect(err).To(gomega.Equal(ErrEmptyPrompt))
		})

		ginkgo.It("should return error when no images provided", func() {
			ctx, agent := setupAgent()
			_, err := agent.Analyze(ctx, "describe", nil)

			gomega.Expect(err).To(gomega.Equal(ErrNoImages))
		})

		ginkgo.It("should return error when only nil images provided", func() {
			ctx, agent := setupAgent()
			_, err := agent.Analyze(ctx, "describe")

			gomega.Expect(err).To(gomega.Equal(ErrNoImages))
		})

		ginkgo.It("should populate raw response with model result", func() {
			ctx, agent := setupAgent()
			img := ImageSrc()
			result, err := agent.Analyze(ctx, "describe", img)

			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			gomega.Expect(result.RawResponse).NotTo(gomega.BeNil())
		})
	})

	ginkgo.Describe("Streaming Analysis", func() {
		ginkgo.It("should stream analysis results via callback", func() {
			ctx, agent := setupAgent()
			img := ImageSrc()
			var chunks []string

			result, err := agent.AnalyzeStream(ctx, "describe", func(text string) error {
				chunks = append(chunks, text)
				return nil
			}, img)

			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			gomega.Expect(result).NotTo(gomega.BeNil())
			gomega.Expect(chunks).ToNot(gomega.BeEmpty())
		})

		ginkgo.It("should handle nil callback without error", func() {
			ctx, agent := setupAgent()
			img := ImageSrc()
			result, err := agent.AnalyzeStream(ctx, "describe", nil, img)

			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			gomega.Expect(result).NotTo(gomega.BeNil())
		})

		ginkgo.It("should return error for empty prompt", func() {
			ctx, agent := setupAgent()
			img := ImageSrc()
			_, err := agent.AnalyzeStream(ctx, "", nil, img)

			gomega.Expect(err).To(gomega.Equal(ErrEmptyPrompt))
		})

		ginkgo.It("should return error when no images provided", func() {
			ctx, agent := setupAgent()
			_, err := agent.AnalyzeStream(ctx, "describe", nil, nil)

			gomega.Expect(err).To(gomega.Equal(ErrNoImages))
		})
	})

	ginkgo.Describe("Request Timeout", func() {
		ginkgo.It("should apply configured timeout to requests", func() {
			a, err := NewAgent(Config{
				Model:          testModel(),
				RequestTimeout: 100 * time.Millisecond,
			})
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			ctx, cancel := a.withTimeout(context.Background())
			defer cancel()

			_, hasDeadline := ctx.Deadline()
			gomega.Expect(hasDeadline).To(gomega.BeTrue())
		})

		ginkgo.It("should not set deadline when timeout is zero", func() {
			a, err := NewAgent(Config{
				Model:          testModel(),
				RequestTimeout: 0,
			})
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			ctx, cancel := a.withTimeout(context.Background())
			defer cancel()

			_, hasDeadline := ctx.Deadline()
			gomega.Expect(hasDeadline).To(gomega.BeFalse())
		})
	})
})

var _ = ginkgo.Describe("AnalyzeStructured", func() {
	ginkgo.It("should return typed structured response", func() {
		ctx, agent := setupAgent()
		img := ImageSrc()
		result, err := AnalyzeStructured[testReview](ctx, agent, "analyze", img)

		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		gomega.Expect(result).NotTo(gomega.BeNil())
		gomega.Expect(result.Object.Layout).To(gomega.Equal(testLayout))
	})

	ginkgo.It("should return error for empty prompt", func() {
		ctx, agent := setupAgent()
		img := ImageSrc()
		_, err := AnalyzeStructured[testReview](ctx, agent, "", img)

		gomega.Expect(err).To(gomega.Equal(ErrEmptyPrompt))
	})

	ginkgo.It("should return error when no images provided", func() {
		ctx, agent := setupAgent()
		_, err := AnalyzeStructured[testReview](ctx, agent, "analyze")

		gomega.Expect(err).To(gomega.Equal(ErrNoImages))
	})
})
