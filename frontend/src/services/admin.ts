import { http } from "./http";
import type {
  AdminProfile,
  AdminUser,
  ApiList,
  AutomationConfig,
  AutomationSyncLog,
  BillingCycle,
  DashboardOverview,
  DashboardRevenue,
  DashboardStatus,
  RevenueAnalyticsQuery,
  RevenueAnalyticsOverviewResponse,
  RevenueAnalyticsTrendPoint,
  RevenueAnalyticsTopItem,
  RevenueAnalyticsDetailsResponse,
  CMSBlock,
  CMSCategory,
  CMSPost,
  CMSPostListResponse,
  Line,
  Order,
  OrderDetailResponse,
  Package,
  PackageCapabilities,
  PermissionItem,
  PermissionGroup,
  PaymentProvider,
  RealNameConfig,
  RealNameProvider,
  RealNameRecordListResponse,
  Region,
  RobotConfig,
  ServerStatus,
  ProbeNode,
  ProbeSLA,
  ProbeLogSession,
  SMTPConfig,
  SMSConfig,
  SMSTemplate,
  SettingItem,
  SystemImage,
  Ticket,
  TicketDetailResponse,
  UserTierAutoRule,
  UserTierDiscountRule,
  UserTierGroup,
  UploadItem,
  UploadListResponse,
  User,
  VPSInstance,
  WalletOrderListResponse,
  WalletOrder,
  DebugStatusResponse,
  DebugLogsResponse,
  PluginListItem,
  PluginDiscoverItem,
  PluginPaymentMethodItem,
  GoodsType,
  Coupon,
  CouponProductGroup
} from "./types";

export const adminLogin = (payload: Record<string, unknown>) => http.post("/admin/api/v1/auth/login", payload);
export const admin2FAUnlock = (payload: { totp_code: string }) => http.post("/admin/api/v1/auth/2fa/unlock", payload);
export const adminSetupTwoFA = (payload: { password?: string; current_code?: string }) =>
  http.post("/admin/api/v1/auth/2fa/setup", payload);
export const adminConfirmTwoFA = (payload: { code: string }) =>
  http.post("/admin/api/v1/auth/2fa/confirm", payload);

export const listAdminUsers = (params?: Record<string, unknown>) => http.get<ApiList<User>>("/admin/api/v1/users", { params });
export const getAdminUserDetail = (id: number | string) => http.get<User>(`/admin/api/v1/users/${id}`);
export const createAdminUser = (payload: Record<string, unknown>) => http.post("/admin/api/v1/users", payload);
export const updateAdminUser = (id: number | string, payload: Record<string, unknown>) =>
  http.patch(`/admin/api/v1/users/${id}`, payload);
export const updateUserStatus = (id: number | string, payload: Record<string, unknown>) =>
  http.patch(`/admin/api/v1/users/${id}/status`, payload);
export const updateAdminUserRealNameStatus = (id: number | string, payload: Record<string, unknown>) =>
  http.patch(`/admin/api/v1/users/${id}/realname-status`, payload);
export const adminImpersonateUser = (id: number | string) =>
  http.post(`/admin/api/v1/users/${id}/impersonate`);
export const resetUserPassword = (id: number | string, payload: Record<string, unknown>) =>
  http.post(`/admin/api/v1/users/${id}/reset-password`, payload);
export const setAdminUserTier = (id: number | string, payload: { group_id: number; expire_at?: string }) =>
  http.patch(`/admin/api/v1/users/${id}/tier`, payload);

export const listUserTierGroups = () => http.get<ApiList<UserTierGroup>>("/admin/api/v1/user-tiers");
export const createUserTierGroup = (payload: Record<string, unknown>) => http.post<UserTierGroup>("/admin/api/v1/user-tiers", payload);
export const updateUserTierGroup = (id: number | string, payload: Record<string, unknown>) =>
  http.patch<UserTierGroup>(`/admin/api/v1/user-tiers/${id}`, payload);
export const deleteUserTierGroup = (id: number | string) => http.delete(`/admin/api/v1/user-tiers/${id}`);
export const rebuildUserTierCaches = (id?: number | string) =>
  http.post(id ? `/admin/api/v1/user-tiers/${id}/rebuild` : "/admin/api/v1/user-tiers/rebuild");

export const listUserTierDiscountRules = (groupId: number | string) =>
  http.get<ApiList<UserTierDiscountRule>>(`/admin/api/v1/user-tiers/${groupId}/discount-rules`);
