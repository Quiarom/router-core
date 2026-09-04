use std::{
    net::{IpAddr, Ipv4Addr, TcpListener},
    process::Command as SystemCommand,
    sync::Mutex,
    time::Duration,
};

use reqwest::Client;
use serde::{Deserialize, Serialize};
use serde_json::{json, Value};
use tauri::{Manager, State};
use tauri_plugin_shell::{process::CommandChild, process::CommandEvent, ShellExt};
use zeroize::Zeroize;

const MAX_SECRET_SIZE: usize = 512;
const MAX_RESPONSE_SIZE: usize = 2 << 20;
const ROUTER_HEALTH_PATH: &str = "/healthz";
const AGENT_HEALTH_PATH: &str = "/healthz";
const ALLOWED_ROUTER_PATHS: &[&str] = &[
    "/healthz",
    "/v0/device",
    "/v0/status",
    "/v0/clients",
    "/v0/capabilities",
    "/v0/security/wireless",
    "/v0/security/wps",
    "/v0/security/dmz",
    "/v0/security/upnp",
    "/v0/security/remote-management",
    "/v0/security/forwarding",
];

#[derive(Default)]
struct RuntimeServices {
    router: Option<CommandChild>,
    agent: Option<CommandChild>,
    router_url: Option<String>,
    agent_url: Option<String>,
}

#[derive(Default)]
struct ServiceState {
    runtime: Mutex<RuntimeServices>,
}

#[derive(Deserialize)]
#[serde(rename_all = "camelCase")]
struct ConnectRequest {
    host: String,
    username: String,
    password: String,
    ai_api_key: Option<String>,
}

impl Drop for ConnectRequest {
    fn drop(&mut self) {
        self.password.zeroize();
        if let Some(key) = self.ai_api_key.as_mut() {
            key.zeroize();
        }
    }
}

#[derive(Serialize)]
#[serde(rename_all = "camelCase")]
struct ConnectionInfo {
    host: String,
    adapter: String,
    assistant_mode: String,
    device: Value,
}

#[derive(Serialize)]
#[serde(rename_all = "camelCase")]
struct LocalResponse {
    status: u16,
    data: Value,
}

#[derive(Deserialize)]
struct ChatRequest {
    question: String,
}

fn validate_host(raw_host: &str) -> Result<String, String> {
    let host = raw_host.trim();
    let ip = host
        .parse::<IpAddr>()
        .map_err(|_| "La dirección del router debe ser una IP local válida".to_string())?;

    let is_allowed = match ip {
        IpAddr::V4(ipv4) => ipv4.is_private() || ipv4.is_loopback(),
        IpAddr::V6(ipv6) => ipv6.is_loopback(),
    };
    if !is_allowed {
        return Err("Solo se permiten direcciones privadas o loopback".to_string());
    }
    Ok(host.to_string())
}

fn reserve_loopback_addr() -> Result<String, String> {
    let listener = TcpListener::bind((Ipv4Addr::LOCALHOST, 0))
        .map_err(|error| format!("No se pudo reservar un puerto local: {error}"))?;
    let address = listener
        .local_addr()
        .map_err(|error| format!("No se pudo leer el puerto local: {error}"))?;
    drop(listener);
    Ok(address.to_string())
}

fn stop_services(state: &ServiceState) {
    let Ok(mut runtime) = state.runtime.lock() else {
        return;
    };
    if let Some(agent) = runtime.agent.take() {
        let _ = agent.kill();
    }
    if let Some(router) = runtime.router.take() {
        let _ = router.kill();
    }
    runtime.router_url = None;
    runtime.agent_url = None;
}

async fn wait_for_service(client: &Client, base_url: &str, path: &str) -> Result<(), String> {
    for _ in 0..40 {
        if let Ok(response) = client.get(format!("{base_url}{path}")).send().await {
            if response.status().is_success() {
                return Ok(());
            }
        }
        tokio::time::sleep(Duration::from_millis(125)).await;
    }
    Err(format!("El servicio local {base_url} no respondió a tiempo"))
}

async fn fetch_json(client: &Client, url: String) -> Result<LocalResponse, String> {
    let response = client
        .get(&url)
        .send()
        .await
        .map_err(|error| format!("No se pudo consultar el servicio local: {error}"))?;
    let status = response.status().as_u16();
    let bytes = response
        .bytes()
        .await
        .map_err(|error| format!("No se pudo leer la respuesta local: {error}"))?;
    if bytes.len() > MAX_RESPONSE_SIZE {
        return Err("La respuesta local excede el límite permitido".to_string());
    }
    let data = serde_json::from_slice(&bytes)
        .map_err(|error| format!("El servicio local devolvió JSON inválido: {error}"))?;
    Ok(LocalResponse { status, data })
}

