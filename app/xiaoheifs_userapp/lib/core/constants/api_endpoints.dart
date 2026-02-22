class ApiEndpoints {
  ApiEndpoints._();

  static const String v1 = '/v1';

  // Auth
  static const String authSettings = '$v1/auth/settings';
  static const String authLogin = '$v1/auth/login';
  static const String authLogout = '$v1/auth/logout';
  static const String authRefresh = '$v1/auth/refresh';
  static const String authRegister = '$v1/auth/register';
  static const String authRegisterCode = '$v1/auth/register/code';
  static const String authForgotPassword = '$v1/auth/forgot-password';
  static const String authResetPassword = '$v1/auth/reset-password';
  static const String authPasswordResetOptions =
      '$v1/auth/password-reset/options';
  static const String authPasswordResetSendCode =
      '$v1/auth/password-reset/send-code';
  static const String authPasswordResetVerifyCode =
      '$v1/auth/password-reset/verify-code';
  static const String authPasswordResetConfirm =
      '$v1/auth/password-reset/confirm';
  static const String me = '$v1/me';
  static const String meUserTier = '$v1/me/user-tier';
  static const String mePasswordChange = '$v1/me/password/change';
  static const String meSecurityContacts = '$v1/me/security/contacts';
  static const String meSecurityEmailVerify2fa =
      '$v1/me/security/email/verify-2fa';
  static const String meSecurityEmailSendCode =
      '$v1/me/security/email/send-code';
  static const String meSecurityEmailConfirm = '$v1/me/security/email/confirm';
  static const String meSecurityPhoneVerify2fa =
      '$v1/me/security/phone/verify-2fa';
  static const String meSecurityPhoneSendCode =
      '$v1/me/security/phone/send-code';
  static const String meSecurityPhoneConfirm = '$v1/me/security/phone/confirm';
  static const String meSecurity2faStatus = '$v1/me/security/2fa/status';
  static const String meSecurity2faSetup = '$v1/me/security/2fa/setup';
  static const String meSecurity2faConfirm = '$v1/me/security/2fa/confirm';

  // Captcha
  static const String captcha = '$v1/captcha';

  // Dashboard
  static const String dashboard = '$v1/dashboard';

  // Catalog
  static const String catalog = '$v1/catalog';
  static const String goodsTypes = '$v1/goods-types';
  static const String planGroups = '$v1/plan-groups';
  static const String packages = '$v1/packages';
  static const String systemImages = '$v1/system-images';
  static const String billingCycles = '$v1/billing-cycles';

  // Cart
  static const String cart = '$v1/cart';
  static String cartItem(int id) => '$v1/cart/$id';

  // Orders
  static const String orders = '$v1/orders';
  static const String ordersItems = '$v1/orders/items';
  static String orderDetail(int id) => '$v1/orders/$id';
  static String orderPay(int id) => '$v1/orders/$id/pay';
  static String orderCancel(int id) => '$v1/orders/$id/cancel';
  static String orderRefresh(int id) => '$v1/orders/$id/refresh';
  static String orderPayments(int id) => '$v1/orders/$id/payments';
  static String orderEvents(int id) => '$v1/orders/$id/events';
  static const String couponsPreview = '$v1/coupons/preview';

  // Payments
  static const String paymentProviders = '$v1/payments/providers';

  // Wallet
  static const String wallet = '$v1/wallet';
  static const String walletTransactions = '$v1/wallet/transactions';
  static const String walletRecharge = '$v1/wallet/recharge';
  static const String walletWithdraw = '$v1/wallet/withdraw';
  static const String walletOrders = '$v1/wallet/orders';

  // Notifications
  static const String notifications = '$v1/notifications';
  static const String notificationsUnreadCount =
      '$v1/notifications/unread-count';
  static const String notificationsReadAll = '$v1/notifications/read-all';
  static String notificationRead(int id) => '$v1/notifications/$id/read';

  // VPS
  static const String vps = '$v1/vps';
  static String vpsDetail(int id) => '$v1/vps/$id';
  static String vpsRefresh(int id) => '$v1/vps/$id/refresh';
  static String vpsStart(int id) => '$v1/vps/$id/start';
  static String vpsShutdown(int id) => '$v1/vps/$id/shutdown';
  static String vpsReboot(int id) => '$v1/vps/$id/reboot';
  static String vpsResetOs(int id) => '$v1/vps/$id/reset-os';
  static String vpsResetOsPassword(int id) => '$v1/vps/$id/reset-os-password';
  static String vpsMonitor(int id) => '$v1/vps/$id/monitor';
  static String vpsPanel(int id) => '$v1/vps/$id/panel';
  static String vpsVnc(int id) => '$v1/vps/$id/vnc';
  static String vpsSnapshots(int id) => '$v1/vps/$id/snapshots';
  static String vpsSnapshotDetail(int id, int snapshotId) =>
      '$v1/vps/$id/snapshots/$snapshotId';
  static String vpsSnapshotRestore(int id, int snapshotId) =>
      '$v1/vps/$id/snapshots/$snapshotId/restore';
  static String vpsBackups(int id) => '$v1/vps/$id/backups';
  static String vpsBackupDetail(int id, int backupId) =>
      '$v1/vps/$id/backups/$backupId';
  static String vpsBackupRestore(int id, int backupId) =>
      '$v1/vps/$id/backups/$backupId/restore';
  static String vpsFirewall(int id) => '$v1/vps/$id/firewall';
  static String vpsFirewallRule(int id, int ruleId) =>
      '$v1/vps/$id/firewall/$ruleId';
  static String vpsPorts(int id) => '$v1/vps/$id/ports';
  static String vpsPortCandidates(int id) => '$v1/vps/$id/ports/candidates';
  static String vpsPortMapping(int id, int mappingId) =>
      '$v1/vps/$id/ports/$mappingId';
  static String vpsRenew(int id) => '$v1/vps/$id/renew';
  static String vpsResizeQuote(int id) => '$v1/vps/$id/resize/quote';
  static String vpsResize(int id) => '$v1/vps/$id/resize';
  static String vpsEmergencyRenew(int id) => '$v1/vps/$id/emergency-renew';
  static String vpsRefund(int id) => '$v1/vps/$id/refund';

  // Tickets
  static const String tickets = '$v1/tickets';
  static String ticketDetail(int id) => '$v1/tickets/$id';
  static String ticketMessages(int id) => '$v1/tickets/$id/messages';
  static String ticketClose(int id) => '$v1/tickets/$id/close';

  // Realname
  static const String realnameStatus = '$v1/realname/status';
  static const String realnameVerify = '$v1/realname/verify';

  // Open API Keys
  static const String openApiKeys = '$v1/open/me/api-keys';
  static String openApiKeyDetail(int id) => '$v1/open/me/api-keys/$id';

  // Site/CMS
  static const String siteSettings = '$v1/site/settings';
  static const String cmsBlocks = '$v1/cms/blocks';
  static const String cmsPosts = '$v1/cms/posts';
}