export const createUserTierDiscountRule = (groupId: number | string, payload: Record<string, unknown>) =>
  http.post<UserTierDiscountRule>(`/admin/api/v1/user-tiers/${groupId}/discount-rules`, payload);
export const updateUserTierDiscountRule = (groupId: number | string, ruleId: number | string, payload: Record<string, unknown>) =>
  http.patch<UserTierDiscountRule>(`/admin/api/v1/user-tiers/${groupId}/discount-rules/${ruleId}`, payload);
export const deleteUserTierDiscountRule = (groupId: number | string, ruleId: number | string) =>
  http.delete(`/admin/api/v1/user-tiers/${groupId}/discount-rules/${ruleId}`);

export const listUserTierAutoRules = (groupId: number | string) =>
  http.get<ApiList<UserTierAutoRule>>(`/admin/api/v1/user-tiers/${groupId}/auto-rules`);
export const createUserTierAutoRule = (groupId: number | string, payload: Record<string, unknown>) =>
  http.post<UserTierAutoRule>(`/admin/api/v1/user-tiers/${groupId}/auto-rules`, payload);
export const updateUserTierAutoRule = (groupId: number | string, ruleId: number | string, payload: Record<string, unknown>) =>
  http.patch<UserTierAutoRule>(`/admin/api/v1/user-tiers/${groupId}/auto-rules/${ruleId}`, payload);
export const deleteUserTierAutoRule = (groupId: number | string, ruleId: number | string) =>
  http.delete(`/admin/api/v1/user-tiers/${groupId}/auto-rules/${ruleId}`);

export const listCouponGroups = () => http.get<ApiList<CouponProductGroup>>("/admin/api/v1/coupon-groups");
export const createCouponGroup = (payload: Record<string, unknown>) =>
  http.post<CouponProductGroup>("/admin/api/v1/coupon-groups", payload);
export const updateCouponGroup = (id: number | string, payload: Record<string, unknown>) =>
  http.patch<CouponProductGroup>(`/admin/api/v1/coupon-groups/${id}`, payload);
export const deleteCouponGroup = (id: number | string) =>
  http.delete(`/admin/api/v1/coupon-groups/${id}`);

export const listCoupons = (params?: Record<string, unknown>) =>
  http.get<ApiList<Coupon>>("/admin/api/v1/coupons", { params });
export const createCoupon = (payload: Record<string, unknown>) =>
  http.post<Coupon>("/admin/api/v1/coupons", payload);
export const updateCoupon = (id: number | string, payload: Record<string, unknown>) =>
  http.patch<Coupon>(`/admin/api/v1/coupons/${id}`, payload);
export const deleteCoupon = (id: number | string) =>
  http.delete(`/admin/api/v1/coupons/${id}`);
export const batchGenerateCoupons = (payload: Record<string, unknown>) =>
  http.post<ApiList<Coupon>>("/admin/api/v1/coupons/batch-generate", payload);

export const listAdminOrders = (params?: Record<string, unknown>) => http.get<ApiList<Order>>("/admin/api/v1/orders", { params });
export const getAdminOrderDetail = (id: number | string) => http.get<OrderDetailResponse>(`/admin/api/v1/orders/${id}`);
export const approveAdminOrder = (id: number | string) => http.post(`/admin/api/v1/orders/${id}/approve`);
export const rejectAdminOrder = (id: number | string, payload: Record<string, unknown>) =>
  http.post(`/admin/api/v1/orders/${id}/reject`, payload);
export const deleteAdminOrder = (id: number | string) => http.delete(`/admin/api/v1/orders/${id}`);
export const markPaidAdminOrder = (id: number | string) => http.post(`/admin/api/v1/orders/${id}/mark-paid`);
export const retryAdminOrder = (id: number | string) => http.post(`/admin/api/v1/orders/${id}/retry`);
export const listAdminScheduledTasks = () => http.get<ApiList<Record<string, unknown>>>("/admin/api/v1/scheduled-tasks");
export const updateAdminScheduledTask = (key: string, payload: Record<string, unknown>) =>
  http.patch(`/admin/api/v1/scheduled-tasks/${key}`, payload);

