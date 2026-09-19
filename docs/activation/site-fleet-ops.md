# Site Fleet Ops — Manual Actions Pending + Machine Notes

_Created 2026-09-19 during the website vision-review sweep follow-up (see
`emeet-pixyd/docs/planning/2026-09-19_15-33_website-vision-review-followup-pareto-plan.md`,
tasks F1/F4/F7/F8/F9/F10, M1/M2/M3)._

Everything in §1 needs a human with console/DNS/sudo access. Everything in §2
is recorded fact for future sessions.

## 1. Manual actions pending (blocked on console / DNS / sudo)

### 1.1 cmdguard.lars.software — attach domain in Firebase console (M1)

Symptom: browser shows a TLS/security error on `https://cmdguard.lars.software`.
The site itself is fine and served at `https://cmdguard.web.app` (HTTP 200).
Root cause: the custom domain was never attached to the `cmdguard` hosting
site in the Firebase console, so Firebase serves a default/fallback cert.

Click-by-click (console.web.app / console.firebase.google.com):

1. Open <https://console.firebase.google.com> → project **lars-software**.
2. Left sidebar → **Hosting**.
3. In the hosting site selector (top of the Hosting page), pick site **cmdguard**.
4. Open the **Add custom domain** / domain list panel → **Add custom domain**.
5. Enter `cmdguard.lars.software` → **Continue**.
6. If Firebase asks for a TXT verification record, add it in Namecheap
   (Domain List → lars.software → Advanced DNS → Add New Record:
   Type `TXT`, Host `_acme-challenge.cmdguard` — copy the exact value Firebase
   shows). Wait 1–2 min, click **Verify**.
7. Firebase shows the A/CNAME records it needs. For an existing `lars.software`
   subdomain, add in Namecheap → Advanced DNS:
   - Type `CNAME`, Host `cmdguard`, Value `cmdguard.web.app.` (keep trailing dot
     if Namecheap shows it), TTL Automatic.
   - Delete any conflicting A record for the same host if present.
8. Back in Firebase, click **Continue**; provisioning takes minutes to ~24 h
   for the managed cert. The console shows "Pending" until the cert is live.
9. When the console shows "Connected": verify with
   `bash scripts/site-monitor.sh` (this repo) — `cmdguard.lars.software` must
   flip from KNOWN to OK and the monitor alerts "recovered".

After attach: re-shoot + re-review cmdguard at the custom domain (plan F3).

### 1.2 typespec-asyncapi.lars.software — Namecheap CNAME + console attach (M2)

Symptom: the host does not resolve at all (NXDOMAIN). The site is deployed and
served at `https://typespec-asyncapi.web.app` (HTTP 200).

Steps:

1. Namecheap → Domain List → **lars.software** → **Advanced DNS**.
2. **Add New Record**: Type `CNAME`, Host `typespec-asyncapi`, Value
   `typespec-asyncapi.web.app.`, TTL Automatic → save.
3. Firebase console → project **lars-software** → Hosting → select site
   **typespec-asyncapi** → **Add custom domain** → `typespec-asyncapi.lars.software`
   → continue through verification (see §1.1 step 6–8).
4. Wait for "Connected", then `bash scripts/site-monitor.sh` must show OK,
   and `https://typespec-asyncapi.lars.software/og.png` must return 200.

After attach: re-shoot + re-review typespec at the custom domain (plan F6).

### 1.3 /data NVMe SMART check — run with sudo (M3, F7)

Measured 2026-09-19 (plain user `dd`, no sudo needed):

| Filesystem | Device | Write (O_DIRECT) | Read (O_DIRECT) | Read (cached) |
| ---------- | ------ | ---------------- | --------------- | ------------- |
| `/data`    | Lexar SSD NQ790 2TB (nvme1n1p8) | **5.3 MB/s** | **8.0 MB/s** | 19.4 MB/s |
| `/`        | Samsung SSD 970 EVO Plus (nvme0n1p2) | 4.2 GB/s | 3.1 GB/s | — |

