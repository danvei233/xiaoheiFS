export interface ApiList<T> {
  items: T[];
  total?: number;
}

export type PluginSignatureStatus = "official" | "untrusted" | "unsigned";

export interface PluginPaymentMethodItem {
  method: string;
  enabled: boolean;
}

export interface PluginManifest {
  plugin_id?: string;
  name?: string;
  version?: string;
  description?: string;
  capabilities?: {
    sms?: { send?: boolean } | null;
    payment?: { methods?: string[] } | null;
    kyc?: { start?: boolean; query_result?: boolean } | null;
    automation?: { features?: string[]; not_supported_reasons?: Record<string, string> } | null;
  };
}

export interface PluginListItem {
  category?: string;
  plugin_id?: string;
  instance_id?: string;
  name?: string;
  version?: string;
  signature_status?: PluginSignatureStatus;
  enabled?: boolean;
  loaded?: boolean;
  installed_at?: string;
  updated_at?: string;
  last_health_at?: string | null;
  health_status?: string;
  health_message?: string;
  manifest?: PluginManifest;
  entry?: {
    platform?: string;
    entry_path?: string;
    entry_supported?: boolean;
    supported_platforms?: string[];
  };
}

export interface PluginDiscoverItem {
  category?: string;
  plugin_id?: string;
  instance_id?: string;
  name?: string;
  version?: string;
  signature_status?: PluginSignatureStatus;
  entry?: {
    platform?: string;
    entry_path?: string;
    entry_supported?: boolean;
    supported_platforms?: string[];
  };
}

export interface CaptchaResponse {
  captcha_provider?: "image" | "geetest";
  captcha_id?: string;
  api_server?: string;
  image_base64?: string;
}

export interface RegisterRequest {
  username: string;
  email?: string;
  qq?: string;
  phone?: string;
  password: string;
  captcha_id?: string;
  captcha_code?: string;
  lot_number?: string;
  captcha_output?: string;
  pass_token?: string;
  gen_time?: string;
  verify_code?: string;
  verify_channel?: "email" | "sms";
}

export interface LoginRequest {
  username: string;
  password: string;
  captcha_id?: string;
  captcha_code?: string;
  lot_number?: string;
  captcha_output?: string;
  pass_token?: string;
  gen_time?: string;
}

export interface AuthSettings {
  register_enabled?: boolean;
  register_required_fields?: string[];
  password_min_len?: number;
  password_require_upper?: boolean;
  password_require_lower?: boolean;
  password_require_number?: boolean;
  password_require_symbol?: boolean;
  register_verify_type?: "none" | "email" | "sms";
  register_verify_channels?: Array<"email" | "sms">;
  register_email_required?: boolean;
  register_verify_ttl_sec?: number;
  register_captcha_enabled?: boolean;
  login_captcha_enabled?: boolean;
  captcha_provider?: "image" | "geetest";
  auth_geetest_captcha_id?: string;
  auth_geetest_api_server?: string;
  auth_login_notify_enabled?: boolean;
  auth_login_notify_on_first_login?: boolean;
  auth_login_notify_on_ip_change?: boolean;
  auth_login_notify_channels?: Array<"email" | "sms">;
  auth_password_reset_enabled?: boolean;
  auth_password_reset_channels?: Array<"email" | "sms">;
  auth_password_reset_verify_ttl_sec?: number;
  auth_sms_code_len?: number;
  auth_sms_code_complexity?: "digits" | "letters" | "alnum";
  auth_email_code_len?: number;
  auth_email_code_complexity?: "digits" | "letters" | "alnum";
  auth_captcha_code_len?: number;
  auth_captcha_code_complexity?: "digits" | "letters" | "alnum";
  auth_email_bind_enabled?: boolean;
  auth_phone_bind_enabled?: boolean;
  auth_contact_change_notify_old_enabled?: boolean;
  auth_contact_bind_verify_ttl_sec?: number;
  auth_bind_require_password_when_no_2fa?: boolean;
  auth_rebind_require_password_when_no_2fa?: boolean;
  auth_2fa_enabled?: boolean;
  auth_2fa_bind_enabled?: boolean;
  auth_2fa_rebind_enabled?: boolean;
}

