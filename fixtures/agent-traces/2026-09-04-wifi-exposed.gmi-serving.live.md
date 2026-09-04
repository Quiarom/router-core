# Live agent trace via GMI Cloud direct — 2026-09-04

This is a real, captured trace from the MiniMax M3 reasoning
layer running against the live lab unit at 192.168.1.1
(firmware 3.15.9 Build 140724 Rel.63227n) via GMI Cloud direct
(api.gmi-serving.com), not via OpenRouter.

The audit is more careful about uncertainty than the
OpenRouter/m2.7 baseline: it explicitly names the cipher-332
ambiguity (could be AES-CCMP or TKIP), the firmware age, and the
hardware EOL risk as the primary concern. Recommendations
include router replacement as a priority.

## Setup

```sh
export GMI_SERVING_API_KEY="<jwt-key>"
router-core serve --host 192.168.1.1 --addr 127.0.0.1:8484 &
router-core-agent \\
    --router-core-url http://127.0.0.1:8484 \\
    --question 'Is my Wi-Fi exposed?'
```

## Trace (verbatim)

```text
## Resumen de auditoría — Seguridad Wi-Fi

### Hechos observados (verified)
- **Wi-Fi habilitado**: sí.
- **SSID**: `TP-LINK_CBEC16` (predecible, derivado del BSSID/MAC; no oculto).
- **Cifrado**: WPA2-PSK (SecurityType=3, Cipher=332). No se observa WPA3 ni modo mixto WPA2/WPA3.
- **PSK configurada**: sí (no se expone el contenido).
- **WPS**: ausente (capability `wps: absent`), por lo que no hay vector de PIN WPS activo.
- **UPnP**: ausente.
- **Administración remota**: ausente.
- **Firmware**: 3.15.9 Build 140724 Rel.63227n (versión 2014; el TP-Link TL-WR841N v8 es un router EOL/antiguo).

### Límites de la evidencia
- `SecurityType=3` y `Cipher=332` son compatibles con **AES-CCMP (WPA2-PSK/AES)**, lo más habitual, pero la observación es de tipo de seguridad genérico; no se confirma con certeza absoluta que el cifrado sea exclusivamente AES y no TKIP.
- No se observó el modo radio (b/g/n), el ancho de canal, ni la potencia de transmisión, que también afectan la exposición.
- Solo se observa **1 cliente DHCP** (`omarchy`, 192.168.1.100, MAC `d8:43:ae:bb:e3:41`); no se observan clientes Wi-Fi aislados, por lo que no puedo confirmar cuántos dispositivos inalámbricos están realmente conectados.

### Riesgos posibles
1. **Firmware muy antiguo (2014) sin parches**: vulnerabilidades conocidas para el chipset/Wi-Fi del WR841N v8 no remediables en esta versión.
2. **SSID predeterminada del fabricante**: revela modelo y facilita ataques dirigidos y deanominación.
3. **WPA2-PSK sin WPA3**: vulnerable a ataques de diccionario/offline si la contraseña es débil, y susceptible a KRACK (parches dependientes del firmware).
4. **Hardware EOL**: es probable que no reciba más actualizaciones de seguridad.

### Recomendaciones
1. **Reemplazo del router** por un modelo con soporte activo y WPA3 (prioridad alta, dado el firmware de 2014).
2. Mientras tanto:
   - Cambiar el SSID a uno neutro que no identifique modelo/marca.
   - Mantener **AES/CCMP** (no TKIP) si la opción está disponible.
   - Usar una **PSK fuerte** (≥ 16 caracteres, aleatoria).
   - Reducir la potencia de transmisión al mínimo necesario.
   - Mantener el firmware actualizado dentro de lo que el fabricante ofrezca para v8.
3. No activar WPS ni UPnP (ya están ausentes, correcto).

### Estado por capacidad
- wireless_security: **verified**
- wps: **absent**
- upnp: **absent**
- remote_management: **absent**
- clients/device/status: **verified**
- dmz, forwarding: no consultados en esta auditoría

No se realizó ningún cambio en el router.
```
