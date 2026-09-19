#!/usr/bin/env bash
# rebuild-redeploy.sh — build every changed family site, deploy green ones.
# Guardrails: build exit code checked strictly (no pipes); deploy only after
# a green build; deploys use the ADC credential path.
#
# Proven 2026-09-19: 14/16 family sites first try during the a11y+og sweep.
# NOT covered here (special build flows, run manually):
#   templcomponents (templ generate + go test goldens), emeet-pixyd
#   (nix build + firebase), learnings (bun + docusaurus binary).
# pnpm v11 sites need `allowBuilds: esbuild: true` in their
# website/pnpm-workspace.yaml or installs fail on the esbuild postinstall.
set -u

export GOOGLE_APPLICATION_CREDENTIALS="${GOOGLE_APPLICATION_CREDENTIALS:-$HOME/.config/gcloud/application_default_credentials.json}"
FB="nix shell nixpkgs#firebase-tools -c firebase"
HOME_DIR="$HOME/projects"

# repo|site|buildcmd
SITES=(
  "art-dupl|art-dupl|pnpm run build"
  "clean-wizard|cleanwizard|pnpm run build"
  "cmdguard|cmdguard|pnpm run build"
  "dynamic-markdown-site|dynamicmarkdown|bun run build"
  "go-atomic-write|atomicwrite|pnpm run build"
  "go-branded-id|brandedid|pnpm run build"
  "go-error-family|errorfamily|bun run build"
  "go-filewatcher|filewatcher|pnpm run build"
  "go-output|go-output|pnpm run build"
  "go-workflow-auditlog|auditlog|pnpm run build"
  "gogenfilter|gogenfilter|bun run build"
  "md-go-validator|md-go-validator|pnpm run build"
  "samber-do-auditlog|do-auditlog|pnpm run build"
  "typespec-asyncapi|typespec-asyncapi|npm run build"
)

declare -a built deployed failed
for s in "${SITES[@]}"; do
  repo="${s%%|*}"; rest="${s#*|}"; site="${rest%%|*}"; cmd="${rest#*|}"
  d="$HOME_DIR/$repo/website"
  if [ ! -d "$d/node_modules" ]; then
    echo "### $repo: installing deps"
    (cd "$d" && pnpm install --silent >/dev/null 2>&1) || { echo "### $repo: INSTALL FAILED"; failed+=("$repo"); continue; }
  fi
  echo "### $repo: build"
  if (cd "$d" && eval "$cmd" >"/tmp/build-$repo.log" 2>&1); then
    built+=("$repo")
    echo "### $repo: build OK — deploying to $site"
    if (cd "$d" && $FB deploy --only "hosting:$site" --project lars-software >"/tmp/deploy-$repo.log" 2>&1); then
      deployed+=("$site")
      echo "### $repo: DEPLOYED"
    else
      failed+=("$repo(deploy)")
      echo "### $repo: DEPLOY FAILED"; tail -3 "/tmp/deploy-$repo.log"
    fi
  else
    failed+=("$repo")
    echo "### $repo: BUILD FAILED"; tail -5 "/tmp/build-$repo.log"
  fi
done

echo
echo "built: ${#built[@]}  deployed: ${#deployed[@]}  failed: ${#failed[@]}"
[ "${#failed[@]}" -eq 0 ]
