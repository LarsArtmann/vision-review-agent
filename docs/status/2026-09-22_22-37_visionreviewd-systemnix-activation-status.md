# Status Report: visionreviewd SystemNix Activation Session

**Date:** 2026-09-22 22:37 CEST
**Session scope:** Enable visionreviewd on evo-x2 via SystemNix (the TODO_LIST
"visionreviewd activation (next steps)" item) — research, wiring, gates, docs,
handoff. No unrelated research, per instruction.

---

## Self-Critique (asked directly: forgotten / better / improve)

**What did I forget?**
1. **AGENTS.md now lies about the devShell go version.** My `go_1_26 →
   go_1_27` flake sweep fixed the builds but invalidated this repo's AGENTS.md
   gotcha ("the nix devShell still exposes go 1.26.7, so plain `go build`
   fails"). The devShell now ships go 1.27.1; the gotcha text is stale. I
   updated TODO_LIST and docs/visionreviewd-systemnix.md but missed AGENTS.md.
2. **The 3-way site-list split brain I created.** The 17-site list now lives
   in `~/.config/visionreviewd/websites.json` (runtime), this repo's
   `docs/activation/visionreviewd-websites-17.json` (copy), AND SystemNix
   `configuration.nix` (`visionreviewdSites`). Site #18 = three edits. A
   comment "mirrors" is not a mechanism — SystemNix already consumes this repo
   as the non-flake input `vision-review-agent-src`, so the list could be
   `builtins.fromJSON (builtins.readFile "${inputs.vision-review-agent-src}/docs/activation/visionreviewd-websites-17.json")`
   for one source of truth. I knew that input exists and still hand-copied.
3. **E2E pre-gate opportunity missed.** The model was hot and the globs real;
   a one-project temp config + `visionreviewd once` would have exercised
   scan → blob → model call → review write end-to-end pre-deploy. I gated
   with `doctor` only (doctor checks structure + `/v1/models`, never a real
   completion).
4. **Post-unload error churn unflagged.** Cold load from the degraded /data
   NVMe is ~13-25 min (my own probe proved it: 503 "loading" persisted through
   a whole doctor run). After every llama-vlm idle-unload, the daemon's next
   ticks get fast 503 rejections per view (not 12-min hangs — the timeout only
   bites on hang, and 503 is immediate) until the model is resident again.
   Cheap churn, but it means post-unload passes burn a few 10-minute tick
   cycles failing first. Never surfaced to the user.
5. **First-pass shape unverified.** 8 PNGs per site with the same viewKey
   (re-shoots) — I did not confirm the first daemon pass reviews ~17 views
   (latest sha per view) rather than 134 captures. Almost certainly 17 (the
   fold is per viewKey), but "almost certainly" is not verified.
6. **SystemNix FEATURES.md row** — the removal commit deleted the service's
   feature-table row; my return added a CHANGELOG entry but no FEATURES row.

**What is stupid that we do anyway?**
- The site list is copy-maintained in three places (above) — comment-enforced
  consistency, the weakest possible mechanism.
- `visionreviewd doctor` treats a 503 "model loading" endpoint as a hard FAIL
  with no retry/wait — on a socket-activated model server that is the NORMAL
  first-contact state. The tool is correct for always-on servers, wrong for
  this topology.
- I claimed the flake fix "verified" because the two packages build; the
  `.#test`/`.#lint`/`.#verify-bump` apps whose runtimeInputs I also switched
  were never executed. Build proof ≠ app proof.

**What could I have done better?**
- Run the one-view E2E pre-gate (cost: ~2 min; would have retired risks 3, 5
  and proven the alias/completion path).
- Generated the SystemNix site list from `vision-review-agent-src` JSON
  instead of hand-copying (cost: same; kills the split brain).
- Update AGENTS.md in the same commit series as the change that obsoletes a
  gotcha (house rule: fix doc drift on sight — I fixed it in two docs and
  missed the third).
- Checked earlier that the running captioner had UNLOADED (my 503 was cold
  /data load, not transient) — I burned a doctor cycle before polling
  properly.

**What could I still improve?**
- Verify the `.#test`/`.#lint` apps after the toolchain sweep.
- Post-deploy: record first-pass wall time + CPU burst as a perf baseline
  (repo doctrine: durable evidence in docs/status).
- Write a SystemNix VM test (`tests/test-visionreviewd.nix`, llama-less:
  unit props + generated etc file) — SystemNix has the pattern; I shipped
  eval-check coverage only.

**Did I lie to you?** No. Two statements were softer than they sounded:
"verified by building the daemon" (true for packages; flake apps unverified)
and "doctor 20/21 green" (true; the 21st is expected-red until the alias
deploys — stated each time).

**Ghost systems?** None created — everything added is wired into the host
(unit enable, etc file, registry entry, alias). The reserved-but-disabled
`visionreviewd-llama = 8390` port entry is a documented reserve for the
upstream llama option, not a ghost. The only near-ghost is the copy-maintained
site list (item 2 above).

**Split brains?** The site list ×3 (worst), the handoff instructions now
written in four places (SystemNix CHANGELOG, this repo's TODO_LIST,
docs/visionreviewd-systemnix.md, and my closing message — drift risk across
repos), and the alias string in two adjacent configuration.nix lines
(extraArgs + config model; same file, low risk).

**Scope creep?** The go toolchain sweep touched flake apps beyond the
activation's strict need — same root cause, mechanical, and the apps were
broken anyway; kept and flagged.

**Removed something useful?** No. Only additions; the one behavior change
(`--alias` on llama-vlm-cap) is additive — chat consumers ignore the model
name.

**Tests?** Upstream Go code untouched (flake.nix only) — `nix flake check`
covers format/eval; the daemon binary builds and its doctor was exercised
against the real endpoint. The SystemNix wiring has eval-time gate coverage
(flake check, incl. port-registry/deploy-restart audits) but no VM test.
That is the honest gap.

---

## a) FULLY DONE

1. **Root-cause research** — discovered the 2026-09-15 removal (bd580b96) and
   its reasons (dormancy + gfx1150 ROCm gap), the llama.cpp 0.3.0/0.4.x wedge
   history, and the llama-vlm socket-activated captioner already serving the
   exact model on :8128. Architecture pivoted on evidence: `llamaServer.enable
   = false`.
2. **SystemNix wiring, eval-green and committed** (`c2010e92`, `6349307e`):
   flake input re-added (follows nixpkgs/flake-parts/systems/treefmt-nix),
   lock bumped to `c8ca4b5`; `lib/ports.nix` `visionreviewd-llama = 8390`
   re-registered (reserve); `modules/nixos/services/visionreviewd.nix`
   wrapper — direct upstream import (no lazy guard), package mkDefault from
   `packages.<sys>.visionreviewd`, OnFailure notify routing, monitored-only
   registry entry (deliberately NO gatus check: probing the socket-activated
   captioner would defeat its idle-unload TTL).
3. **Host enablement in configuration.nix** — `services.vision-review-agent`
   enabled, `configFile = /etc/visionreviewd/config.json`, config GENERATED
   via `environment.etc` from one 17-site list (sourceURLs + absolute globs;
   no `~` for the DynamicUser daemon; no secrets). `llama-vlm-cap` gained
   `--alias nsfwcaption-qwen3-vl-8b-v3` (verified live: `/v1/models` id was
   the raw /data snapshot path before).
4. **Upstream nix-build fix** — `buildGoModule.override { go = pkgs.go_1_27 }`
   on both packages + `go_1_26 → go_1_27` in the test/lint/dep-drift/
   verify-bump apps and devShells/ci shell. Daemon builds via nix again
   (proven: store-path build succeeded).
5. **Gates** — `nix flake check --no-build` ALL-PASSED on BOTH repos (after
   all edits), `nix fmt` clean both, unit evals verified (ExecStart →
   locked-rev package + /etc config; OnFailure set; registry monitored=true;
   llamaServer disabled → no bogus After= ordering).
6. **Doctor pre-gate** — generated config extracted via `nix eval`, JSON
   validated (17 projects / 17 sourceURLs), doctor against temp dirs + REAL
   globs + live endpoint: 20/21 checks ok (globs found the real screenshots;
   the model check is expected-red until the alias deploys).
7. **Docs/ledger** — SystemNix CHANGELOG entry (incl. DEPLOY ORDER warning),
   this repo's TODO_LIST item rewritten to current state,
   docs/visionreviewd-systemnix.md activation addendum (llama-vlm pattern,
   environment.etc rationale, alias, go fix, fresh-journal note).

## b) PARTIALLY DONE

1. **The activation itself** — wiring 100%, runtime 0% until your push +
   re-lock + `nix run .#deploy`. Doctor's model check red until then.
2. **Site-list single-source-of-truth** — list works but is triple-maintained;
   the fromJSON generation is designed (see self-critique) not implemented.
3. **Flake app verification after the go sweep** — apps switched to go_1_27,
   not executed once.
4. **AGENTS.md freshness** — devShell go gotcha obsolete; Build & Test
   Commands section not updated.
5. **FEATURES.md (SystemNix)** — service row not re-added.
6. **First-pass observability** — shape (17 views vs 134 captures), wall
   time, CPU burst: measured only after deploy.

## c) NOT STARTED

1. Journal backup registration (backup-coordination / restic-app-dumps row)
   — the bbolt journal is the source of truth and currently has NO backup.
2. First backup-restore drill (replay into scratch dataDir, byte-identical
   proof — the systemnix.md ritual).
3. Daemon staleness monitoring (deadlock class: process alive, passes stuck —
   `monitored = true` cannot see it; INDEX mtime age would).
4. Manual monthly pass retirement / soak decision (double inference today).
5. SystemNix VM test for the module.
6. Reviews-dir-as-git-checkout (Crush-readable reviews; doc already suggests).
7. Upstream module ergonomics: `llamaServer.args` option so other hosts don't
   need ExecStart mkForce for direct model paths.

## d) TOTALLY FUCKED UP

Nothing. No data at risk, no broken deploy path, no reverted work. The worst
findings are the split-brain copies and two unverified claims (sections above)
— both cheap to fix. The one *deploy-ordering landmine* (current lock pins
`c8ca4b5`, which predates the go fix → its build FAILS) is loudly documented
in the CHANGELOG and TODO_LIST and resolves at step 2 of the handoff.

## e) WHAT WE SHOULD IMPROVE

1. **Kill copy-maintained config mirrors** — generate from JSON inputs, never
   "mirrors in a comment" (applies to the site list; same class as the old
   cv.nix drift incident).
2. **Pre-deploy gates should include one real E2E action**, not only static
   checks — doctor proved structure, not behavior.
3. **Doc gotchas die the same day their cause dies** — the AGENTS.md go
   gotcha aged badly within one session; the repo already has a doctrine
   note for exactly this failure (fix-on-sight owner grant 2026-09-06).
4. **Prove what you switch** — toolchain sweeps need the swept apps executed,
   not just the happy-path build.
5. **Monitoring must match topology** — socket-activated backends must not be
   health-probed (TTL defeat); conversely the daemon needs a staleness signal
   beyond unit state.

## f) Up to 50 things to get done next

*(1-8 = the activation tail; 9-20 = high-value follow-ups; 21+ = ROADMAP
fuel. Not commitments — harvest input.)*

1. Push both repos (user; auto-commits included).
2. `nix flake lock --update-input vision-review-agent` in SystemNix —
   MANDATORY (locked rev predates the go fix; its build fails).
3. `nix run .#deploy` (user).
4. Post-deploy: `systemctl status visionreviewd` + `sudo visionreviewd doctor
   -config /etc/visionreviewd/config.json` → expect 21/21.
5. Watch first pass (`journalctl -u visionreviewd -f`); confirm ~17 views
   reviewed (not 134 captures) and reviews land under
   `/var/lib/visionreviewd/reviews` with INDEX.md.
6. Record first-pass wall time + CPU burst + model-residency pattern as a
   perf/status note (docs/status, durable evidence).
7. Verify the alias: `GET :8128/v1/models` post-deploy shows
   `nsfwcaption-qwen3-vl-8b-v3`.
8. Confirm system-health picked up the monitored unit + a deliberate restart
   routes OnFailure notify (once).
9. Generate `visionreviewdSites` in SystemNix from
   `vision-review-agent-src`'s `visionreviewd-websites-17.json` via fromJSON
   — kill the 3-way split brain.
10. Run `nix run .#test` and `.#lint` (go_1_27 apps) once to close the
    unverified-toolchain-sweep gap.
11. Update this repo's AGENTS.md: devShell now ships go 1.27.1; retire the
    "use the nix-store go-1.27.1 binary" gotcha.
12. Re-add the SystemNix FEATURES.md row for visionreviewd.
13. Register the journal in backup-coordination (+ restic-app-dumps row) —
    the event journal is the source of truth and currently unprotected.
14. First backup + restore drill: `visionreviewd backup` → scratch dataDir →
    `replay` → byte-identical INDEX (the documented ritual, executed once).
15. Staleness monitor: alarm when reviews INDEX mtime goes stale while the
    unit is "active" (the deadlock class `monitored` cannot see).
16. Decide: retire the manual monthly `review-fleet.sh once` pass (daemon
    becomes the only reviewer) vs soak in parallel — today both review the
    same shoots (double CPU inference).
17. Document in site-fleet-ops.md that shoots are now auto-reviewed (or the
    soak outcome of 16) — the "never overlap a shoot with a pass" doctrine
    needs its daemon-era wording.
18. doctor: treat 503-loading as a wait-and-retry condition (readiness
    semantics for socket-activated model servers) — upstream UX fix.
19. Single-view E2E pre-gate harness (one-project temp config + `once`) for
    future host activations.
20. SystemNix VM test `test-visionreviewd.nix` (llama-less: unit props,
    generated etc file, service starts with a stub endpoint).
21. Measure the post-unload churn window (503-fast-fail ticks until model
    resident) and decide whether acceptable or needs a daemon-side
    "wait-for-model" backoff.
22. Confirm scan/fold semantics: same-viewKey re-shoots collapse to one
    review per pass (read pipeline.go once, record in AGENTS.md).
23. KeepAlive vs daemon cadence: 10 m tick keeps the 10 GB model effectively
    always resident — decide if that's wanted (hourly tick vs accept).
24. Reviews-dir-as-git-checkout (readable by Crush; writer doesn't care).
25. Homepage tile for visionreviewd (link the reviews INDEX; needs 24 or an
    internal vHost first).
26. Internal read-only vHost over `/var/lib/visionreviewd/reviews` if remote
    review reading is wanted (Caddy layer decision — registry `vHost.layer`).
27. Discordsync full-watch TODO (next section in TODO_LIST): fold its 220-PNG
    glob into the daemon config once cadence is decided (user decision).
28. Upstream: `llamaServer.args` option (direct-path ergonomics without
    host-level mkForce) — for other hosts using the bundled unit.
29. Upstream: version var prep — the go fix + activation are v0.8.0 release
    content; cut when the user says.
30. Post-deploy smoke into `post-deploy-check.sh`: INDEX existence after the
    first tick.
31. SigNoz/dashboard panel from `visionreviewd events` output (pass duration,
    error counts) — optional monitoring enrichment.
32. Snapshot-hash model path: when the caption model is ever re-downloaded,
    the llama-vlm modelPath changes — add a comment cross-link in both repos
    (alias keeps the daemon config stable; only llama-vlm moves).
33. Decide whether `visionreviewd-llama` (8390) reserve should carry a
    "reserved" marker convention in lib/ports.nix (it has a comment; fine —
    revisit only if the audit grows teeth about unused ports).
34. Sweep other handoff instructions into ONE canonical place (the deploy
    steps live in 4 documents now — pick TODO_LIST as canonical, link the
    rest).
35. Consider `RandomizedDelaySec` on the daemon tick to avoid thundering
    alignment with other 10 m timers (minor).
36. Consider MemoryMax headroom check for the DAEMON under a 17-view pass
    (upstream caps at 1G; images are streamed — verify no OOM at fleet size).
37. After first pass: review quality spot-check (read 2-3 generated reviews;
    the persona prompts were refactored in dedup commits — lock rev
    `c8ca4b5` predates them; quality should be identical, confirm once).
38. Pin-policy note: input rides `?ref=master` — flake-update workflow will
    bump it weekly; confirm the weekly auto-update treats it fine (it will;
    note the go-fix coupling: bumps now REQUIRE upstream green builds —
    upstream CI runs the same gates, so trust-but-verify the first bump).
39. Delete my /tmp doctor artifacts (`/tmp/visionreviewd-doctor-config.json`
    etc.) after deploy — throwaway hygiene.
40. Once 13 lands: add the journal to the btrbk/receipt documentation so the
    backup story is discoverable from the repo (not only SystemNix).

## g) Questions I can NOT figure out myself

1. **Should the daemon become the ONLY reviewer?** After deploy, both the
   daemon (10 m tick, fresh /var/lib journal) and the manual monthly
   `review-fleet.sh once` pass (user-space journal) will review the same
   shoots — double CPU inference. Retire the manual pass now, or soak the
   daemon for a cycle (e.g. one month) before retiring? (Owner call on risk
   appetite; I can prepare either.)
2. **First-pass burst acceptable?** On first start the daemon reviews all 17
   sites immediately — CPU inference for the full fleet with the caption
   model resident for the duration. OK to let it rip, or should I flip the
   generated config to a small pilot subset (e.g. 2 sites) for the first
   deploy and widen after you've watched one pass?
3. **Journal backup policy?** The event journal is the source of truth and
   currently has no backup registration. I propose: nightly copy via the
   restic-app-dumps pattern + backup-coordination row (freshness watched),
   retention 14d daily. Confirm retention/cadence — or do you want the
   journal on the btrbk snapshot tier instead?
