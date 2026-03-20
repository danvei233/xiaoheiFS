# Cloud VPS Console Backend

## Run

```bash
go run ./cmd/server
```

## Config file (YAML)

Backend supports a local YAML config file for basic settings (bind address, site domain, DB, etc.).

Load order (later wins):

1) Defaults
2) Config file (`app.config.yaml`, then `app.config.yml`, then legacy `app.config.json`)

Example `app.config.yaml`:

```yaml
addr: ":8080"
api_base_url: "http://localhost:8080"
db:
  type: "sqlite"
  path: "./data/app.db"
  dsn: ""
```

Notes:

- The config file is for non-sensitive runtime settings (bind address/db/etc.).
- `jwt_secret` is auto-generated on first run if missing, and written into the config file.
- The installer (`POST /api/v1/install`) creates the first admin user in the database (password stored as a bcrypt hash); admin credentials are not stored in the config file.
- Integration settings (e.g. automation base URL / API key) are stored in the database `settings` table and can be managed from the admin APIs/UI.

## Docs

```bash
go run ./cmd/tools/gendocs
```

This updates `docs/openapi.yaml` and `docs/api.md`.

Plugin system (go-plugin GRPC + protobuf) migration notes: `docs/plugin-system-migration.md`.

## Examples

See `examples/curl.md`.
