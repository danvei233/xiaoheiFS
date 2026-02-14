/// 应用字符串常量
/// 集中管理所有 UI 文本
class AppStrings {
  AppStrings._();

  // 应用信息
  static const String appName = '云享互联';
  static const String appTitle = '云享互联';

  // 通用
  static const String loading = '加载中...';
  static const String error = '错误';
  static const String success = '成功';
  static const String failed = '失败';
  static const String retry = '重试';
  static const String confirm = '确认';
  static const String cancel = '取消';
  static const String save = '保存';
  static const String delete = '删除';
  static const String edit = '编辑';
  static const String submit = '提交';
  static const String search = '搜索';
  static const String filter = '筛选';
  static const String refresh = '刷新';
  static const String close = '关闭';
  static const String back = '返回';
  static const String next = '下一步';
  static const String done = '完成';

  // 认证相关
  static const String login = '登录';
  static const String logout = '退出登录';
  static const String username = '用户名';
  static const String password = '<REDACTED>';
  static const String apiUrl = 'API 地址';
  static const String rememberMe = '记住我';
  static const String loginSuccess = '登录成功';
  static const String loginFailed = '登录失败';
  static const String logoutConfirm = '确定要退出登录吗？';
  static const String inputUsername = '请输入用户名';
  static const String inputPassword = '请输入密码';
  static const String inputApiUrl = '请输入 API 地址';

  // 导航菜单
  static const String navDashboard = '总览';
  static const String navVps = '云服务器';
  static const String navCart = '购物车';
  static const String navOrders = '订单管理';
  static const String navWallet = '钱包充值';
  static const String navTickets = '工单中心';
  static const String navRealname = '实名认证';
  static const String navProfile = '个人设置';
  static const String navMore = '更多';
  static const String navNotifications = '消息中心';

  // Dashboard
  static const String dashboard = '控制台';
  static const String accountBalance = '账户余额';
  static const String vpsCount = '云服务器';
  static const String orderCount = '订单';
  static const String spendTrend = '30天消费';
  static const String expiringSoon = '即将到期';
  static const String quickActions = '快捷操作';
  static const String gotoRealname = '去实名认证';
  static const String viewCart = '查看购物车';
  static const String pendingOrders = '待处理订单';

  // VPS
  static const String vpsManagement = '云服务器管理';
  static const String vpsList = '实例列表';
  static const String vpsDetail = '实例详情';
  static const String vpsName = '实例名称';
  static const String vpsIp = 'IP 地址';
  static const String vpsStatus = '状态';
  static const String vpsRegion = '地区线路';
  static const String vpsPackage = '套餐配置';
  static const String vpsExpireAt = '到期时间';
  static const String vpsCreatedAt = '创建时间';
  static const String vpsBuy = '购买实例';
  static const String vpsStart = '开机';
  static const String vpsShutdown = '关机';
  static const String vpsReboot = '重启';
  static const String vpsReinstall = '重装系统';
  static const String vpsConsole = '控制台';
  static const String vpsMonitor = '监控';
  static const String vpsRenew = '续费';

  // 购物车
  static const String shoppingCart = '购物车';
  static const String cartEmpty = '购物车为空';
  static const String addToCart = '加入购物车';
  static const String removeFromCart = '移出购物车';
  static const String checkout = '结算';
  static const String totalAmount = '总金额';
  static const String quantity = '数量';

  // 订单
  static const String ordersManagement = '订单管理';
  static const String orderNo = '订单号';
  static const String orderStatus = '订单状态';
  static const String orderAmount = '订单金额';
  static const String orderTime = '下单时间';
  static const String orderDetail = '订单详情';
  static const String payNow = '立即支付';
  static const String cancelOrder = '取消订单';

  // 钱包
  static const String walletBalance = '钱包余额';
  static const String recharge = '充值';
  static const String withdraw = '提现';
  static const String transactionHistory = '交易记录';
  static const String transactionType = '交易类型';
  static const String transactionAmount = '金额';
  static const String transactionTime = '交易时间';
  static const String transactionStatus = '状态';

  // 工单
  static const String ticketManagement = '工单中心';
  static const String ticketList = '工单列表';
  static const String ticketDetail = '工单详情';
  static const String createTicket = '新建工单';
  static const String ticketTitle = '工单标题';
  static const String ticketContent = '工单内容';
  static const String ticketStatus = '工单状态';
  static const String ticketCreateTime = '创建时间';
  static const String sendMessage = '发送消息';
  static const String closeTicket = '关闭工单';

  // 实名认证
  static const String realnameVerification = '实名认证';
  static const String realnameStatus = '认证状态';
  static const String realname = '真实姓名';
  static const String idNumber = '身份证号';
  static const String submitVerification = '提交认证';
  static const String verificationPending = '待审核';
  static const String verificationApproved = '已认证';
  static const String verificationRejected = '审核未通过';
  static const String verificationNotSubmit = '未提交';

  // 个人设置
  static const String profileSettings = '个人设置';
  static const String userInfo = '用户信息';
  static const String email = '邮箱';
  static const String phone = '手机号';
  static const String qq = 'QQ';
  static const String bio = '个人简介';
  static const String avatar = '头像';
  static const String changePassword = '修改密码';
  static const String oldPassword = '原密码';
  static const String newPassword = '新密码';
  static const String confirmPassword = '确认密码';

  // 空状态
  static const String noData = '暂无数据';
  static const String noVps = '暂无云服务器实例';
  static const String noOrders = '暂无订单';
  static const String noTickets = '暂无工单';
  static const String noTransactions = '暂无交易记录';
  static const String noNotifications = '暂无消息';

  // Notifications
  static const String notifications = '消息中心';
  static const String notification = '通知';
  static const String markAllRead = '全部已读';

  // 错误信息
  static const String networkError = '网络连接失败';
  static const String serverError = '服务器错误';
  static const String unauthorized = '未授权，请重新登录';
  static const String forbidden = '无权限访问';
  static const String notFound = '请求的资源不存在';
  static const String requestTimeout = '请求超时';
  static const String unknownError = '未知错误';
}

