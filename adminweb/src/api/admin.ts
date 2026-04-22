import request from '@/utils/http'

export interface ApiList<T> {
  items?: T[]
  total?: number
}

export interface AdminLoginParams {
  username: string
  password: string
  admin_path?: string
}

export interface AdminLoginResponse {
  access_token?: string
  refresh_token?: string
  expires_in?: number
  totp_enabled?: boolean
  mfa_required?: boolean
  mfa_bind_required?: boolean
  mfa_unlocked?: boolean
  user?: {
    id?: number
    username?: string
    role?: string
  }
}

export interface Admin2FASetupResponse {
  secret?: string
  otpauth_url?: string
}

export interface Admin2FAUnlockResponse {
  access_token?: string
  refresh_token?: string
  expires_in?: number
  mfa_unlocked?: boolean
}

export interface AdminProfile {
  id?: number
  username?: string
  email?: string
  qq?: string
  avatar?: string
  avatar_url?: string
  permission_group_id?: number | null
  permission_group_name?: string
  permissions?: string[]
  status?: string
  role?: string
  created_at?: string
  updated_at?: string
}

export interface AdminAuthUserInfo {
  buttons: string[]
  roles: string[]
  permissions: string[]
  userId: number
  userName: string
  email: string
  avatar?: string
  qq?: string
  permissionGroupName?: string
  status?: string
}

export interface AdminUserSummary {
  id?: number
  username?: string
  role?: string
}

export interface UserRecord {
  id?: number
  username?: string
  email?: string
  qq?: string
  phone?: string
  bio?: string
  intro?: string
  avatar?: string
  avatar_url?: string
  role?: string
  status?: string
  created_at?: string
  updated_at?: string
  permission_group_id?: number | null
  permissions?: string[]
  user_tier_group_id?: number | null
  user_tier_expire_at?: string | null
}

export interface UserTierGroup {
  id?: number
  name?: string
  color?: string
  icon?: string
  priority?: number
  auto_approve_enabled?: boolean
  is_default?: boolean
  created_at?: string
  updated_at?: string
}

export interface UserTierDiscountRule {
  id?: number
  group_id?: number
  scope?: string
  goods_type_id?: number
  region_id?: number
  plan_group_id?: number
  package_id?: number
  discount_permille?: number
  fixed_price?: number | null
  add_core_permille?: number
  add_mem_permille?: number
  add_disk_permille?: number
  add_bw_permille?: number
}

export interface UserTierAutoRule {
  id?: number
  group_id?: number
  duration_days?: number
  conditions_json?: string
  sort_order?: number
}

export interface CouponProductRule {
  scope?: string
  goods_type_id?: number
  region_id?: number
  plan_group_id?: number
  package_id?: number
  addon_core_enabled?: boolean
  addon_mem_enabled?: boolean
  addon_disk_enabled?: boolean
  addon_bw_enabled?: boolean
}

export interface CouponProductGroupRecord {
  id?: number
  name?: string
  rules?: CouponProductRule[]
  scope?: string
  goods_type_id?: number
  region_id?: number
  plan_group_id?: number
  package_id?: number
  addon_core?: number
  addon_mem_gb?: number
  addon_disk_gb?: number
  addon_bw_mbps?: number
}

export interface CouponRecord {
  id?: number
  code?: string
  discount_permille?: number
  product_group_id?: number
  total_limit?: number
  per_user_limit?: number
  starts_at?: string | null
  ends_at?: string | null
  new_user_only?: boolean
  active?: boolean
  note?: string
  created_at?: string
  updated_at?: string
}

export interface WalletInfo {
  balance?: number
  currency?: string
}

export interface WalletTransaction {
  id?: number
  type?: string
  amount?: number
  note?: string
  created_at?: string
}

export interface WalletOrderRecord {
  id?: number
  user_id?: number
  type?: string
  amount?: number
  currency?: string
  status?: string
  note?: string
  meta?: Record<string, unknown>
  reviewed_by?: number | null
  review_reason?: string
  created_at?: string
  updated_at?: string
}

export interface OrderRecord {
  id?: number
  user_id?: number
  order_no?: string
  source?: string
  status?: string
  can_review?: boolean
  total_amount?: number
  currency?: string
  coupon_id?: number | null
  coupon_code?: string
  coupon_discount?: number
  idempotency_key?: string
  pending_reason?: string
  approved_by?: number | null
  approved_at?: string | null
  rejected_reason?: string
  created_at?: string
  updated_at?: string
}

export interface OrderItemRecord {
  id?: number
  order_id?: number
  package_id?: number
  system_id?: number
  spec?: Record<string, unknown>
  qty?: number
  amount?: number
  status?: string
  automation_instance_id?: string
  action?: string
  duration_months?: number
  created_at?: string
  updated_at?: string
}

export interface OrderPaymentRecord {
  id?: number
  order_id?: number
  user_id?: number
  method?: string
  amount?: number
  currency?: string
  trade_no?: string
  note?: string
  screenshot_url?: string
  status?: string
  idempotency_key?: string
  reviewed_by?: number | null
  review_reason?: string
  created_at?: string
  updated_at?: string
}

export interface OrderEventRecord {
  id?: number
  order_id?: number
  seq?: number
  type?: string
  data?: Record<string, unknown> | string
  created_at?: string
}

export interface OrderDetailResponse {
  order?: OrderRecord
  items?: OrderItemRecord[]
  payments?: OrderPaymentRecord[]
  events?: OrderEventRecord[]
}

export interface CatalogGoodsType {
  id?: number
  code?: string
  name?: string
  active?: boolean
  sort_order?: number
  automation_plugin_id?: string
  automation_instance_id?: string
}

export interface CatalogRegion {
  id?: number
  goods_type_id?: number
  name?: string
  code?: string
  active?: boolean
  visible?: boolean
  sort_order?: number
}

export interface CatalogPlanGroup {
  id?: number
  goods_type_id?: number
  region_id?: number
  line_id?: number
  name?: string
  unit_core?: number
  unit_mem?: number
  unit_disk?: number
  unit_bw?: number
  add_core_min?: number
  add_core_max?: number
  add_core_step?: number
  add_mem_min?: number
  add_mem_max?: number
  add_mem_step?: number
  add_disk_min?: number
  add_disk_max?: number
  add_disk_step?: number
  add_bw_min?: number
  add_bw_max?: number
  add_bw_step?: number
  active?: boolean
  visible?: boolean
  capacity_remaining?: number
  sort_order?: number
}

export interface CatalogPackage {
  id?: number
  goods_type_id?: number
  plan_group_id?: number
  product_id?: number
  integration_package_id?: number
  name?: string
  cores?: number
  memory_gb?: number
  disk_gb?: number
  bandwidth_mbps?: number
  cpu_model?: string
  port_num?: number
  monthly_price?: number
  active?: boolean
  visible?: boolean
  capacity_remaining?: number
  sort_order?: number
}

export interface CatalogSystemImage {
  id?: number
  image_id?: number
  name?: string
  type?: string
  enabled?: boolean
}

export interface CatalogBillingCycle {
  id?: number
  name?: string
  months?: number
  multiplier?: number
  min_qty?: number
  max_qty?: number
  active?: boolean
  sort_order?: number
  created_at?: string
  updated_at?: string
}

export interface CatalogCapabilityRecord {
  goods_type_id?: number
  package_id?: number
  resize_enabled?: boolean
  refund_enabled?: boolean
  resize_source?: string
  refund_source?: string
  goods_type_resize_enabled?: boolean | null
  goods_type_refund_enabled?: boolean | null
  package_resize_enabled?: boolean | null
  package_refund_enabled?: boolean | null
}

