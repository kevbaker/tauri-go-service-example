mod service;

use service::{ServiceRequest, ServiceState};
use tauri::Manager;

#[tauri::command]
async fn service_invoke(
    app: tauri::AppHandle,
    state: tauri::State<'_, ServiceState>,
    request: ServiceRequest,
) -> Result<serde_json::Value, String> {
    state.invoke(&app, request).await
}

#[cfg_attr(mobile, tauri::mobile_entry_point)]
pub fn run() {
    let application = tauri::Builder::default()
        .plugin(tauri_plugin_opener::init())
        .plugin(tauri_plugin_shell::init())
        .manage(ServiceState::new())
        .invoke_handler(tauri::generate_handler![service_invoke])
        .build(tauri::generate_context!())
        .expect("error while building Tauri application");

    application.run(|app, event| {
        if matches!(event, tauri::RunEvent::Exit) {
            tauri::async_runtime::block_on(app.state::<ServiceState>().shutdown());
        }
    });
}
