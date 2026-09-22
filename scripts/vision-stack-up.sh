#!/usr/bin/env bash
# vision-stack-up.sh — ensure the VL model server for visionreviewd is healthy.
#
# - if http://127.0.0.1:8390/health already says ok, done
# - otherwise start llama-server from DIRECT LOCAL PATHS (the -hf form hangs
#   ~10 min on this machine; see docs/activation/site-fleet-ops.md §1.3)
# - wait up to TIMEOUT for health, then report
#
# Usage: scripts/vision-stack-up.sh [--port 8390] [--wait 900]
# Default wait is 900 s: the 8B model cold-reloads from the degraded /data NVMe
# in ~13 min (measured 2026-09-19); 120 s aborted the first monthly cycle.
set -u

PORT=8390
WAIT=900
MODEL_SNAP="${VISION_MODEL_SNAP:-/data/ai/cache/huggingface/hub/models--GitMyLo--nsfwcaption-qwen3-vl-8b-v3-gguf/snapshots/eb52b76411f34ea197558ec03eb15b2814d1b0c2}"
LLAMA_SERVER="${VISION_LLAMA_SERVER:-llama-server}"

while [ $# -gt 0 ]; do
	case "$1" in
	--port)
		PORT="$2"
		shift 2
		;;
	--wait)
		WAIT="$2"
		shift 2
		;;
	*)
		echo "unknown arg: $1" >&2
		exit 2
		;;
	esac
done

healthy() {
	python3 - "$PORT" <<'PYEOF'
import http.client, sys
try:
    c = http.client.HTTPConnection("127.0.0.1", int(sys.argv[1]), timeout=5)
    c.request("GET", "/health")
    r = c.getresponse()
    sys.exit(0 if r.status == 200 else 1)
except Exception:
    sys.exit(1)
PYEOF
}

if healthy; then
	echo "vision stack already healthy on :$PORT"
	exit 0
fi

MODEL="$MODEL_SNAP/NSFWCaption-v3-Qwen3-VL-8B-Q8_0.gguf"
MMPROJ="$MODEL_SNAP/mmproj-NSFWCaption-v3.gguf"
if [ ! -f "$MODEL" ] || [ ! -f "$MMPROJ" ]; then
	echo "model files missing under $MODEL_SNAP" >&2
	echo "set VISION_MODEL_SNAP to the correct snapshot dir" >&2
	exit 1
fi

if ! command -v "$LLAMA_SERVER" >/dev/null 2>&1; then
	echo "$LLAMA_SERVER not in PATH; set VISION_LLAMA_SERVER (e.g. /nix/store/<hash>-llama-cpp-0.4.0/bin/llama-server)" >&2
	exit 1
fi

echo "starting $LLAMA_SERVER on :$PORT (direct paths)"
start=$(date +%s)
nohup "$LLAMA_SERVER" -m "$MODEL" --mmproj "$MMPROJ" \
	--host 127.0.0.1 --port "$PORT" \
	>"/tmp/vision-stack-$PORT.log" 2>&1 &
echo "pid $!; log /tmp/vision-stack-$PORT.log"

deadline=$(($(date +%s) + WAIT))
until healthy; do
	[ "$(date +%s)" -ge "$deadline" ] && {
		echo "server not healthy after ${WAIT}s — check the log" >&2
		exit 1
	}
	sleep 2
done
echo "vision stack healthy on :$PORT (reload took $(($(date +%s) - start))s)"
