# Status Report — art-dupl `-t 1` Dedup Pass (Session 2026-10-05)

- **Date:** 2026-10-05 16:31 CEST
- **Scope:** Full-repo clone scan (`art-dupl --sort total-tokens -t 1
  --suggest-generics --type-aware`), judgment of all 8 shown clone groups,
  one extraction, marker encoding, docs re-baseline.
- **Commits (auto-commit daemon, chronological):** `4b101ec` → `fb2e138` →
  `4f43585` (7 files) → `70f67ec` (5) → `abcc53d` (2) → `610145f` (4).
  Working tree clean at report time.
- **Format note:** user explicitly named a `.md` path; skill default is
  styled HTML. Honored the user override — flagged here per skill contract.

## Session in One Paragraph

The user ran art-dupl at threshold 1 with `--suggest-generics` and got 8
clone groups (the documented baseline was 4 accepted groups from
2026-09-22, never re-swept under v0.7 at `-t 1`). All 8 groups were read,
judged, and dispositioned: 1 extracted (`countVisionModels`), 1 accepted
with in-place `// art-dupl:accept` markers (the three per-package
`ModelBrand`/`ModelID` definitions from the 2026-10-05 strong-id pass),
6 accepted as documented idiom groups. Result: canonical `-t 1` scan now
shows 4 groups (all accepted), `--suggest-generics` shows 6 (all accepted).
Bonus discovery: v0.7 empirically honors the marker (marked group vanishes
from output), closing an open question from the 2026-09-22 re-baseline and
resolving a `TODO_LIST.md` item.

---

## a) FULLY DONE

| # | Item | Evidence |
|---|------|----------|
| 1 | **All 8 clone groups judged** (extract / accept / accept-marked) — none left unexamined | Final scan: canonical `-t 1` = 4 shown (all documented-accepted); `--suggest-generics` = 6 shown (all documented-accepted) |
| 2 | **`countVisionModels` extraction** — the 2-site `SupportsImages` counting loop in `cmd/vision/listing.go` collapsed into one named helper (both call sites inlined; the duplicate loop is gone) | `cmd/vision/listing.go:19`; clone group absent from both scan variants |
| 3 | **Table-driven test for the new helper** — empty / mixed synthetic / real-catalog-provider cases | `cmd/vision/listing_test.go` (`TestCountVisionModels`) |
| 4 | **`// art-dupl:accept` markers on the three per-package brand definitions** (`internal/catalog/ids.go`, `pkg/vision/ids.go`, `internal/reviewd/reviewer.go`) with coupling rationale | Marked group no longer appears in `-t 1` output |
| 5 | **Marker support empirically verified** — v0.7 honors `// art-dupl:accept`; the 2026-09-22 open question ("markers unverified under v0.7") is closed | Before/after scan diff: 6 shown → 4 shown after marking |
| 6 | **`docs/DUPLICATION_POLICY.md` re-baselined** — new 2026-10-05 Current State paragraph, `countVisionModels` helper row, 2 new accepted-group rows (ModelBrand ×3, decompile scalar ×2), widened-window note on the messages.go row and the accumulate-append row | Policy doc diff |
| 7 | **`AGENTS.md` duplication section updated** — current state now cites the 2026-10-05 verification and the pass summary | `AGENTS.md` Code Duplication section |
| 8 | **`TODO_LIST.md` item resolved** — "art-dupl v0.7 `-t 1` sweep" checked off with resolution details | `TODO_LIST.md` |
| 9 | **Verification on touched code** — `go build ./...`, `go vet ./...` (exit 0), `gofmt` clean, `go test -race ./cmd/vision/` green, full `go test ./...` run (sole failure = the documented pre-existing known-red), pinned `golangci-lint` on all 4 touched packages (only the 2 documented pre-existing findings, in untouched files) | Session transcript; vet exit code captured cleanly on re-run |

## b) PARTIALLY DONE

