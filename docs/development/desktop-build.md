# Desktop build

The Gavetero desktop app is a Tauri 2.x wrapper around the
web frontend. It spawns `router-core` and `router-core-agent`
as sidecar processes on startup.

## Prerequisites

- Rust toolchain (cargo, rustc)
- Tauri CLI 2.11+: `cargo install tauri-cli --version '^2.0' --locked`
- webkit2gtk-4.1, gtk3, libsoup-3.0, librsvg-2
- The Go toolchain (for the sidecar build script)

## Build

```sh
# 1. Build the Go sidecars into the directory Tauri expects
scripts/build-desktop-sidecars.sh

# 2. Build the desktop bundle
cd frontend/src-tauri
cargo tauri build --bundles deb,appimage
```

The output is at:

```
frontend/src-tauri/target/release/bundle/deb/router-core_0.1.0_amd64.deb
frontend/src-tauri/target/release/bundle/appimage/router-core_0.1.0_amd64.AppImage
```

## Verified on Arch / Omarchy (2026-09-03)

- `cargo build --release`: clean (Tauri 2.11.4, webkit2gtk-4.1 2.52.6)
- `.deb` bundle: 19 MB, contains binary + 2 sidecars + .desktop + 4 icons
- AppImage: not verified in this environment (linuxdeploy plugin issue)

The desktop app is functionally complete but not part of the
MiniMax-Week submission (the submission is the gavetero CLI
and the web frontend). The desktop is post-submission work.
