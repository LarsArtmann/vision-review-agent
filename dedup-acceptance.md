# Dedup Acceptance Log

Remaining clone groups after deduplication. Each entry is evaluated and
deliberately accepted with a one-line reason.

## Production Code

### `if err != nil { return nil, err }` + `defer prep.cancel()` (5 occurrences)

**Locations:**
- `pkg/vision/vision.go` — Analyze, AnalyzeStream, AnalyzeConversation, AnalyzeConversationStream
- `pkg/vision/structured.go` — AnalyzeStructured, AnalyzeStructuredStream

**Reason:** Standard Go error-check idiom after calling `prepare()`. The 4-line
pattern (`if err != nil / return nil, err / } / defer prep.cancel()`) cannot be
abstracted without making the code worse — Go has no `?` operator, and every
caller must still handle the error. This is the most fundamental Go control-flow
pattern; flagging it is noise, not signal.

## Examples

### CLI setup boilerplate: `cli.RequireArgc(2)` + `ctx` + `model` (5 occurrences)

**Locations:**
- `examples/batch/main.go`
- `examples/url-loading/main.go`
- `examples/error-handling/main.go`
- `examples/hooks/main.go`
- `examples/openai/main.go`

**Reason:** Each example is a self-contained, copy-pasteable teaching program.
Extracting shared setup into a helper would break the "one file tells the whole
story" property that makes examples useful. The duplicated lines (3-4 per file)
are minimal entry-point boilerplate.

### `UIReview` struct (2 occurrences)

**Locations:**
- `examples/structured/main.go`
- `examples/structured-stream/main.go`

**Reason:** The struct tags have intentionally different descriptions tailored to
each example's focus. Moving to a shared package would couple two independently
runnable examples and obscure the schema definition from the reader.