`/data` is ~800× slower than the healthy root NVMe — the device or its
partition is failing (or stuck in a degenerate state). Run:

```bash
sudo smartctl -a /dev/nvme1n1          # full SMART incl. media errors
sudo smartctl -x /dev/nvme1n1          # extended info
sudo nvme smart-log /dev/nvme1n1       # if nvme-cli installed
```

Look for: `media_errors`, `critical_warning`, `percentage_used`,
`available_spare` below threshold, growing `unsafe_shutdowns`.

### 1.4 Decision input: what rides on /data

`/data/ai` holds **378G** (101G HF cache + 262G `models/`) — root has only
123G free, so a wholesale migration is impossible. The failing disk changes
the priority: SMART first (§1.3), then either (a) replace the Lexar NQ790 and
copy the pool over, or (b) selectively migrate the actively-used models
(~9G for the VL 8B Q8 + mmproj) to `/` and re-acquire the rest on demand.

Given the measured speeds, **loading models from /data is the bottleneck of
every local ML workload** (an 8 GB Q8 GGUF takes ~25 min to read cold from
/data vs ~2 s from `/`). Field-measured 2026-09-19: a full llama-server
reload of the VL 8B Q8 + mmproj took **~13 min** — this is why
`scripts/vision-stack-up.sh` defaults to a 900 s health wait (the original
120 s default aborted the first monthly fleet-review cycle while the model
was still loading).

Selective migration of the pipeline-critical model (do this even before the
SMART verdict — it only reads ~9G at ~8 MB/s once, ~20 min):

```bash
mkdir -p ~/ai-models    # on the healthy nvme0n1 root
rsync -a --info=progress2 \
  /data/ai/cache/huggingface/hub/models--GitMyLo--nsfwcaption-qwen3-vl-8b-v3-gguf/ \
  ~/ai-models/models--GitMyLo--nsfwcaption-qwen3-vl-8b-v3-gguf/
# then repoint llama-server -m/--mmproj to ~/ai-models/... and re-test
```

## 2. Machine notes (durable facts)

### 2.1 Deploy/auth path that works

```bash
export GOOGLE_APPLICATION_CREDENTIALS=$HOME/.config/gcloud/application_default_credentials.json
nix shell nixpkgs#firebase-tools -c firebase <cmd> --project lars-software
```

The Firebase **domains API returns 403** even with these credentials —
custom-domain attach is console-only (done in the browser, §1.1/§1.2).
`hosting:sites:list` and deploys work fine from the CLI.

### 2.2 /data contents relevant to the pipeline

(measured 2026-09-19; refresh with `du -sh /data/ai/*` if it matters)

- Models used by the vision review stack live under
  `/data/ai/cache/huggingface/hub/models--GitMyLo--nsfwcaption-qwen3-vl-8b-v3-gguf/`
  (snapshot `eb52b76411f34ea197558ec03eb15b2814d1b0c2`).
- llama-server must be started with **direct local paths** (`-m …/NSFWCaption-v3-Qwen3-VL-8B-Q8_0.gguf --mmproj …/mmproj-NSFWCaption-v3.gguf --host 127.0.0.1 --port 8390`); the `-hf GitMyLo/…` flag hangs ~10 min resolving/downloading through the degraded disk + network. See AGENTS.md for the full working invocation.

### 2.3 Site fleet monitor

`scripts/site-monitor.sh` (this repo) checks all fleet URLs hourly via the
`site-monitor.timer` user timer (installed in `~/.config/systemd/user/`,
enabled through `timers.target.wants`). It alerts on state transitions via
`notify-send`, logs to `~/.local/state/site-monitor/log`, and tracks the two
KNOWN-broken custom domains (§1.1, §1.2) without alerting until they are
fixed. If a home-manager switch wipes the `timers.target.wants` symlink,
re-create it:

```bash
ln -sf ../site-monitor.timer ~/.config/systemd/user/timers.target.wants/site-monitor.timer
systemctl --user daemon-reload
```