export interface AuthResponse {
  access_token?: string;
  expires_in?: number;
  user?: User;
}

export interface User {
  id?: number;
  username?: string;
  email?: string;
  email_masked?: string;
  email_bound?: boolean;
  qq?: string;
  phone?: string;
  phone_masked?: string;
  phone_bound?: boolean;
  totp_enabled?: boolean;
  avatar?: string;
  avatar_url?: string;
  bio?: string;
  intro?: string;
  role?: string;
  status?: string;
  created_at?: string;
  updated_at?: string;
  balance?: number;
}

export interface Region {
  id?: number;
  goods_type_id?: number;
  code?: string;
  name?: string;
  active?: boolean;
}

export interface Line {
  id?: number;
  goods_type_id?: number;
  region_id?: number;
  name?: string;
  line_id?: number;
  unit_core?: number;
  unit_mem?: number;
  unit_disk?: number;
  unit_bw?: number;
  add_core_min?: number;
  add_core_max?: number;
  add_core_step?: number;
  add_mem_min?: number;
  add_mem_max?: number;
  add_mem_step?: number;
  add_disk_min?: number;
  add_disk_max?: number;
  add_disk_step?: number;
  add_bw_min?: number;
  add_bw_max?: number;
  add_bw_step?: number;
  active?: boolean;
  visible?: boolean;
  capacity_remaining?: number;
  sort_order?: number;
}

export interface Product {
  id?: number;
  goods_type_id?: number;
  plan_group_id?: number;
  product_id?: number;
  name?: string;
  cores?: number;
  memory_gb?: number;
  disk_gb?: number;
  bandwidth_mbps?: number;
  cpu_model?: string;
  monthly_price?: number;
  port_num?: number;
  sort_order?: number;
  active?: boolean;
  visible?: boolean;
  capacity_remaining?: number;
}

export interface Package extends Product {}

export interface SystemImage {
  id?: number;
  line_id?: number;
  plan_group_id?: number;
  image_id?: number;
  name?: string;
  type?: string;
  enabled?: boolean;
}

export interface BillingCycle {
  id?: number;
  name?: string;
  months?: number;
  multiplier?: number;
  min_qty?: number;
  max_qty?: number;
  active?: boolean;
  sort_order?: number;
}

export interface CartSpec {
  add_cores?: number;
  add_mem_gb?: number;
  add_disk_gb?: number;
  add_bw_mbps?: number;
  billing_cycle_id?: number;
  cycle_qty?: number;
  duration_months?: number;
}

export interface CartItem {
  id?: number;
  user_id?: number;
  package_id?: number;
  system_id?: number;
  spec?: CartSpec;
  qty?: number;
  amount?: number;
  created_at?: string;
  updated_at?: string;
}

export interface CartItemRequest {
  package_id?: number;
  system_id?: number;
  spec?: CartSpec;
  qty?: number;
}

export interface Order {
  id?: number;
  user_id?: number;
  order_no?: string;
  status?: string;
  total_amount?: number;
  currency?: string;
  idempotency_key?: string;
  pending_reason?: string;
  approved_by?: number;
  approved_at?: string;
  rejected_reason?: string;
  created_at?: string;
  updated_at?: string;
}

export interface OrderItem {
  id?: number;
  order_id?: number;
  package_id?: number;
  system_id?: number;
  spec?: Record<string, unknown>;
  qty?: number;
  amount?: number;
  status?: string;
  automation_instance_id?: string;
  action?: string;
  duration_months?: number;
  created_at?: string;
  updated_at?: string;
}

