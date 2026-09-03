# Aplicación de escritorio universal

La aplicación de escritorio usa Tauri 2, React y los servicios Go de
`router-core`. Inicia con una pantalla de conexión, detecta la
puerta de enlace activa y solicita las credenciales para consultar el router.

## Requisitos

- Linux x86_64 (Fedora, Debian, Ubuntu, Arch o distribuciones basadas en AppImage).
- Go 1.25 o posterior.
- Rust y Cargo.
- Node.js y npm.
- Dependencias del sistema para Tauri y WebKitGTK:

En distribuciones basadas en RPM (Fedora / RHEL):

```bash
sudo dnf install webkit2gtk4.1-devel openssl-devel curl wget file \
  libappindicator-gtk3-devel librsvg2-devel
```

En distribuciones basadas en DEB (Debian / Ubuntu):

```bash
sudo apt-get install libwebkit2gtk-4.1-dev build-essential curl wget file \
  libssl-dev libayatana-appindicator3-dev librsvg2-dev
```

## Ejecutar en desarrollo

Desde la raíz del repositorio:

```bash
make desktop-dev
```

La aplicación detecta el gateway mediante `ip route`, abre el formulario local
y arranca los sidecars únicamente después de que el usuario envía la
contraseña. La contraseña del router y la clave opcional del asistente se
mantienen en memoria y nunca se guardan en disco.

## Generar paquetes universales

```bash
make desktop-build
```

Los artefactos empaquetados se escriben en:

```text
frontend/src-tauri/target/release/bundle/
```

El script `scripts/build-desktop-sidecars.sh` compila los binarios Go con el
triple de Rust correspondiente a la máquina.

## Compatibilidad de routers

La aplicación admite la conexión universal con routers locales mediante su
interfaz de administración HTTP. Al conectar, identifica el fabricante, modelo
y versión de firmware, y consulta el estado, los clientes conectados y las
capacidades de seguridad admitidas por el dispositivo.

Los datos se consultan exclusivamente en modo lectura. La aplicación nunca
ejecuta operaciones de escritura ni modifica la configuración de los equipos.
