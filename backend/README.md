# Cloud VPS Console Backend

## Run

```bash
go run ./cmd/server
```

Environment variables:
- `APP_ADDR` (default `:8080`)
- `APP_DB_PATH` (default `./data/app.db`)
- `ADMIN_USER` (default `admin`)
- `ADMIN_PASS` (default `admin123`)
- `ADMIN_JWT_SECRET` (default `dev_secret`)
- `API_BASE_URL` (default `http://localhost:8080`)
- `AUTOMATION_BASE_URL` (default `https://idc.duncai.top/index.php/api/cloud`)
- `AUTOMATION_API_KEY` (required for automation calls)

## Docs

```bash
go run ./cmd/tools/gendocs
```

This updates `docs/openapi.yaml` and `docs/api.md`.

## Examples

See `examples/curl.md`.
