# BuildFlow — What Breaks Builds Here, and What To Do

The repo's build is healthy; the flakiness documented across past sessions came
from two sources OUTSIDE the code: system OOM kills and an external daemon
rewriting imports. This file exists so future sessions stop re-triaging them.

## 1. OOM kills (the "random build failure" class)

**Evidence (2026-08-18 16:27:58, `journalctl -k`):** a global OOM
(`constraint=CONSTRAINT_NONE`, `global_oom`) killed 7 user-session daemons
(aw-watcher, aw-server, dms, xdg-desktop-portal ×2, wireplumber,
gnome-keyring). The dominant consumer in the kernel's process dump was
**llama-server** holding the Q8_0 8B model: ~15 GB total_vm, ~1 GB RSS plus
~4.6 GB in swap. Victims were chosen by oom_score, not by size — small daemons
died while the actual hog survived.

**Symptoms you will see:** `go test -race` / `nix build` / `golangci-lint`
dying without a compile error, shells dropping, or (nix-specific) a
buildGoModule "succeeding" with an EMPTY output. Any of these under memory
pressure is an OOM kill until proven otherwise.

**Policy:**

1. Diagnose before retrying: `journalctl -k --since "<time window>" | grep -i
   oom`. If OOM lines exist, the failure was environmental — do not touch repo
   code.
2. Free memory before the heavy commands: stop llama-server (`pkill
   llama-server`) during `go test -race ./...` or nix evaluations, or accept
   flakiness.
3. Reduce parallelism instead of retrying blindly: `go test -p 1 -race ./...`,
   or build one package at a time.
4. The verification matrix in `AGENTS.md` assumes a quiet machine; run it as
   such before suspecting the repo.

## 2. json/v2 import rewrites (the "compilation suddenly broke" class)

An external go-auto-upgrade daemon repeatedly rewrote `encoding/json` imports
to `encoding/json/v2` / `encoding/json/jsontext` across sessions (4 documented
incidents). Those paths **cannot compile in this repo**: they would require a
`go.mod` replace directive and have a different low-level API. The repo's
policy is dual-regime support — import only `encoding/json`; the jsonv2
`GOEXPERIMENT` swaps the implementation underneath while preserving the v1 API
(see AGENTS.md "Dual json v1+v2 support").

**The guard:** the first step of the `build-and-test` CI job fails on any
`.go` file importing `encoding/json/v2` or `encoding/json/jsontext`:

```bash
git grep -nE --untracked '"encoding/json/(v2|jsontext)(/[a-z0-9]+)?"'
```

`--untracked` is load-bearing: fresh daemon-edited files are exactly the
threat model, and plain `git grep` misses them. Probe-verified in both
directions (offending import caught; comments/strings mentioning json/v2 not
caught). When it fires: revert the import to `encoding/json`, keep everything
else. The root fix (configuring the daemon) is a user action, tracked in
TODO_LIST.

## 3. Lint-noise policy (verdicts, 2026-08-18)

Ambient warning noise from external tools has an explicit verdict per source —
"fix, configure, or ignore-by-design", never silent. Configs live in the repo
root (`.markdownlint.json`, `.markdownlint-cli2.yaml`, `.codespellrc`).

| Source                                      | Finding                                                                                                          | Verdict                                                                                                                                                    | Where                                       |
| ------------------------------------------- | ---------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------- |
| markdownlint MD013                          | Line-length hits across living + historical docs (tables, URLs cannot wrap; treefmt/dprint owns markdown layout) | ignore-by-design                                                                                                                                           | `.markdownlint.json` (`MD013: false`)       |
| markdownlint MD024                          | Duplicate `### Matching` / `### Changed` headings across sibling sections (semantic pattern)                     | configure                                                                                                                                                  | `.markdownlint.json` (`siblings_only`)      |
| markdownlint MD010                          | Hard tabs inside Go code fences (Go style IS tabs)                                                               | configure                                                                                                                                                  | `.markdownlint.json` (`code_blocks: false`) |
| markdownlint (historical docs)              | Frozen findings in `docs/status/**`, `docs/planning/**`                                                          | ignore-by-design (point-in-time docs are never rewritten)                                                                                                  | `.markdownlint-cli2.yaml` `ignores`         |
| codespell                                   | `onText` (callback param name), `unparseable` (accepted variant); findings in generated `go.sum` + frozen docs   | configure / ignore-by-design                                                                                                                               | `.codespellrc`                              |
| go-structure-linter `assets-directory`      | Suggests an `assets/` dir for static files                                                                       | ignore-by-design — Go convention is `testdata/` (excluded from builds, shipped in the module); moving the pinned schemas would break `go test` data lookup | not configured; reasoning recorded here     |
| go-structure-linter `coverage-out-location` | `coverage.out` in repo root                                                                                      | fix at source — transient artifact (gitignored); `trash` it when flagged                                                                                   | `.gitignore` already covers                 |
| go-licenses missing from devShell           | Tool absent for license reports                                                                                  | fixed                                                                                                                                                      | added to `devShell.default` (flake.nix)     |

Living docs are markdownlint-clean (16 files, 0 issues) and codespell-clean as
of 2026-08-18; `nix run nixpkgs#markdownlint-cli2` and `nix run
nixpkgs#codespell` reproduce both checks.

## 4. Quick reference

- Canonical verification matrix: `AGENTS.md` → "Verification matrix".
- Pre-commit hook (BuildFlow) runs dprint + markdown link checks; broken links
  block commits — fix the link, do not bypass the hook.
- `nix build` hash mismatch after a dep bump: update `vendorHash.nix`
  (procedure in that file) — a mismatch is expected, not a flake failure.
- Nix files must be git-tracked to be visible to the flake (`git add` before
  `nix build` evaluates new files).