export const listAdminVps = (params?: Record<string, unknown>) => http.get<ApiList<VPSInstance>>("/admin/api/v1/vps", { params });
export const createAdminVps = (payload: Record<string, unknown>) => http.post<VPSInstance>("/admin/api/v1/vps", payload);
export const getAdminVpsDetail = (id: number | string) => http.get<VPSInstance>(`/admin/api/v1/vps/${id}`);
export const lockAdminVps = (id: number | string) => http.post(`/admin/api/v1/vps/${id}/lock`);
export const unlockAdminVps = (id: number | string) => http.post(`/admin/api/v1/vps/${id}/unlock`);
export const deleteAdminVps = (id: number | string, payload?: { reason?: string }) =>
  http.post(`/admin/api/v1/vps/${id}/delete`, payload || {});
export const resizeAdminVps = (id: number | string, payload: Record<string, unknown>) =>
  http.post(`/admin/api/v1/vps/${id}/resize`, payload);
export const refreshAdminVps = (id: number | string) => http.post(`/admin/api/v1/vps/${id}/refresh`);
export const updateAdminVps = (id: number | string, payload: Record<string, unknown>) =>
  http.patch(`/admin/api/v1/vps/${id}`, payload);
export const updateAdminVpsStatus = (id: number | string, payload: Record<string, unknown>) =>
  http.post(`/admin/api/v1/vps/${id}/status`, payload);
export const emergencyRenewAdminVps = (id: number | string, payload: Record<string, unknown>) =>
  http.post(`/admin/api/v1/vps/${id}/emergency-renew`, payload);
export const updateAdminVpsExpire = (id: number | string, payload: Record<string, unknown>) =>
  http.patch(`/admin/api/v1/vps/${id}/expire-at`, payload);

export const listRegions = (params?: Record<string, unknown>) => http.get<ApiList<Region>>("/admin/api/v1/regions", { params });
export const createRegion = (payload: Record<string, unknown>) => http.post("/admin/api/v1/regions", payload);
export const updateRegion = (id: number | string, payload: Record<string, unknown>) =>
  http.patch(`/admin/api/v1/regions/${id}`, payload);
export const deleteRegion = (id: number | string) => http.delete(`/admin/api/v1/regions/${id}`);
export const bulkDeleteRegions = (ids: Array<number | string>) =>
  http.post("/admin/api/v1/regions/bulk-delete", { ids });

export const listLines = (params?: Record<string, unknown>) => http.get<ApiList<Line>>("/admin/api/v1/lines", { params });
export const createLine = (payload: Record<string, unknown>) => http.post("/admin/api/v1/lines", payload);
export const updateLine = (id: number | string, payload: Record<string, unknown>) => http.patch(`/admin/api/v1/lines/${id}`, payload);
export const deleteLine = (id: number | string) => http.delete(`/admin/api/v1/lines/${id}`);
export const bulkDeleteLines = (ids: Array<number | string>) =>
  http.post("/admin/api/v1/lines/bulk-delete", { ids });

export const listPlanGroups = (params?: Record<string, unknown>) => http.get<ApiList<Line>>("/admin/api/v1/plan-groups", { params });
export const createPlanGroup = (payload: Record<string, unknown>) => http.post("/admin/api/v1/plan-groups", payload);
export const updatePlanGroup = (id: number | string, payload: Record<string, unknown>) =>
  http.patch(`/admin/api/v1/plan-groups/${id}`, payload);
export const deletePlanGroup = (id: number | string) => http.delete(`/admin/api/v1/plan-groups/${id}`);
export const bulkDeletePlanGroups = (ids: Array<number | string>) =>
  http.post("/admin/api/v1/plan-groups/bulk-delete", { ids });

export const listPackages = (params?: Record<string, unknown>) => http.get<ApiList<Package>>("/admin/api/v1/packages", { params });
export const createPackage = (payload: Record<string, unknown>) => http.post("/admin/api/v1/packages", payload);
export const updatePackage = (id: number | string, payload: Record<string, unknown>) =>
  http.patch(`/admin/api/v1/packages/${id}`, payload);
export const deletePackage = (id: number | string) => http.delete(`/admin/api/v1/packages/${id}`);
export const bulkDeletePackages = (ids: Array<number | string>) =>
  http.post("/admin/api/v1/packages/bulk-delete", { ids });
export const getPackageCapabilities = (id: number | string) =>
  http.get<PackageCapabilities>(`/admin/api/v1/packages/${id}/capabilities`);
export const updatePackageCapabilities = (id: number | string, payload: { resize_enabled?: boolean | null; refund_enabled?: boolean | null }) =>
  http.patch(`/admin/api/v1/packages/${id}/capabilities`, payload);