export interface VpsRecord {
  id?: number
  user_id?: number
  order_item_id?: number
  goods_type_id?: number
  automation_instance_id?: string
  name?: string
  region?: string
  region_id?: number
  line_id?: number
  package_id?: number
  package_name?: string
  cpu?: number
  memory_gb?: number
  disk_gb?: number
  bandwidth_mbps?: number
  port_num?: number
  monthly_price?: number
  spec?: Record<string, unknown>
  system_id?: number
  status?: string
  automation_state?: number
  admin_status?: string
  expire_at?: string
  destroy_at?: string | null
  destroy_in_days?: number | null
  panel_url_cache?: string
  access_info?: Record<string, unknown>
  last_emergency_renew_at?: string | null
  created_at?: string
  updated_at?: string
}

export interface DashboardOverview {
  total_orders?: number
  pending_review?: number
  revenue?: number
  vps_count?: number
  expiring_soon?: number
}

export interface RevenuePoint {
  date?: string
  amount?: number
}

export interface DashboardRevenue {
  items?: RevenuePoint[]
  points?: RevenuePoint[]
}

export type RevenueAnalyticsLevel = 'overall' | 'goods_type' | 'region' | 'line' | 'package'

export interface RevenueAnalyticsQuery {
  from_at: string
  to_at: string
  level: RevenueAnalyticsLevel
  user_id?: number
  goods_type_id?: number
  region_id?: number
  line_id?: number
  package_id?: number
  page?: number
  page_size?: number
  sort_field?: 'paid_at' | 'amount'
  sort_order?: 'asc' | 'desc'
}

export interface RevenueAnalyticsSummary {
  total_revenue_cents?: number
  order_count?: number
  yoy_ratio?: number | null
  mom_ratio?: number | null
  yoy_comparable?: boolean
  mom_comparable?: boolean
}

export interface RevenueAnalyticsShareItem {
  dimension_id?: number
  dimension_name?: string
  revenue_cents?: number
  ratio?: number
}

export interface RevenueAnalyticsTopItem {
  rank?: number
  dimension_id?: number
  dimension_name?: string
  revenue_cents?: number
  ratio?: number
}

export interface RevenueAnalyticsTrendPoint {
  bucket?: string
  revenue_cents?: number
  order_count?: number
}

export interface RevenueAnalyticsDetailRecord {
  payment_id?: number
  order_id?: number
  order_no?: string
  user_id?: number
  goods_type_id?: number
  region_id?: number
  line_id?: number
  package_id?: number
  amount_cents?: number
  paid_at?: string
  status?: string
}

export interface RevenueAnalyticsOverviewResponse {
  summary?: RevenueAnalyticsSummary
  share_items?: RevenueAnalyticsShareItem[]
  top_items?: RevenueAnalyticsTopItem[]
}

export interface RevenueAnalyticsDetailsResponse {
  items?: RevenueAnalyticsDetailRecord[]
  page?: number
  page_size?: number
  total?: number
}

export interface StatusPoint {
  status?: string
  count?: number
}

export interface DashboardStatus {
  items?: StatusPoint[]
  points?: StatusPoint[]
}

export interface ServerStatus {
  hostname?: string
  os?: string
  platform?: string
  kernel_version?: string
  uptime_seconds?: number
  cpu_model?: string
  cpu_cores?: number
  cpu_usage_percent?: number
  mem_total?: number
  mem_used?: number
  mem_usage_percent?: number
  disk_total?: number
  disk_used?: number
  disk_usage_percent?: number
  status?: string
}

export interface RealNameVerification {
  id?: number
  user_id?: number
  real_name?: string
  id_number?: string
  status?: string
  provider?: string
  reason?: string
  redirect_url?: string
  created_at?: string
  verified_at?: string
}

export interface RealNameConfigRecord {
  enabled?: boolean
  provider?: string
  block_actions?: string[]
}

export interface RealNameProviderRecord {
  key?: string
  name?: string
}

export interface SettingItemRecord {
  key?: string
  value?: string
  created_at?: string
  updated_at?: string
}

export interface CMSCategoryRecord {
  id?: number
  key?: string
  name?: string
  lang?: string
  sort_order?: number
  visible?: boolean
  created_at?: string
  updated_at?: string
}

export interface CMSPostRecord {
  id?: number
  category_id?: number
  title?: string
  slug?: string
  summary?: string
  content_html?: string
  cover_url?: string
  lang?: string
  status?: string
  pinned?: boolean
  sort_order?: number
  published_at?: string | null
  created_at?: string
  updated_at?: string
}

export interface CMSBlockRecord {
  id?: number
  page?: string
  type?: string
  title?: string
  subtitle?: string
  content_json?: string
  custom_html?: string
  lang?: string
  visible?: boolean
  sort_order?: number
  created_at?: string
  updated_at?: string
}

export interface UploadAssetRecord {
  id?: number
  name?: string
  path?: string
  url?: string
  mime?: string
  size?: number
  uploader_id?: number
  created_at?: string
}

export interface AdminPluginRecord {
  category?: string
  plugin_id?: string
  instance_id?: string
  name?: string
  version?: string
  signature_status?: 'official' | 'untrusted' | 'unsigned'
  enabled?: boolean
  loaded?: boolean
  installed_at?: string
  updated_at?: string
  last_health_at?: string | null
  health_status?: string
  health_message?: string
  entry?: {
    platform?: string
    entry_path?: string
    entry_supported?: boolean
    supported_platforms?: string[]
  }
  manifest?: {
    plugin_id?: string
    name?: string
    version?: string
    description?: string
    capabilities?: {
      sms?: { send?: boolean } | null
      payment?: { methods?: string[] } | null
      kyc?: { start?: boolean; query_result?: boolean } | null
      automation?: {
        features?: string[]
        not_supported_reasons?: Record<string, string>
        catalog_readonly?: boolean
      } | null
    }
  }
}

export interface SmsConfigRecord {
  enabled?: boolean
  plugin_id?: string
  instance_id?: string
  default_template_id?: string
  provider_template_id?: string
}

export interface SmsTemplateRecord {
  id?: number
  name?: string
  content?: string
  enabled?: boolean
  created_at?: string
  updated_at?: string
}

export interface SmtpConfigRecord {
  host?: string
  port?: string
  user?: string
  pass?: string
  from?: string
  enabled?: boolean
}

export interface EmailTemplateRecord {
  id?: number
  name?: string
  subject?: string
  body?: string
  enabled?: boolean
  created_at?: string
  updated_at?: string
}

export interface PaymentProviderRecord {
  key?: string
  name?: string
  enabled?: boolean
  order_enabled?: boolean
  wallet_enabled?: boolean
  schema_json?: string
  config_json?: string
  balance?: number
}

export interface PluginPaymentMethodRecord {
  method?: string
  enabled?: boolean
}

export interface PluginDiscoverRecord {
  category?: string
  plugin_id?: string
  name?: string
  version?: string
  signature_status?: 'official' | 'untrusted' | 'unsigned'
  entry?: {
    platform?: string
    entry_path?: string
    entry_supported?: boolean
    supported_platforms?: string[]
  }
}

export interface APIKeyRecord {
  id?: number
  name?: string
  key_hash?: string
  status?: string
  scopes?: string[]
  permission_group_id?: number | null
  created_at?: string
  updated_at?: string
  last_used_at?: string | null
}

export interface PermissionGroupRecord {
  id?: number
  ID?: number
  name?: string
  Name?: string
  description?: string
  Description?: string
  permissions_json?: string
  PermissionsJSON?: string
  created_at?: string
  CreatedAt?: string
  updated_at?: string
  UpdatedAt?: string
}

export interface PermissionRecord {
  code?: string
  name?: string
  friendly_name?: string
  category?: string
  parent_code?: string
  sort_order?: number
  children?: PermissionRecord[]
}

export interface RobotWebhookRecord {
  name?: string
  url?: string
  secret?: string
  enabled?: boolean
  events?: string[]
}

export interface RobotConfigRecord {
  url?: string
  secret?: string
  enabled?: boolean
  webhooks?: RobotWebhookRecord[]
}

export interface DebugStatusResponse {
  enabled?: boolean
}

