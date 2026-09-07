#!/usr/bin/env bash
# Reports dependency drift: every direct requirement in go.mod compared
# against the latest stable version the module proxy knows about.
# Exits 1 when drift exists so it can gate automation. Run from the repo
# root (the flake app `.#dep-drift` and CI both do).
set -euo pipefail

drift=0

while read -r module current; do
	latest=$(go list -m -versions "$module" 2>/dev/null | tr ' ' '\n' | grep -v -- '-' | tail -1 || true)

	if [ -z "$latest" ]; then
		printf 'UNKNOWN %-55s (no stable tag published)\n' "$module"
		continue
	fi

	if [ "$current" != "$latest" ]; then
		printf 'DRIFT   %-55s %s -> %s\n' "$module" "$current" "$latest"
		drift=1
	else
		printf 'OK      %-55s %s\n' "$module" "$current"
	fi
done < <(go list -m -f '{{if not .Indirect}}{{.Path}} {{.Version}}{{end}}' all | grep -v '^github.com/larsartmann/vision-review-agent' | sort)

exit "$drift"
