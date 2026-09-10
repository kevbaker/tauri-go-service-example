# Configuration

Shared, non-secret configuration files and the canonical configuration schema belong here.

The Go service loads and validates the complete configuration. The frontend receives only an explicitly allowlisted, read-only subset through the typed application client. Local overrides and secrets must remain outside version control.

Precedence is built-in defaults, the selected YAML file, `TGS_` environment values, then explicit command-line overrides. Desktop authentication comes only from `TGS_SERVICE_TOKEN`; remote-development authentication comes only from `TGS_REMOTE_ACCESS_TOKEN`. Neither token is represented in the shared files or public configuration.
