# Aplicación de escritorio en Fedora

La aplicación de escritorio usa Tauri 2, React y los servicios Go de
`router-core`. En Fedora inicia con una pantalla de conexión, detecta la
puerta de enlace y solicita las credenciales antes de consultar el router.

## Requisitos

- Fedora Linux x86_64.
- Go 1.25 o posterior.
- Rust y Cargo.
- Node.js y npm.
- Dependencias de desarrollo de Tauri para Fedora:

```bash
sudo dnf install webkit2gtk4.1-devel openssl-devel curl wget file \
  libappindicator-gtk3-devel librsvg2-devel
```

El paquete RPM generado necesita que el sistema tenga WebKitGTK 4.1 instalado.

## Ejecutar en desarrollo

Desde la raíz del repositorio:

```bash
make desktop-dev
```

La aplicación detecta el gateway mediante `ip route`, abre el formulario local
y arranca los sidecars únicamente después de que el usuario envía la
contraseña. La contraseña del router y la clave opcional del asistente no se
pasan como argumentos ni se guardan en disco.

## Generar paquetes

```bash
make desktop-build
```

Los artefactos se escriben en:

```text
frontend/src-tauri/target/release/bundle/
```

El script `scripts/build-desktop-sidecars.sh` compila los binarios Go con el
triple de Rust de la máquina. Los binarios generados y `target/` están
ignorados por Git.

## Estado de compatibilidad

La aplicación distingue entre conexión de red y compatibilidad del firmware.
Responder al gateway no es suficiente para marcar un router como compatible:
la identificación debe incluir autenticación y una versión de firmware
observada.

Actualmente está verificado el adaptador TP-Link TL-WR841N/ND v8.4. El
Sercomm IP3442M-L/US se detecta como firmware no compatible hasta incorporar su
receta de autenticación y sus consultas JSON-RPC a partir de una captura
sanitizada. No se muestran datos mock cuando la aplicación está en modo real.

Los adaptadores futuros deben implementar solo observaciones verificadas. Las
operaciones JSON-RPC de mutación, como `SET`, no forman parte de la aplicación.
