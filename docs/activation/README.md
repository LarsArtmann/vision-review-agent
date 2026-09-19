# visionreviewd Activation — Worked Examples

Configs used for the first real activation (2026-08-18, user-space stack:
llama-server on `127.0.0.1:8390`, model cached under a shared `HF_HOME`).
They are reference material — paths are machine-specific; copy and adjust.

- `visionreviewd-single-view.json` — one DiscordSync golden; used for the
  first real `once` + `doctor` gate.
- `visionreviewd-eight-views.json` — Dashboard ×4 + Messages_hide_bots ×4;
  used for the 8-view pass (7 reviewed, 1 correctly skipped by skip-seen).
- `visionreviewd-websites-17.json` — the 17-site website fleet config
  (2026-09-19); runtime copy lives at `~/.config/visionreviewd/websites.json`.
  Pair with `scripts/shoot-sites.sh`, which produces the screenshots it globs.

Both first two point `dataDir`/`reviewsDir` at `~/.local/share/vision-review-agent`
(durable), so wiping `reviewsDir` and running `replay` rebuilds everything
from the event store.

Model server (restart command — **direct local paths, do NOT use `-hf`**):

```bash
SNAP=/data/ai/cache/huggingface/hub/models--GitMyLo--nsfwcaption-qwen3-vl-8b-v3-gguf/snapshots/eb52b76411f34ea197558ec03eb15b2814d1b0c2
llama-server -m "$SNAP/NSFWCaption-v3-Qwen3-VL-8B-Q8_0.gguf" \
  --mmproj "$SNAP/mmproj-NSFWCaption-v3.gguf" \
  --host 127.0.0.1 --port 8390
```

The `-hf GitMyLo/...` form hangs ~10 min (model resolution + download over
the degraded `/data` disk, see `site-fleet-ops.md` §1.3); direct paths start
in seconds. Health gate: `http://127.0.0.1:8390/health` returns
`{"status":"ok"}`.

Smoke sequence that was verified end-to-end:

```bash
visionreviewd doctor  -config visionreviewd-single-view.json
visionreviewd once    -config visionreviewd-single-view.json
visionreviewd compare -config visionreviewd-single-view.json -project discordsync before.png after.png
visionreviewd events  -config visionreviewd-single-view.json -project discordsync
visionreviewd replay  -config visionreviewd-single-view.json
```

Website fleet pipeline (17 sites, verified 2026-09-19):

```bash
llama-server ...                # direct-path command above
scripts/shoot-sites.sh          # capture + verify all 17 homes (~10 min)
visionreviewd once -config ~/.config/visionreviewd/websites.json
```

Keep capture and review serialized — never overlap a shoot with a running
`once` pass.

### `sourceURLs` — page provenance in reviews

The fleet config maps each project to the live URL its screenshots came from
(`Config.SourceURLs`, added 2026-09-19). When set, each capture event records
`sourceURL` and the review markdown renders it as a provenance line right
under the heading:

```json
{
  "sourceURLs": {
    "gogenfilter": "https://gogenfilter.lars.software"
  }
}
```

Rendered per view in `reviews/<project>/<view>.md`:

```markdown
- **Page:** <https://gogenfilter.lars.software>
```

Omit the field (or a single project from it) and reviews render exactly as
before — the feature degrades by omission.

For host-level (NixOS/SystemNix) enablement see
[`../visionreviewd-systemnix.md`](../visionreviewd-systemnix.md).
