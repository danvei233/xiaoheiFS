# Cloud VPS Console Backend

## Run

```bash
go run ./cmd/server
```

## Config file (YAML)

Backend supports a local YAML config file for basic settings (bind address, site domain, DB, etc.).

Load order (later wins):
1) Defaults
2) Config file (`APP_CONFIG_PATH`, or `app.config.yaml`, then `app.config.yml`, then legacy `app.config.json`)
3) Environment variables

Example `app.config.yaml`:

```yaml
addr: ":8080"
api_base_url: "http://localhost:8080"
site:
  name: "小黑云控制台"
  url: "https://example.com"
db:
  type: "sqlite"
  path: "./data/app.db"
  dsn: ""
admin:
  user: "admin"
  pass: "admin123"
jwt_secret: "dev_secret"
automation:
  base_url: "https://idc.example.com/index.php/api/cloud"
  api_key: "xxx"
```

Environment variables:
- `APP_ADDR` (default `:8080`)
- `APP_DB_TYPE` (default `sqlite`)
- `APP_DB_PATH` (default `./data/app.db`)
- `APP_DB_DSN` (default empty)
- `SITE_NAME` (default empty; also written into DB setting `site_name` on startup)
- `SITE_URL` (default empty; also written into DB setting `site_url` on startup)
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
