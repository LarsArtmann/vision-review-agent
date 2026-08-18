# Hash of the vendored Go dependency set for buildGoModule.
#
# Update procedure after any dependency change (`go get`, `go mod tidy`):
#   1. Run `nix build . 2>&1 | grep got:` and copy the `sha256-...` value.
#   2. Replace the hash below with it.
#   3. Re-run `nix build .` and `nix build .#visionreviewd` — both must succeed.
#
# Kept in its own file so dependency bumps touch exactly one line and the
# diff of a bump is unambiguous in review.
"sha256-kNLZZUTvkz8skF94i6UOgN9NK5ujfChHJCgfWlCqr6Y="