export const listBillingCycles = () => http.get<ApiList<BillingCycle>>("/admin/api/v1/billing-cycles");
export const createBillingCycle = (payload: Record<string, unknown>) => http.post("/admin/api/v1/billing-cycles", payload);
export const updateBillingCycle = (id: number | string, payload: Record<string, unknown>) =>
  http.patch(`/admin/api/v1/billing-cycles/${id}`, payload);
export const deleteBillingCycle = (id: number | string) => http.delete(`/admin/api/v1/billing-cycles/${id}`);
export const bulkDeleteBillingCycles = (ids: Array<number | string>) =>
  http.post("/admin/api/v1/billing-cycles/bulk-delete", { ids });

export const listSystemImages = (params?: Record<string, unknown>) =>
  http.get<ApiList<SystemImage>>("/admin/api/v1/system-images", { params });
export const createSystemImage = (payload: Record<string, unknown>) => http.post("/admin/api/v1/system-images", payload);
export const updateSystemImage = (id: number | string, payload: Record<string, unknown>) =>
  http.patch(`/admin/api/v1/system-images/${id}`, payload);
export const deleteSystemImage = (id: number | string) => http.delete(`/admin/api/v1/system-images/${id}`);
export const bulkDeleteSystemImages = (ids: Array<number | string>) =>
  http.post("/admin/api/v1/system-images/bulk-delete", { ids });
export const setLineSystemImages = (id: number | string, payload: { image_ids: Array<number | string> }) =>
  http.post(`/admin/api/v1/lines/${id}/system-images`, payload);
export const syncSystemImages = (params?: Record<string, unknown>) =>
  http.post("/admin/api/v1/system-images/sync", null, { params });

export const listApiKeys = (params?: Record<string, unknown>) => http.get<ApiList<Record<string, unknown>>>("/admin/api/v1/api-keys", { params });
export const createApiKey = (payload: Record<string, unknown>) => http.post("/admin/api/v1/api-keys", payload);
export const updateApiKeyStatus = (id: number | string, payload: Record<string, unknown>) =>
  http.patch(`/admin/api/v1/api-keys/${id}`, payload);

export const listSettings = () => http.get<ApiList<SettingItem>>("/admin/api/v1/settings");
export const updateSetting = (payload: Record<string, unknown>) => http.patch("/admin/api/v1/settings", payload);

export const listAdminPaymentProviders = () => http.get<ApiList<PaymentProvider>>("/admin/api/v1/payments/providers");
export const updateAdminPaymentProvider = (key: string, payload: Record<string, unknown>) =>
  http.patch(`/admin/api/v1/payments/providers/${key}`, payload);

export const listEmailTemplates = () => http.get<ApiList<Record<string, unknown>>>("/admin/api/v1/email-templates");
export const upsertEmailTemplate = (payload: Record<string, unknown>) => http.post("/admin/api/v1/email-templates", payload);
export const updateEmailTemplate = (id: number | string, payload: Record<string, unknown>) =>
  http.patch(`/admin/api/v1/email-templates/${id}`, payload);
export const deleteEmailTemplate = (id: number | string) => http.delete(`/admin/api/v1/email-templates/${id}`);

export const listAuditLogs = (params?: Record<string, unknown>) => http.get<ApiList<Record<string, unknown>>>("/admin/api/v1/audit-logs", { params });

export const getDebugStatus = () => http.get<DebugStatusResponse>("/admin/api/v1/debug/status");
export const updateDebugStatus = (payload: { enabled: boolean }) =>
  http.patch("/admin/api/v1/debug/status", payload);
export const getDebugLogs = (params?: Record<string, unknown>) =>
  http.get<DebugLogsResponse>("/admin/api/v1/debug/logs", { params });

// 钱包订单
export const listAdminWalletOrders = (params?: Record<string, unknown>) =>
  http.get<WalletOrderListResponse>("/admin/api/v1/wallet/orders", { params });
export const approveAdminWalletOrder = (id: number | string) =>
  http.post<{ order?: WalletOrder }>("/admin/api/v1/wallet/orders/{id}/approve".replace("{id}", String(id)));
export const rejectAdminWalletOrder = (id: number | string, payload?: { reason?: string }) =>
  http.post(`/admin/api/v1/wallet/orders/{id}/reject`.replace("{id}", String(id)), payload || {});