fn validate_device(response: &LocalResponse) -> Result<(), String> {
    if response.status != 200 {
        return Err("El router no respondió correctamente".to_string());
    }
    let authenticated = response
        .data
        .get("authenticated")
        .and_then(Value::as_str);

    if authenticated == Some("false") {
        return Err("Autenticación rechazada por el router".to_string());
    }

    let vendor = response
        .data
        .get("vendor")
        .and_then(Value::as_str)
        .unwrap_or("");
    let model = response
        .data
        .get("model")
        .and_then(Value::as_str)
        .unwrap_or("");
    let hardware = response
        .data
        .pointer("/hardwareVersion/value")
        .and_then(Value::as_str)
        .unwrap_or("");
    let firmware = response
        .data
        .pointer("/firmwareVersion/value")
        .and_then(Value::as_str)
        .unwrap_or("");

    if vendor.is_empty() && model.is_empty() && hardware.is_empty() && firmware.is_empty() {
        return Err("No se pudo obtener información de identidad del router".to_string());
    }

    Ok(())
}

#[tauri::command]
fn detect_gateway() -> Result<String, String> {
    let output = SystemCommand::new("ip")
        .args(["route", "show", "default"])
        .output()
        .map_err(|error| format!("No se pudo ejecutar ip route: {error}"))?;
    if !output.status.success() {
        return Err("No se pudo detectar la puerta de enlace".to_string());
    }

    let stdout = String::from_utf8_lossy(&output.stdout);
    let gateway = stdout
        .split_whitespace()
        .collect::<Vec<_>>()
        .windows(2)
        .find_map(|tokens| (tokens[0] == "via").then_some(tokens[1]))
        .ok_or_else(|| "No se encontró una puerta de enlace activa".to_string())?;
    validate_host(gateway)
}

