#!/usr/bin/env bash
# Repairs vendorHash.nix after a go.mod/go.sum change. Nix reports the
# expected output hash in the build failure ("got: sha256-..."); this script
# harvests it and rewrites vendorHash.nix, then rebuilds to prove the fix.
# Run from the repo root (the flake app `.#update-vendor-hash` does).
# Usage: nix run .#update-vendor-hash [attr]   (default attr: visionreviewd)
set -euo pipefail

attr="${1:-visionreviewd}"
hash_file="vendorHash.nix"

fail() {
	printf 'update-vendor-hash: %s\n' "$1" >&2
	exit 1
}

[ -f "$hash_file" ] || fail "$hash_file not found (run from the repo root)"

printf 'update-vendor-hash: probing nix build .#%s\n' "$attr"
build_log="$(nix build ".#$attr" --no-link 2>&1 || true)"

if ! printf '%s' "$build_log" | grep -q 'hash mismatch'; then
	# No mismatch: either the build succeeded or it failed for another
	# reason; surface the tail either way.
	printf '%s\n' "$build_log" | tail -5
	if printf '%s' "$build_log" | grep -q '^error'; then
		fail "build failed without a hash mismatch; see log above"
	fi
	printf 'update-vendor-hash: %s already matches; nothing to do\n' "$hash_file"
	exit 0
fi

got="$(printf '%s' "$build_log" | sed -n 's/.*got:[[:space:]]*\(sha256-[A-Za-z0-9+/=]*\).*/\1/p' | tail -1)"

[ -n "$got" ] || fail "could not find a 'got: sha256-...' hash in the build log"

printf '"%s"\n' "$got" >"$hash_file"

printf 'update-vendor-hash: %s -> %s\n' "$hash_file" "$got"
printf 'update-vendor-hash: rebuilding to verify\n'

nix build ".#$attr" --no-link || fail "rebuild with the new hash failed"

printf 'update-vendor-hash: OK\n'