| # | Item | What exists / what's missing |
|---|------|------------------------------|
| 1 | **Full verification matrix for this change** — build/vet/gofmt/tests/lint done; **not** run: `GOEXPERIMENT=jsonv2` + `GOEXPERIMENT=none` regime builds, `go mod verify` + `tidy -diff`, `govulncheck`, `nix run .#test`/`.#lint`, `nix build`, `nix flake check` | Defensible for comment-edits + one CLI helper, but the canonical matrix says run the full set after cross-cutting changes; the marker touches `pkg/vision` + `internal/reviewd` (both regimes matter there) |
| 2 | **`--suggest-generics` ladder** — only `-t 1` with the flag was mapped; `-t 2` / `-t 3` with the flag never run, so the flag's full effect on shown-count is undocumented (policy cites plain-flag numbers only) | One command each would complete the picture |
| 3 | **Accepted-group marker consistency** — policy says the two 2026-09-22 *judgment* groups carry markers while idiom rows deliberately stay visible; that convention is now applied to the new judgment group (ModelBrand) but never written down as a rule, and `buildObjectCall` (`pkg/vision/structured.go`) shares the params-copy idiom with **no marker and no why-not note** (2026-09-22 round-3 leftover #34) | A 3-line policy note + a one-line decision |
| 4 | **Self-critique fixes** (see section e): the real-catalog test case I added works but is convoluted — `countNonOpenAIVision` derives the expectation from `VisionModels()` instead of synthetic fixtures, and the name reads badly | Functional, not clean |

## c) NOT STARTED

| # | Item | Source |
|---|------|--------|
| 1 | **Fold product decision** — tolerate-and-ignore vs strict fold for unknown event types in `ApplyViewState`; the known-red test and the repo's only `err113` lint finding both hang on it | AGENTS.md KNOWN RED note (pre-existing) |
| 2 | **`no-jsonv2` CI blocker resolution** — red since 2026-09-29 for a transitive reason (fantasy ≥ v0.45.2 floors kaptinlin v0.9.9+ which imports `encoding/json/v2`); options: pin fantasy v0.41.1 / redefine job scope to the fantasy-free subset / wait for upstream | AGENTS.md jsonv2 block |
| 3 | **Direct table test for `defaultIDs`** (covered only transitively today) | `TODO_LIST.md` |
| 4 | **go-cqrs-lite v5 migration** tracking | `ROADMAP.md` via AGENTS.md |
| 5 | **A2UI v1.0 spec upgrade** (`VersionV091` → v1.0; `theme` → `surfaceProperties` rename) — blocked upstream until v1.0 leaves candidate status | AGENTS.md a2ui section |
| 6 | **go-cqrs-lite asks #22/#23** — both fixed upstream and absorbed by the 2026-09-08 re-bump; TODO item stands as a deliberate no-op unless the user wants different handling | `TODO_LIST.md` |
| 7 | **docs-health HARVEST of this report's section (f)** into `TODO_LIST.md`/`ROADMAP.md` — user instructed WAIT, so not run this session | This report, section f |

## d) TOTALLY FUCKED UP

Nothing shipped this session is broken. Honest incident list:

| # | Item | Severity | State |
|---|------|----------|-------|
| 1 | **Self-inflicted `/tmp` exhaustion mid-session** — each command created a fresh `mktemp -d` GOCACHE (AGENTS.md documents the `/mnt/buildcache` gotcha but I followed it wastefully); `/tmp` hit 99%, one `go vet` run died with `no space left on device`, costing two diagnostic round trips. Fixed by cleaning my temp dirs and using ONE shared cache (`/tmp/go-cache-dedup`) for the rest of the session | Medium (session-local, no repo damage) | Resolved in-session; root cause was my own command pattern |
| 2 | **Two broken test-compile round trips** — the helper's doc comment lost a `//` prefix (caught by vet as `expected declaration, found table`), and I wrote `catalog.Provider` before checking that `Providers()` returns `[]catwalk.Provider` (caught by vet as `undefined`) | Low | Both fixed immediately; caused by editing before fully reading the target file's shape |
| 3 | **Pre-existing master reds observed, untouched by design** — `TestApplyViewStateIgnoresUnknownEventType` fails on master (fold default errors; test expects ignore — product decision pending), and the `no-jsonv2` CI job is red for the transitive fantasy/kaptinlin reason. Both documented; **confirming whether the known-red currently reds the required `build-and-test` check on master (admin-bypassed) was NOT done** per the no-unrelated-research instruction | High (pre-existing, repo-level) | Documented, decision-pending |

## e) WHAT WE SHOULD IMPROVE

**What I forgot / could have done better this session:**

1. **Read before writing (again).** Both compile errors came from writing the
   test before reading `catalog.go`'s actual types (`Providers()` →
   `[]catwalk.Provider`, `FindProvider` takes `ProviderID`). The vet loop
   caught it, but the rule is read-first, and I skipped it for the test file.