export const getAdminWalletInfo = (userId: number | string) =>
  http.get(`/admin/api/v1/wallets/${userId}`);
export const listAdminWalletTransactions = (userId: number | string, params?: Record<string, unknown>) =>
  http.get(`/admin/api/v1/wallets/${userId}/transactions`, { params });

// 工单
export const listAdminTickets = (params?: Record<string, unknown>) =>
  http.get<ApiList<Ticket>>("/admin/api/v1/tickets", { params });
export const getAdminTicketDetail = (id: number | string) =>
  http.get<TicketDetailResponse>(`/admin/api/v1/tickets/${id}`);
export const updateAdminTicket = (id: number | string, payload: Record<string, unknown>) =>
  http.patch<Ticket>(`/admin/api/v1/tickets/${id}`, payload);
export const addAdminTicketMessage = (id: number | string, payload: Record<string, unknown>) =>
  http.post(`/admin/api/v1/tickets/${id}/messages`, payload);
export const deleteAdminTicket = (id: number | string) => http.delete(`/admin/api/v1/tickets/${id}`);

export const getAutomationConfig = () => http.get<AutomationConfig>("/admin/api/v1/integrations/automation");
export const updateAutomationConfig = (payload: Record<string, unknown>) =>
  http.patch("/admin/api/v1/integrations/automation", payload);
export const syncAutomationCatalog = () => http.post("/admin/api/v1/integrations/automation/sync");
export const listAutomationSyncLogs = () => http.get<ApiList<AutomationSyncLog>>("/admin/api/v1/integrations/automation/sync-logs");

export const getRobotConfig = () => http.get<RobotConfig>("/admin/api/v1/integrations/robot");
export const updateRobotConfig = (payload: Record<string, unknown>) => http.patch("/admin/api/v1/integrations/robot", payload);
export const testRobotWebhook = (payload: Record<string, unknown>) => http.post("/admin/api/v1/integrations/robot/test", payload);

// 实名认证
export const getRealNameConfig = () => http.get<RealNameConfig>("/admin/api/v1/realname/config");
export const updateRealNameConfig = (payload: RealNameConfig) => http.patch("/admin/api/v1/realname/config", payload);
export const listRealNameProviders = () => http.get<{ items: RealNameProvider[] }>("/admin/api/v1/realname/providers");
export const listRealNameRecords = (params?: Record<string, unknown>) =>
  http.get<RealNameRecordListResponse>("/admin/api/v1/realname/records", { params });
export const getSmtpConfig = () => http.get<SMTPConfig>("/admin/api/v1/integrations/smtp");
export const updateSmtpConfig = (payload: Record<string, unknown>) => http.patch("/admin/api/v1/integrations/smtp", payload);
export const testSmtpConfig = (payload: Record<string, unknown>) => http.post("/admin/api/v1/integrations/smtp/test", payload);
export const getSmsConfig = () => http.get<SMSConfig>("/admin/api/v1/integrations/sms");
export const updateSmsConfig = (payload: Record<string, unknown>) => http.patch("/admin/api/v1/integrations/sms", payload);
export const previewSmsConfig = (payload: Record<string, unknown>) => http.post<{ content?: string }>("/admin/api/v1/integrations/sms/preview", payload);
export const testSmsConfig = (payload: Record<string, unknown>) => http.post("/admin/api/v1/integrations/sms/test", payload);
export const listSmsTemplates = () => http.get<ApiList<SMSTemplate>>("/admin/api/v1/sms-templates");
export const upsertSmsTemplate = (payload: Record<string, unknown>) => http.post<SMSTemplate>("/admin/api/v1/sms-templates", payload);
export const updateSmsTemplate = (id: number | string, payload: Record<string, unknown>) =>
  http.patch<SMSTemplate>(`/admin/api/v1/sms-templates/${id}`, payload);
export const deleteSmsTemplate = (id: number | string) => http.delete(`/admin/api/v1/sms-templates/${id}`);

export const getAdminDashboardOverview = () => http.post<DashboardOverview>("/admin/api/v1/dashboard/overview");
export const getAdminDashboardRevenue = (params?: Record<string, unknown>) =>
  http.post<DashboardRevenue>("/admin/api/v1/dashboard/revenue", null, { params });
