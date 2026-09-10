# Tauri Rust Project

This is the root of the standard Cargo project used by Tauri. It contains `Cargo.toml`, `build.rs`, `tauri.conf.json`, Rust source, capabilities, and generated icons. `Cargo.lock` is created and committed when Cargo first resolves the project.

Rust is limited to Tauri setup, bridge validation, authenticated proxying, and Go sidecar lifecycle. `service_invoke` is the Electron `ipcMain.handle` equivalent: it accepts only the task envelope, keeps the sidecar token and address private, streams responses into a bounded buffer, and forwards to Go. On exit, Rust waits for the service's bounded graceful shutdown and only forces termination after that deadline. The webview runs under an explicit restrictive content security policy. See the repository's [Tauri patterns](../../../.agents/patterns/tauri.md) and [architecture explanation](../../../ARCHITECTURE-EXPLAINED.md).