2. **One cache, from the start.** The `/tmp` exhaustion was preventable: a
   single shared `GOCACHE` per session should be the reflex, not four
   throwaway `mktemp` dirs. Worth adding the concrete command to the
   AGENTS.md build-cache gotcha.
3. **Baseline should have matched the user's exact flags.** I baselined with
   the skill's canonical command (6 groups) and only later noticed the
   user's `--suggest-generics` run showed 8; comparing both up front would
   have split the "new vs flag-widened" groups a pass earlier.
4. **Exit-code hygiene in pipelines.** `… | grep -v downloads; echo $?`
   reported grep's exit, twice. Use `> file; echo $?` directly — I
   eventually did, but only after burning a round trip.
5. **Marker placement verified only empirically for one shape.** The marker
   works above the doc comment of a top-level type; I verified by re-scan
   (good) but didn't document the placement convention, so the next person
   may place it differently and silently lose suppression.
6. **The new test's "real catalog provider" case leans on a helper with a
   poor name** (`countNonOpenAIVision`) and an indirect expectation
   derivation — a synthetic `[]catwalk.Model` fixture would be clearer and
   would not re-derive counting logic inside a test of a counting function.
7. **Did not write down the idiom-vs-judgment marker rule** while it was
   fresh — policy rows document *what* was accepted but the marker
   convention itself lives only in session history and 3 archived status
   docs.

## f) UP TO 50 THINGS WE SHOULD GET DONE NEXT

*Brainstorm, sorted roughly by impact within theme. This section is HARVEST
input — most items below the first ~10 are ROADMAP fuel, not commitments.*

**Correctness / unblock CI:**

| # | Task | Why |
|---|------|-----|
| 1 | Make the fold product decision (ignore vs strict) and land it — fixes the known-red test AND the `err113` lint finding at `store.go:97` together | Two documented reds, one root decision |
| 2 | Resolve the `no-jsonv2` CI job: pin fantasy v0.41.1, scope the job to the fantasy-free subset, or track upstream kaptinlin | Job red on master since 2026-09-29 for transitive reasons |
| 3 | Confirm current master CI state: is `build-and-test` red via the known-red (admin-bypassed), and are dependabot PRs affected? | Red required checks on master contradict the "full-green" narrative |
| 4 | If the fold decision needs time, quarantine the known-red test with a `t.Skip` + tracking link so master CI goes green without hiding the decision | Reversibility + visibility |
| 5 | Verify `printProviderInfo`'s `API Key: %s` output only ever prints catwalk env-var metadata, never a real secret | I noticed the print this session and never verified the field's semantics |

**Finish this pass properly:**

| # | Task | Why |
|---|------|-----|
| 6 | Run the remaining verification-matrix steps for this change (jsonv2 both regimes, `mod verify`/`tidy -diff`, nix steps) | Matrix discipline after touching `pkg/vision` + `internal/reviewd` |
| 7 | Map the `--suggest-generics` ladder (`-t 2`, `-t 3`) and record the flag's effect in the policy doc | The user's command and the canonical command report different counts; document both |
| 8 | Write the marker convention into the policy doc: judgment groups get in-place markers (with placement note: above the doc comment works), idiom rows stay visible on purpose | Prevents silent suppression-loss by the next editor |
| 9 | Decide `buildObjectCall` marker consistency (mark it, or document why not) — leftover #34 from the 2026-09-22 round-3 | Closes the last open dedup leftover |
| 10 | Simplify `TestCountVisionModels`' real-catalog case to synthetic fixtures; rename `countNonOpenAIVision` | My own technical debt from this session |
| 11 | Add a machine check that re-runs the dup ladder and asserts markers exist at claimed sites (2026-09-22 leftover) | Policy doc is the source of truth; make it verifiable |
| 12 | Record the one-shared-GOCACHE-per-session command in AGENTS.md's build-cache gotcha | This session's `/tmp` incident is the second cache-related gotcha entry |
| 13 | Consider upstreaming `// art-dupl:accept` marker documentation to art-dupl's README (2026-09-22 leftover #40) | Marker semantics are load-bearing but undocumented upstream |
| 14 | Note in the policy that the installed art-dupl reports `version dev` — cite how the build is produced so "v0.7.x" claims stay checkable | Policy precision |