export const getAdminDashboardVpsStatus = () => http.get<DashboardStatus>("/admin/api/v1/dashboard/vps-status");
export const getServerStatus = () => http.get<ServerStatus>("/admin/api/v1/server/status");
export const getRevenueAnalyticsOverview = (payload: RevenueAnalyticsQuery) =>
  http.post<RevenueAnalyticsOverviewResponse>("/admin/api/v1/dashboard/revenue-analytics/overview", payload);
export const getRevenueAnalyticsTrend = (payload: RevenueAnalyticsQuery) =>
  http.post<{ items?: RevenueAnalyticsTrendPoint[] }>("/admin/api/v1/dashboard/revenue-analytics/trend", payload);
export const getRevenueAnalyticsTop = (payload: RevenueAnalyticsQuery) =>
  http.post<{ items?: RevenueAnalyticsTopItem[] }>("/admin/api/v1/dashboard/revenue-analytics/top", payload);
export const getRevenueAnalyticsDetails = (payload: RevenueAnalyticsQuery) =>
  http.post<RevenueAnalyticsDetailsResponse>("/admin/api/v1/dashboard/revenue-analytics/details", payload);
export const exportRevenueAnalyticsAudit = (payload: RevenueAnalyticsQuery) =>
  http.post("/admin/api/v1/dashboard/revenue-analytics/export", payload, { responseType: "blob" });

// Probes
export const listAdminProbes = (params?: Record<string, unknown>) =>
  http.get<ApiList<ProbeNode>>("/admin/api/v1/probes", { params });
export const createAdminProbe = (payload: Record<string, unknown>) =>
  http.post<{ probe?: ProbeNode; enroll_token?: string }>("/admin/api/v1/probes", payload);
export const getAdminProbeDetail = (id: number | string, params?: Record<string, unknown>) =>
  http.get<{ probe?: ProbeNode; online?: boolean }>(`/admin/api/v1/probes/${id}`, { params });
export const updateAdminProbe = (id: number | string, payload: Record<string, unknown>) =>
  http.patch(`/admin/api/v1/probes/${id}`, payload);
export const deleteAdminProbe = (id: number | string) =>
  http.delete(`/admin/api/v1/probes/${id}`);
export const resetAdminProbeEnrollToken = (id: number | string) =>
  http.post<{ enroll_token?: string }>(`/admin/api/v1/probes/${id}/enroll-token/reset`);
export const getAdminProbeSla = (id: number | string, params?: Record<string, unknown>) =>
  http.get<{ sla?: ProbeSLA }>(`/admin/api/v1/probes/${id}/sla`, { params });
export const adminProbePortCheck = (id: number | string, payload?: Record<string, unknown>) =>
  http.post<{ ok?: boolean; request_id?: string }>(`/admin/api/v1/probes/${id}/port-check`, payload || {});
export const createAdminProbeLogSession = (id: number | string, payload: Record<string, unknown>) =>
  http.post<{ session_id?: string; stream_path?: string; log_session?: ProbeLogSession }>(`/admin/api/v1/probes/${id}/log-sessions`, payload);

// 管理员管理
export const listAdmins = (params?: Record<string, unknown>) => http.get<ApiList<AdminUser>>("/admin/api/v1/admins", { params });
export const createAdmin = (payload: Record<string, unknown>) => http.post("/admin/api/v1/admins", payload);
export const updateAdmin = (id: number | string, payload: Record<string, unknown>) =>
  http.patch(`/admin/api/v1/admins/${id}`, payload);
export const updateAdminStatus = (id: number | string, payload: Record<string, unknown>) =>
  http.patch(`/admin/api/v1/admins/${id}/status`, payload);
export const deleteAdmin = (id: number | string) => http.delete(`/admin/api/v1/admins/${id}`);

// 权限组管理
export const listPermissions = () => http.get<ApiList<PermissionItem>>("/admin/api/v1/permissions/list");
export const listPermissionGroups = () => http.get<ApiList<PermissionGroup>>("/admin/api/v1/permission-groups");
export const createPermissionGroup = (payload: Record<string, unknown>) => http.post("/admin/api/v1/permission-groups", payload);
export const updatePermissionGroup = (id: number | string, payload: Record<string, unknown>) =>
  http.patch(`/admin/api/v1/permission-groups/${id}`, payload);
export const deletePermissionGroup = (id: number | string) => http.delete(`/admin/api/v1/permission-groups/${id}`);

