# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Cloud VPS Console (小黑云控制台) - A Vue 3 full-stack web application for managing cloud VPS services with dual interfaces:
- **User Console** (`/console`) - Customer-facing VPS purchasing and management
- **Admin Console** (`/admin`) - Operations staff interface for system management

## Development Commands

```bash
# Install dependencies
npm i

# Development server (runs on port 5173)
npm run dev

# Production build
npm run build

# Preview production build
npm run preview
```

## API Proxy Configuration

The development server proxies API requests to the backend (Go) at `http://localhost:8080`:
- `/api` → `http://localhost:8080`
- `/admin/api` → `http://localhost:8080`
- `/sdk` → `http://localhost:8080`

To connect directly to a backend, set:
```bash
VITE_API_BASE=http://localhost:8080
```

## Architecture

### Directory Structure

```
src/
├── components/       # Reusable UI components (ProTable, StatusTag, Charts, etc.)
├── layouts/          # UserLayout.vue and AdminLayout.vue
├── pages/
│   ├── auth/         # Login/Register pages
│   ├── console/      # User-facing pages
│   └── admin/        # Admin pages
├── router/           # Vue Router with auth guards
├── services/         # API layer (http.ts, user.ts, admin.ts, types.ts, sse.ts)
├── stores/           # Pinia state management
├── styles/           # Global theme.css with CSS variables
├── App.vue           # Root component with route transitions
└── main.ts           # Entry point
```

### Key Integration Files

- **User API**: `src/services/user.ts`
- **Admin API**: `src/services/admin.ts`
- **HTTP Client**: `src/services/http.ts` - Axios instance with interceptors
- **SSE**: `src/services/sse.ts` - Server-Sent Events for real-time order updates
- **Types**: `src/services/types.ts` - All TypeScript interfaces

### State Management (Pinia)

All stores use `defineStore` with Options API. Key stores:
- `auth.ts` / `adminAuth.ts` - User/Admin authentication (tokens in localStorage)
- `cart.ts` - Shopping cart state
- `catalog.ts` - Product catalog
- `orders.ts` - Orders with SSE event streaming
- `vps.ts` - VPS instances
- `app.ts` - API keys configuration

### API Layer Patterns

- Centralized Axios instance in `services/http.ts`
- Automatic Bearer token injection based on URL prefix (`/api` or `/admin/api`)
- API Key support via `X-API-Key` header
- Automatic 401 handling with redirect to login
- Error messaging via Ant Design message/notification

### Backend Compatibility

The frontend supports field name variations from the Go backend:
- Both camelCase and PascalCase (e.g., `id`/`ID`, `name`/`Name`)

### Routing

- Route guards for user and admin authentication
- Lazy-loaded page components
- Meta fields: `requiresUser`, `requiresAdmin`

## Component Patterns

- Vue 3 Composition API with `<script setup>`
- Scoped CSS with CSS variables for theming
- Ant Design components for UI
- Custom components: ProTable (column visibility), StatusTag, ConfirmAction, FilterBar

## Key Settings Keys

- `robot_webhook_url`, `robot_webhook_key`
- `smtp_host`, `smtp_port`, `smtp_user`, `smtp_pass`, `smtp_from`
- `email_enabled`, `email_expire_enabled`
- `expire_reminder_days`
- `emergency_renew_days`, `emergency_renew_interval_hours`

## API Endpoints

**User API** (`/api/v1/`):
- Auth: `captcha`, `auth/register`, `auth/login`, `me`
- Dashboard, catalog, cart, orders
- VPS: `vps`, `vps/{id}/*`, `vps/{id}/renew`, `vps/{id}/resize`
- Order events: `orders/{id}/events` (SSE)

**Admin API** (`/admin/api/v1/`):
- Auth: `auth/login`
- Users, orders (approve/reject/retry)
- VPS (lock/unlock/delete/resize/refresh/status/emergency-renew)
- Catalog: regions, plan-groups, packages, system-images
- Settings, API keys, email templates, audit logs

## Tech Stack

- Vue 3.4.21 + Vite 5.2.6 + TypeScript 5.4.3
- Vue Router 4.3.0 + Pinia 2.1.7
- Ant Design Vue 4.2.1
- Axios 1.6.8
- ECharts 5.5.0
- Day.js 1.11.10
- CKEditor 5 / TinyMCE (rich text editors)