export interface OrderPayment {
  id?: number;
  order_id?: number;
  user_id?: number;
  method?: string;
  amount?: number;
  currency?: string;
  trade_no?: string;
  note?: string;
  screenshot_url?: string;
  status?: string;
  idempotency_key?: string;
  reviewed_by?: number;
  review_reason?: string;
  created_at?: string;
  updated_at?: string;
}

export interface OrderDetailResponse {
  order?: Order;
  items?: OrderItem[];
  payments?: OrderPayment[];
}

export interface OrderCreateRequest {
  items?: Array<{
    package_id?: number;
    system_id?: number;
    spec?: CartSpec;
    qty?: number;
  }>;
}

export interface OrderCreateResponse {
  order?: Order;
  items?: OrderItem[];
}

export interface PaymentRequest {
  method: string;
  amount: number;
  currency?: string;
  trade_no?: string;
  note?: string;
  screenshot_url?: string;
}

export interface PaymentProvider {
  key?: string;
  name?: string;
  enabled?: boolean;
  schema_json?: string;
  config_json?: string;
  balance?: number;
}

export interface PaymentCreateResult {
  status?: string;
  pay_url?: string;
  trade_no?: string;
  extra?: Record<string, unknown>;
  paid?: boolean;
}

export interface WalletInfo {
  balance?: number;
  currency?: string;
}

export interface WalletOrder {
  id?: number;
  user_id?: number;
  type?: string;
  amount?: number;
  currency?: string;
  status?: string;
  note?: string;
  meta?: Record<string, unknown>;
  reviewed_by?: number;
  review_reason?: string;
  created_at?: string;
  updated_at?: string;
}

export interface WalletOrderCreateRequest {
  amount: number;
  currency?: string;
  note?: string;
  meta?: Record<string, unknown>;
}

export interface WalletOrderListResponse {
  items?: WalletOrder[];
  total?: number;
}

export interface WalletTransaction {
  id?: number;
  type?: string;
  amount?: number;
  note?: string;
  created_at?: string;
}

export interface Notification {
  id?: number;
  user_id?: number;
  type?: string;
  title?: string;
  content?: string;
  status?: string;
  created_at?: string;
  read_at?: string;
}

export interface UnreadCountResponse {
  unread?: number;
}

export interface ServerStatus {
  hostname?: string;
  os?: string;
  platform?: string;
  kernel_version?: string;
  uptime_seconds?: number;
  cpu_model?: string;
  cpu_cores?: number;
  cpu_usage_percent?: number;
  mem_total?: number;
  mem_used?: number;
  mem_usage_percent?: number;
  disk_total?: number;
  disk_used?: number;
  disk_usage_percent?: number;
  status?: string;
}

export interface ProbeSnapshot {
  system?: Record<string, unknown>;
  cpu?: Record<string, unknown>;
  memory?: Record<string, unknown>;
  disks?: Record<string, unknown>[];
  ports?: Record<string, unknown>[];
  raw?: Record<string, unknown>;
}

export interface ProbeNode {
  id?: number;
  name?: string;
  agent_id?: string;
  status?: string;
  os_type?: string;
  tags?: string[];
  last_heartbeat_at?: string;
  last_snapshot_at?: string;
  snapshot?: ProbeSnapshot;
  created_at?: string;
  updated_at?: string;
}

export interface ProbeStatusEvent {
  id?: number;
  probe_id?: number;
  status?: string;
  at?: string;
  reason?: string;
  created_at?: string;
}

export interface ProbeSLA {
  window_from?: string;
  window_to?: string;
  total_seconds?: number;
  online_seconds?: number;
  uptime_percent?: number;
  events?: ProbeStatusEvent[];
}

export interface ProbeLogSession {
  id?: number;
  probe_id?: number;
  operator_id?: number;
  source?: string;
  status?: string;
  started_at?: string;
  ended_at?: string;
  created_at?: string;
}

