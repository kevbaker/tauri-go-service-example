# Rust Source

Tauri's generated desktop entrypoint belongs in `main.rs`; keep it thin. Application setup, commands, and the mobile-compatible entrypoint belong in `lib.rs` and focused modules called from it.
