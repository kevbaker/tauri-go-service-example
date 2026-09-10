use serde::{Deserialize, Serialize};
use serde_json::Value;
use std::{fs, time::Duration};
use tauri::{AppHandle, Manager};
use tauri_plugin_shell::{
    process::{CommandChild, CommandEvent},
    ShellExt,
};
use tokio::sync::{oneshot, Mutex};

const PROTOCOL_VERSION: u64 = 1;
const MAX_REQUEST_BYTES: usize = 64 << 10;
const MAX_RESPONSE_BYTES: u64 = 512 << 10;
const STARTUP_TIMEOUT: Duration = Duration::from_secs(10);
const REQUEST_TIMEOUT: Duration = Duration::from_secs(12);
const SHUTDOWN_TIMEOUT: Duration = Duration::from_secs(6);

#[derive(Clone, Debug, Deserialize, Serialize)]
#[serde(rename_all = "camelCase", deny_unknown_fields)]
pub(crate) struct ServiceRequest {
    protocol_version: u64,
    request_id: String,
    operation: String,
    #[serde(default)]
    payload: Value,
}

#[derive(Debug, Deserialize)]
#[serde(rename_all = "camelCase")]
struct Readiness {
    address: String,
    protocol_version: u64,
}

struct RunningService {
    address: String,
    token: String,
    child: CommandChild,
    terminated: oneshot::Receiver<()>,
}

pub(crate) struct ServiceState {
    service: Mutex<Option<RunningService>>,
    http: reqwest::Client,
}

impl ServiceState {
    pub(crate) fn new() -> Self {
        Self {
            service: Mutex::new(None),
            http: reqwest::Client::builder()
                .timeout(REQUEST_TIMEOUT)
                .build()
                .expect("build loopback HTTP client"),
        }
    }

    pub(crate) async fn invoke(
        &self,
        app: &AppHandle,
        request: ServiceRequest,
    ) -> Result<Value, String> {
        validate_request(&request)?;
        let encoded = serde_json::to_vec(&request)
            .map_err(|_| "PROTOCOL_ERROR: request could not be encoded".to_owned())?;
        if encoded.len() > MAX_REQUEST_BYTES {
            return Err("PROTOCOL_ERROR: request is too large".to_owned());
        }

        let (address, token) = self.connection(app).await?;
        let mut response = self
            .http
            .post(format!("http://{address}/v1/invoke"))
            .bearer_auth(token)
            .header(reqwest::header::CONTENT_TYPE, "application/json")
            .body(encoded)
            .send()
            .await
            .map_err(|_| "SERVICE_UNAVAILABLE: task service request failed".to_owned())?;

        if !response.status().is_success() {
            return Err("SERVICE_UNAVAILABLE: task service rejected the request".to_owned());
        }
        if response
            .content_length()
            .is_some_and(|size| size > MAX_RESPONSE_BYTES)
        {
            return Err("PROTOCOL_ERROR: task service response is too large".to_owned());
        }
        let mut response_bytes = Vec::new();
        while let Some(chunk) = response
            .chunk()
            .await
            .map_err(|_| "PROTOCOL_ERROR: task service response could not be read".to_owned())?
        {
            if response_bytes.len().saturating_add(chunk.len()) > MAX_RESPONSE_BYTES as usize {
                return Err("PROTOCOL_ERROR: task service response is too large".to_owned());
            }
            response_bytes.extend_from_slice(&chunk);
        }
        let response_value: Value = serde_json::from_slice(&response_bytes)
            .map_err(|_| "PROTOCOL_ERROR: task service returned invalid JSON".to_owned())?;
        validate_response(&response_value, &request.request_id)?;
        Ok(response_value)
    }

    async fn connection(&self, app: &AppHandle) -> Result<(String, String), String> {
        let mut service = self.service.lock().await;
        if service.is_none() {
            *service = Some(start_sidecar(app).await?);
        }
        let running = service.as_ref().expect("service was initialized");
        Ok((running.address.clone(), running.token.clone()))
    }

    pub(crate) async fn shutdown(&self) {
        let Some(mut running) = self.service.lock().await.take() else {
            return;
        };
        let shutdown_request = self
            .http
            .post(format!("http://{}/v1/shutdown", running.address))
            .bearer_auth(&running.token)
            .send();
        let _ = tokio::time::timeout(Duration::from_millis(500), shutdown_request).await;
        if tokio::time::timeout(SHUTDOWN_TIMEOUT, &mut running.terminated)
            .await
            .is_err()
        {
            let _ = running.child.kill();
        }
    }
}

