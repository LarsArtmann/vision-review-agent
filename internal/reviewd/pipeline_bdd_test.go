package reviewed_test

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"path/filepath"

	reviewed "github.com/larsartmann/vision-review-agent/internal/reviewd"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Pipeline Pass", func() {
	const (
		project    = "myapp"
		viewKeyStr = "Home--dark--desktop"
	)

	var (
		model      *stubLanguageModel
		store      *reviewed.Store
		pipeline   *reviewed.Pipeline
		shotsDir   string
		dataDir    string
		reviewsDir string
		writer     *reviewed.Writer
	)

	newProjects := func() map[string][]string {
		return map[string][]string{project: {filepath.Join(shotsDir, "*.png")}}
	}

	BeforeEach(func() {
		var err error

		shotsDir, err = os.MkdirTemp("", "reviewd-shots-*")
		Expect(err).NotTo(HaveOccurred())

		dataDir, err = os.MkdirTemp("", "reviewd-data-*")
		Expect(err).NotTo(HaveOccurred())

		reviewsDir, err = os.MkdirTemp("", "reviewd-reviews-*")
		Expect(err).NotTo(HaveOccurred())

		model = &stubLanguageModel{markdown: "## Review\nLooks good.\n\n**Score: 8/10**"}

		reviewer, reviewerErr := reviewed.NewReviewer(model, "stub-model", 0)
		Expect(reviewerErr).NotTo(HaveOccurred())

		store, err = reviewed.OpenStore(filepath.Join(dataDir, "events.db"), slog.Default())
		Expect(err).NotTo(HaveOccurred())

		writer = reviewed.NewWriter(reviewsDir)

		pipeline, err = reviewed.NewPipeline(reviewer, store, reviewed.NewBlobStore(dataDir), writer, nil)
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		Expect(store.Close()).To(Succeed())
		Expect(os.RemoveAll(shotsDir)).To(Succeed())
		Expect(os.RemoveAll(dataDir)).To(Succeed())
		Expect(os.RemoveAll(reviewsDir)).To(Succeed())
	})

	Context("when a project has a new screenshot", func() {
		BeforeEach(func() {
			Expect(writeShotPNG(filepath.Join(shotsDir, viewKeyStr+".png"))).To(Succeed())
		})

		It("captures, reviews, and writes the review plus INDEX", func(ctx SpecContext) {
			result, err := pipeline.Pass(ctx, newProjects())
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal(reviewed.PassResult{
				Projects: 1,
				Views:    1,
				Captured: 1,
				Skipped:  0,
				Reviewed: 1,
				Compared: 0,
			}))

			state, _, loadErr := store.LoadView(ctx, project, homeViewKey())
			Expect(loadErr).NotTo(HaveOccurred())
			Expect(state.Captures).To(Equal(1))
			Expect(state.Reviews).To(Equal(1))
			Expect(state.LastScore).To(Equal(8))
			Expect(state.PrevScore).To(Equal(reviewed.ScoreUnknown))
			Expect(state.ReviewedSHA).To(Equal(state.SHA256))
			Expect(state.NeedsReview()).To(BeFalse())

			events, eventsErr := store.AllEvents(ctx)
			Expect(eventsErr).NotTo(HaveOccurred())
			Expect(countEvents(events, reviewed.EventViewCaptured)).To(Equal(1))
			Expect(countEvents(events, reviewed.EventViewReviewed)).To(Equal(1))
			Expect(countEvents(events, reviewed.EventViewCompared)).To(Equal(0))

			reviewPath := writer.ViewReviewPath(project, homeViewKey())
			Expect(readFileOrFail(reviewPath)).To(ContainSubstring("## Review"))
			Expect(readFileOrFail(reviewPath)).To(ContainSubstring("8/10"))

			index := readFileOrFail(writer.IndexPath(project))
			Expect(index).To(ContainSubstring("`" + viewKeyStr + "`"))
			Expect(index).To(ContainSubstring("8/10"))
			Expect(index).To(ContainSubstring("·"))

			entries, readErr := os.ReadDir(filepath.Join(dataDir, "images"))
			Expect(readErr).NotTo(HaveOccurred())
			Expect(entries).To(HaveLen(1))
		})
	})

	Context("when the screenshot is unchanged since the last pass", func() {
		BeforeEach(func() {
			Expect(writeShotPNG(filepath.Join(shotsDir, viewKeyStr+".png"))).To(Succeed())

			_, err := pipeline.Pass(context.Background(), newProjects())
			Expect(err).NotTo(HaveOccurred())
		})

		It("skips the view without calling the model again", func(ctx SpecContext) {
			result, err := pipeline.Pass(ctx, newProjects())
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal(reviewed.PassResult{
				Projects: 1,
				Views:    1,
				Captured: 0,
				Skipped:  1,
				Reviewed: 0,
				Compared: 0,
			}))

			Expect(model.calls()).To(Equal(1))

			events, eventsErr := store.AllEvents(ctx)
			Expect(eventsErr).NotTo(HaveOccurred())
			Expect(countEvents(events, reviewed.EventViewCaptured)).To(Equal(1))
			Expect(countEvents(events, reviewed.EventViewReviewed)).To(Equal(1))
		})
	})

	Context("when the screenshot changed since the last pass", func() {
		BeforeEach(func() {
			shotPath := filepath.Join(shotsDir, viewKeyStr+".png")
			Expect(writeShotPNG(shotPath)).To(Succeed())

			_, err := pipeline.Pass(context.Background(), newProjects())
			Expect(err).NotTo(HaveOccurred())

			Expect(writeChangedShotPNG(shotPath)).To(Succeed())

			model.setMarkdown("## Diff\nBetter spacing.\n\n**Score: 9/10**")
		})

		It("compares before against after and records the new review", func(ctx SpecContext) {
			result, err := pipeline.Pass(ctx, newProjects())
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal(reviewed.PassResult{
				Projects: 1,
				Views:    1,
				Captured: 1,
				Skipped:  0,
				Reviewed: 1,
				Compared: 1,
			}))

			state, _, loadErr := store.LoadView(ctx, project, homeViewKey())
			Expect(loadErr).NotTo(HaveOccurred())
			Expect(state.Captures).To(Equal(2))
			Expect(state.Reviews).To(Equal(2))
			Expect(state.Comparisons).To(Equal(1))
			Expect(state.LastScore).To(Equal(9))
			Expect(state.PrevScore).To(Equal(8))

			events, eventsErr := store.AllEvents(ctx)
			Expect(eventsErr).NotTo(HaveOccurred())
			Expect(countEvents(events, reviewed.EventViewCaptured)).To(Equal(2))
			Expect(countEvents(events, reviewed.EventViewCompared)).To(Equal(1))

			comparisons, readErr := os.ReadDir(filepath.Join(reviewsDir, project, "comparisons"))
			Expect(readErr).NotTo(HaveOccurred())
			Expect(comparisons).To(HaveLen(1))
			Expect(comparisons[0].Name()).To(HaveSuffix(viewKeyStr + ".md"))

			Expect(readFileOrFail(writer.IndexPath(project))).To(ContainSubstring("▲ +1"))

			Expect(model.calls()).To(Equal(3))
			Expect(countFileParts(model.promptAt(1))).To(Equal(2))
		})
	})

	Context("when the model fails on a new screenshot", func() {
		BeforeEach(func() {
			Expect(writeShotPNG(filepath.Join(shotsDir, viewKeyStr+".png"))).To(Succeed())

			model.setGenerateErr(errors.New("model exploded"))
		})

		It("still records the capture and reports the failure", func(ctx SpecContext) {
			result, err := pipeline.Pass(ctx, newProjects())
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("review"))
			Expect(result.Captured).To(Equal(1))
			Expect(result.Reviewed).To(Equal(0))

			state, _, loadErr := store.LoadView(ctx, project, homeViewKey())
			Expect(loadErr).NotTo(HaveOccurred())
			Expect(state.Captures).To(Equal(1))
			Expect(state.Reviews).To(Equal(0))
			Expect(state.LastScore).To(Equal(reviewed.ScoreUnknown))
			Expect(state.NeedsReview()).To(BeTrue())

			index := readFileOrFail(writer.IndexPath(project))
			Expect(index).To(ContainSubstring("`" + viewKeyStr + "`"))
			Expect(index).To(ContainSubstring("?"))
		})
	})
})

// homeViewKey is the static view key every spec fixture uses.
func homeViewKey() reviewed.ViewKey {
	const name = "Home--dark--desktop"

	viewKey, err := reviewed.ParseViewKey(name)
	if err != nil {
		panic("parse view key " + name + ": " + err.Error())
	}

	return viewKey
}
