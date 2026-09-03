/**
 * Web shim for @tauri-apps/api/core.
 *
 * The web build does not have the Tauri runtime. The functions
 * here are placeholders that throw a clear error if called from
 * a web context. The Vite resolve.alias is configured to map
 * "@tauri-apps/api/core" to this file in the web build.
 */

export function invoke(cmd, args) {
  throw new Error(
    `Tauri command "${cmd}" is not available in the web build. ` +
    `Use the desktop build (npm run desktop:build) to access it.`
  );
}
