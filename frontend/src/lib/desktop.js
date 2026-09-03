import { invoke } from "@tauri-apps/api/core";

/**
 * Indica si el frontend se está ejecutando dentro del contenedor nativo de Tauri.
 *
 * @returns {boolean} `true` cuando las API IPC de Tauri están disponibles.
 */
export function isDesktopRuntime() {
  return typeof window !== "undefined" && "__TAURI_INTERNALS__" in window;
}

/**
 * Detecta la puerta de enlace activa mediante el backend nativo de Fedora.
 *
 * @returns {Promise<string>} Dirección IPv4 privada de la puerta de enlace.
 */
export async function detectGateway() {
  return invoke("detect_gateway");
}

/**
 * Inicia los servicios locales y autentica el router con credenciales efímeras.
 *
 * @param {{host: string, username: string, password: string, aiApiKey?: string}} request Datos de conexión introducidos por el usuario.
 * @returns {Promise<{host: string, adapter: string, assistantMode: string, device: object}>} Resultado validado de la conexión.
 */
export async function connectRouter(request) {
  return invoke("connect_router", { request });
}

/**
 * Detiene los servicios locales y elimina la sesión mantenida en memoria.
 *
 * @returns {Promise<void>} Promesa resuelta al solicitar la desconexión.
 */
export async function disconnectRouter() {
  return invoke("disconnect_router");
}

/**
 * Consulta uno de los endpoints de observación permitidos por el backend nativo.
 *
 * @param {string} path Ruta fija de la API local de router-core.
 * @returns {Promise<{status: number, data: object}>} Estado HTTP y cuerpo JSON observado.
 */
export async function getRouterData(path) {
  if (isDesktopRuntime()) {
    return invoke("router_get", { path });
  }

  try {
    const response = await fetch(`/api/router${path}`);
    const data = await response.json().catch(() => null);
    return { status: response.status, data };
  } catch (error) {
    return {
      status: 0,
      data: null,
      error: error instanceof Error ? error.message : "No se pudo consultar router-core"
    };
  }
}

/**
 * Envía una pregunta al agente local usando exclusivamente observaciones del router.
 *
 * @param {string} question Pregunta escrita por el usuario.
 * @returns {Promise<{status: number, data: object}>} Respuesta HTTP normalizada del agente.
 */
export async function askAssistant(question) {
  if (isDesktopRuntime()) {
    return invoke("assistant_chat", { request: { question } });
  }

  try {
    const response = await fetch("/api/chat", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ question })
    });
    const data = await response.json().catch(() => null);
    return { status: response.status, data };
  } catch (error) {
    return {
      status: 0,
      data: null,
      error: error instanceof Error ? error.message : "No se pudo consultar el asistente"
    };
  }
}