export interface PluginConfigSchemaResponse {
  json_schema?: string
  ui_schema?: string
}

export interface PluginConfigResponse {
  config_json?: string
}

export interface AdminAuditLogRecord {
  id?: number
  admin_id?: number
  action?: string
  target_type?: string
  target_id?: string
  detail?: Record<string, unknown> | string
  created_at?: string
}

export interface AutomationLogRecord {
  id?: number
  order_id?: number
  order_item_id?: number
  action?: string
  request_json?: unknown
  response_json?: unknown
  success?: boolean
  message?: string
  created_at?: string
}

export interface IntegrationSyncLogRecord {
  id?: number
  target?: string
  mode?: string
  status?: string
  message?: string
  created_at?: string
}

export interface AutomationConfigRecord {
  base_url?: string
  api_key?: string
  enabled?: boolean
  timeout_sec?: number
  retry?: number
  dry_run?: boolean
  configured?: boolean
  compat_mode?: boolean
  plugins_ready?: boolean
  config_source?: string
  plugin_id?: string
  instance_id?: string
  code?: string
  redirect_path?: string
}

export interface DebugLogList<T> {
  items?: T[]
  total?: number
}

export interface DebugLogsResponse {
  audit_logs?: DebugLogList<AdminAuditLogRecord>
  automation_logs?: DebugLogList<AutomationLogRecord>
  sync_logs?: DebugLogList<IntegrationSyncLogRecord>
}

export interface TicketResourceRecord {
  id?: number
  ticket_id?: number
  resource_type?: string
  resource_id?: number
  resource_name?: string
  created_at?: string
}

export interface TicketMessageRecord {
  id?: number
  ticket_id?: number
  sender_id?: number
  sender_role?: string
  sender_name?: string
  sender_qq?: string
  content?: string
  created_at?: string
}

export interface TicketRecord {
  id?: number
  user_id?: number
  subject?: string
  status?: string
  resource_count?: number
  last_reply_at?: string | null
  last_reply_by?: number | null
  last_reply_role?: string
  closed_at?: string | null
  created_at?: string
  updated_at?: string
}

export interface TicketDetailResponse {
  ticket?: TicketRecord
  messages?: TicketMessageRecord[]
  resources?: TicketResourceRecord[]
}

export interface ProbeSnapshotRecord {
  system?: Record<string, unknown>
  cpu?: Record<string, unknown>
  memory?: Record<string, unknown>
  disks?: Array<Record<string, unknown>>
  ports?: Array<Record<string, unknown>>
  raw?: Record<string, unknown>
}

export interface ProbeRecord {
  id?: number
  name?: string
  agent_id?: string
  status?: string
  os_type?: string
  tags?: string[]
  last_heartbeat_at?: string | null
  last_snapshot_at?: string | null
  snapshot?: ProbeSnapshotRecord | null
  created_at?: string
  updated_at?: string
}

export interface ProbeStatusEventRecord {
  id?: number
  probe_id?: number
  status?: string
  at?: string
  reason?: string
  created_at?: string
}

export interface ProbeSlaRecord {
  window_from?: string
  window_to?: string
  total_seconds?: number
  online_seconds?: number
  uptime_percent?: number
  events?: ProbeStatusEventRecord[]
}

export interface ProbeLogSessionResponse {
  session_id?: string
  stream_path?: string
  request_id?: string
  probe_online?: boolean
}

export type ScheduledTaskStrategy = 'interval' | 'daily'

export interface ScheduledTaskRecord {
  key?: string
  name?: string
  description?: string
  enabled?: boolean
  strategy?: ScheduledTaskStrategy | string
  interval_sec?: number
  daily_at?: string
  last_run_at?: string | null
  next_run_at?: string | null
  running?: boolean
  last_status?: string
  last_error?: string
  last_elapsed_sec?: number
}

export interface ScheduledTaskRunRecord {
  id?: number
  task_key?: string
  status?: string
  started_at?: string
  finished_at?: string | null
  duration_sec?: number
  message?: string
  created_at?: string
  ID?: number
  TaskKey?: string
  Status?: string
  StartedAt?: string
  FinishedAt?: string | null
  DurationSec?: number
  Message?: string
  CreatedAt?: string
}

export interface ImpersonateResponse {
  access_token?: string
  expires_in?: number
  user?: AdminUserSummary
}

export function hasAdminPermission(
  grantedPermissions: string[] | undefined,
  requiredPermissions: string | string[]
): boolean {
  const granted = Array.isArray(grantedPermissions) ? grantedPermissions : []
  const required = Array.isArray(requiredPermissions) ? requiredPermissions : [requiredPermissions]

  if (!required.length) {
    return true
  }

  return required.some((permission) =>
    granted.some((grantedPermission) => {
      if (!grantedPermission) {
        return false
      }

      if (grantedPermission === '*' || grantedPermission === permission) {
        return true
      }

      if (grantedPermission.endsWith('*')) {
        return permission.startsWith(grantedPermission.slice(0, -1))
      }

      return false
    })
  )
}

export function mapAdminProfileToUserInfo(profile: AdminProfile): AdminAuthUserInfo {
  const permissions = Array.isArray(profile.permissions) ? profile.permissions.filter(Boolean) : []
  const roles = hasAdminPermission(permissions, '*') ? ['R_SUPER'] : ['R_ADMIN']

  return {
    buttons: [...permissions],
    roles,
    permissions,
    userId: Number(profile.id || 0),
    userName: String(profile.username || '管理员'),
    email: String(profile.email || ''),
    avatar: String(profile.avatar || profile.avatar_url || ''),
    qq: String(profile.qq || ''),
    permissionGroupName: String(profile.permission_group_name || ''),
    status: String(profile.status || '')
  }
}

export function fetchAdminLogin(data: AdminLoginParams) {
  return request.post<AdminLoginResponse>({
    url: '/admin/api/v1/auth/login',
    data
  })
}

export function setupAdminTwoFactor(data: { password?: string; current_code?: string }) {
  return request.post<Admin2FASetupResponse>({
    url: '/admin/api/v1/auth/2fa/setup',
    data
  })
}

export function confirmAdminTwoFactor(data: { code: string }) {
  return request.post<{ ok?: boolean }>({
    url: '/admin/api/v1/auth/2fa/confirm',
    data
  })
}

export function unlockAdminTwoFactor(data: { totp_code: string }) {
  return request.post<Admin2FAUnlockResponse>({
    url: '/admin/api/v1/auth/2fa/unlock',
    data
  })
}

export function fetchAdminForgotPassword(data: { email: string }) {
  return request.post<{ ok?: boolean }>({
    url: '/api/v1/auth/forgot-password',
    data
  })
}

export function fetchAdminResetPassword(data: { token: string; new_password: string }) {
  return request.post<{ ok?: boolean }>({
    url: '/api/v1/auth/reset-password',
    data
  })
}

export function fetchAdminProfile() {
  return request.get<AdminProfile>({
    url: '/admin/api/v1/profile'
  })
}

export function fetchAdminAutomationConfig() {
  return request.get<AutomationConfigRecord>({
    url: '/admin/api/v1/integrations/automation'
  })
}

export function fetchAdminAutomationSyncLogs(params?: Record<string, unknown>) {
  return request.get<ApiList<IntegrationSyncLogRecord>>({
    url: '/admin/api/v1/integrations/automation/sync-logs',
    params
  })
}

export function updateAdminProfile(data: { email?: string; qq?: string }) {
  return request.request<{ ok?: boolean }>({
    url: '/admin/api/v1/profile',
    method: 'PATCH',
    data
  })
}

export function changeAdminPassword(data: { old_password: string; new_password: string }) {
  return request.post<{ ok?: boolean }>({
    url: '/admin/api/v1/profile/change-password',
    data
  })
}

export function fetchAdminUsers(params?: Record<string, unknown>) {
  return request.get<ApiList<UserRecord>>({
    url: '/admin/api/v1/users',
    params
  })
}