export interface VPSInstance {
  id?: number;
  user_id?: number;
  order_item_id?: number;
  goods_type_id?: number;
  automation_instance_id?: string;
  name?: string;
  region?: string;
  region_id?: number;
  line_id?: number;
  package_id?: number;
  package_name?: string;
  cpu?: number;
  memory_gb?: number;
  disk_gb?: number;
  bandwidth_mbps?: number;
  port_num?: number;
  spec?: Record<string, unknown>;
  system_id?: number;
  status?: string;
  automation_state?: number;
  admin_status?: string;
  expire_at?: string;
  destroy_at?: string;
  destroy_in_days?: number;
  panel_url_cache?: string;
  access_info?: Record<string, unknown>;
  last_emergency_renew_at?: string;
  created_at?: string;
  updated_at?: string;
  monthly_price?: number;
}

export interface MonitorResponse {
  cpu?: number;
  memory?: number;
  bytes_in?: number;
  bytes_out?: number;
  storage?: number;
}

export interface RevenuePoint {
  date?: string;
  amount?: number;
}

export interface StatusPoint {
  status?: string;
  count?: number;
}

export interface SettingItem {
  key?: string;
  value?: string;
  value_json?: string;
  created_at?: string;
}

export interface AutomationConfig {
  base_url?: string;
  api_key?: string;
  enabled?: boolean;
  timeout_sec?: number;
  retry?: number;
  dry_run?: boolean;
}

export interface AutomationSyncLog {
  id?: number;
  status?: string;
  message?: string;
  created_at?: string;
}

export interface GoodsType {
  id?: number;
  code?: string;
  name?: string;
  active?: boolean;
  sort_order?: number;
  automation_category?: string;
  automation_plugin_id?: string;
  automation_instance_id?: string;
  created_at?: string;
  updated_at?: string;
}

export interface DashboardOverview {
  total_revenue?: number;
  today_revenue?: number;
  pending_orders?: number;
  provisioning?: number;
  failed?: number;
  vps_total?: number;
  expiring?: number;
}

export interface DashboardRevenue {
  granularity?: string;
  points?: RevenuePoint[];
}

export interface DashboardStatus {
  points?: StatusPoint[];
}

export interface UserDashboard {
  orders?: number;
  vps?: number;
  expiring?: number;
  pending_review?: number;
  spend_30d?: number;
}

export interface RobotConfig {
  webhooks?: RobotWebhook[];
}

export interface RobotWebhook {
  name?: string;
  url?: string;
  secret?: string;
  enabled?: boolean;
  events?: string[];
}

export interface SMTPConfig {
  host?: string;
  port?: string;
  user?: string;
  pass?: string;
  from?: string;
  enabled?: boolean;
}

export interface SMSConfig {
  enabled?: boolean;
  plugin_id?: string;
  instance_id?: string;
  default_template_id?: string;
  provider_template_id?: string;
}

export interface SMSTemplate {
  id?: number;
  name?: string;
  content?: string;
  enabled?: boolean;
  created_at?: string;
  updated_at?: string;
}

// 管理员相关
export interface AdminUser {
  id?: number;
  username?: string;
  email?: string;
  qq?: string;
  avatar?: string;
  permission_group_id?: number;
  permission_group_name?: string;
  permissions?: string[];
  status?: string;
  created_at?: string;
  updated_at?: string;
}

export interface PermissionItem {
  code?: string;
  name?: string;
  friendly_name?: string;
  category?: string;
  parent_code?: string;
  sort_order?: number;
  children?: PermissionItem[];
}

export interface PermissionGroup {
  id?: number;
  name?: string;
  description?: string;
  permissions?: string[];
  created_at?: string;
  updated_at?: string;
}

export interface AdminProfile {
  id?: number;
  username?: string;
  email?: string;
  qq?: string;
  avatar?: string;
  permission_group_name?: string;
  permissions?: string[];
  created_at?: string;
}

export interface SiteSetting {
  key?: string;
  value?: string;
}

