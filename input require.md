# Input Requirement Standard

## Scope
- Backend: `./backend`
- Frontend: `./frontend`
- Goal: all user-input text fields must have unified length limits to prevent oversized payload abuse.

## Global Rules
- All text input must be validated on backend; frontend `maxlength` is only UX assistance.
- Length unit: Unicode rune count (not bytes).
- Trim leading/trailing spaces before validation where business allows.
- Empty-string handling follows field semantics (required vs optional).
- Return consistent error style: `字段名长度不能超过 N 个字符`.

## Field Standards

### User/Admin Info
- `email`: max 120
- `nickname`: max 32
- `qq`: max 20
- `signature` / `bio` / `intro`: max 500
- `password`: max 128 (min length still follows existing policy)
- `avatar_url`: max 500
- `phone`: max 32

### Ticket (expanded x2)
- `title` / `subject`: max 240
- `content`: max 10000
- `resource_name`: max 200

### Payment / Approval
- `approval` (review remark/reason): max 500
- `method`: max 50
- `note`: max 500
- `screenshot_url`: max 500

### Refund
- `reason` / `refund_reason`: max 500

### VPS Management
- `reset password`: max 128
- `reinstall password`: max 128
- `port mapping name`: max 100
- `vps display/name`: max 100

## Implementation Baseline

### Backend (already applied)
- Unified constants and validators:
  - `backend/internal/usecase/input_limits.go`
- Applied in usecases:
  - `auth_service.go`
  - `admin_service.go`
  - `ticket_service.go`
  - `order_service.go`
  - `wallet_order_service.go`
  - `vps_service.go`
  - `password_reset_service.go`
- DB migration column-size alignment:
  - `backend/internal/adapter/repo/migrate_gorm.go`

### Frontend (already applied)
- Shared constants:
  - `frontend/src/constants/inputLimits.ts`
- Added `maxlength` and submit-time checks in auth/profile/ticket/order/vps/admin pages.

## Acceptance Criteria
- Oversized input is rejected by backend with clear message.
- Frontend input controls block overlong typing where applicable.
- Ticket limits are expanded to:
  - subject 240
  - content 10000
- Backend/frontend limits are consistent.

## Change Control
- New text field must be added to both:
  - backend `input_limits.go`
  - frontend `inputLimits.ts`
- If DB column size is lower than app limit, migration must be updated first.
