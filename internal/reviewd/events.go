package reviewed

import "time"

// Event type constants recorded on View streams. One stream exists per
// project:Page--theme--viewport and holds the full history of that view.
const (
	EventViewCaptured = "view.captured"
	EventViewReviewed = "view.reviewed"
	EventViewCompared = "view.compared"
)

// ScoreUnknown is the Score value used when the model output contained no
// parseable score line.
const ScoreUnknown = -1

// Captured is the payload of view.captured: a screenshot with a hash the
// scanner has not seen before was found and archived in the blob store.
type Captured struct {
	SourcePath string    `json:"sourcePath"`
	BlobPath   string    `json:"blobPath"`
	SHA256     string    `json:"sha256"`
	CapturedAt time.Time `json:"capturedAt"`
}

// Reviewed is the payload of view.reviewed: the model reviewed the capture
// with the given hash.
type Reviewed struct {
	SHA256     string    `json:"sha256"`
	Model      string    `json:"model"`
	Markdown   string    `json:"markdown"`
	Score      int       `json:"score"`
	ReviewedAt time.Time `json:"reviewedAt"`
}

// Compared is the payload of view.compared: the model diffed two captures of
// the same view (before the change, after the change).
type Compared struct {
	BeforeSHA256   string    `json:"beforeSha256"`
	BeforeBlobPath string    `json:"beforeBlobPath"`
	AfterSHA256    string    `json:"afterSha256"`
	AfterBlobPath  string    `json:"afterBlobPath"`
	Model          string    `json:"model"`
	Markdown       string    `json:"markdown"`
	ComparedAt     time.Time `json:"comparedAt"`
}