export function fetchAdminUserDetail(id: number | string) {
  return request.get<UserRecord>({
    url: `/admin/api/v1/users/${id}`
  })
}

export function createAdminUser(data: Record<string, unknown>) {
  return request.post<UserRecord>({
    url: '/admin/api/v1/users',
    data
  })
}

export function updateAdminUser(id: number | string, data: Record<string, unknown>) {
  return request.request<UserRecord>({
    url: `/admin/api/v1/users/${id}`,
    method: 'PATCH',
    data
  })
}

export function updateAdminUserStatus(id: number | string, data: { status: string }) {
  return request.request<{ ok?: boolean }>({
    url: `/admin/api/v1/users/${id}/status`,
    method: 'PATCH',
    data
  })
}

export function updateAdminUserRealNameStatus(
  id: number | string,
  data: { status: string; reason?: string }
) {
  return request.request<{ ok?: boolean }>({
    url: `/admin/api/v1/users/${id}/realname-status`,
    method: 'PATCH',
    data
  })
}

export function resetAdminUserPassword(id: number | string, data: { password: string }) {
  return request.post<{ ok?: boolean }>({
    url: `/admin/api/v1/users/${id}/reset-password`,
    data
  })
}

export function impersonateAdminUser(id: number | string) {
  return request.post<ImpersonateResponse>({
    url: `/admin/api/v1/users/${id}/impersonate`
  })
}

export function fetchUserTierGroups() {
  return request.get<ApiList<UserTierGroup>>({
    url: '/admin/api/v1/user-tiers'
  })
}

export function createUserTierGroup(data: Record<string, unknown>) {
  return request.post<UserTierGroup>({
    url: '/admin/api/v1/user-tiers',
    data
  })
}

export function updateUserTierGroup(id: number | string, data: Record<string, unknown>) {
  return request.request<UserTierGroup>({
    url: `/admin/api/v1/user-tiers/${id}`,
    method: 'PATCH',
    data
  })
}

export function deleteUserTierGroup(id: number | string) {
  return request.request<{ ok?: boolean }>({
    url: `/admin/api/v1/user-tiers/${id}`,
    method: 'DELETE'
  })
}

export function rebuildUserTierCaches(id?: number | string) {
  return request.post<{ ok?: boolean }>({
    url: id ? `/admin/api/v1/user-tiers/${id}/rebuild` : '/admin/api/v1/user-tiers/rebuild'
  })
}

export function fetchUserTierDiscountRules(groupId: number | string) {
  return request.get<ApiList<UserTierDiscountRule>>({
    url: `/admin/api/v1/user-tiers/${groupId}/discount-rules`
  })
}

export function createUserTierDiscountRule(
  groupId: number | string,
  data: Record<string, unknown>
) {
  return request.post<UserTierDiscountRule>({
    url: `/admin/api/v1/user-tiers/${groupId}/discount-rules`,
    data
  })
}

export function updateUserTierDiscountRule(
  groupId: number | string,
  ruleId: number | string,
  data: Record<string, unknown>
) {
  return request.request<UserTierDiscountRule>({
    url: `/admin/api/v1/user-tiers/${groupId}/discount-rules/${ruleId}`,
    method: 'PATCH',
    data
  })
}

export function deleteUserTierDiscountRule(groupId: number | string, ruleId: number | string) {
  return request.request<{ ok?: boolean }>({
    url: `/admin/api/v1/user-tiers/${groupId}/discount-rules/${ruleId}`,
    method: 'DELETE'
  })
}

export function fetchUserTierAutoRules(groupId: number | string) {
  return request.get<ApiList<UserTierAutoRule>>({
    url: `/admin/api/v1/user-tiers/${groupId}/auto-rules`
  })
}

export function createUserTierAutoRule(groupId: number | string, data: Record<string, unknown>) {
  return request.post<UserTierAutoRule>({
    url: `/admin/api/v1/user-tiers/${groupId}/auto-rules`,
    data
  })
}

export function updateUserTierAutoRule(
  groupId: number | string,
  ruleId: number | string,
  data: Record<string, unknown>
) {
  return request.request<UserTierAutoRule>({
    url: `/admin/api/v1/user-tiers/${groupId}/auto-rules/${ruleId}`,
    method: 'PATCH',
    data
  })
}

export function deleteUserTierAutoRule(groupId: number | string, ruleId: number | string) {
  return request.request<{ ok?: boolean }>({
    url: `/admin/api/v1/user-tiers/${groupId}/auto-rules/${ruleId}`,
    method: 'DELETE'
  })
}

export function updateAdminUserTier(
  id: number | string,
  data: { group_id: number; expire_at?: string }
) {
  return request.request<{ ok?: boolean }>({
    url: `/admin/api/v1/users/${id}/tier`,
    method: 'PATCH',
    data
  })
}

export function fetchAdminCouponGroups() {
  return request.get<ApiList<CouponProductGroupRecord>>({
    url: '/admin/api/v1/coupon-groups'
  })
}

export function createAdminCouponGroup(data: Record<string, unknown>) {
  return request.post<CouponProductGroupRecord>({
    url: '/admin/api/v1/coupon-groups',
    data
  })
}

export function updateAdminCouponGroup(id: number | string, data: Record<string, unknown>) {
  return request.request<CouponProductGroupRecord>({
    url: `/admin/api/v1/coupon-groups/${id}`,
    method: 'PATCH',
    data
  })
}

export function deleteAdminCouponGroup(id: number | string) {
  return request.request<{ ok?: boolean }>({
    url: `/admin/api/v1/coupon-groups/${id}`,
    method: 'DELETE'
  })
}

export function fetchAdminCoupons(params?: Record<string, unknown>) {
  return request.get<ApiList<CouponRecord>>({
    url: '/admin/api/v1/coupons',
    params
  })
}

export function createAdminCoupon(data: Record<string, unknown>) {
  return request.post<CouponRecord>({
    url: '/admin/api/v1/coupons',
    data
  })
}

export function updateAdminCoupon(id: number | string, data: Record<string, unknown>) {
  return request.request<CouponRecord>({
    url: `/admin/api/v1/coupons/${id}`,
    method: 'PATCH',
    data
  })
}

export function deleteAdminCoupon(id: number | string) {
  return request.request<{ ok?: boolean }>({
    url: `/admin/api/v1/coupons/${id}`,
    method: 'DELETE'
  })
}

export function batchGenerateAdminCoupons(data: Record<string, unknown>) {
  return request.post<ApiList<CouponRecord>>({
    url: '/admin/api/v1/coupons/batch-generate',
    data
  })
}

export function fetchAdminWalletInfo(userId: number | string) {
  return request.get<{ wallet?: WalletInfo }>({
    url: `/admin/api/v1/wallets/${userId}`
  })
}

export function fetchAdminWalletOrders(params?: Record<string, unknown>) {
  return request.get<ApiList<WalletOrderRecord>>({
    url: '/admin/api/v1/wallet/orders',
    params
  })
}

export function approveAdminWalletOrder(id: number | string) {
  return request.post<{ order?: WalletOrderRecord }>({
    url: `/admin/api/v1/wallet/orders/${id}/approve`
  })
}

export function rejectAdminWalletOrder(
  id: number | string,
  data?: {
    reason?: string
  }
) {
  return request.post<{ ok?: boolean }>({
    url: `/admin/api/v1/wallet/orders/${id}/reject`,
    data: data || {}
  })
}

export function fetchAdminWalletTransactions(
  userId: number | string,
  params?: Record<string, unknown>
) {
  return request.get<ApiList<WalletTransaction>>({
    url: `/admin/api/v1/wallets/${userId}/transactions`,
    params
  })
}

export function fetchRealNameRecords(params?: Record<string, unknown>) {
  return request.get<ApiList<RealNameVerification>>({
    url: '/admin/api/v1/realname/records',
    params
  })
}