// 管理员个人资料
export const getAdminProfile = () => http.get<AdminProfile>("/admin/api/v1/profile");
export const updateAdminProfile = (payload: Record<string, unknown>) => http.patch("/admin/api/v1/profile", payload);
export const changeAdminPassword = (payload: { old_password: string; new_password: string }) =>
  http.post("/admin/api/v1/profile/change-password", payload);

// CMS
export const listCmsCategories = (params?: Record<string, unknown>) =>
  http.get<ApiList<CMSCategory>>("/admin/api/v1/cms/categories", { params });
export const createCmsCategory = (payload: Record<string, unknown>) =>
  http.post<CMSCategory>("/admin/api/v1/cms/categories", payload);
export const updateCmsCategory = (id: number | string, payload: Record<string, unknown>) =>
  http.patch<CMSCategory>(`/admin/api/v1/cms/categories/${id}`, payload);
export const deleteCmsCategory = (id: number | string) => http.delete(`/admin/api/v1/cms/categories/${id}`);

export const listCmsPosts = (params?: Record<string, unknown>) =>
  http.get<CMSPostListResponse>("/admin/api/v1/cms/posts", { params });
export const createCmsPost = (payload: Record<string, unknown>) => http.post<CMSPost>("/admin/api/v1/cms/posts", payload);
export const updateCmsPost = (id: number | string, payload: Record<string, unknown>) =>
  http.patch<CMSPost>(`/admin/api/v1/cms/posts/${id}`, payload);
export const deleteCmsPost = (id: number | string) => http.delete(`/admin/api/v1/cms/posts/${id}`);

export const listCmsBlocks = (params?: Record<string, unknown>) =>
  http.get<ApiList<CMSBlock>>("/admin/api/v1/cms/blocks", { params });
export const createCmsBlock = (payload: Record<string, unknown>) => http.post<CMSBlock>("/admin/api/v1/cms/blocks", payload);
export const updateCmsBlock = (id: number | string, payload: Record<string, unknown>) =>
  http.patch<CMSBlock>(`/admin/api/v1/cms/blocks/${id}`, payload);
export const deleteCmsBlock = (id: number | string) => http.delete(`/admin/api/v1/cms/blocks/${id}`);

export const listUploads = (params?: Record<string, unknown>) =>
  http.get<UploadListResponse>("/admin/api/v1/uploads", { params });
export const uploadFile = (file: File) => {
  const formData = new FormData();
  formData.append("file", file);
  return http.post<UploadItem>("/admin/api/v1/uploads", formData, {
    headers: { "Content-Type": "multipart/form-data" }
  });
};

export const uploadPaymentPlugin = (file: File, password: string) => {
  const formData = new FormData();
  formData.append("file", file);
  formData.append("password", password);
  return http.post("/admin/api/v1/plugins/payment/upload", formData, {
    headers: { "Content-Type": "multipart/form-data" }
  });
};

// Plugins (new unified plugin system)
export const listAdminPlugins = () => http.get<{ items?: PluginListItem[] }>("/admin/api/v1/plugins");
export const discoverAdminPlugins = () => http.get<{ items?: PluginDiscoverItem[] }>("/admin/api/v1/plugins/discover");

export const installAdminPlugin = (file: File, adminPassword?: string) => {
  const formData = new FormData();
  formData.append("file", file);
  if (adminPassword) formData.append("admin_password", adminPassword);
  return http.post<{ ok?: boolean; plugin?: Record<string, unknown> }>("/admin/api/v1/plugins/install", formData, {
    headers: { "Content-Type": "multipart/form-data" }
  });
};

export const enableAdminPlugin = (category: string, pluginId: string) =>
  http.post<{ ok?: boolean }>(`/admin/api/v1/plugins/${category}/${pluginId}/enable`);

export const disableAdminPlugin = (category: string, pluginId: string) =>
  http.post<{ ok?: boolean }>(`/admin/api/v1/plugins/${category}/${pluginId}/disable`);

export const uninstallAdminPlugin = (category: string, pluginId: string) =>
  http.delete<{ ok?: boolean }>(`/admin/api/v1/plugins/${category}/${pluginId}`);

export const createAdminPluginInstance = (category: string, pluginId: string, payload: { instance_id?: string; config_json?: string }) =>
  http.post<{ ok?: boolean; plugin?: Record<string, unknown> }>(`/admin/api/v1/plugins/${category}/${pluginId}/instances`, payload || {});

