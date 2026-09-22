#!/usr/bin/env bash
# review-fleet.sh — full monthly review cycle for the 17-site fleet:
#   0. pre-flight: load guard, binary freshness, canary review latency
#   1. ensure the VL model server is healthy (vision-stack-up.sh)
#   2. re-capture all home pages with verify gates (cdp-shoot.py, which
#      now asserts its own shot inventory)
#   3. run the review pass (visionreviewd once; skip-seen keeps it incremental)
#   4. print + log the score table; desktop notification on completion/failure
#
# Designed for the fleet-review.timer systemd user timer; safe to run manually.
set -u
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
STATE_DIR="${XDG_STATE_HOME:-$HOME/.local/state}/site-monitor"
mkdir -p "$STATE_DIR"
LOG="$STATE_DIR/fleet-review.log"
BIN="$HOME/projects/vision-review-agent/visionreviewd"
CFG="$HOME/.config/visionreviewd/websites.json"
MAX_LOAD="${FLEET_MAX_LOAD:-10}"
CANARY_LIMIT="${FLEET_CANARY_SECS:-45}"
START_TS=$SECONDS

notify() {
	command -v notify-send >/dev/null 2>&1 && notify-send -a fleet-review "$1" "$2" ${3:-normal} 2>/dev/null
	return 0
}

# average of a "7/10 8/10 ..." score list, in tenths (bash has no floats)
avg10() {
	local sum=0 n=0 s
	for s in $1; do
		sum=$((sum + ${s%%/*}))
		n=$((n + 1))
	done
	[ "$n" -eq 0 ] && {
		echo 0
		return
	}
	echo $((sum * 10 / n))
}

stamp=$(date -Is)
echo "$stamp === fleet review start" >>"$LOG"

# 0a. load guard: an overloaded box turns every review into a 12-minute
#     timeout grind; defer instead of burning the window.
load1=$(cut -d' ' -f1 /proc/loadavg)
if awk -v l="$load1" -v m="$MAX_LOAD" 'BEGIN{exit !(l>m)}'; then
	echo "$stamp DEFERRED: load $load1 > $MAX_LOAD" >>"$LOG"
	notify "fleet-review deferred" "load $load1 > $MAX_LOAD; rerun later"
	exit 1
fi

# 0b. binary freshness: a stale visionreviewd silently lacks the newest
#     pipeline fixes (one full session ran on a months-old build).
if [ ! -x "$BIN" ]; then
	echo "$stamp ABORT: visionreviewd binary missing at $BIN" >>"$LOG"
	notify "fleet-review aborted" "binary missing, see $LOG" critical
	exit 1
fi
if [ -n "$(find "$HOME/projects/vision-review-agent" -name '*.go' ! -name '*_test.go' -newer "$BIN" -print -quit)" ]; then
	echo "$stamp ABORT: visionreviewd binary older than sources - rebuild first" >>"$LOG"
	notify "fleet-review aborted" "stale binary, rebuild + rerun" critical
	exit 1
fi

# 0c. canary: one 1-token completion measures queue latency; if the server
#     cannot even start responding, the pass would burn 12-minute timeouts.
canary_out=$(
	python3 - "$CANARY_LIMIT" <<'PYEOF'
import http.client, json, sys, time
limit = float(sys.argv[1])
try:
    c = http.client.HTTPConnection("127.0.0.1", 8390, timeout=limit)
    body = json.dumps({"model": "canary", "messages": [{"role": "user", "content": "hi"}], "max_tokens": 1})
    t0 = time.time()
    c.request("POST", "/v1/chat/completions", body, {"Content-Type": "application/json"})
    r = c.getresponse()
    r.read()
    print(f"{time.time() - t0:.1f}")
except Exception as e:
    print(f"FAIL {type(e).__name__}: {e}")
PYEOF
)
if [[ "$canary_out" == FAIL* ]]; then
	echo "$stamp ABORT: canary request failed: $canary_out" >>"$LOG"
	notify "fleet-review aborted" "canary failed: $canary_out" critical
	exit 1
fi
if awk -v c="$canary_out" -v m="$CANARY_LIMIT" 'BEGIN{exit !(c>m)}'; then
	echo "$stamp DEFERRED: canary ${canary_out}s > ${CANARY_LIMIT}s (server saturated)" >>"$LOG"
	notify "fleet-review deferred" "canary ${canary_out}s > ${CANARY_LIMIT}s"
	exit 1
fi
echo "canary ok: ${canary_out}s" >>"$LOG"

# 1. model server
"$SCRIPT_DIR/vision-stack-up.sh" >>"$LOG" 2>&1 || {
	echo "$stamp vision stack DOWN - aborting" >>"$LOG"
	notify "fleet-review: model server down" "cycle aborted, see $LOG" critical
	exit 1
}

# 2. capture (CDP full-page light+dark; supersedes the tall-viewport
#    shoot-sites.sh, which is kept as a fallback for CDP breakage)
if ! "$SCRIPT_DIR/cdp-shoot.py" >>"$LOG" 2>&1; then
	echo "$stamp CDP capture failed - retrying with shoot-sites.sh" >>"$LOG"
	if ! "$SCRIPT_DIR/shoot-sites.sh" >>"$LOG" 2>&1; then
		echo "$stamp shoot verify FAILED - reviews NOT run (bad captures must not be reviewed)" >>"$LOG"
		notify "fleet-review: capture verification failed" "reviews skipped, see $LOG" critical
		exit 1
	fi
fi

# 3. review
if ! "$BIN" once -config "$CFG" >>"$LOG" 2>&1; then
	echo "$stamp review pass FAILED" >>"$LOG"
	notify "fleet-review: review pass failed" "see $LOG" critical
	exit 1
fi

# 4. summary
echo "$stamp === scores" >>"$LOG"
summary=""
declare -a score_lines
for d in "$HOME"/.local/share/vision-review-agent/reviews/*/; do
	p=$(basename "$d")
	[ "$p" = "discordsync" ] && continue
	f="$d/INDEX.md"
	[ -f "$f" ] || continue
	scores=$(grep -oE '\| [0-9]+/10 \|' "$f" | grep -oE '\b[0-9]+/10' | paste -sd' ')
	echo "$p: $scores" >>"$LOG"
	score_lines+=("$p: $scores")
	summary="$summary$p $scores\n"
done

# trend digest: compare per-site averages against the previous cycle
# (|delta| >= 0.2 avg is reported; inside that is capture noise)
PREV_SCORES="$STATE_DIR/fleet-scores.prev"
if [ -f "$PREV_SCORES" ]; then
	for line in "${score_lines[@]}"; do
		site="${line%%:*}"
		newscores="${line#*: }"
		oldscores=$(grep -m1 "^$site:" "$PREV_SCORES" | cut -d' ' -f2-)
		[ -z "$oldscores" ] && continue
		navg=$(avg10 "$newscores")
		oavg=$(avg10 "$oldscores")
		delta=$((navg - oavg))
		if [ "$delta" -le -2 ] || [ "$delta" -ge 2 ]; then
			echo "$stamp TREND $site avg $((oavg / 10)).$((oavg % 10)) -> $((navg / 10)).$((navg % 10))" >>"$LOG"
		fi
	done
fi
printf '%s\n' "${score_lines[@]}" >"$PREV_SCORES"

echo "$stamp === cycle took $((SECONDS - START_TS))s (canary ${canary_out}s)" >>"$LOG"
notify "fleet-review complete" "scores in $LOG"
echo "fleet review complete: $LOG"