export function fetchRealNameConfig() {
  return request.get<RealNameConfigRecord>({
    url: '/admin/api/v1/realname/config'
  })
}

export function updateRealNameConfig(data: {
  enabled: boolean
  provider: string
  block_actions: string[]
}) {
  return request.request<{ ok?: boolean }>({
    url: '/admin/api/v1/realname/config',
    method: 'PATCH',
    data
  })
}

export function fetchRealNameProviders() {
  return request.get<ApiList<RealNameProviderRecord>>({
    url: '/admin/api/v1/realname/providers'
  })
}

export function fetchAdminSettings() {
  return request.get<ApiList<SettingItemRecord>>({
    url: '/admin/api/v1/settings'
  })
}

export function fetchCMSCategories(params?: Record<string, unknown>) {
  return request.get<ApiList<CMSCategoryRecord>>({
    url: '/admin/api/v1/cms/categories',
    params
  })
}

export function createCMSCategory(data: Record<string, unknown>) {
  return request.post<CMSCategoryRecord>({
    url: '/admin/api/v1/cms/categories',
    data
  })
}

export function updateCMSCategory(id: number | string, data: Record<string, unknown>) {
  return request.request<CMSCategoryRecord>({
    url: `/admin/api/v1/cms/categories/${id}`,
    method: 'PATCH',
    data
  })
}

export function deleteCMSCategory(id: number | string) {
  return request.request<{ ok?: boolean }>({
    url: `/admin/api/v1/cms/categories/${id}`,
    method: 'DELETE'
  })
}

export function fetchCMSPosts(params?: Record<string, unknown>) {
  return request.get<ApiList<CMSPostRecord>>({
    url: '/admin/api/v1/cms/posts',
    params
  })
}

export function createCMSPost(data: Record<string, unknown>) {
  return request.post<CMSPostRecord>({
    url: '/admin/api/v1/cms/posts',
    data
  })
}

export function updateCMSPost(id: number | string, data: Record<string, unknown>) {
  return request.request<CMSPostRecord>({
    url: `/admin/api/v1/cms/posts/${id}`,
    method: 'PATCH',
    data
  })
}

export function deleteCMSPost(id: number | string) {
  return request.request<{ ok?: boolean }>({
    url: `/admin/api/v1/cms/posts/${id}`,
    method: 'DELETE'
  })
}

export function fetchCMSBlocks(params?: Record<string, unknown>) {
  return request.get<ApiList<CMSBlockRecord>>({
    url: '/admin/api/v1/cms/blocks',
    params
  })
}

export function createCMSBlock(data: Record<string, unknown>) {
  return request.post<CMSBlockRecord>({
    url: '/admin/api/v1/cms/blocks',
    data
  })
}

export function updateCMSBlock(id: number | string, data: Record<string, unknown>) {
  return request.request<CMSBlockRecord>({
    url: `/admin/api/v1/cms/blocks/${id}`,
    method: 'PATCH',
    data
  })
}

export function deleteCMSBlock(id: number | string) {
  return request.request<{ ok?: boolean }>({
    url: `/admin/api/v1/cms/blocks/${id}`,
    method: 'DELETE'
  })
}

export function fetchAdminUploads(params?: Record<string, unknown>) {
  return request.get<ApiList<UploadAssetRecord>>({
    url: '/admin/api/v1/uploads',
    params
  })
}

export function uploadAdminAsset(file: File) {
  const formData = new FormData()
  formData.append('file', file)

  return request.post<UploadAssetRecord>({
    url: '/admin/api/v1/uploads',
    data: formData
  })
}

export function fetchAdminApiKeys(params?: Record<string, unknown>) {
  return request.get<ApiList<APIKeyRecord>>({
    url: '/admin/api/v1/api-keys',
    params
  })
}

export function createAdminApiKey(data: {
  name?: string
  permission_group_id?: number | null
  scopes?: string[]
}) {
  return request.post<{ api_key?: string; record?: APIKeyRecord }>({
    url: '/admin/api/v1/api-keys',
    data
  })
}

export function updateAdminApiKeyStatus(
  id: number | string,
  data: {
    status: string
  }
) {
  return request.request<{ ok?: boolean }>({
    url: `/admin/api/v1/api-keys/${id}`,
    method: 'PATCH',
    data
  })
}

export function fetchPermissionGroups() {
  return request.get<ApiList<PermissionGroupRecord>>({
    url: '/admin/api/v1/permission-groups'
  })
}

export function createPermissionGroup(data: {
  name: string
  description?: string
  permissions: string[]
}) {
  return request.post<PermissionGroupRecord>({
    url: '/admin/api/v1/permission-groups',
    data
  })
}

export function updatePermissionGroup(
  id: number | string,
  data: {
    name: string
    description?: string
    permissions: string[]
  }
) {
  return request.request<{ ok?: boolean }>({
    url: `/admin/api/v1/permission-groups/${id}`,
    method: 'PATCH',
    data
  })
}

export function deletePermissionGroup(id: number | string) {
  return request.request<{ ok?: boolean }>({
    url: `/admin/api/v1/permission-groups/${id}`,
    method: 'DELETE'
  })
}

export function fetchAdminPermissions() {
  return request.get<ApiList<PermissionRecord>>({
    url: '/admin/api/v1/permissions/list'
  })
}

export function fetchAdminAccounts(params?: Record<string, unknown>) {
  return request.get<ApiList<UserRecord>>({
    url: '/admin/api/v1/admins',
    params
  })
}

export function createAdminAccount(data: {
  username: string
  email: string
  qq?: string
  password: string
  permission_group_id?: number | null
}) {
  return request.post<UserRecord>({
    url: '/admin/api/v1/admins',
    data
  })
}

export function updateAdminAccount(
  id: number | string,
  data: {
    username: string
    email: string
    qq?: string
    permission_group_id?: number | null
  }
) {
  return request.request<{ ok?: boolean }>({
    url: `/admin/api/v1/admins/${id}`,
    method: 'PATCH',
    data
  })
}

export function updateAdminAccountStatus(
  id: number | string,
  data: {
    status: string
  }
) {
  return request.request<{ ok?: boolean }>({
    url: `/admin/api/v1/admins/${id}/status`,
    method: 'PATCH',
    data
  })
}

export function deleteAdminAccount(id: number | string) {
  return request.request<{ ok?: boolean }>({
    url: `/admin/api/v1/admins/${id}`,
    method: 'DELETE'
  })
}

export function updateAdminSettings(data: {
  key?: string
  value?: string
  items?: Array<{
    key: string
    value: string
  }>
}) {
  return request.request<{ ok?: boolean }>({
    url: '/admin/api/v1/settings',
    method: 'PATCH',
    data
  })
}

export function fetchAdminPlugins() {
  return request.get<ApiList<AdminPluginRecord>>({
    url: '/admin/api/v1/plugins'
  })
}

export function fetchAdminPluginDiscover() {
  return request.get<ApiList<PluginDiscoverRecord>>({
    url: '/admin/api/v1/plugins/discover'
  })
}

export function installAdminPlugin(file: File, adminPassword?: string) {
  const data = new FormData()
  data.append('file', file)
  if (adminPassword) {
    data.append('admin_password', adminPassword)
  }

  return request.post<{ ok?: boolean; plugin?: AdminPluginRecord }>({
    url: '/admin/api/v1/plugins/install',
    data
  })
}

export function uploadAdminPaymentPlugin(file: File, password: string) {
  const data = new FormData()
  data.append('file', file)
  data.append('password', password)

  return request.post<{ ok?: boolean; path?: string }>({
    url: '/admin/api/v1/plugins/payment/upload',
    data
  })
}

export function importAdminPluginFromDisk(
  category: string,
  pluginId: string,
  adminPassword?: string
) {
  return request.post<{ ok?: boolean; plugin?: AdminPluginRecord }>({
    url: `/admin/api/v1/plugins/${category}/${pluginId}/import`,
    data: adminPassword ? { admin_password: adminPassword } : {}
  })
}