export const enableAdminPluginInstance = (category: string, pluginId: string, instanceId: string) =>
  http.post<{ ok?: boolean }>(`/admin/api/v1/plugins/${category}/${pluginId}/${instanceId}/enable`);

export const disableAdminPluginInstance = (category: string, pluginId: string, instanceId: string) =>
  http.post<{ ok?: boolean }>(`/admin/api/v1/plugins/${category}/${pluginId}/${instanceId}/disable`);

export const deleteAdminPluginInstance = (category: string, pluginId: string, instanceId: string) =>
  http.delete<{ ok?: boolean }>(`/admin/api/v1/plugins/${category}/${pluginId}/${instanceId}`);

export const deleteAdminPluginFiles = (category: string, pluginId: string) =>
  http.delete<{ ok?: boolean }>(`/admin/api/v1/plugins/${category}/${pluginId}/files`);

export const importAdminPluginFromDisk = (category: string, pluginId: string, adminPassword?: string) =>
  http.post<{ ok?: boolean; plugin?: Record<string, unknown> }>(`/admin/api/v1/plugins/${category}/${pluginId}/import`, {
    admin_password: adminPassword || ""
  });

export const getAdminPluginConfigSchema = (category: string, pluginId: string) =>
  http.get<{ json_schema?: string; ui_schema?: string }>(`/admin/api/v1/plugins/${category}/${pluginId}/config/schema`);

export const getAdminPluginConfig = (category: string, pluginId: string) =>
  http.get<{ config_json?: string }>(`/admin/api/v1/plugins/${category}/${pluginId}/config`);

export const updateAdminPluginConfig = (category: string, pluginId: string, configJson: string) =>
  http.put<{ ok?: boolean }>(`/admin/api/v1/plugins/${category}/${pluginId}/config`, { config_json: configJson });

export const getAdminPluginInstanceConfigSchema = (category: string, pluginId: string, instanceId: string) =>
  http.get<{ json_schema?: string; ui_schema?: string }>(`/admin/api/v1/plugins/${category}/${pluginId}/${instanceId}/config/schema`);

export const getAdminPluginInstanceConfig = (category: string, pluginId: string, instanceId: string) =>
  http.get<{ config_json?: string }>(`/admin/api/v1/plugins/${category}/${pluginId}/${instanceId}/config`);

export const updateAdminPluginInstanceConfig = (category: string, pluginId: string, instanceId: string, configJson: string) =>
  http.put<{ ok?: boolean }>(`/admin/api/v1/plugins/${category}/${pluginId}/${instanceId}/config`, { config_json: configJson });

export const listAdminPluginPaymentMethods = (params: { category?: string; plugin_id: string; instance_id?: string }) =>
  http.get<{ items?: PluginPaymentMethodItem[] }>("/admin/api/v1/plugins/payment-methods", { params });

export const updateAdminPluginPaymentMethod = (payload: {
  category?: string;
  plugin_id: string;
  instance_id?: string;
  method: string;
  enabled: boolean;
}) => http.patch<{ ok?: boolean }>("/admin/api/v1/plugins/payment-methods", payload);

// Goods types
export const listGoodsTypes = () => http.get<{ items?: GoodsType[] }>("/admin/api/v1/goods-types");
export const createGoodsType = (payload: Record<string, unknown>) => http.post<GoodsType>("/admin/api/v1/goods-types", payload);
export const updateGoodsType = (id: number | string, payload: Record<string, unknown>) =>
  http.put(`/admin/api/v1/goods-types/${id}`, payload);
export const deleteGoodsType = (id: number | string) => http.delete(`/admin/api/v1/goods-types/${id}`);
export const syncGoodsTypeAutomation = (id: number | string, mode?: string) =>
  http.post(`/admin/api/v1/goods-types/${id}/sync-automation`, null, { params: { mode: mode || "merge" } });
export const getGoodsTypeAutomationOptions = (id: number | string) =>
  http.get<{
    line_items?: Array<{ id?: number; name?: string; area_id?: number; state?: number }>;
    product_type_items?: Array<{ id?: number; name?: string; area_id?: number; state?: number }>;
    package_items?: Array<{ line_id?: number; id?: number; name?: string }>;
    product_items?: Array<{ line_id?: number; id?: number; name?: string }>;
    billing_cycle_items?: Array<{ value?: string; label?: string }>;
    cancel_type_items?: Array<{ value?: string; label?: string }>;
  }>(`/admin/api/v1/goods-types/${id}/automation-options`);
