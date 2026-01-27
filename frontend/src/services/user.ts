import { http, withApiKey } from "./http";
import type {
  ApiList,
  AuthResponse,
  CaptchaResponse,
  CartItem,
  CartItemRequest,
  Order,
  OrderCreateResponse,
  OrderCreateRequest,
  OrderDetailResponse,
  PaymentRequest,
  BillingCycle,
  Package,
  Line,
  SystemImage,
  User,
  UserDashboard,
  VPSInstance,
  MonitorResponse,
  Ticket,
  TicketDetailResponse,
  PaymentProvider,
  PaymentCreateResult,
  WalletInfo,
  WalletOrderCreateRequest,
  WalletOrderListResponse,
  WalletTransaction,
  Notification,
  RealNameStatusResponse,
  UnreadCountResponse,
  RealNameVerification,
  CMSBlock,
  CMSPost
} from "./types";

export const getCaptcha = () => http.get<CaptchaResponse>("/api/v1/captcha");
export const getInstallStatus = () => http.get<{ installed: boolean }>("/api/v1/install/status");
export const checkInstallDB = (payload: Record<string, unknown>) =>
  http.post<{ ok: boolean; error?: string }>("/api/v1/install/db/check", payload);
export const runInstall = (payload: Record<string, unknown>) => http.post("/api/v1/install", payload);
export const userRegister = (payload: Record<string, unknown>) => http.post("/api/v1/auth/register", payload);
export const userLogin = (payload: Record<string, unknown>) => http.post<AuthResponse>("/api/v1/auth/login", payload);
export const getMe = () => http.get<User>("/api/v1/me");
export const updateMe = (payload: Record<string, unknown>) => http.patch<User>("/api/v1/me", payload);

export const getDashboard = () => http.get<UserDashboard>("/api/v1/dashboard");
export const getCatalog = () => http.get("/api/v1/catalog");
export const listPlanGroups = (params?: { region_id?: number }) =>
  http.get<ApiList<Line>>("/api/v1/plan-groups", { params });
export const listPackages = (params?: { plan_group_id?: number }) =>
  http.get<ApiList<Package>>("/api/v1/packages", { params });
export const listBillingCycles = () => http.get<ApiList<BillingCycle>>("/api/v1/billing-cycles");
export const listSystemImages = (params?: { line_id?: number; plan_group_id?: number }) =>
  http.get<ApiList<SystemImage>>("/api/v1/system-images", { params });

export const listCart = () => http.get<ApiList<CartItem>>("/api/v1/cart");
export const addCartItem = (payload: CartItemRequest) => http.post("/api/v1/cart", payload);
export const updateCartItem = (id: number | string, payload: CartItemRequest) => http.patch(`/api/v1/cart/${id}`, payload);
export const deleteCartItem = (id: number | string) => http.delete(`/api/v1/cart/${id}`);
export const clearCart = () => http.delete("/api/v1/cart");

export const listOrders = (params?: Record<string, unknown>) => http.get<ApiList<Order>>("/api/v1/orders", { params });
export const createOrderFromCart = (idempotencyKey?: string) =>
  http.post<OrderCreateResponse>("/api/v1/orders", null, {
    headers: idempotencyKey ? { "Idempotency-Key": idempotencyKey } : {}
  });
export const createOrder = (payload: OrderCreateRequest, idempotencyKey?: string) =>
  http.post<OrderCreateResponse>("/api/v1/orders/items", payload, {
    headers: idempotencyKey ? { "Idempotency-Key": idempotencyKey } : {}
  });
export const getOrderDetail = (id: number | string) => http.get<OrderDetailResponse>(`/api/v1/orders/${id}`);
export const refreshOrder = (id: number | string) => http.post(`/api/v1/orders/${id}/refresh`);
export const cancelOrder = (id: number | string) => http.post(`/api/v1/orders/${id}/cancel`);
export const submitOrderPayment = (id: number | string, payload: PaymentRequest, idempotencyKey?: string) =>
  http.post(`/api/v1/orders/${id}/payments`, payload, {
    headers: idempotencyKey ? { "Idempotency-Key": idempotencyKey } : {}
  });

