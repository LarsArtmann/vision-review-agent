package vision

import (
	"context"
	"errors"
	"net/http"

	"charm.land/fantasy"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

var _ = ginkgo.Describe("Error Classification", func() {
	ginkgo.Describe("Consumer retry decisions", func() {
		ginkgo.DescribeTable(
			"should classify provider errors into actionable kinds",
			func(statusCode int, expectedKind ErrorKind, shouldRetry bool) {
				model := &mockModel{generateErr: newTestProviderErr(statusCode)}
				_, agent := setupAgentWithModel(model)

				_, err := agent.Analyze(context.Background(), "prompt", ImageSrc())
				gomega.Expect(err).To(gomega.HaveOccurred())

				me, ok := errors.AsType[*ModelError](err)
				gomega.Expect(ok).To(gomega.BeTrue(), "error should be extractable as *ModelError")
				gomega.Expect(me.Kind).To(gomega.Equal(expectedKind))
				gomega.Expect(me.IsRetryable()).To(gomega.Equal(shouldRetry))
			},
			ginkgo.Entry("429 rate limited → retry", http.StatusTooManyRequests, KindRateLimited, true),
			ginkgo.Entry("500 server error → retry", http.StatusInternalServerError, KindServerError, true),
			ginkgo.Entry("502 bad gateway → retry", http.StatusBadGateway, KindServerError, true),
			ginkgo.Entry(
				"503 service unavailable → retry",
				http.StatusServiceUnavailable,
				KindServiceUnavailable,
				true,
			),
			ginkgo.Entry("408 request timeout → retry", http.StatusRequestTimeout, KindTimeout, true),
			ginkgo.Entry("401 unauthorized → no retry", http.StatusUnauthorized, KindAuthentication, false),
			ginkgo.Entry("403 forbidden → no retry", http.StatusForbidden, KindAuthentication, false),
			ginkgo.Entry("404 not found → no retry", http.StatusNotFound, KindNotFound, false),
			ginkgo.Entry("400 bad request → no retry", http.StatusBadRequest, KindBadRequest, false),
			ginkgo.Entry("501 not implemented → no retry", http.StatusNotImplemented, KindNotImplemented, false),
		)

		ginkgo.It("should classify context cancellation as non-retryable", func() {
			model := &mockModel{generateErr: context.Canceled}
			_, agent := setupAgentWithModel(model)

			_, err := agent.Analyze(context.Background(), "prompt", ImageSrc())
			gomega.Expect(err).To(gomega.HaveOccurred())

			me, ok := errors.AsType[*ModelError](err)
			gomega.Expect(ok).To(gomega.BeTrue())
			gomega.Expect(me.Kind).To(gomega.Equal(KindCancelled))
			gomega.Expect(me.IsRetryable()).To(gomega.BeFalse())
		})

		ginkgo.It("should classify deadline exceeded as retryable timeout", func() {
			model := &mockModel{generateErr: context.DeadlineExceeded}
			_, agent := setupAgentWithModel(model)

			_, err := agent.Analyze(context.Background(), "prompt", ImageSrc())
			gomega.Expect(err).To(gomega.HaveOccurred())

			me, ok := errors.AsType[*ModelError](err)
			gomega.Expect(ok).To(gomega.BeTrue())
			gomega.Expect(me.Kind).To(gomega.Equal(KindTimeout))
			gomega.Expect(me.IsRetryable()).To(gomega.BeTrue())
		})

		ginkgo.It("should preserve the original cause through the wrapper", func() {
			providerErr := newTestProviderErr(http.StatusTooManyRequests)
			model := &mockModel{generateErr: providerErr}
			_, agent := setupAgentWithModel(model)

			_, err := agent.Analyze(context.Background(), "prompt", ImageSrc())
			gomega.Expect(err).To(gomega.HaveOccurred())

			extracted, ok := errors.AsType[*fantasy.ProviderError](err)
			gomega.Expect(ok).To(gomega.BeTrue(), "ProviderError must be extractable through ModelError")
			gomega.Expect(extracted.StatusCode).To(gomega.Equal(http.StatusTooManyRequests))
		})
	})

	ginkgo.Describe("IsRetryable convenience function", func() {
		ginkgo.It("should return true for retryable classified errors", func() {
			model := &mockModel{generateErr: newTestProviderErr(http.StatusTooManyRequests)}
			_, agent := setupAgentWithModel(model)

			_, err := agent.Analyze(context.Background(), "prompt", ImageSrc())
			gomega.Expect(IsRetryable(err)).To(gomega.BeTrue())
		})

		ginkgo.It("should return false for non-retryable classified errors", func() {
			model := &mockModel{generateErr: newTestProviderErr(http.StatusUnauthorized)}
			_, agent := setupAgentWithModel(model)

			_, err := agent.Analyze(context.Background(), "prompt", ImageSrc())
			gomega.Expect(IsRetryable(err)).To(gomega.BeFalse())
		})
	})
})
