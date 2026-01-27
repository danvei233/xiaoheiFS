# Realname providers

The realname module uses a provider interface and registry:
- Interface: `backend/internal/usecase/ports.go` (RealNameProvider)
- Registry: `backend/internal/adapter/realname/registry.go`

## Built-in provider
- `idcard_cn`: China ID card format checks.

## Demo provider
The repository includes a demo provider source at:
- `backend/pkg/realname_demo`

This demo does not register automatically. It shows how to implement and register a
provider in code.

Build/run example:
```
go run ./pkg/realname_demo
```

To add a new provider in the backend:
1) Create a provider type implementing `RealNameProvider`.
2) Register it in `realname.NewRegistry()` (see `backend/cmd/server/main.go`).
3) Set `realname_provider` in settings to the provider key.
