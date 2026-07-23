# Roadmap

Long-term direction and raw ideas not yet refined into actionable tasks.

## Near-term

- **Tool/function calling** — fantasy supports `WithTools` and `NewAgentTool[T]`; expose typed tools so the agent can call back into Go code during analysis (e.g., "fetch accessibility guidelines", "measure contrast ratio")
- **Provider-defined tools** — Expose `WithProviderDefinedTools` for provider-native tools like web search and computer use
- **PrepareStep interceptor** — Expose `WithPrepareStep` for per-step model/system-prompt/tool manipulation
- **Stop conditions** — Expose `WithStopConditions` for composable stopping logic (step count, tool call detection, token budget)

## Mid-term

- **Retry middleware with backoff** — Automatic retry on retryable errors using configurable backoff strategies (exponential, jittered)
- **Cost tracking** — Track and expose token costs across analysis calls (input, output, cache read/creation, reasoning tokens)
- **Image preprocessing** — Resize, compress, or convert images before sending to the model (reduce token usage, support provider-specific limits)
- **Custom HTTP client** — Allow injecting a custom `*http.Client` for LoadImageFromURL and provider connections (proxies, timeouts, TLS)
- **Context-aware batch** — Let batch analysis share context or conversation state across images

## Long-term Ideas

- **Agent orchestration** — Multi-agent workflows where one agent's output feeds another's input (e.g., accessibility checker feeds recommendations to fixer)
- **Plugin system** — Extensible middleware/interceptor chain for pre/post-processing of analysis requests and results
- **Result caching** — Cache analysis results by image hash + prompt to avoid redundant API calls
- **Provider failover** — Automatically try a secondary provider if the primary fails with a retryable error
- **Observability integration** — OpenTelemetry spans for analysis lifecycle (start, model call, finish)
- **Prompt templates** — Pre-built, parameterized prompt templates for common use cases (accessibility audit, UX review, layout analysis, bug detection)
- **Diff analysis** — Compare two screenshots and describe the differences structurally
- **Video frame analysis** — Extract frames from video and analyze them as a batch or sequence
