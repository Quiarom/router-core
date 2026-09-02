#!/bin/bash
# Full demo: serve + agent + frontend dev + integration test + live agent.
# Not committed; lives in /tmp.
set -e

cd /home/quiarom/Documents/Hackathon/router-core

# Build
go build -o /tmp/router-core ./cmd/router-core
go build -o /tmp/router-core-agent ./cmd/router-core-agent

export OPENROUTER_API_KEY="$(cat /tmp/openrouter_key 2>/dev/null || echo '')"

pkill -f 'router-core serve' 2>/dev/null || true
pkill -f 'node.*vite' 2>/dev/null || true
sleep 1

# Start serve in background
(echo 'admin'; sleep 8) | timeout 20 /tmp/router-core serve \
    --host 192.168.1.1 --addr 127.0.0.1:8484 > /tmp/serve.log 2>&1 &
SERVE_PID=$!

# Start vite in background
(cd frontend && npx vite --port 5180 --host 127.0.0.1 > /tmp/vite.log 2>&1 &)

# Wait for serve + vite
sleep 8

# Start agent in background (after serve is up)
if [ -n "$OPENROUTER_API_KEY" ]; then
    /tmp/router-core-agent \
        --router-core-url http://127.0.0.1:8484 \
        --serve 127.0.0.1:8585 \
        --model minimax/minimax-m2.7:free \
        > /tmp/agent.log 2>&1 &
    AGENT_PID=$!
    sleep 2
fi

echo '== integration test =='
node --test frontend/tests/integration.test.mjs 2>&1
INTEGRATION_STATUS=$?

if [ -n "$OPENROUTER_API_KEY" ]; then
    echo
    echo '== live agent trace (Is my Wi-Fi exposed?) =='
    curl -sS -X POST http://127.0.0.1:8585/v0/chat \
        -H 'Content-Type: application/json' \
        -d '{"question":"Is my Wi-Fi exposed?"}' \
        | head -c 800
    echo
fi

# Cleanup
kill $AGENT_PID 2>/dev/null || true
kill $SERVE_PID 2>/dev/null || true
pkill -f 'node.*vite' 2>/dev/null || true
wait 2>/dev/null
exit $INTEGRATION_STATUS
