# Adapter Layout

This directory follows a split-by-responsibility structure:

- `plugins/core`: plugin runtime manager and lifecycle wiring.
- `plugins/automation`: automation plugin adapter client/resolver.
- `plugins/payment`: payment provider RPC plugin contract.
- `plugins/realname`: real-name (KYC) plugin provider adapter.
- `repo/core`: repository implementations and DB migration wiring.
- `http`: HTTP transport adapter.

Deprecated locations (must stay empty of `.go` files):

- `automation/`
- `payment/plugin/`
