#!/usr/bin/env bash
# site-monitor.sh — HTTP status + TLS cert-expiry monitor for the lars.software site fleet.
#
# Checks each target over HTTPS (SNI), classifies:
#   OK      — 2xx/3xx status and cert valid for > WARN_DAYS
#   WARN    — reachable but cert expires within WARN_DAYS
#   FAIL    — unreachable, TLS error, or non-2xx/3xx status
#   KNOWN   — FAIL on a target already known to be broken pending a manual
#             (console/DNS) action; logged, not alerted.
#
# Alerts via notify-send on state TRANSITIONS only (ok->fail, fail->ok),
# so an hourly timer does not spam. Full log every run:
#   ~/.local/state/site-monitor/log
#
# Usage:
#   site-monitor.sh               check all targets, alert on transitions
#   site-monitor.sh --test-alert  force a fake failure to verify the alert path
#   site-monitor.sh --json        machine-readable summary to stdout
set -u

WARN_DAYS=21
TIMEOUT=15
KNOWN_ESCALATE_DAYS=7
STATE_DIR="${XTEST_STATE_DIR:-${XDG_STATE_HOME:-$HOME/.local/state}/site-monitor}"
mkdir -p "$STATE_DIR"
LOG="$STATE_DIR/log"

# target list: host
# (each is checked at https://host/; entries also in KNOWN_BROKEN are
#  classified KNOWN when they fail, so they are tracked without alerting)
TARGETS=(
  "art-dupl.lars.software"
  "cleanwizard.lars.software"
  "dynamicmarkdown.lars.software"
  "emeet-pixyd.lars.software"
  "atomicwrite.lars.software"
  "branded-id.lars.software"
  "errorfamily.lars.software"
  "filewatcher.lars.software"
  "go-output.lars.software"
  "go-workflow-auditlog.lars.software"
  "gogenfilter.lars.software"
  "md-go-validator.lars.software"
  "do-auditlog.lars.software"
  "templcomponents.lars.software"
  "cmdguard.web.app"
  "typespec-asyncapi.web.app"
  "lars-learnings.web.app"
  "cmdguard.lars.software"
  "typespec-asyncapi.lars.software"
)

# KNOWN-broken targets (pending manual console/DNS action) — do not alert.
KNOWN_BROKEN=(
  "cmdguard.lars.software:TLS cert invalid - domain not attached in Firebase console (site works at cmdguard.web.app)"
  "typespec-asyncapi.lars.software:DNS missing - Namecheap CNAME to typespec-asyncapi.web.app not created yet (site works at typespec-asyncapi.web.app)"
)

is_known_broken() {
  local host="$1" entry
  for entry in "${KNOWN_BROKEN[@]}"; do
    [ "${entry%%:*}" = "$host" ] && return 0
  done
  return 1
}

check_host() {
  local host="$1"
  # status line from a single TLS session, capped by timeout
  printf 'GET / HTTP/1.1\r\nHost: %s\r\nConnection: close\r\n\r\n' "$host" |
    timeout "$TIMEOUT" openssl s_client -quiet -connect "$host:443" -servername "$host" 2>/dev/null
}

cert_days() {
  local host="$1" end ts now
  end=$(echo | timeout "$TIMEOUT" openssl s_client -connect "$host:443" -servername "$host" 2>/dev/null \
        | openssl x509 -noout -enddate 2>/dev/null | cut -d= -f2)
  [ -z "$end" ] && { echo "-1"; return; }
  ts=$(date -d "$end" +%s 2>/dev/null) || { echo "-1"; return; }
  now=$(date +%s)
  echo $(( (ts - now) / 86400 ))
}

prev_state() {
  local f="$STATE_DIR/$1.state"
  [ -f "$f" ] && cat "$f" || echo "unknown"
}

set_state() {
  printf '%s' "$2" > "$STATE_DIR/$1.state"
}

notify() {
  command -v notify-send >/dev/null 2>&1 || return 0
  notify-send -a site-monitor -u "${3:-normal}" "$1" "$2" 2>/dev/null
}

