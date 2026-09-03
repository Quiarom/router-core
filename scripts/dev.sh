#!/usr/bin/env bash
set -euo pipefail

# router-core Full Stack Environment.
#
# Two dimensions of mode, both required to be explicit:
#
#   --router mock   : load a fixture-backed adapter (no hardware)
#   --router live   : talk to a real TP-Link WR841N on --host
#
#   --agent stub    : deterministic offline agent (no API key)
#   --agent live    : real MiniMax M3 (or M2.7 fallback) via GMI Cloud
#
# Combinations:
#   --router mock --agent stub   (no network, no API key)        -> CI / smoke
#   --router mock --agent live   (no hardware, real M3)          -> submission demo
#   --router live --agent live   (real router, real M3)          -> end-to-end
#
# The script never silently switches to paid/live inference because a
# key happens to exist. Live M3 needs GMI_SERVING_API_KEY; if the
# flag --agent live is set and the key is missing, fail fast.
#
# Usage:
#   ./scripts/dev.sh --router mock --agent stub
#   ./scripts/dev.sh --router mock --agent live
#   ./scripts/dev.sh --router live --agent live --host 192.168.1.1

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

ROUTER_MODE=""
AGENT_MODE=""
ROUTER_HOST=""

usage() {
    cat <<USAGE
Usage: $0 --router <mock|live> --agent <stub|live> [--host <IP>]

  --router mock           use a fixture-backed adapter (no hardware)
  --router live           talk to a real TP-Link WR841N
  --agent  stub           deterministic offline agent (no API key)
  --agent  live           MiniMax M3 (or fallback) via GMI Cloud
  --host   <IP>           router address for --router live (RFC1918 literal)
  --help                  this message

Both --router and --agent are required. There is no implicit default.
USAGE
    exit 1
}

while [[ $# -gt 0 ]]; do
    case "$1" in
        --router)
            ROUTER_MODE="$2"
            shift 2
            ;;
        --agent)
            AGENT_MODE="$2"
            shift 2
            ;;
        --host)
            ROUTER_HOST="$2"
            shift 2
            ;;
        --help|-h)
            usage
            ;;
        *)
            echo "Unknown flag: $1" >&2
            usage
            ;;
    esac
done

if [[ -z "$ROUTER_MODE" || -z "$AGENT_MODE" ]]; then
    echo "Error: both --router and --agent are required." >&2
    usage
fi

case "$ROUTER_MODE" in
    mock|live) ;;
    *) echo "Error: --router must be 'mock' or 'live', got '$ROUTER_MODE'." >&2; exit 1 ;;
esac
case "$AGENT_MODE" in
    stub|live) ;;
    *) echo "Error: --agent must be 'stub' or 'live', got '$AGENT_MODE'." >&2; exit 1 ;;
esac
if [[ "$ROUTER_MODE" == "live" && -z "$ROUTER_HOST" ]]; then
    echo "Error: --router live requires --host <IP>." >&2
    usage
fi
if [[ "$AGENT_MODE" == "live" && -z "${GMI_SERVING_API_KEY:-}" ]]; then
    echo "Error: --agent live requires GMI_SERVING_API_KEY in the environment." >&2
    echo "       Refusing to silently fall back to a stub agent." >&2
    exit 1
fi

# Build if needed
if [[ ! -f "$ROOT_DIR/bin/router-core" || ! -f "$ROOT_DIR/bin/router-core-agent" ]]; then
    echo "==> Building router-core binaries..."
    mkdir -p bin
    if command -v go >/dev/null 2>&1; then
        go build -o bin/router-core      ./cmd/router-core
        go build -o bin/router-core-agent ./cmd/router-core-agent
    elif command -v podman >/dev/null 2>&1; then
        podman run --rm -v "$ROOT_DIR:/app:Z" -w /app docker.io/library/golang:alpine \
            sh -c "go build -o bin/router-core ./cmd/router-core && \
                   go build -o bin/router-core-agent ./cmd/router-core-agent"
    else
        echo "Error: no 'go' or 'podman' available." >&2
        exit 1
    fi
fi

PIDS=()
cleanup() {
    echo
    echo "==> Stopping services..."
    for pid in "${PIDS[@]}"; do
        if kill -0 "$pid" 2>/dev/null; then
            kill "$pid" 2>/dev/null || true
        fi
    done
    exit 0
}
trap cleanup SIGINT SIGTERM EXIT

echo "============================================================"
echo " router-core Full Stack Environment"
echo "   router: $ROUTER_MODE$([ "$ROUTER_MODE" = "live" ] && echo " ($ROUTER_HOST)" || echo " (fixtures)")"
echo "   agent : $AGENT_MODE$([ "$AGENT_MODE" = "live" ] && echo " (MiniMax M3 via GMI Cloud)" || echo " (deterministic)")"
echo "============================================================"

# 1. Start router-core serve
if [[ "$ROUTER_MODE" == "mock" ]]; then
    echo "==> [1/3] router-core serve (mock fixtures) on http://127.0.0.1:8484"
    "$ROOT_DIR/bin/router-core" serve --mock --addr 127.0.0.1:8484 > /tmp/router-core-serve.log 2>&1 &
    PIDS+=($!)
else
    echo "==> [1/3] router-core serve (router: $ROUTER_HOST) on http://127.0.0.1:8484"
    "$ROOT_DIR/bin/router-core" serve --host "$ROUTER_HOST" --addr 127.0.0.1:8484 > /tmp/router-core-serve.log 2>&1 &
    PIDS+=($!)
fi
sleep 1

# 2. Start router-core-agent
if [[ "$AGENT_MODE" == "stub" ]]; then
    echo "==> [2/3] router-core-agent (deterministic stub) on http://127.0.0.1:8585"
    "$ROOT_DIR/bin/router-core-agent" \
        --serve 127.0.0.1:8585 \
        --router-core-url http://127.0.0.1:8484 \
        --dry-run > /tmp/router-core-agent.log 2>&1 &
else
    echo "==> [2/3] router-core-agent (MiniMax M3 via GMI Cloud) on http://127.0.0.1:8585"
    "$ROOT_DIR/bin/router-core-agent" \
        --serve 127.0.0.1:8585 \
        --router-core-url http://127.0.0.1:8484 > /tmp/router-core-agent.log 2>&1 &
fi
PIDS+=($!)
sleep 1

# 3. Start Frontend
echo "==> [3/3] frontend Vite on http://localhost:5173"
cd "$ROOT_DIR/frontend"
exec npm run dev