export interface CMSBlock {
  id?: number;
  page?: string;
  type?: string;
  title?: string;
  subtitle?: string;
  content_json?: string;
  custom_html?: string;
  lang?: string;
  visible?: boolean;
  sort_order?: number;
  created_at?: string;
  updated_at?: string;
}

// CMS Block Content Types
export interface CMSBlockContent {
  [key: string]: unknown;
}

// Hero Block Content
export interface HeroBlockContent extends CMSBlockContent {
  kicker?: string;
  title?: string;
  subtitle?: string;
  primary_cta_text?: string;
  primary_cta_url?: string;
  secondary_cta_text?: string;
  secondary_cta_url?: string;
  stats?: Array<{ value: string; label: string }>;
  media_url?: string;
}

// Product List Block Content
export interface ProductListBlockContent extends CMSBlockContent {
  items?: Array<{
    name: string;
    price: string;
    unit?: string;
    description: string;
    tags?: string[];
    cta_url?: string;
    cta_text?: string;
  }>;
}

// Feature Cards Block Content
export interface FeatureCardsBlockContent extends CMSBlockContent {
  items?: Array<{
    title: string;
    description: string;
    icon?: string;
    color?: string;
  }>;
}

// Announcement/Doc List Block Content
export interface ListBlockContent extends CMSBlockContent {
  items?: Array<{
    title: string;
    summary?: string;
    description?: string;
    url?: string;
  }>;
}

// Activity Banner Block Content
export interface ActivityBannerBlockContent extends CMSBlockContent {
  kicker?: string;
  title?: string;
  subtitle?: string;
  cta_text?: string;
  cta_url?: string;
}

// Custom HTML Block Content
export interface CustomHTMLBlockContent extends CMSBlockContent {
  html?: string;
}

// 3D Hero Block Content
export interface Hero3DBlockContent extends CMSBlockContent {
  badge?: string;
  title_lines?: string[];
  description_lines?: string[];
  buttons?: Array<{ text: string; url?: string; type?: string; size?: string }>;
  trust_badges?: string[];
  card1_icon?: string;
  card1_label?: string;
  card1_value?: string;
  card2_icon?: string;
  card2_label?: string;
  card2_value?: string;
  card3_icon?: string;
  card3_label?: string;
  card3_value?: string;
  ring_value?: string;
  ring_label?: string;
}

// Stats Bar Block Content
export interface StatsBarBlockContent extends CMSBlockContent {
  stats?: Array<{
    icon: string;
    value: string;
    unit: string;
    label: string;
    gradient?: string;
    chart?: string[];
  }>;
}

// Product Cards Block Content
export interface ProductCardsBlockContent extends CMSBlockContent {
  products?: Array<{
    emoji?: string;
    name: string;
    desc: string;
    gradient?: string;
    features?: string[];
    link?: string;
    cta_text?: string;
  }>;
}

// Feature Metrics Block Content
export interface FeatureMetricsBlockContent extends CMSBlockContent {
  features?: Array<{
    icon: string;
    title: string;
    desc: string;
    gradient?: string;
    metrics?: Array<{ value: string; label: string }>;
  }>;
}

// Solutions Tabs Block Content
export interface SolutionsTabsBlockContent extends CMSBlockContent {
  solutions?: Array<{
    icon: string;
    name: string;
    title: string;
    desc: string;
    items?: string[];
    link?: string;
    cta_text?: string;
    cards?: Array<{ icon: string; title: string; value: string }>;
  }>;
}

// Customers Block Content
export interface CustomersBlockContent extends CMSBlockContent {
  logos?: Array<{ text: string }>;
  stats?: Array<{ value: string; label: string }>;
}

// CTA Gift Block Content
export interface CTAGiftBlockContent extends CMSBlockContent {
  badge?: string;
  title?: string;
  currency?: string;
  amount?: string;
  unit?: string;
  desc?: string;
  gradient?: string;
  buttons?: Array<{ text: string; url?: string; type?: string; size?: string }>;
}

