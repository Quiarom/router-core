# Live agent trace — 2026-09-04

This is a real, captured trace from the MiniMax M2.7 reasoning
layer running against the live lab unit at `192.168.1.1`
(firmware 3.15.9 Build 140724 Rel.63227n) via OpenRouter.

The trace was captured on the `develop` branch at commit
`8090e55 docs: add PHASE5_AGENT_RUN.md with the reproducible run
playbook` (the state of the repo immediately before the live
OpenRouter run). The API key has been redacted; everything else
is verbatim from the live run.

## Setup

```sh
export OPENROUTER_API_KEY="<redacted>"  # user-tier key with chat completions
# Build
go build -o /tmp/router-core ./cmd/router-core
go build -o /tmp/router-core-agent ./cmd/router-core-agent
# Start serve
(echo 'admin'; sleep 6) | timeout 8 /tmp/router-core serve \
    --host 192.168.1.1 --addr 127.0.0.1:8500 &
# Ask
/tmp/router-core-agent \
    --router-core-url http://127.0.0.1:8500 \
    --model minimax/minimax-m2.7:free \
    --question 'Is my Wi-Fi exposed?'
```

## Trace (verbatim)

```text
router-core-agent: pregunta="Is my Wi-Fi exposed?"
router-core-agent: get_security -> HTTP 200
Tu red Wi-Fi **no está abierta**, pero tiene configuraciones que conviene revisar:

---

## Lo observado

| Aspecto | Valor | Evaluación |
|---|---|---|
| Estado | Habilitada | ✅ La red existe |
| Seguridad | WPA2-PSK | ✅ Aceptable para uso doméstico |
| Clave pre-compartida | Presente | ✅ No esabierta |
| SSID | TP-LINK_CBEC16 | ⚠️ Visible para cualquiera que escanee redes |

---

## Límites de la evidencia

- **No puedo ver la contraseña** ni evaluar su fortaleza.
- **No puedo confirmar** si el SSID está oculto o si hay filtrado MAC.
- **La visibilidad del nombre** no es en sí un riesgo grave, pero facilita que terceros intenten conectarse.

---

## Recomendación

1. **Cambia la contraseña WPA2-PSK** si hace más de un año que no lo haces o si la compartiste con personas que ya no viven en tu hogar.
2. **No puedo verificar WPS** (esta consulta no está disponible), pero si tu router lo tiene activado, desactívalo desde la interfaz web en `192.168.1.1` — es una vulnerabilidad conocida.

¿Quieres que revise también el estado del puerto o el UPnP?
```

## What the agent called

The model decided to call one tool: `get_security("wireless")`,
which hits `GET /v0/security/wireless` on the local serve. That
endpoint returned 200 with the parsed `WirelessSecurity` struct
including `SSID`, `SecurityType` (WPA2-PSK), `Cipher` (332 = AES),
`KeyRenewalSecs` (1812), and `HasPreSharedKey` (true).

The model received the parsed JSON, interpreted the evidence
against its own knowledge of Wi-Fi security best practices, and
produced the answer above. It also named its uncertainty ("no
puedo ver la contraseña") and asked a follow-up question about
port/UPnP.

## How the frontend can replay this

The frontend dev can wire `fixtures/agent-traces/2026-09-04-wifi-exposed.live.jsonl`
(one JSON object per turn) as a dev-mode mock for `/v0/chat`.
That gives the dashboard a real, captured agent trace to render
without needing an OpenRouter key in CI.
