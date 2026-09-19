#!/usr/bin/env bash
# review-fleet.sh — full monthly review cycle for the 17-site fleet:
#   1. ensure the VL model server is healthy (vision-stack-up.sh)
#   2. re-capture all home pages with verify gates (shoot-sites.sh)
#   3. run the review pass (visionreviewd once; skip-seen keeps it incremental)
#   4. print + log the score table; desktop notification on completion/failure
#
# Designed for the fleet-review.timer systemd user timer; safe to run manually.
set -u
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
STATE_DIR="${XDG_STATE_HOME:-$HOME/.local/state}/site-monitor"
mkdir -p "$STATE_DIR"
LOG="$STATE_DIR/fleet-review.log"

notify() {
  command -v notify-send >/dev/null 2>&1 && notify-send -a fleet-review "$1" "$2" 2>/dev/null
  return 0
}

stamp=$(date -Is)
echo "$stamp === fleet review start" >> "$LOG"

# 1. model server
"$SCRIPT_DIR/vision-stack-up.sh" >> "$LOG" 2>&1 || {
  echo "$stamp vision stack DOWN - aborting" >> "$LOG"
  notify "fleet-review: model server down" "cycle aborted, see $LOG" critical
  exit 1
}

# 2. capture
if ! "$SCRIPT_DIR/shoot-sites.sh" >> "$LOG" 2>&1; then
  echo "$stamp shoot verify FAILED - reviews NOT run (bad captures must not be reviewed)" >> "$LOG"
  notify "fleet-review: capture verification failed" "reviews skipped, see $LOG" critical
  exit 1
fi

# 3. review
if ! "$HOME/projects/vision-review-agent/visionreviewd" once \
     -config "$HOME/.config/visionreviewd/websites.json" >> "$LOG" 2>&1; then
  echo "$stamp review pass FAILED" >> "$LOG"
  notify "fleet-review: review pass failed" "see $LOG" critical
  exit 1
fi

# 4. summary
echo "$stamp === scores" >> "$LOG"
summary=""
for d in "$HOME"/.local/share/vision-review-agent/reviews/*/; do
  p=$(basename "$d")
  [ "$p" = "discordsync" ] && continue
  f="$d/INDEX.md"
  [ -f "$f" ] || continue
  scores=$(grep -oE '\| [0-9]+/10 \|' "$f" | grep -oE '\b[0-9]+/10' | paste -sd' ')
  echo "$p: $scores" >> "$LOG"
  summary="$summary$p $scores\n"
done
notify "fleet-review complete" "scores in $LOG"
echo "fleet review complete: $LOG"