export interface CMSCategory {
  id?: number;
  key?: string;
  name?: string;
  lang?: string;
  sort_order?: number;
  visible?: boolean;
  created_at?: string;
  updated_at?: string;
}

export interface CMSPost {
  id?: number;
  category_id?: number;
  title?: string;
  slug?: string;
  summary?: string;
  content_html?: string;
  cover_url?: string;
  lang?: string;
  status?: string;
  pinned?: boolean;
  sort_order?: number;
  published_at?: string;
  created_at?: string;
  updated_at?: string;
}

export interface CMSBlockListResponse {
  items?: CMSBlock[];
}

export interface CMSPostListResponse {
  items?: CMSPost[];
  total?: number;
}

export interface UploadItem {
  id?: number;
  name?: string;
  path?: string;
  url?: string;
  mime?: string;
  size?: number;
  uploader_id?: number;
  created_at?: string;
}

export interface UploadListResponse {
  items?: UploadItem[];
  total?: number;
}

export interface TicketResource {
  id?: number;
  ticket_id?: number;
  resource_type?: string;
  resource_id?: number;
  resource_name?: string;
  created_at?: string;
}

export interface TicketMessage {
  id?: number;
  ticket_id?: number;
  sender_id?: number;
  sender_role?: string;
  sender_name?: string;
  sender_avatar?: string;
  sender_qq?: string;
  role?: string;
  content?: string;
  created_at?: string;
}

export interface Ticket {
  id?: number;
  user_id?: number;
  subject?: string;
  status?: string;
  resource_count?: number;
  last_reply_role?: string;
  created_at?: string;
  updated_at?: string;
}

export interface TicketDetailResponse {
  ticket?: Ticket;
  messages?: TicketMessage[];
  resources?: TicketResource[];
}

export interface RealNameVerification {
  id?: number;
  user_id?: number;
  real_name?: string;
  id_number?: string;
  status?: string;
  provider?: string;
  reason?: string;
  created_at?: string;
  verified_at?: string;
}

export interface RealNameStatusResponse {
  enabled?: boolean;
  provider?: string;
  block_actions?: string[];
  verified?: boolean;
  verification?: RealNameVerification;
}

export interface RealNameConfig {
  enabled?: boolean;
  provider?: string;
  block_actions?: string[];
}

export interface RealNameProvider {
  key?: string;
  name?: string;
}

export interface RealNameRecordListResponse {
  items?: RealNameVerification[];
  total?: number;
}

export interface DebugStatusResponse {
  enabled?: boolean;
}

export interface AdminAuditLog {
  id?: number;
  admin_id?: number;
  action?: string;
  target_type?: string;
  target_id?: string;
  detail?: Record<string, unknown>;
  created_at?: string;
}

export interface AutomationLog {
  id?: number;
  order_id?: number;
  order_item_id?: number;
  action?: string;
  request_json?: unknown;
  response_json?: unknown;
  success?: boolean;
  message?: string;
  created_at?: string;
}

export interface IntegrationSyncLog {
  id?: number;
  target?: string;
  mode?: string;
  status?: string;
  message?: string;
  created_at?: string;
}

export interface DebugLogList<T> {
  items?: T[];
  total?: number;
}

export interface DebugLogsResponse {
  audit_logs?: DebugLogList<AdminAuditLog>;
  automation_logs?: DebugLogList<AutomationLog>;
  sync_logs?: DebugLogList<IntegrationSyncLog>;
}

export interface OrderEvent {
  id?: number;
  order_id?: number;
  seq?: number;
  type?: string;
  data?: Record<string, unknown>;
  created_at?: string;
}

export interface OrderDetailWithEventsResponse {
  order?: Order;
  items?: OrderItem[];
  payments?: OrderPayment[];
  events?: OrderEvent[];
}