**Docs health:**

| # | Task | Why |
|---|------|-----|
| 15 | HARVEST this report's (f) into `TODO_LIST.md` / `ROADMAP.md` (needs user go-ahead — instructed to WAIT) | Section f must not be entombed |
| 16 | Verify `docs/DOMAIN_LANGUAGE.md` documents the branded-ID vocabulary (brand rendering `Model:<id>`) from the strong-id pass | Shared-vocabulary doc should own the new terms |
| 17 | Check FEATURES.md / README for `ModelInfo.ID` breaking-change fallout (pre-1.0 field-type change; literal readers must use `.ID.Get()`) | Consumer-facing surface |
| 18 | CHANGELOG: add the strong-id + dedup passes if refactors are tracked there | Change history completeness |
| 19 | Run a docs-health VERIFY pass over the policy doc's new claims (counts, dates, flag behavior) | Fresh claims need the same verification standard as old ones |
| 20 | Schedule this report for `archived/` once its items resolve (git mv, annotate first) | Snapshot hygiene |

**Pre-existing backlog seen this session:**

| # | Task | Why |
|---|------|-----|
| 21 | Direct table test for `defaultIDs` (both empty / one empty / neither) | `TODO_LIST.md`, 10-line test |
| 22 | go-cqrs-lite v5 migration tracking (ROADMAP item; pair-form guard test already in place) | Upstream v5 prep is on their master |
| 23 | A2UI v1.0 upgrade prep — inventory the `theme` → `surfaceProperties` rename touchpoints now so the upgrade is mechanical later | Blocked upstream, prep isn't |
| 24 | Fix or `nolint` the `exhaustive` finding in `cmd/vision/main.go:438` (ErrorKind switch missing 3 kinds) | Pre-existing diagnostic noise on every session |
| 25 | Audit that no raw strings leak into `reviewer.model` positions (config→`NewModelID` boundary claim from the strong-id pass) | One grep + review; validates the pass's own claim |
| 26 | Run `nix run .#dep-drift` (not run this session) | Advisory; cheap |
| 27 | Run `govulncheck ./...` locally (CI covers it; local parity after the fantasy/kaptinlin pin question) | Security posture |
| 28 | Decide whether auto-commit heuristic messages ("N changed file(s)") should carry structured subjects for scan-ability — this session's work landed as 6 opaque commits | History readability; the jsonv2 recurrences rode exactly such commits |
| 29 | Consider a pre-commit/auto-commit hook that runs the depguard grep for `encoding/json/v2` imports | The documented recurrence guard currently relies on humans |
| 30 | Re-check the llama-server / fleet-ops docs pointers used by AGENTS.md still resolve (links + file existence) | Docs-health quick win |

**Smaller polish / ideas:**

