#!/usr/bin/env bash
# family-a11y-guard.sh — check-mode guard for the 14-repo Astro template family.
#
# Flags the three violation classes found by the 2026-09-19 axe audit
# (see docs/activation/template-family-divergence.md and ADR 0001):
#   a) empty <th ...></th> / <th ... /> in table components (empty-table-header)
#   b) scrollable pre/code/.overflow-* elements without tabindex="0"
#      (scrollable-region-focusable)
#   c) solid accent CTAs without the --color-on-accent token scheme
#      FAIL: `text-on-accent` class used but token undefined (renders wrong)
#      INFO: bg-accent CTA in a repo without the token (sync priority only,
#            NOT a verified violation — the pair may still pass contrast)
#
# Check-mode only: reports, never edits. Exit 1 on FAIL findings (INFO is
# reported but does not fail the run).
#
# Usage:
#   scripts/family-a11y-guard.sh              # all repos, human output
#   scripts/family-a11y-guard.sh gogenfilter  # subset
set -u

REPOS=(
	art-dupl clean-wizard cmdguard dynamic-markdown-site go-atomic-write
	go-branded-id go-error-family go-filewatcher go-output
	go-workflow-auditlog gogenfilter md-go-validator samber-do-auditlog
	typespec-asyncapi
)

filter=("$@")
total=0
info=0
for repo in "${REPOS[@]}"; do
	if [ "${#filter[@]}" -gt 0 ]; then
		keep=0
		for f in "${filter[@]}"; do [ "$f" = "$repo" ] && keep=1; done
		[ "$keep" = 0 ] && continue
	fi
	src="$HOME/projects/$repo/website/src"
	if [ ! -d "$src" ]; then
		echo "?? $repo: no website/src (skipped)"
		continue
	fi
	has_token=0
	grep -q -- '--color-on-accent' "$src/styles/global.css" 2>/dev/null && has_token=1
	repo_findings=0
	while IFS= read -r hit; do
		[ -z "$hit" ] && continue
		echo "FAIL $repo: $hit"
		repo_findings=$((repo_findings + 1))
	done <<EOF
$(grep -rnE '<th[^>]*>\s*</th>|<th[^>]*/>' "$src" --include='*.astro' |
		sed "s|^$src/||; s|^|a) empty th: |")
$(grep -rnE '<(pre|code)[^>]*overflow-[xy]-auto' "$src" --include='*.astro' |
			grep -v 'tabindex="0"' |
			sed "s|^$src/||; s|^|b) scrollable w/o tabindex: |")
$(if [ "$has_token" = 0 ]; then
			grep -rln 'text-on-accent' "$src" --include='*.astro' |
				sed "s|^$src/||; s|^|c) text-on-accent class, token undefined: |"
		fi)
EOF
	while IFS= read -r hit; do
		[ -z "$hit" ] && continue
		echo "INFO $repo: $hit"
		info=$((info + 1))
	done <<EOF
$(if [ "$has_token" = 0 ]; then
		grep -rlnE 'bg-accent[^/]' "$src" --include='*.astro' |
			sed "s|^$src/||; s|^|c-info) bg-accent CTA, no token scheme: |"
	fi)
EOF
	if [ "$repo_findings" = 0 ]; then
		echo "ok   $repo"
	else
		total=$((total + repo_findings))
	fi
done
echo "---"
echo "family-a11y-guard: $total FAIL finding(s), $info info"
[ "$total" = 0 ]