export function createAdminPluginInstance(
  category: string,
  pluginId: string,
  data?: {
    instance_id?: string
    config_json?: string
  }
) {
  return request.post<{ ok?: boolean; plugin?: AdminPluginRecord }>({
    url: `/admin/api/v1/plugins/${category}/${pluginId}/instances`,
    data: data || {}
  })
}

export function enableAdminPluginInstance(category: string, pluginId: string, instanceId: string) {
  return request.post<{ ok?: boolean }>({
    url: `/admin/api/v1/plugins/${category}/${pluginId}/${instanceId}/enable`
  })
}

export function disableAdminPluginInstance(category: string, pluginId: string, instanceId: string) {
  return request.post<{ ok?: boolean }>({
    url: `/admin/api/v1/plugins/${category}/${pluginId}/${instanceId}/disable`
  })
}

export function deleteAdminPluginInstance(category: string, pluginId: string, instanceId: string) {
  return request.del<{ ok?: boolean }>({
    url: `/admin/api/v1/plugins/${category}/${pluginId}/${instanceId}`
  })
}

export function fetchAdminPluginInstanceConfigSchema(
  category: string,
  pluginId: string,
  instanceId: string
) {
  return request.get<PluginConfigSchemaResponse>({
    url: `/admin/api/v1/plugins/${category}/${pluginId}/${instanceId}/config/schema`
  })
}

export function fetchAdminPluginInstanceConfig(
  category: string,
  pluginId: string,
  instanceId: string
) {
  return request.get<PluginConfigResponse>({
    url: `/admin/api/v1/plugins/${category}/${pluginId}/${instanceId}/config`
  })
}

export function updateAdminPluginInstanceConfig(
  category: string,
  pluginId: string,
  instanceId: string,
  configJson: string
) {
  return request.put<{ ok?: boolean }>({
    url: `/admin/api/v1/plugins/${category}/${pluginId}/${instanceId}/config`,
    data: {
      config_json: configJson
    }
  })
}

export function fetchAdminPaymentProviders(params?: {
  include_disabled?: boolean
  include_legacy?: boolean
  scene?: 'order' | 'wallet'
}) {
  return request.get<ApiList<PaymentProviderRecord>>({
    url: '/admin/api/v1/payments/providers',
    params
  })
}

export function updateAdminPaymentProvider(
  key: string,
  data: {
    enabled?: boolean
    config_json?: string
    scene?: 'order' | 'wallet'
  }
) {
  return request.request<{ ok?: boolean }>({
    url: `/admin/api/v1/payments/providers/${key}`,
    method: 'PATCH',
    data
  })
}

export function fetchAdminPluginPaymentMethods(params: {
  category?: string
  plugin_id: string
  instance_id?: string
}) {
  return request.get<ApiList<PluginPaymentMethodRecord>>({
    url: '/admin/api/v1/plugins/payment-methods',
    params
  })
}

export function updateAdminPluginPaymentMethod(data: {
  category?: string
  plugin_id: string
  instance_id?: string
  method: string
  enabled: boolean
}) {
  return request.request<{ ok?: boolean }>({
    url: '/admin/api/v1/plugins/payment-methods',
    method: 'PATCH',
    data
  })
}

export function fetchSmtpConfig() {
  return request.get<SmtpConfigRecord>({
    url: '/admin/api/v1/integrations/smtp'
  })
}

export function updateSmtpConfig(data: Record<string, unknown>) {
  return request.request<{ ok?: boolean }>({
    url: '/admin/api/v1/integrations/smtp',
    method: 'PATCH',
    data
  })
}

export function testSmtpConfig(data: Record<string, unknown>) {
  return request.request<{ ok?: boolean }>({
    url: '/admin/api/v1/integrations/smtp/test',
    method: 'POST',
    data
  })
}

export function fetchEmailTemplates() {
  return request.get<ApiList<EmailTemplateRecord>>({
    url: '/admin/api/v1/email-templates'
  })
}

export function createEmailTemplate(data: Record<string, unknown>) {
  return request.request<EmailTemplateRecord>({
    url: '/admin/api/v1/email-templates',
    method: 'POST',
    data
  })
}

export function updateEmailTemplate(id: number | string, data: Record<string, unknown>) {
  return request.request<EmailTemplateRecord>({
    url: `/admin/api/v1/email-templates/${id}`,
    method: 'PATCH',
    data
  })
}

export function deleteEmailTemplate(id: number | string) {
  return request.request<{ ok?: boolean }>({
    url: `/admin/api/v1/email-templates/${id}`,
    method: 'DELETE'
  })
}

export function fetchRobotConfig() {
  return request.get<RobotConfigRecord>({
    url: '/admin/api/v1/integrations/robot'
  })
}

export function updateRobotConfig(data: Record<string, unknown>) {
  return request.request<{ ok?: boolean }>({
    url: '/admin/api/v1/integrations/robot',
    method: 'PATCH',
    data
  })
}

export function testRobotWebhook(data: Record<string, unknown>) {
  return request.request<{ ok?: boolean }>({
    url: '/admin/api/v1/integrations/robot/test',
    method: 'POST',
    data
  })
}

export function fetchSmsConfig() {
  return request.get<SmsConfigRecord>({
    url: '/admin/api/v1/integrations/sms'
  })
}

export function updateSmsConfig(data: Record<string, unknown>) {
  return request.request<{ ok?: boolean }>({
    url: '/admin/api/v1/integrations/sms',
    method: 'PATCH',
    data
  })
}

export function previewSmsConfig(data: Record<string, unknown>) {
  return request.request<{ content?: string }>({
    url: '/admin/api/v1/integrations/sms/preview',
    method: 'POST',
    data
  })
}

export function testSmsConfig(data: Record<string, unknown>) {
  return request.request<{ ok?: boolean }>({
    url: '/admin/api/v1/integrations/sms/test',
    method: 'POST',
    data
  })
}

export function fetchSmsTemplates() {
  return request.get<ApiList<SmsTemplateRecord>>({
    url: '/admin/api/v1/sms-templates'
  })
}

export function createSmsTemplate(data: Record<string, unknown>) {
  return request.request<SmsTemplateRecord>({
    url: '/admin/api/v1/sms-templates',
    method: 'POST',
    data
  })
}

export function updateSmsTemplate(id: number | string, data: Record<string, unknown>) {
  return request.request<SmsTemplateRecord>({
    url: `/admin/api/v1/sms-templates/${id}`,
    method: 'PATCH',
    data
  })
}

export function deleteSmsTemplate(id: number | string) {
  return request.request<{ ok?: boolean }>({
    url: `/admin/api/v1/sms-templates/${id}`,
    method: 'DELETE'
  })
}

export function fetchAdminDebugStatus() {
  return request.get<DebugStatusResponse>({
    url: '/admin/api/v1/debug/status'
  })
}

export function updateAdminDebugStatus(data: { enabled: boolean }) {
  return request.request<{ ok?: boolean }>({
    url: '/admin/api/v1/debug/status',
    method: 'PATCH',
    data
  })
}

export function fetchAdminDebugLogs(params?: Record<string, unknown>) {
  return request.get<DebugLogsResponse>({
    url: '/admin/api/v1/debug/logs',
    params
  })
}

export function fetchAdminAuditLogs(params?: Record<string, unknown>) {
  return request.get<ApiList<AdminAuditLogRecord>>({
    url: '/admin/api/v1/audit-logs',
    params
  })
}

export function fetchAdminScheduledTasks() {
  return request.get<ApiList<ScheduledTaskRecord>>({
    url: '/admin/api/v1/scheduled-tasks'
  })
}

export function updateAdminScheduledTask(
  key: string,
  data: {
    enabled?: boolean
    strategy?: ScheduledTaskStrategy | string
    interval_sec?: number
    daily_at?: string
  }
) {
  return request.request<ScheduledTaskRecord>({
    url: `/admin/api/v1/scheduled-tasks/${key}`,
    method: 'PATCH',
    data
  })
}

