# visionreviewd Activation — Worked Examples

Configs used for the first real activation (2026-08-18, user-space stack:
llama-server on `127.0.0.1:8390`, model cached under a shared `HF_HOME`).
They are reference material — paths are machine-specific; copy and adjust.

- `visionreviewd-single-view.json` — one DiscordSync golden; used for the
  first real `once` + `doctor` gate.
- `visionreviewd-eight-views.json` — Dashboard ×4 + Messages_hide_bots ×4;
  used for the 8-view pass (7 reviewed, 1 correctly skipped by skip-seen).

Both point `dataDir`/`reviewsDir` at `~/.local/share/vision-review-agent`
(durable), so wiping `reviewsDir` and running `replay` rebuilds everything
from the event store.

Model server (restart command):

```bash
HF_HOME=/data/ai/cache/huggingface llama-server \
  -hf GitMyLo/nsfwcaption-qwen3-vl-8b-v3-gguf:Q8_0 \
  --host 127.0.0.1 --port 8390
```

Smoke sequence that was verified end-to-end:

```bash
visionreviewd doctor  -config visionreviewd-single-view.json
visionreviewd once    -config visionreviewd-single-view.json
visionreviewd compare -config visionreviewd-single-view.json -project discordsync before.png after.png
visionreviewd events  -config visionreviewd-single-view.json -project discordsync
visionreviewd replay  -config visionreviewd-single-view.json
```

For host-level (NixOS/SystemNix) enablement see
[`visionreviewd-systemnix.md`](visionreviewd-systemnix.md).
