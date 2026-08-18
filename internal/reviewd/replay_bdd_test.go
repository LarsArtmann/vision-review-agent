package reviewed_test

import (
	"context"
	"os"
	"path/filepath"

	reviewed "github.com/larsartmann/vision-review-agent/internal/reviewd"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Replay", func() {
	const (
		project = "myapp"
		viewKey = "Home--dark--desktop"
	)

	var (
		reviewsDir string
		store      *reviewed.Store
		pipeline   *reviewed.Pipeline
	)

	BeforeEach(func() {
		dataDir := GinkgoT().TempDir()
		reviewsDir = GinkgoT().TempDir()

		var err error

		store, err = reviewed.OpenStore(filepath.Join(dataDir, "events.db"), nil)
		Expect(err).NotTo(HaveOccurred())

		reviewer, err := reviewed.NewReviewer(
			&stubLanguageModel{markdown: "## Diff\nSpacing improved.\n\nScore: 9/10"},
			"stub-review-model",
			0,
		)
		Expect(err).NotTo(HaveOccurred())

		pipeline, err = reviewed.NewPipeline(
			reviewer, store, reviewed.NewBlobStore(dataDir), reviewed.NewWriter(reviewsDir), nil,
		)
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		Expect(store.Close()).To(Succeed())
	})

	When("a stream only ever saw a manual comparison", func() {
		var beforePath, afterPath string

		BeforeEach(func() {
			dir := GinkgoT().TempDir()
			beforePath = filepath.Join(dir, "before.png")
			afterPath = filepath.Join(dir, viewKey+".png")

			Expect(writeShotPNG(beforePath)).To(Succeed())
			Expect(writeChangedShotPNG(afterPath)).To(Succeed())

			Expect(pipeline.CompareManually(context.Background(), project, beforePath, afterPath)).To(Succeed())
		})

		It("records the comparison event", func() {
			key, err := reviewed.ParseViewKey(viewKey)
			Expect(err).NotTo(HaveOccurred())

			state, _, err := store.LoadView(context.Background(), project, key)
			Expect(err).NotTo(HaveOccurred())
			Expect(state.Comparisons).To(Equal(1))
		})

		It("rebuilds byte-identical markdown from the journal after a wipe", func() {
			comparisonGlob := filepath.Join(reviewsDir, project, "comparisons", "*_"+viewKey+".md")
			before, err := filepath.Glob(comparisonGlob)
			Expect(err).NotTo(HaveOccurred())
			Expect(before).To(HaveLen(1))

			original, err := os.ReadFile(before[0])
			Expect(err).NotTo(HaveOccurred())

			// Wipe the whole projection; the journal is the source of truth.
			Expect(os.RemoveAll(reviewsDir)).To(Succeed())
			Expect(os.MkdirAll(reviewsDir, 0o750)).To(Succeed())

			result, err := reviewed.Replay(context.Background(), store, reviewed.NewWriter(reviewsDir))
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Projects).To(Equal(1))
			Expect(result.Comparisons).To(Equal(1))

			after, err := filepath.Glob(comparisonGlob)
			Expect(err).NotTo(HaveOccurred())
			Expect(after).To(HaveLen(1))

			rebuilt, err := os.ReadFile(after[0])
			Expect(err).NotTo(HaveOccurred())
			Expect(rebuilt).To(Equal(original), "replay must rebuild the comparison byte-identically")
		})

		It("lists the view in the replayed INDEX", func() {
			Expect(os.RemoveAll(reviewsDir)).To(Succeed())
			Expect(os.MkdirAll(reviewsDir, 0o750)).To(Succeed())

			_, err := reviewed.Replay(context.Background(), store, reviewed.NewWriter(reviewsDir))
			Expect(err).NotTo(HaveOccurred())

			index, err := os.ReadFile(filepath.Join(reviewsDir, project, "INDEX.md"))
			Expect(err).NotTo(HaveOccurred())
			Expect(string(index)).To(ContainSubstring(viewKey))
		})
	})
})