export function fetchAdminScheduledTaskRuns(key: string, params?: { limit?: number }) {
  return request.get<ApiList<ScheduledTaskRunRecord>>({
    url: `/admin/api/v1/scheduled-tasks/${key}/runs`,
    params
  })
}

export function fetchAdminTickets(params?: Record<string, unknown>) {
  return request.get<ApiList<TicketRecord>>({
    url: '/admin/api/v1/tickets',
    params
  })
}

export function fetchAdminProbes(params?: Record<string, unknown>) {
  return request.get<ApiList<ProbeRecord>>({
    url: '/admin/api/v1/probes',
    params
  })
}

export function createAdminProbe(data: {
  name: string
  agent_id: string
  os_type: string
  tags?: string[]
}) {
  return request.post<{ probe?: ProbeRecord; enroll_token?: string }>({
    url: '/admin/api/v1/probes',
    data
  })
}

export function fetchAdminProbeDetail(id: number | string, params?: Record<string, unknown>) {
  return request.get<{ probe?: ProbeRecord; online?: boolean }>({
    url: `/admin/api/v1/probes/${id}`,
    params
  })
}

export function updateAdminProbe(
  id: number | string,
  data: {
    name?: string
    os_type?: string
    tags?: string[]
  }
) {
  return request.request<{ ok?: boolean }>({
    url: `/admin/api/v1/probes/${id}`,
    method: 'PATCH',
    data
  })
}

export function deleteAdminProbe(id: number | string) {
  return request.request<{ ok?: boolean }>({
    url: `/admin/api/v1/probes/${id}`,
    method: 'DELETE'
  })
}

export function resetAdminProbeEnrollToken(id: number | string) {
  return request.post<{ enroll_token?: string }>({
    url: `/admin/api/v1/probes/${id}/enroll-token/reset`
  })
}

export function fetchAdminProbeSla(id: number | string, params?: Record<string, unknown>) {
  return request.get<{ sla?: ProbeSlaRecord }>({
    url: `/admin/api/v1/probes/${id}/sla`,
    params
  })
}

export function createAdminProbeLogSession(
  id: number | string,
  data: {
    source?: string
    keyword?: string
    follow?: boolean
    lines?: number
  }
) {
  return request.post<ProbeLogSessionResponse>({
    url: `/admin/api/v1/probes/${id}/log-sessions`,
    data
  })
}

export function fetchAdminTicketDetail(id: number | string) {
  return request.get<TicketDetailResponse>({
    url: `/admin/api/v1/tickets/${id}`
  })
}

export function updateAdminTicket(id: number | string, data: Record<string, unknown>) {
  return request.request<TicketRecord>({
    url: `/admin/api/v1/tickets/${id}`,
    method: 'PATCH',
    data
  })
}

export function createAdminTicketMessage(id: number | string, data: { content: string }) {
  return request.post<TicketMessageRecord>({
    url: `/admin/api/v1/tickets/${id}/messages`,
    data
  })
}

export function deleteAdminTicket(id: number | string) {
  return request.del<{ ok?: boolean }>({
    url: `/admin/api/v1/tickets/${id}`
  })
}

export function fetchAdminOrders(params?: Record<string, unknown>) {
  return request.get<ApiList<OrderRecord>>({
    url: '/admin/api/v1/orders',
    params
  })
}

export function fetchAdminOrderDetail(id: number | string) {
  return request.get<OrderDetailResponse>({
    url: `/admin/api/v1/orders/${id}`
  })
}

export function approveAdminOrder(id: number | string) {
  return request.post<{ ok?: boolean }>({
    url: `/admin/api/v1/orders/${id}/approve`
  })
}

export function rejectAdminOrder(
  id: number | string,
  data: {
    reason?: string
  }
) {
  return request.post<{ ok?: boolean }>({
    url: `/admin/api/v1/orders/${id}/reject`,
    data
  })
}

export function deleteAdminOrder(id: number | string) {
  return request.del<{ ok?: boolean }>({
    url: `/admin/api/v1/orders/${id}`
  })
}

export function retryAdminOrder(id: number | string) {
  return request.post<{ ok?: boolean }>({
    url: `/admin/api/v1/orders/${id}/retry`
  })
}

export function fetchAdminGoodsTypes() {
  return request.get<ApiList<CatalogGoodsType>>({
    url: '/admin/api/v1/goods-types'
  })
}

export function createAdminGoodsType(data: Record<string, unknown>) {
  return request.post<CatalogGoodsType>({
    url: '/admin/api/v1/goods-types',
    data
  })
}

export function updateAdminGoodsType(id: number | string, data: Record<string, unknown>) {
  return request.put<{ ok?: boolean }>({
    url: `/admin/api/v1/goods-types/${id}`,
    data
  })
}

export function deleteAdminGoodsType(id: number | string) {
  return request.del<{ ok?: boolean }>({
    url: `/admin/api/v1/goods-types/${id}`
  })
}

export function syncAdminGoodsTypeAutomation(
  id: number | string,
  mode: 'merge' | 'replace' = 'merge'
) {
  return request.post<Record<string, unknown>>({
    url: `/admin/api/v1/goods-types/${id}/sync-automation`,
    params: { mode }
  })
}

export function fetchAdminGoodsTypeCapabilities(id: number | string) {
  return request.get<CatalogCapabilityRecord>({
    url: `/admin/api/v1/goods-types/${id}/capabilities`
  })
}

export function updateAdminGoodsTypeCapabilities(
  id: number | string,
  data: {
    resize_enabled?: boolean | null
    refund_enabled?: boolean | null
  }
) {
  return request.request<{ ok?: boolean }>({
    url: `/admin/api/v1/goods-types/${id}/capabilities`,
    method: 'PATCH',
    data
  })
}

export function fetchAdminRegions(params?: Record<string, unknown>) {
  return request.get<ApiList<CatalogRegion>>({
    url: '/admin/api/v1/regions',
    params
  })
}

export function createAdminRegion(data: Record<string, unknown>) {
  return request.post<CatalogRegion>({
    url: '/admin/api/v1/regions',
    data
  })
}

export function updateAdminRegion(id: number | string, data: Record<string, unknown>) {
  return request.request<CatalogRegion>({
    url: `/admin/api/v1/regions/${id}`,
    method: 'PATCH',
    data
  })
}

export function deleteAdminRegion(id: number | string) {
  return request.del<{ ok?: boolean }>({
    url: `/admin/api/v1/regions/${id}`
  })
}

export function bulkDeleteAdminRegions(ids: Array<number | string>) {
  return request.post<{ ok?: boolean }>({
    url: '/admin/api/v1/regions/bulk-delete',
    data: { ids }
  })
}

export function fetchAdminPlanGroups(params?: Record<string, unknown>) {
  return request.get<ApiList<CatalogPlanGroup>>({
    url: '/admin/api/v1/plan-groups',
    params
  })
}

export function fetchAdminLines(params?: Record<string, unknown>) {
  return request.get<ApiList<CatalogPlanGroup>>({
    url: '/admin/api/v1/lines',
    params
  })
}

export function createAdminPlanGroup(data: Record<string, unknown>) {
  return request.post<CatalogPlanGroup>({
    url: '/admin/api/v1/plan-groups',
    data
  })
}

export function updateAdminPlanGroup(id: number | string, data: Record<string, unknown>) {
  return request.request<CatalogPlanGroup>({
    url: `/admin/api/v1/plan-groups/${id}`,
    method: 'PATCH',
    data
  })
}

export function deleteAdminPlanGroup(id: number | string) {
  return request.del<{ ok?: boolean }>({
    url: `/admin/api/v1/plan-groups/${id}`
  })
}

export function bulkDeleteAdminPlanGroups(ids: Array<number | string>) {
  return request.post<{ ok?: boolean }>({
    url: '/admin/api/v1/plan-groups/bulk-delete',
    data: { ids }
  })
}