export const listPaymentProviders = () => http.get<ApiList<PaymentProvider>>("/api/v1/payments/providers");
export const createOrderPayment = (id: number | string, payload: Record<string, unknown>) =>
  http.post<PaymentCreateResult>(`/api/v1/orders/${id}/pay`, payload);
export const getWallet = () => http.get<WalletInfo>("/api/v1/wallet");
export const createWalletRecharge = (payload: WalletOrderCreateRequest) =>
  http.post<{ order?: Record<string, unknown> }>("/api/v1/wallet/recharge", payload);
export const createWalletWithdraw = (payload: WalletOrderCreateRequest) =>
  http.post<{ order?: Record<string, unknown> }>("/api/v1/wallet/withdraw", payload);
export const listWalletOrders = (params?: Record<string, unknown>) =>
  http.get<WalletOrderListResponse>("/api/v1/wallet/orders", { params });
export const listWalletTransactions = (params?: Record<string, unknown>) =>
  http.get<ApiList<WalletTransaction>>("/api/v1/wallet/transactions", { params });

// 消息中心
export const listNotifications = (params?: Record<string, unknown>) =>
  http.get<ApiList<Notification>>("/api/v1/notifications", { params });
export const getUnreadCount = () => http.get<UnreadCountResponse>("/api/v1/notifications/unread-count");
export const markNotificationRead = (id: number | string) => http.post(`/api/v1/notifications/${id}/read`);
export const markAllNotificationsRead = () => http.post("/api/v1/notifications/read-all");

export const listVps = () => http.get<ApiList<VPSInstance>>("/api/v1/vps");
export const getVpsDetail = (id: number | string) => http.get<VPSInstance>(`/api/v1/vps/${id}`);
export const refreshVps = (id: number | string) => http.post(`/api/v1/vps/${id}/refresh`);
export const getVpsPanel = (id: number | string) => http.get(`/api/v1/vps/${id}/panel`);
export const getVpsMonitor = (id: number | string) => http.get<MonitorResponse>(`/api/v1/vps/${id}/monitor`);
export const getVpsVnc = (id: number | string) => http.get(`/api/v1/vps/${id}/vnc`);
export const startVps = (id: number | string) => http.post(`/api/v1/vps/${id}/start`);
export const shutdownVps = (id: number | string) => http.post(`/api/v1/vps/${id}/shutdown`);
export const rebootVps = (id: number | string) => http.post(`/api/v1/vps/${id}/reboot`);
export const resetVpsOS = (id: number | string, payload: { template_id: number | string; password: string }) =>
  http.post(`/api/v1/vps/${id}/reset-os`, { host_id: id, ...payload });
export const resetVpsOsPassword = (id: number | string, payload: { password: string }) =>
  http.post(`/api/v1/vps/${id}/reset-os-password`, payload);
export const getVpsSnapshots = (id: number | string) => http.get(`/api/v1/vps/${id}/snapshots`);
export const createVpsSnapshot = (id: number | string) => http.post(`/api/v1/vps/${id}/snapshots`);
export const deleteVpsSnapshot = (id: number | string, snapshotId: number | string) =>
  http.delete(`/api/v1/vps/${id}/snapshots/${snapshotId}`);
export const restoreVpsSnapshot = (id: number | string, snapshotId: number | string) =>
  http.post(`/api/v1/vps/${id}/snapshots/${snapshotId}/restore`);
export const getVpsBackups = (id: number | string) => http.get(`/api/v1/vps/${id}/backups`);
export const createVpsBackup = (id: number | string) => http.post(`/api/v1/vps/${id}/backups`);
export const deleteVpsBackup = (id: number | string, backupId: number | string) =>
  http.delete(`/api/v1/vps/${id}/backups/${backupId}`);