async fn start_sidecar(app: &AppHandle) -> Result<RunningService, String> {
    let app_data = app
        .path()
        .app_data_dir()
        .map_err(|_| "SERVICE_UNAVAILABLE: application data directory is unavailable".to_owned())?;
    fs::create_dir_all(&app_data).map_err(|_| {
        "SERVICE_UNAVAILABLE: application data directory could not be created".to_owned()
    })?;
    let database_path = app_data.join("tasks.db");
    let token = uuid::Uuid::new_v4().to_string();
    let command = app
        .shell()
        .sidecar("task-service")
        .map_err(|_| "SERVICE_UNAVAILABLE: bundled task service was not found".to_owned())?
        .args([
            "--mode",
            "desktop",
            "--host",
            "127.0.0.1",
            "--port",
            "0",
            "--database-path",
            &database_path.to_string_lossy(),
        ])
        .env("TGS_SERVICE_TOKEN", &token);

    let (mut events, child) = command
        .spawn()
        .map_err(|_| "SERVICE_UNAVAILABLE: task service could not be started".to_owned())?;

    let readiness_result = tokio::time::timeout(STARTUP_TIMEOUT, async {
        let mut stdout = Vec::new();
        while let Some(event) = events.recv().await {
            match event {
                CommandEvent::Stdout(bytes) => {
                    stdout.extend(bytes);
                    if let Some(end) = stdout.iter().position(|byte| *byte == b'\n') {
                        return serde_json::from_slice::<Readiness>(&stdout[..end]).map_err(|_| {
                            "SERVICE_UNAVAILABLE: task service readiness was invalid".to_owned()
                        });
                    }
                }
                CommandEvent::Stderr(bytes) => {
                    eprintln!("{}", String::from_utf8_lossy(&bytes));
                }
                CommandEvent::Error(message) => {
                    eprintln!("task-service sidecar error: {message}");
                }
                CommandEvent::Terminated(_) => {
                    return Err(
                        "SERVICE_UNAVAILABLE: task service stopped during startup".to_owned()
                    );
                }
                _ => {}
            }
        }
        Err("SERVICE_UNAVAILABLE: task service readiness stream closed".to_owned())
    })
    .await;
    let readiness = match readiness_result {
        Ok(Ok(readiness)) => readiness,
        Ok(Err(error)) => {
            let _ = child.kill();
            return Err(error);
        }
        Err(_) => {
            let _ = child.kill();
            return Err("SERVICE_UNAVAILABLE: task service startup timed out".to_owned());
        }
    };

    if readiness.protocol_version != PROTOCOL_VERSION || !is_loopback_address(&readiness.address) {
        let _ = child.kill();
        return Err("SERVICE_UNAVAILABLE: task service readiness was rejected".to_owned());
    }

    let (terminated_tx, terminated_rx) = oneshot::channel();
    tauri::async_runtime::spawn(async move {
        while let Some(event) = events.recv().await {
            match event {
                CommandEvent::Stderr(bytes) => eprintln!("{}", String::from_utf8_lossy(&bytes)),
                CommandEvent::Error(message) => eprintln!("task-service sidecar error: {message}"),
                CommandEvent::Terminated(payload) => {
                    eprintln!("task-service sidecar stopped: {payload:?}");
                    break;
                }
                _ => {}
            }
        }
        let _ = terminated_tx.send(());
    });

    Ok(RunningService {
        address: readiness.address,
        token,
        child,
        terminated: terminated_rx,
    })
}

fn validate_request(request: &ServiceRequest) -> Result<(), String> {
    if request.protocol_version != PROTOCOL_VERSION {
        return Err("PROTOCOL_ERROR: unsupported protocol version".to_owned());
    }
    if request.request_id.is_empty() || request.request_id.len() > 128 {
        return Err("PROTOCOL_ERROR: invalid request ID".to_owned());
    }
    if !matches!(
        request.operation.as_str(),
        "config.getPublic"
            | "tasks.list"
            | "tasks.get"
            | "tasks.create"
            | "tasks.update"
            | "tasks.delete"
    ) {
        return Err("PROTOCOL_ERROR: operation is not allowed".to_owned());
    }
    Ok(())
}

fn validate_response(response: &Value, request_id: &str) -> Result<(), String> {
    let Some(object) = response.as_object() else {
        return Err("PROTOCOL_ERROR: task service returned an invalid envelope".to_owned());
    };
    if object.get("protocolVersion").and_then(Value::as_u64) != Some(PROTOCOL_VERSION)
        || object.get("requestId").and_then(Value::as_str) != Some(request_id)
        || object.get("ok").and_then(Value::as_bool).is_none()
    {
        return Err("PROTOCOL_ERROR: task service response did not match the request".to_owned());
    }
    Ok(())
}

fn is_loopback_address(address: &str) -> bool {
    address
        .parse::<std::net::SocketAddr>()
        .is_ok_and(|socket| socket.ip().is_loopback())
}

#[cfg(test)]
mod tests {
    use super::{is_loopback_address, validate_request, validate_response, ServiceRequest};
    use serde_json::json;

    fn request(operation: &str) -> ServiceRequest {
        ServiceRequest {
            protocol_version: 1,
            request_id: "request-1".to_owned(),
            operation: operation.to_owned(),
            payload: json!({}),
        }
    }

    #[test]
    fn accepts_only_application_operations() {
        assert!(validate_request(&request("config.getPublic")).is_ok());
        assert!(validate_request(&request("tasks.create")).is_ok());
        assert!(validate_request(&request("system.shell")).is_err());
    }

    #[test]
    fn requires_matching_response_envelopes() {
        assert!(validate_response(
            &json!({"protocolVersion": 1, "requestId": "request-1", "ok": true}),
            "request-1"
        )
        .is_ok());
        assert!(validate_response(
            &json!({"protocolVersion": 1, "requestId": "wrong", "ok": true}),
            "request-1"
        )
        .is_err());
    }

    #[test]
    fn accepts_only_loopback_readiness_addresses() {
        assert!(is_loopback_address("127.0.0.1:49152"));
        assert!(is_loopback_address("[::1]:49152"));
        assert!(!is_loopback_address("0.0.0.0:49152"));
    }
}
