package a2ui_test

import (
	"context"
	"errors"

	"github.com/larsartmann/vision-review-agent/pkg/vision"
	"github.com/larsartmann/vision-review-agent/pkg/vision/a2ui"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

var _ = ginkgo.Describe("Generate", func() {
	ginkgo.Context("when the model returns a well-formed surface spec", func() {
		ginkgo.It("compiles it into validated wire messages", func() {
			model := &fakeModel{object: validSpecObject()}

			result, err := a2ui.Generate(
				ginkgoCtx(), newAgent(model), a2ui.GenerateOptions{}, testImage(),
			)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			gomega.Expect(result.Messages).To(gomega.HaveLen(3))
			gomega.Expect(result.Messages[0]).To(gomega.BeAssignableToTypeOf(&a2ui.CreateSurface{}))
			gomega.Expect(result.Messages[1]).To(gomega.BeAssignableToTypeOf(&a2ui.UpdateComponents{}))
			gomega.Expect(result.Messages[2]).To(gomega.BeAssignableToTypeOf(&a2ui.UpdateDataModel{}))

			gomega.Expect(a2ui.Validate(result.Messages)).To(gomega.Succeed())
			gomega.Expect(result.Usage.TotalTokens).To(gomega.Equal(int64(42)))

			create, ok := result.Messages[0].(*a2ui.CreateSurface)
			gomega.Expect(ok).To(gomega.BeTrue())
			gomega.Expect(create.SurfaceID).To(gomega.Equal("review-dashboard"))
			gomega.Expect(create.CatalogID).To(gomega.Equal(a2ui.DefaultCatalogID))
		})

		ginkgo.It("roundtrips the messages through the JSONL wire format", func() {
			model := &fakeModel{object: validSpecObject()}

			result, err := a2ui.Generate(
				ginkgoCtx(), newAgent(model), a2ui.GenerateOptions{}, testImage(),
			)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			wire, err := a2ui.MarshalJSONL(result.Messages)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			decoded, err := a2ui.UnmarshalJSONL(wire)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			gomega.Expect(decoded).To(gomega.HaveLen(len(result.Messages)))
		})
	})

	ginkgo.Context("when the model omits surface and catalog ids", func() {
		ginkgo.It("applies the option defaults", func() {
			object := validSpecObject()
			object["surfaceId"] = ""
			object["catalogId"] = ""
			object["dataModel"] = nil
			model := &fakeModel{object: object}

			result, err := a2ui.Generate(ginkgoCtx(), newAgent(model), a2ui.GenerateOptions{
				SurfaceID: "from-options",
			}, testImage())
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			create, ok := result.Messages[0].(*a2ui.CreateSurface)
			gomega.Expect(ok).To(gomega.BeTrue())
			gomega.Expect(create.SurfaceID).To(gomega.Equal("from-options"))
			gomega.Expect(create.CatalogID).To(gomega.Equal(a2ui.DefaultCatalogID))
		})
	})

	ginkgo.Context("when the model returns a structurally broken spec", func() {
		ginkgo.It("fails with ErrValidation instead of broken messages", func() {
			object := validSpecObject()
			object["components"] = []any{
				map[string]any{"id": "no-root", "component": "Text", "properties": map[string]any{"text": "x"}},
			}
			model := &fakeModel{object: object}

			result, err := a2ui.Generate(ginkgoCtx(), newAgent(model), a2ui.GenerateOptions{}, testImage())
			gomega.Expect(result).To(gomega.BeNil())
			gomega.Expect(errors.Is(err, a2ui.ErrValidation)).To(gomega.BeTrue())
		})
	})

	ginkgo.Context("when the model call fails", func() {
		ginkgo.It("propagates a classified model error", func() {
			model := &fakeModel{err: errors.New("boom")}

			result, err := a2ui.Generate(ginkgoCtx(), newAgent(model), a2ui.GenerateOptions{}, testImage())
			gomega.Expect(result).To(gomega.BeNil())
			gomega.Expect(err).To(gomega.HaveOccurred())

			modelErr := extractModelError(err)
			gomega.Expect(modelErr).To(gomega.HaveOccurred(), "error should be a classified vision.ModelError")
		})
	})

	ginkgo.Context("the prompt sent to the model", func() {
		ginkgo.It("carries the catalog signatures, structure rules, and task", func() {
			model := &fakeModel{object: validSpecObject()}

			_, err := a2ui.Generate(ginkgoCtx(), newAgent(model), a2ui.GenerateOptions{
				Task: "Build a review dashboard from the screenshot",
			}, testImage())
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			gomega.Expect(model.called).To(gomega.Equal(1))
			gomega.Expect(model.schemaSeen).To(gomega.BeTrue(), "structured output schema must be attached")

			prompt := promptText(model.promptSeen)
			gomega.Expect(prompt).To(gomega.ContainSubstring("A2UI"))
			gomega.Expect(prompt).To(gomega.ContainSubstring("Column: children*"))
			gomega.Expect(prompt).To(gomega.ContainSubstring(`"root"`))
			gomega.Expect(prompt).To(gomega.ContainSubstring("Build a review dashboard from the screenshot"))
		})
	})
})

// ginkgoCtx returns the suite's context.
func ginkgoCtx() context.Context {
	return context.Background()
}

// extractModelError pulls the classified model error out of the chain, if
// any; nil when the error is not classified.
func extractModelError(err error) error {
	modelErr, ok := errors.AsType[*vision.ModelError](err)
	if !ok {
		return nil
	}

	return modelErr
}
