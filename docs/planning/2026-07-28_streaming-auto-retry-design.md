# Streaming Auto-Retry Design

**Status:** Design proposal (not implemented)
**Date:** 2026-07-28

## Problem

`AnalyzeStream`, `AnalyzeConversationStream`, and `AnalyzeStructuredStream`
do **not** auto-retry on transient failures (`Config.Retry` is ignored). This
is a deliberate gap: once the caller has received partial output via the
streaming callback, retrying the entire request would either:

1. **Re-send already-delivered chunks** (duplicate output), or
2. **Require buffering** all output until completion (defeats the purpose of
   streaming), or
3. **Require delta tracking** so only new chunks are sent after retry (complex,
   error-prone, provider-specific).

## Current Behavior

```
Caller → AnalyzeStream → Agent.Generate (stream)
                          ↓
                    chunk → callback("hello ")
                    chunk → callback("wor")  ← provider disconnects
                    error → return ModelError{Kind: Network}
Caller must manually wrap in WithRetry[T]
```

## Design Options

### Option A: Buffer-and-Replay (Rejected)

Buffer all chunks. On failure, retry transparently. On success, flush the
buffer to the callback.

**Pros:** Caller-transparent, simple semantics.
**Cons:** Defeats streaming (caller sees nothing until the entire response
arrives). Memory usage for long responses.

### Option B: Retry-Before-First-Chunk (Recommended)

Retry is only allowed **before** the first chunk is delivered to the caller.
Once streaming begins, failures are returned as-is.

```
AnalyzeStream:
  1. for attempt := 1; attempt <= maxAttempts; attempt++ {
  2.   stream = agent.Generate(...)
  3.   firstChunk = true
  4.   for chunk in stream {
  5.     if firstChunk && isRetryableError(chunk.err) → retry
  6.     callback(chunk.text)
  7.     firstChunk = false
  8.   }
  9.   return  // success or non-retryable error
 10. }
```

**Pros:** No buffering, no duplicate output, caller still gets streaming
benefits. Covers the common case where the provider fails during connection
setup (before any output).

**Cons:** Does not help when the provider fails mid-stream (after delivering
some chunks). But that's an inherent limitation of streaming.

### Option C: Caller-Controlled Retry Token

Provide a `RetryToken` that the caller can pass back to resume from where
streaming left off. The token encodes the number of chunks already delivered.

**Pros:** Full retry capability even mid-stream.
**Cons:** Complex API. Provider-specific semantics (some providers support
resume, most don't). Over-engineered for a vision-analysis SDK.

## Recommendation

**Option B** (retry-before-first-chunk) is the pragmatic choice. It handles
the most common transient failures (connection setup, rate limiting before
first byte) without the complexity of buffering or resume tokens.

## Implementation Sketch

```go
func (a *Agent) AnalyzeStream(ctx context.Context, prompt string, cb func(string) error, imgs ...*ImageSource) (*AnalyzeResult, error) {
    if a.config.Retry == nil {
        return a.analyzeStreamOnce(ctx, prompt, cb, imgs...)
    }

    cfg := effectiveRetryConfig(a.config.Retry)
    var lastErr error

    for attempt := range cfg.MaxAttempts {
        var firstChunk bool = true
        result, err := a.analyzeStreamOnce(ctx, prompt, func(text string) error {
            firstChunk = false
            return cb(text)
        }, imgs...)

        if err == nil {
            return result, nil
        }

        lastErr = err
        if !firstChunk {
            // Already delivered output — cannot retry without duplicates
            return nil, err
        }

        if me, ok := errors.AsType[*apperrors.ModelError](err); ok && !me.IsRetryable() {
            return nil, err
        }

        backoff := cfg.backoff(attempt)
        select {
        case <-time.After(backoff):
        case <-ctx.Done():
            return nil, ctx.Err()
        }
    }

    return nil, lastErr
}
```

## Open Questions

1. Should the `firstChunk` guard be configurable? (Default: no, keep it simple)
2. Should structured streaming (`AnalyzeStructuredStream`) use the same pattern?
   (Yes — same semantics apply)
3. Should the retry counter be reset if the provider sends data then fails?
   (No — firstChunk=false means no retry, period)