#[tauri::command]
async fn connect_router(
    app: tauri::AppHandle,
    state: State<'_, ServiceState>,
    mut request: ConnectRequest,
) -> Result<ConnectionInfo, String> {
    let host = validate_host(&request.host)?;
    let username = request.username.trim().to_string();
    if username.is_empty() || username.len() > 128 {
        return Err("El usuario del router no es válido".to_string());
    }
    if request.password.is_empty()
        || request.password.len() > MAX_SECRET_SIZE
        || request.password.contains(['\r', '\n'])
    {
        request.password.zeroize();
        return Err("La contraseña del router no es válida".to_string());
    }
    if request
        .ai_api_key
        .as_ref()
        .is_some_and(|key| key.len() > MAX_SECRET_SIZE || key.contains(['\r', '\n']))
    {
        request.password.zeroize();
        if let Some(key) = request.ai_api_key.as_mut() {
            key.zeroize();
        }
        return Err("La clave del asistente no es válida".to_string());
    }

    stop_services(&state);

    let router_addr = reserve_loopback_addr()?;
    let router_url = format!("http://{router_addr}");
    let router_command = app
        .shell()
        .sidecar("router-core")
        .map_err(|error| format!("No se encontró router-core: {error}"))?
        .args([
            "serve",
            "--host",
            &host,
            "--username",
            &username,
            "--addr",
            &router_addr,
            // The router password is written to the child stdin
            // below (not as --password, which is intentionally
            // not a supported flag in router-core serve).
            // The --password-stdin flag tells the child binary
            // to read the password from stdin, not from a TTY.
            "--password-stdin",
        ]);
    let (mut router_events, mut router_child) = router_command
        .spawn()
        .map_err(|error| format!("No se pudo iniciar router-core: {error}"))?;

    let mut password_input = format!("{}\n", request.password);
    let write_result = router_child.write(password_input.as_bytes());
    password_input.zeroize();
    request.password.zeroize();
    if let Err(error) = write_result {
        let _ = router_child.kill();
        return Err(format!("No se pudo entregar la contraseña: {error}"));
    }

    tauri::async_runtime::spawn(async move {
        while let Some(event) = router_events.recv().await {
            if let CommandEvent::Error(error) = event {
                log::error!("router-core terminó con error: {error}");
            }
        }
    });

    {
        let mut runtime = state
            .runtime
            .lock()
            .map_err(|_| "No se pudo guardar el estado local".to_string())?;
        runtime.router = Some(router_child);
        runtime.router_url = Some(router_url.clone());
    }

    let client = Client::builder()
        .timeout(Duration::from_secs(8))
        .build()
        .map_err(|error| format!("No se pudo crear el cliente local: {error}"))?;
    if let Err(error) = wait_for_service(&client, &router_url, ROUTER_HEALTH_PATH).await {
        stop_services(&state);
        return Err(format!(
            "{error}. Comprueba el usuario, la contraseña y la compatibilidad del router"
        ));
    }

    let device = match fetch_json(&client, format!("{router_url}/v0/device")).await {
        Ok(response) => response,
        Err(error) => {
            stop_services(&state);
            return Err(error);
        }
    };
    if let Err(error) = validate_device(&device) {
        stop_services(&state);
        return Err(error);
    }

    let agent_addr = match reserve_loopback_addr() {
        Ok(address) => address,
        Err(error) => {
            stop_services(&state);
            return Err(error);
        }
    };
    let agent_url = format!("http://{agent_addr}");
    let mut agent_args = vec![
        "--serve".to_string(),
        agent_addr,
        "--router-core-url".to_string(),
        router_url.clone(),
    ];
    let has_live_assistant = request
        .ai_api_key
        .as_ref()
        .is_some_and(|key| !key.trim().is_empty());
    if !has_live_assistant {
        agent_args.push("--dry-run".to_string());
    }

    let mut agent_command = match app.shell().sidecar("router-core-agent") {
        Ok(command) => command.args(agent_args),
        Err(error) => {
            stop_services(&state);
            return Err(format!("No se encontró router-core-agent: {error}"));
        }
    };
    if let Some(key) = request.ai_api_key.as_mut() {
        if !key.trim().is_empty() {
            agent_command = agent_command.env("OPENROUTER_API_KEY", key.trim());
        }
        key.zeroize();
    }

    let (mut agent_events, agent_child) = match agent_command.spawn() {
        Ok(result) => result,
        Err(error) => {
            stop_services(&state);
            return Err(format!("No se pudo iniciar router-core-agent: {error}"));
        }
    };
    tauri::async_runtime::spawn(async move {
        while let Some(event) = agent_events.recv().await {
            if let CommandEvent::Error(error) = event {
                log::error!("router-core-agent terminó con error: {error}");
            }
        }
    });

    {
        let mut runtime = match state.runtime.lock() {
            Ok(runtime) => runtime,
            Err(_) => {
                let _ = agent_child.kill();
                stop_services(&state);
                return Err("No se pudo guardar el estado del asistente".to_string());
            }
        };
        runtime.agent = Some(agent_child);
        runtime.agent_url = Some(agent_url.clone());
    }
    if let Err(error) = wait_for_service(&client, &agent_url, AGENT_HEALTH_PATH).await {
        stop_services(&state);
        return Err(error);
    }

    let vendor = device
        .data
        .get("vendor")
        .and_then(Value::as_str)
        .unwrap_or("router");
    let model = device
        .data
        .get("model")
        .and_then(Value::as_str)
        .unwrap_or("generic");
    let raw_name = format!("{vendor}-{model}")
        .to_lowercase()
        .replace([' ', '/'], "-");
    let adapter = if raw_name.trim_matches('-').is_empty() {
        "universal-router".to_string()
    } else {
        raw_name.trim_matches('-').to_string()
    };

    Ok(ConnectionInfo {
        host,
        adapter,
        assistant_mode: if has_live_assistant {
            "minimax".to_string()
        } else {
            "deterministic".to_string()
        },
        device: device.data,
    })
}

#[tauri::command]
async fn router_get(
    state: State<'_, ServiceState>,
    path: String,
) -> Result<LocalResponse, String> {
    if !ALLOWED_ROUTER_PATHS.contains(&path.as_str()) {
        return Err("La ruta solicitada no está permitida".to_string());
    }
    let router_url = state
        .runtime
        .lock()
        .map_err(|_| "No se pudo leer el estado local".to_string())?
        .router_url
        .clone()
        .ok_or_else(|| "No hay un router conectado".to_string())?;
    let client = Client::builder()
        .timeout(Duration::from_secs(8))
        .build()
        .map_err(|error| format!("No se pudo crear el cliente local: {error}"))?;
    fetch_json(&client, format!("{router_url}{path}")).await
}