export const restoreVpsBackup = (id: number | string, backupId: number | string) =>
  http.post(`/api/v1/vps/${id}/backups/${backupId}/restore`);
export const getVpsFirewallRules = (id: number | string) => http.get(`/api/v1/vps/${id}/firewall`);
export const addVpsFirewallRule = (id: number | string, payload: Record<string, unknown>) =>
  http.post(`/api/v1/vps/${id}/firewall`, payload);
export const deleteVpsFirewallRule = (id: number | string, ruleId: number | string) =>
  http.delete(`/api/v1/vps/${id}/firewall/${ruleId}`);
export const getVpsPortMappings = (id: number | string) => http.get(`/api/v1/vps/${id}/ports`);
export const addVpsPortMapping = (id: number | string, payload: Record<string, unknown>) =>
  http.post(`/api/v1/vps/${id}/ports`, payload);
export const getVpsPortCandidates = (id: number | string, params?: { keywords?: string }) =>
  http.get(`/api/v1/vps/${id}/ports/candidates`, { params });
export const deleteVpsPortMapping = (id: number | string, mappingId: number | string) =>
  http.delete(`/api/v1/vps/${id}/ports/${mappingId}`);
export const createVpsRenewOrder = (id: number | string, payload: Record<string, unknown>) =>
  http.post(`/api/v1/vps/${id}/renew`, payload);
export const emergencyRenewVps = (id: number | string) => http.post(`/api/v1/vps/${id}/emergency-renew`);
export const createVpsResizeOrder = (id: number | string, payload: Record<string, unknown>) =>
  http.post(`/api/v1/vps/${id}/resize`, payload);
export const quoteVpsResizeOrder = (id: number | string, payload: Record<string, unknown>) =>
  http.post(`/api/v1/vps/${id}/resize/quote`, payload);
export const requestVpsRefund = (id: number | string, payload: { reason?: string }) =>
  http.post(`/api/v1/vps/${id}/refund`, payload);

export const triggerRobotWebhook = (payload: Record<string, unknown>, headers?: Record<string, string>) =>
  http.post("/api/v1/integrations/robot/webhook", payload, withApiKey(headers));

// 实名认证
export const getRealNameStatus = () => http.get<RealNameStatusResponse>("/api/v1/realname/status");
export const submitRealNameVerification = (payload: { real_name: string; id_number: string }) =>
  http.post<RealNameVerification>("/api/v1/realname/verify", payload);

// 密码找回
export const forgotPassword = (email: string) => http.post("/api/v1/auth/forgot-password", { email });
export const resetPassword = (token: string, new_password: string) =>
  http.post("/api/v1/auth/reset-password", { token, new_password });

// 工单
export const listTickets = (params?: Record<string, unknown>) => http.get<ApiList<Ticket>>("/api/v1/tickets", { params });
export const createTicket = (payload: Record<string, unknown>) => http.post<TicketDetailResponse>("/api/v1/tickets", payload);
export const getTicketDetail = (id: number | string) => http.get<TicketDetailResponse>(`/api/v1/tickets/${id}`);
export const addTicketMessage = (id: number | string, payload: Record<string, unknown>) =>
  http.post(`/api/v1/tickets/${id}/messages`, payload);
export const closeTicket = (id: number | string) => http.post(`/api/v1/tickets/${id}/close`);

// CMS Public API
export const getSiteSettings = () => http.get("/api/v1/site/settings");
export const getCmsBlocks = (params?: Record<string, unknown>) =>
  http.get<{ items?: CMSBlock[] }>("/api/v1/cms/blocks", { params });
export const getCmsPosts = (params?: Record<string, unknown>) =>
  http.get<{ items?: CMSPost[]; total?: number }>("/api/v1/cms/posts", { params });
export const getCmsPostBySlug = (slug: string) =>
  http.get<CMSPost>(`/api/v1/cms/posts/${slug}`);