// The backend accepts a PlanGroup id here and resolves it to the underlying LineID.
export function setAdminLineSystemImages(
  id: number | string,
  data: {
    image_ids: Array<number | string>
  }
) {
  return request.post<{ ok?: boolean }>({
    url: `/admin/api/v1/lines/${id}/system-images`,
    data
  })
}

export function fetchAdminPackages(params?: Record<string, unknown>) {
  return request.get<ApiList<CatalogPackage>>({
    url: '/admin/api/v1/packages',
    params
  })
}

export function createAdminPackage(data: Record<string, unknown>) {
  return request.post<CatalogPackage>({
    url: '/admin/api/v1/packages',
    data
  })
}

export function updateAdminPackage(id: number | string, data: Record<string, unknown>) {
  return request.request<CatalogPackage>({
    url: `/admin/api/v1/packages/${id}`,
    method: 'PATCH',
    data
  })
}

export function deleteAdminPackage(id: number | string) {
  return request.del<{ ok?: boolean }>({
    url: `/admin/api/v1/packages/${id}`
  })
}

export function bulkDeleteAdminPackages(ids: Array<number | string>) {
  return request.post<{ ok?: boolean }>({
    url: '/admin/api/v1/packages/bulk-delete',
    data: { ids }
  })
}

export function fetchAdminSystemImages(params?: Record<string, unknown>) {
  return request.get<ApiList<CatalogSystemImage>>({
    url: '/admin/api/v1/system-images',
    params
  })
}

export function createAdminSystemImage(data: Record<string, unknown>) {
  return request.post<CatalogSystemImage>({
    url: '/admin/api/v1/system-images',
    data
  })
}

export function updateAdminSystemImage(id: number | string, data: Record<string, unknown>) {
  return request.request<CatalogSystemImage>({
    url: `/admin/api/v1/system-images/${id}`,
    method: 'PATCH',
    data
  })
}

export function deleteAdminSystemImage(id: number | string) {
  return request.del<{ ok?: boolean }>({
    url: `/admin/api/v1/system-images/${id}`
  })
}

export function bulkDeleteAdminSystemImages(ids: Array<number | string>) {
  return request.post<{ ok?: boolean }>({
    url: '/admin/api/v1/system-images/bulk-delete',
    data: { ids }
  })
}

export function syncAdminSystemImages(params?: {
  goods_type_id?: number | string
  plan_group_id?: number | string
  line_id?: number | string
}) {
  return request.post<Record<string, unknown>>({
    url: '/admin/api/v1/system-images/sync',
    params
  })
}

export function fetchAdminBillingCycles() {
  return request.get<ApiList<CatalogBillingCycle>>({
    url: '/admin/api/v1/billing-cycles'
  })
}

export function createAdminBillingCycle(data: Record<string, unknown>) {
  return request.post<CatalogBillingCycle>({
    url: '/admin/api/v1/billing-cycles',
    data
  })
}

export function updateAdminBillingCycle(id: number | string, data: Record<string, unknown>) {
  return request.request<CatalogBillingCycle>({
    url: `/admin/api/v1/billing-cycles/${id}`,
    method: 'PATCH',
    data
  })
}

export function deleteAdminBillingCycle(id: number | string) {
  return request.del<{ ok?: boolean }>({
    url: `/admin/api/v1/billing-cycles/${id}`
  })
}

export function bulkDeleteAdminBillingCycles(ids: Array<number | string>) {
  return request.post<{ ok?: boolean }>({
    url: '/admin/api/v1/billing-cycles/bulk-delete',
    data: { ids }
  })
}

export function fetchAdminVps(params?: Record<string, unknown>) {
  return request.get<ApiList<VpsRecord>>({
    url: '/admin/api/v1/vps',
    params
  })
}

export function fetchAdminVpsDetail(id: number | string) {
  return request.get<VpsRecord>({
    url: `/admin/api/v1/vps/${id}`
  })
}

export function createAdminVps(data: Record<string, unknown>) {
  return request.post<VpsRecord>({
    url: '/admin/api/v1/vps',
    data
  })
}

export function updateAdminVps(id: number | string, data: Record<string, unknown>) {
  return request.request<VpsRecord>({
    url: `/admin/api/v1/vps/${id}`,
    method: 'PATCH',
    data
  })
}

export function lockAdminVps(id: number | string) {
  return request.post<{ ok?: boolean }>({
    url: `/admin/api/v1/vps/${id}/lock`
  })
}

export function unlockAdminVps(id: number | string) {
  return request.post<{ ok?: boolean }>({
    url: `/admin/api/v1/vps/${id}/unlock`
  })
}

export function deleteAdminVps(id: number | string, data?: { reason?: string }) {
  return request.post<{ ok?: boolean }>({
    url: `/admin/api/v1/vps/${id}/delete`,
    data: data || {}
  })
}

export function resizeAdminVps(id: number | string, data: Record<string, unknown>) {
  return request.post<{ ok?: boolean }>({
    url: `/admin/api/v1/vps/${id}/resize`,
    data
  })
}

export function refreshAdminVps(id: number | string) {
  return request.post<VpsRecord>({
    url: `/admin/api/v1/vps/${id}/refresh`
  })
}

export function updateAdminVpsStatus(id: number | string, data: Record<string, unknown>) {
  return request.post<{ ok?: boolean }>({
    url: `/admin/api/v1/vps/${id}/status`,
    data
  })
}

export function emergencyRenewAdminVps(id: number | string, data?: Record<string, unknown>) {
  return request.post<VpsRecord>({
    url: `/admin/api/v1/vps/${id}/emergency-renew`,
    data: data || {}
  })
}

export function updateAdminVpsExpire(id: number | string, data: { expire_at: string }) {
  return request.request<VpsRecord>({
    url: `/admin/api/v1/vps/${id}/expire-at`,
    method: 'PATCH',
    data
  })
}

export function fetchAdminDashboardOverview() {
  return request.post<DashboardOverview>({
    url: '/admin/api/v1/dashboard/overview'
  })
}

export function fetchAdminDashboardRevenue(params?: Record<string, unknown>) {
  return request.request<DashboardRevenue>({
    url: '/admin/api/v1/dashboard/revenue',
    method: 'POST',
    params,
    data: {}
  })
}

export function fetchAdminRevenueAnalyticsOverview(data: RevenueAnalyticsQuery) {
  return request.post<RevenueAnalyticsOverviewResponse>({
    url: '/admin/api/v1/dashboard/revenue-analytics/overview',
    data
  })
}

export function fetchAdminRevenueAnalyticsTrend(data: RevenueAnalyticsQuery) {
  return request.post<{ items?: RevenueAnalyticsTrendPoint[] }>({
    url: '/admin/api/v1/dashboard/revenue-analytics/trend',
    data
  })
}

export function fetchAdminRevenueAnalyticsTop(data: RevenueAnalyticsQuery) {
  return request.post<{ items?: RevenueAnalyticsTopItem[] }>({
    url: '/admin/api/v1/dashboard/revenue-analytics/top',
    data
  })
}

export function fetchAdminRevenueAnalyticsDetails(data: RevenueAnalyticsQuery) {
  return request.post<RevenueAnalyticsDetailsResponse>({
    url: '/admin/api/v1/dashboard/revenue-analytics/details',
    data
  })
}

export function exportAdminRevenueAnalyticsAudit(data: RevenueAnalyticsQuery) {
  return request.request<Blob>({
    url: '/admin/api/v1/dashboard/revenue-analytics/export',
    method: 'POST',
    data,
    responseType: 'blob',
    showSuccessMessage: false
  })
}

export function fetchAdminDashboardVpsStatus() {
  return request.get<DashboardStatus>({
    url: '/admin/api/v1/dashboard/vps-status'
  })
}

export function fetchAdminServerStatus() {
  return request.get<ServerStatus>({
    url: '/admin/api/v1/server/status'
  })
}