TEST_ALERT=0; JSON=0
for arg in "$@"; do
  case "$arg" in
    --test-alert) TEST_ALERT=1 ;;
    --json) JSON=1 ;;
  esac
done

now_iso=$(date -Is)
declare -a summary
unexpected_fail=0; warn_count=0; known_count=0; ok_count=0

if [ "$TEST_ALERT" = 1 ]; then
  notify "site-monitor TEST" "This is a forced test alert for host test-alert.invalid - if you can read this, the alert path works." critical
  echo "$now_iso TEST-ALERT sent" >> "$LOG"
  exit 0
fi

for t in "${TARGETS[@]}"; do
  host="$t"
  out=$(check_host "$host")
  status=$(printf '%s' "$out" | head -1 | grep -oE 'HTTP/1\.[01] [0-9]+' | grep -oE '[0-9]+$')
  days=$(cert_days "$host")

  state="FAIL"; detail=""
  if [ -n "$status" ] && [[ "$status" =~ ^[23] ]]; then
    if [ "$days" -ge 0 ] && [ "$days" -le "$WARN_DAYS" ]; then
      state="WARN"; detail="cert expires in ${days}d"
    else
      state="OK"; detail="HTTP $status, cert ${days}d"
    fi
  else
    detail="no HTTP status (TLS/conn error)"
  fi

  if [ "$state" = "FAIL" ] && is_known_broken "$host"; then
    state="KNOWN"
  fi

  case "$state" in
    OK)   ok_count=$((ok_count+1)) ;;
    WARN) warn_count=$((warn_count+1)) ;;
    KNOWN) known_count=$((known_count+1)) ;;
    FAIL) unexpected_fail=$((unexpected_fail+1)) ;;
  esac

  # KNOWN-broken escalation: a domain pending a console/DNS action must not
  # rot silently — re-alert (critical) every KNOWN_ESCALATE_DAYS while it
  # stays KNOWN.
  since_f="$STATE_DIR/$host.known-since"
  esc_f="$STATE_DIR/$host.last-escalation"
  if [ "$state" = "KNOWN" ]; then
    [ -f "$since_f" ] || printf '%s' "$(date +%s)" > "$since_f"
    since=$(cat "$since_f")
    age_days=$(( ($(date +%s) - since) / 86400 ))
    today=$(date +%F)
    if [ "$age_days" -ge "$KNOWN_ESCALATE_DAYS" ] && [ "$(cat "$esc_f" 2>/dev/null)" != "$today" ]; then
      printf '%s' "$today" > "$esc_f"
      notify "site-monitor: $host still broken" "known-broken for ${age_days}d - console/DNS action still pending" critical
      echo "$now_iso ESCALATION $host known-broken ${age_days}d" >> "$LOG"
    fi
  else
    rm -f "$since_f" "$esc_f"
  fi

  prev=$(prev_state "$host")
  if [ "$prev" != "$state" ]; then
    if [ "$state" = "FAIL" ]; then
      notify "site-monitor: $host DOWN" "$detail" critical
    elif [ "$state" = "WARN" ]; then
      notify "site-monitor: $host cert warning" "$detail"
    elif [ "$prev" = "FAIL" ] || [ "$prev" = "WARN" ]; then
      notify "site-monitor: $host recovered" "$detail"
    fi
    set_state "$host" "$state"
  fi

  summary+=("$state $host ($detail)")
  echo "$now_iso $state $host $detail" >> "$LOG"
done

if [ "$JSON" = 1 ]; then
  printf '{"ok":%d,"warn":%d,"known":%d,"fail":%d,"at":"%s"}\n' \
    "$ok_count" "$warn_count" "$known_count" "$unexpected_fail" "$now_iso"
else
  printf '%s\n' "${summary[@]}"
  echo "site-monitor: $ok_count ok, $warn_count warn, $known_count known-broken, $unexpected_fail UNEXPECTED FAIL"
fi

[ "$unexpected_fail" -eq 0 ]
