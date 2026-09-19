#!/usr/bin/env bash
# shoot.sh v2 — capture all 17 site homes (desktop + mobile), with verify gates.
#
# Fixes vs v1 (pass-1 lessons):
#   * lazy images are force-loaded (--blink-settings=lazyImageLoadingEnabled=false)
#     — without it, image-heavy sections screenshot as blank cards
#   * per-site URL map uses WORKING urls (web.app fallbacks for the two
#     console/DNS-blocked custom domains, Firebase for learnings)
#   * blank-capture detection (ffmpeg luma variance) + error-page detection
#     (dump-dom grep) run BEFORE the shot is accepted; failures are reported
#     and exit non-zero so reviews never run on broken captures
#
# Usage: scripts/shoot-sites.sh [project ...]
set -u

OUT="${VISION_SHOOT_OUT:-$HOME/.local/share/vision-review-agent/screenshots}"
CHROME="${VISION_CHROME:-/nix/store/lgg6q8vg34lpnm69xxfn8nsj9pvv6jss-chromium-153.0.8010.47/bin/chromium}"
FFMPEG=ffmpeg

# project|url
SITES=(
  "art-dupl|https://art-dupl.lars.software"
  "cleanwizard|https://cleanwizard.lars.software"
  "cmdguard|https://cmdguard.web.app"
  "dynamicmarkdown|https://dynamicmarkdown.lars.software"
  "emeet-pixyd|https://emeet-pixyd.lars.software"
  "atomicwrite|https://atomicwrite.lars.software"
  "branded-id|https://branded-id.lars.software"
  "errorfamily|https://errorfamily.lars.software"
  "filewatcher|https://filewatcher.lars.software"
  "go-output|https://go-output.lars.software"
  "go-workflow-auditlog|https://go-workflow-auditlog.lars.software"
  "gogenfilter|https://gogenfilter.lars.software"
  "md-go-validator|https://md-go-validator.lars.software"
  "do-auditlog|https://do-auditlog.lars.software"
  "typespec-asyncapi|https://typespec-asyncapi.web.app"
  "templcomponents|https://templcomponents.lars.software"
  "learnings|https://lars-learnings.web.app"
)

VARIANCE_MIN=8        # min (YMAX - YMIN) luma spread; flat pages below this are "blank"
MIN_BYTES=8000        # PNG smaller than this is suspicious

url_for() {
  local p="$1" s
  for s in "${SITES[@]}"; do
    [ "${s%%|*}" = "$p" ] && { echo "${s#*|}"; return 0; }
  done
  return 1
}

luma_spread() {
  # print (YMAX - YMIN) for the image; -1 if ffmpeg fails
  local img="$1" out
  out=$($FFMPEG -hide_banner -i "$img" -vf "signalstats,metadata=print:key=lavfi.signalstats.YMAX:file=-" -f null - 2>/dev/null; \
        $FFMPEG -hide_banner -i "$img" -vf "signalstats,metadata=print:key=lavfi.signalstats.YMIN:file=-" -f null - 2>/dev/null)
  local ymax ymin
  ymax=$(printf '%s' "$out" | grep -o 'YMAX=[0-9.]*' | head -1 | cut -d= -f2)
  ymin=$(printf '%s' "$out" | grep -o 'YMIN=[0-9.]*' | head -1 | cut -d= -f2)
  [ -z "$ymax" ] || [ -z "$ymin" ] && { echo -1; return; }
  echo "${ymax%%.*} ${ymin%%.*}" | awk '{printf "%d", $1-$2}'
}

dom_has_error() {
  # capture the DOM and look for common error-page markers
  local url="$1" dom
  dom=$("$CHROME" --headless --no-sandbox --disable-gpu \
        --virtual-time-budget=8000 --dump-dom "$url" 2>/dev/null)
  printf '%s' "$dom" | grep -qiE 'ERR_(NAME_NOT_RESOLVED|INTERNET_DISCONNECTED|CONNECTION|SSL|CERT)|Site not found|This site can.t be reached|404 Not Found' \
    && return 0
  return 1
}

verify_or_die() {
  local proj="$1" view="$2" img="$OUT/$proj/$view.png" size spread
  if [ ! -s "$img" ]; then
    echo "  VERIFY-FAIL $proj/$view: missing/empty"; return 1
  fi
  size=$(stat -c%s "$img")
  if [ "$size" -lt "$MIN_BYTES" ]; then
    echo "  VERIFY-FAIL $proj/$view: only ${size}B"; return 1
  fi
  spread=$(luma_spread "$img")
  if [ "$spread" -lt 0 ]; then
    echo "  VERIFY-WARN $proj/$view: luma analysis failed (kept)"
  elif [ "$spread" -lt "$VARIANCE_MIN" ]; then
    echo "  VERIFY-FAIL $proj/$view: blank (luma spread $spread < $VARIANCE_MIN)"; return 1
  else
    echo "  verify ok $proj/$view (${size}B, spread $spread)"
  fi
  return 0
}

filter=("$@")
fail=0; shot_n=0
for s in "${SITES[@]}"; do
  proj="${s%%|*}"
  if [ "${#filter[@]}" -gt 0 ]; then
    keep=0
    for f in "${filter[@]}"; do [ "$f" = "$proj" ] && keep=1; done
    [ "$keep" = 0 ] && continue
  fi
  url="${s#*|}"
  echo "=== $proj ($url)"
  if dom_has_error "$url"; then
    echo "  VERIFY-FAIL $proj: error page detected in DOM"; fail=$((fail+1)); continue
  fi
  for spec in "Home--light--desktop|1440,4320" "Home--light--mobile|412,3000"; do
    view="${spec%%|*}"; geo="${spec#*|}"
    mkdir -p "$OUT/$proj"
    "$CHROME" --headless --no-sandbox --disable-gpu --hide-scrollbars \
      --blink-settings=lazyImageLoadingEnabled=false \
      --window-size="$geo" --virtual-time-budget=15000 \
      --screenshot="$OUT/$proj/$view.png" "$url" >/dev/null 2>&1
    shot_n=$((shot_n+1))
    verify_or_die "$proj" "$view" || fail=$((fail+1))
  done
done

echo "DONE: $shot_n shots, $fail verify failures"
[ "$fail" -eq 0 ]