#[tauri::command]
async fn assistant_chat(
    state: State<'_, ServiceState>,
    request: ChatRequest,
) -> Result<LocalResponse, String> {
    let question = request.question.trim();
    if question.is_empty() || question.len() > 64 * 1024 {
        return Err("La pregunta no es válida".to_string());
    }
    let agent_url = state
        .runtime
        .lock()
        .map_err(|_| "No se pudo leer el estado local".to_string())?
        .agent_url
        .clone()
        .ok_or_else(|| "El asistente no está iniciado".to_string())?;
    let client = Client::builder()
        .timeout(Duration::from_secs(55))
        .build()
        .map_err(|error| format!("No se pudo crear el cliente local: {error}"))?;
    let response = client
        .post(format!("{agent_url}/v0/chat"))
        .json(&json!({ "question": question }))
        .send()
        .await
        .map_err(|error| format!("No se pudo consultar el asistente: {error}"))?;
    let status = response.status().as_u16();
    let bytes = response
        .bytes()
        .await
        .map_err(|error| format!("No se pudo leer la respuesta del asistente: {error}"))?;
    if bytes.len() > MAX_RESPONSE_SIZE {
        return Err("La respuesta del asistente excede el límite permitido".to_string());
    }
    let data = serde_json::from_slice(&bytes)
        .map_err(|error| format!("El asistente devolvió JSON inválido: {error}"))?;
    Ok(LocalResponse { status, data })
}

#[tauri::command]
fn disconnect_router(state: State<'_, ServiceState>) {
    stop_services(&state);
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn validate_host_accepts_private_ipv4() {
        assert_eq!(validate_host("192.168.50.1"), Ok("192.168.50.1".to_string()));
    }

    #[test]
    fn validate_host_rejects_public_ipv4() {
        assert!(validate_host("8.8.8.8").is_err());
    }

    #[test]
    fn validate_device_rejects_empty_identity() {
        let response = LocalResponse {
            status: 200,
            data: json!({
                "authenticated": "unknown",
                "vendor": "",
                "model": "",
                "hardwareVersion": {"value": ""},
                "firmwareVersion": {"value": ""}
            }),
        };
        assert!(validate_device(&response).is_err());
    }

    #[test]
    fn validate_device_accepts_verified_identity() {
        let response = LocalResponse {
            status: 200,
            data: json!({
                "authenticated": "true",
                "vendor": "TP-Link",
                "model": "TL-WR841N/ND",
                "hardwareVersion": {"value": "hardware"},
                "firmwareVersion": {"value": "firmware"}
            }),
        };
        assert!(validate_device(&response).is_ok());
    }

    #[test]
    fn validate_device_accepts_universal_models() {
        let response = LocalResponse {
            status: 200,
            data: json!({
                "authenticated": "true",
                "vendor": "Sercomm",
                "model": "IP3442M-L/US",
                "hardwareVersion": {"value": "hardware"},
                "firmwareVersion": {"value": "firmware"}
            }),
        };
        assert!(validate_device(&response).is_ok());
    }

    #[test]
    fn validate_device_rejects_unauthenticated() {
        let response = LocalResponse {
            status: 200,
            data: json!({
                "authenticated": "false",
                "vendor": "TP-Link",
                "model": "TL-WR841N/ND",
                "hardwareVersion": {"value": "hardware"},
                "firmwareVersion": {"value": "firmware"}
            }),
        };
        assert!(validate_device(&response).is_err());
    }
}

/// Inicia la aplicación de escritorio y registra exclusivamente los comandos locales permitidos.
#[cfg_attr(mobile, tauri::mobile_entry_point)]
pub fn run() {
    tauri::Builder::default()
        .plugin(tauri_plugin_shell::init())
        .plugin(
            tauri_plugin_log::Builder::default()
                .level(log::LevelFilter::Info)
                .build(),
        )
        .manage(ServiceState::default())
        .invoke_handler(tauri::generate_handler![
            detect_gateway,
            connect_router,
            router_get,
            assistant_chat,
            disconnect_router
        ])
        .on_window_event(|window, event| {
            if matches!(event, tauri::WindowEvent::CloseRequested { .. }) {
                let state = window.state::<ServiceState>();
                stop_services(&state);
            }
        })
        .run(tauri::generate_context!())
        .expect("no se pudo ejecutar la aplicación de escritorio");
}
