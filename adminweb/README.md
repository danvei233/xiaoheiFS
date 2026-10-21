# adminweb

Standalone admin web project built with Vue 3, Vite, and Element Plus.

## Requirements

- Node.js `>=20.19.0`
- pnpm `>=8.8.0`

## Local Development

```bash
cd adminweb
pnpm install
pnpm dev
```

Vite proxies `/api` and `/admin/api` to `VITE_API_PROXY_URL`.

## Build

```bash
cd adminweb
pnpm build
```

Production builds output to the repository root `static-admin/` directory by default.
Set `VITE_BUILD_OUT_DIR` to override it.

## Common Commands

```bash
pnpm lint
pnpm serve
```
