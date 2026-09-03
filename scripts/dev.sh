#!/usr/bin/env bash
set -euo pipefail

# Ejecuta toda la pila de router-core.
# Inicia router-core serve, router-core-agent (MiniMax M3) y el frontend de Vite.

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

MODE="mock"
ROUTER_HOST="192.168.1.1"

while [[ $# -gt 0 ]]; do
  case "$1" in
    --live)
      MODE="live"
      shift
      ;;
    --mock)
      MODE="mock"
      shift
      ;;
    --host)
      ROUTER_HOST="$2"
      MODE="live"
      shift 2
      ;;
    *)
      echo "Uso: $0 [--mock | --live] [--host <IP>]"
      exit 1
      ;;
  esac
done

# Asegura que existan los binarios.
if [[ ! -f "$ROOT_DIR/bin/router-core" || ! -f "$ROOT_DIR/bin/router-core-agent" ]]; then
  echo "==> Compilando binarios de Go..."
  if command -v go >/dev/null 2>&1; then
    mkdir -p bin
    go build -o bin/router-core ./cmd/router-core
    go build -o bin/router-core-agent ./cmd/router-core-agent
  elif command -v podman >/dev/null 2>&1; then
    podman run --rm -v "$ROOT_DIR:/app:Z" -w /app docker.io/library/golang:alpine sh -c "mkdir -p bin && go build -o bin/router-core ./cmd/router-core && go build -o bin/router-core-agent ./cmd/router-core-agent"
  else
    echo "Error: No se encontró 'go' ni 'podman' para compilar los binarios."
    exit 1
  fi
fi

PIDS=()
cleanup() {
  echo -e "\n==> Deteniendo servicios..."
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
echo " Modo: $MODE"
echo "============================================================"

# 1. Start router-core serve
if [[ "$MODE" == "mock" ]]; then
  echo "==> [1/3] Iniciando router-core serve en http://127.0.0.1:8484 (Modo Mock Fixtures)..."
  "$ROOT_DIR/bin/router-core" serve --mock --addr 127.0.0.1:8484 &
  PIDS+=($!)
else
  echo "==> [1/3] Iniciando router-core serve en http://127.0.0.1:8484 (Router: $ROUTER_HOST)..."
  "$ROOT_DIR/bin/router-core" serve --host "$ROUTER_HOST" --addr 127.0.0.1:8484 &
  PIDS+=($!)
fi
sleep 1

# 2. Start router-core-agent
echo "==> [2/3] Iniciando router-core-agent en http://127.0.0.1:8585 (MiniMax M3)..."
"$ROOT_DIR/bin/router-core-agent" --serve 127.0.0.1:8585 --router-core-url http://127.0.0.1:8484 &
PIDS+=($!)
sleep 1

# 3. Start Frontend
echo "==> [3/3] Iniciando frontend Vite en http://localhost:5173..."
cd "$ROOT_DIR/frontend"
npm run dev