| # | Task | Why |
|---|------|-----|
| 31 | Add a second descent function trigger note: if `decompile.go` grows a third scalar-default, extract the named error constructor | The accept verdict has an explicit future trigger |
| 32 | Evaluate whether `slices.Sorted(maps.Keys(m))` uses cover all four former helper sites (2026-09-22 claim) | One grep; validates an old extraction |
| 33 | Pin the canonical art-dupl invocation (flags + purpose) at the top of the policy doc so future passes compare like-for-like | Flag variance caused this pass's baseline confusion |
| 34 | Consider `art-dupl` HTML output default (the user's console run dumped a full HTML document) — add `--rich-text` to the documented interactive command | Usability |
| 35 | Sweep for other `visionCount`-style local counters that hide domain concepts (naming-review lens on listing code) | Same class as this session's extraction |
| 36 | Verify the embedded-catalog tests still pass offline (`catalog.New()` in tests uses embedded data) | Test-environment assumption check |
| 37 | Confirm `findProvider`/`FindProvider` ProviderID alias boundary is covered by a test with a raw user string input | Strong-ID boundary claim |
| 38 | Check whether `printProviders`' sort comparator (`strings.ToLower`) matches `printVisionModels`' ordering logic or diverges | Two tables, one mental model |
| 39 | Decide the fate of `docs/status/archived/2026-09-22` round-3 leftover list (#34/#40 done via this report's items 9/13) — annotate those items done | Archived snapshots deserve the same ANNOTATE treatment |
| 40 | Add `countVisionModels` to any CLI-facing docs if the listing flags are documented anywhere user-facing | Feature-inventory completeness |
| 41 | Review whether `--suggest-generics`-flag-only groups (messages.go SurfaceID window) should be pinned by a regression test (assert the 4-site shape stays) | Cheap tripwire for wire-format drift |
| 42 | Verify journal golden fixture still folds clean after the marker comment edits (trivial, but the golden tests are the journal's contract) | Paranoia-cheap |
| 43 | Look at `internal/reviewd` package name (`reviewed`) vs directory (`reviewd`) split-brain risk for newcomers — one line in DOMAIN_LANGUAGE.md if intentional | Naming review candidate noticed while reading reviewer.go |
| 44 | Timebox a `library-deep-dive` on go-branded-id v0.7.0 (now a direct dep) — are we using its full surface? | New dependency from the strong-id pass |
| 45 | Check if `.golangci.yaml` still carries the depguard deny block after any config rework (documented recurrence risk) | Guard-erosion tripwire; next config touch must re-grep |
| 46 | Evaluate adding the dup-ladder counts to CI as an advisory job (fail-on-new-groups vs the machine check in item 11) | Automated baseline enforcement |
| 47 | Confirm `GOEXPERIMENT` env assumptions in local shells (exported var vs `go env -w`) once more before the next release cut | Release gotcha |
| 48 | Pre-release checklist: both `version` vars flip together (two mains, both flake ldflags entries) — write the flip into a release-runbook line | Documented GOTCHA, easy to miss |
| 49 | Sweep `docs/activation/` fleet-ops claims against current reality only if fleet work resumes (do not research now) | Deferred |
| 50 | After the fold decision lands: re-run the full 9-step matrix and re-baseline AGENTS.md's "Latest full-green run" stamp | Restores the green narrative |

## g) QUESTIONS I CANNOT FIGURE OUT MYSELF

1. **Fold semantics (product decision):** Should `ApplyViewState`'s fold
   **tolerate-and-ignore** unknown event types (forward compatibility for
   journal schema evolution — what the test asserts) or **fail strict**
   (what the code does)? This single decision unblocks the known-red test,
   the `store.go:97` `err113` lint finding, and possibly a CI red. The
   journal is golden-pinned, so I won't pick a semantics unilaterally.
2. **`no-jsonv2` strategy:** Do you want fantasy pinned back to v0.41.1
   (restores the v1-JSON regime but fights the auto-upgrade daemon), the
   CI job's scope redefined to the fantasy-free SDK subset (honest but
   weaker guarantee), or hold for an upstream kaptinlin dual-regime
   release (time unknown)? Each option trades a different risk; it's a
   dependency-strategy call I shouldn't make alone.
3. **Auto-commit daemon:** Do you want the heuristic "auto-commit N
   changed file(s)" messages upgraded (e.g. file-name summaries or a
   diff-derived subject), or is the current noise acceptable given the
   required-checks gate? This session's entire pass landed as 6
   unreadable commits — and the documented jsonv2 recurrences rode
   exactly such commits — but the daemon is your infrastructure and the
   fix belongs in its config, not this repo.

---

*Point-in-time snapshot — generated 2026-10-05 16:31 CEST. When a later task
brings this current: docs-health ANNOTATE, never rewrite. Section (f) is
HARVEST input pending user instruction.*
