import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../app_state.dart';
import '../services/api_client.dart';
import '../services/avatar.dart';
import 'orders_screen.dart';
import 'users_screen.dart';
import 'servers_screen.dart';
import 'wallet_orders_screen.dart';
import 'tickets_screen.dart';
import 'audit_logs_screen.dart';
import 'scheduled_tasks_screen.dart';
import 'api_keys_screen.dart';
import 'payment_providers_screen.dart';
import 'settings_kv_screen.dart';
import 'catalog/catalog_hub_screen.dart';
import 'permissions_screen.dart';
import 'settings_screen.dart';

class HomeScreen extends StatefulWidget {
  const HomeScreen({super.key});

  @override
  State<HomeScreen> createState() => _HomeScreenState();
}

class _HomeScreenState extends State<HomeScreen> {
  Future<DashboardData>? _future;
  ApiClient? _client;

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    final client = context.read<AppState>().apiClient;
    if (client?.baseUrl != _client?.baseUrl ||
        client?.apiKey != _client?.apiKey ||
        client?.token != _client?.token) {
      _client = client;
      if (client != null) {
        _future = _load(client);
      }
    }
  }

  Future<DashboardData> _load(ApiClient client) async {
    final overview = await client.postJson('/admin/api/v1/dashboard/overview');
    final usersTotal = await _safeTotal(client, '/admin/api/v1/users');
    final walletTotal = await _safeTotal(client, '/admin/api/v1/wallet/orders');
    return DashboardData(
      totalOrders: _asInt(overview['total_orders']),
      pendingReview: _asInt(overview['pending_review']),
      revenueCents: _asInt(overview['revenue']),
      vpsCount: _asInt(overview['vps_count']),
      expiringSoon: _asInt(overview['expiring_soon']),
      usersTotal: usersTotal,
      walletOrdersTotal: walletTotal,
    );
  }

  Future<int?> _safeTotal(ApiClient client, String path) async {
    try {
      final resp = await client.getJson(path, query: {'limit': '1'});
      return _asInt(resp['total']);
    } catch (_) {
      return null;
    }
  }

  int _asInt(dynamic value) {
    if (value is int) return value;
    if (value is double) return value.toInt();
    if (value is String) return int.tryParse(value) ?? 0;
    return 0;
  }

  String _formatAmount(int cents) {
    final amount = cents / 100.0;
    return '¥${amount.toStringAsFixed(2)}';
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      body: FutureBuilder<DashboardData>(
        future: _future,
        builder: (context, snapshot) {
          if (snapshot.connectionState == ConnectionState.waiting) {
            return const Center(child: CircularProgressIndicator());
          }
          if (snapshot.hasError) {
            return _ErrorState(
              message: '加载概览失败，请检查 API Key 权限或 API 地址。',
              onRetry: () {
                final client = context.read<AppState>().apiClient;
                if (client != null) {
                  setState(() {
                    _future = _load(client);
                  });
                }
              },
            );
          }
          final data = snapshot.data;
          if (data == null) {
            return const _EmptyState();
          }

          return RefreshIndicator(
            onRefresh: () async {
              final client = context.read<AppState>().apiClient;
              if (client != null) {
                setState(() {
                  _future = _load(client);
                });
              }
              await _future;
            },
            child: CustomScrollView(
              slivers: [
                _HeaderSliver(data: data),
                SliverToBoxAdapter(
                  child: _StatsSection(data: data, formatAmount: _formatAmount),
                ),
                const SliverToBoxAdapter(child: _QuickEntrySection()),
                const SliverToBoxAdapter(child: _ManagementSection()),
                const SliverToBoxAdapter(child: SizedBox(height: 100)),
              ],
            ),
          );
        },
      ),
    );
  }
}

class _HeaderSliver extends StatefulWidget {
  final DashboardData data;

  const _HeaderSliver({required this.data});

  @override
  State<_HeaderSliver> createState() => _HeaderSliverState();
}

class _HeaderSliverState extends State<_HeaderSliver> {
  Future<Map<String, dynamic>>? _future;
  ApiClient? _client;

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    final client = context.read<AppState>().apiClient;
    if (client?.baseUrl != _client?.baseUrl ||
        client?.apiKey != _client?.apiKey ||
        client?.token != _client?.token) {
      _client = client;
      if (client != null) {
        _future = client.getJson('/admin/api/v1/profile');
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    final session = context.read<AppState>().session;
    final username = session?.username ?? '管理员';

    return SliverAppBar(
      expandedHeight: 220,
      pinned: true,
      floating: false,
      backgroundColor: const Color(0xFF00BFA6),
      actions: [
        IconButton(
          icon: const Icon(Icons.settings, color: Colors.white),
          onPressed: () => Navigator.push(
            context,
            MaterialPageRoute(builder: (_) => const SettingsScreen()),
          ),
        ),
      ],
      flexibleSpace: FlexibleSpaceBar(
        collapseMode: CollapseMode.pin,
        background: Container(
          decoration: const BoxDecoration(
            gradient: LinearGradient(
              colors: [Color(0xFF00BFA6), Color(0xFF008B7A)],
              begin: Alignment.topLeft,
              end: Alignment.bottomRight,
            ),
          ),
          child: SafeArea(
            bottom: false,
            child: Padding(
              padding: const EdgeInsets.fromLTRB(16, 12, 16, 12),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Row(
                    children: [
                      const Text(
                        '小黑云',
                        style: TextStyle(
                          fontSize: 24,
                          fontWeight: FontWeight.bold,
                          color: Colors.white,
                        ),
                      ),
                      const Spacer(),
                      FutureBuilder<Map<String, dynamic>>(
                        future: _future,
                        builder: (context, snapshot) {
                          final profile = snapshot.data ?? {};
                          final baseUrl = _client?.baseUrl ?? '';
                          final avatarUrl = resolveAvatarUrl(
                            baseUrl: baseUrl,
                            qq: profile['qq']?.toString(),
                            avatarUrl: profile['avatar_url'] as String?,
                          );
                          final headers = avatarHeaders(
                            token: session?.token,
                            apiKey: session?.apiKey,
                          );
                          return CircleAvatar(
                            radius: 18,
                            backgroundColor: Colors.white24,
                            child: avatarUrl.isNotEmpty
                                ? ClipOval(
                                    child: Image.network(
                                      avatarUrl,
                                      width: 36,
                                      height: 36,
                                      fit: BoxFit.cover,
                                      headers: headers.isEmpty ? null : headers,
                                      errorBuilder: (context, error, stack) {
                                        return const Icon(
                                          Icons.person,
                                          color: Colors.white,
                                        );
                                      },
                                    ),
                                  )
                                : Text(
                                    username.isNotEmpty
                                        ? username.characters.first
                                        : '?',
                                    style: const TextStyle(
                                      color: Colors.white,
                                      fontWeight: FontWeight.bold,
                                      fontSize: 16,
                                    ),
                                  ),
                          );
                        },
                      ),
                    ],
                  ),
                  const SizedBox(height: 8),
                  Expanded(
                    child: Container(
                      decoration: BoxDecoration(
                        color: Colors.white,
                        borderRadius: BorderRadius.circular(16),
                      ),
                      child: Row(
                        children: [
                          Expanded(
                            child: Padding(
                              padding: const EdgeInsets.all(14),
                              child: Column(
                                crossAxisAlignment: CrossAxisAlignment.start,
                                mainAxisAlignment: MainAxisAlignment.center,
                                children: [
                                  Text(
                                    '今日概览',
                                    style: TextStyle(
                                      fontSize: 13,
                                      color: Colors.grey[600],
                                    ),
                                  ),
                                  const SizedBox(height: 4),
                                  Row(
                                    children: [
                                      Text(
                                        '${widget.data.totalOrders}',
                                        style: const TextStyle(
                                          fontSize: 24,
                                          fontWeight: FontWeight.bold,
                                          color: Color(0xFF00BFA6),
                                        ),
                                      ),
                                      const SizedBox(width: 8),
                                      Text(
                                        '笔订单',
                                        style: TextStyle(
                                          fontSize: 13,
                                          color: Colors.grey[600],
                                        ),
                                      ),
                                      if (widget.data.pendingReview > 0) ...[
                                        const SizedBox(width: 8),
                                        Container(
                                          padding: const EdgeInsets.symmetric(
                                            horizontal: 8,
                                            vertical: 2,
                                          ),
                                          decoration: BoxDecoration(
                                            color: const Color(0xFFFF6B6B),
                                            borderRadius: BorderRadius.circular(
                                              10,
                                            ),
                                          ),
                                          child: Text(
                                            '${widget.data.pendingReview} 待审核',
                                            style: const TextStyle(
                                              fontSize: 11,
                                              color: Colors.white,
                                              fontWeight: FontWeight.w500,
                                            ),
                                          ),
                                        ),
                                      ],
                                    ],
                                  ),
                                ],
                              ),
                            ),
                          ),
                          Container(
                            width: 68,
                            height: double.infinity,
                            decoration: const BoxDecoration(
                              color: Color(0xFFF0F7F6),
                              borderRadius: BorderRadius.only(
                                topRight: Radius.circular(16),
                                bottomRight: Radius.circular(16),
                              ),
                            ),
                            child: const Icon(
                              Icons.receipt_long_rounded,
                              size: 34,
                              color: Color(0xFF00BFA6),
                            ),
                          ),
                        ],
                      ),
                    ),
                  ),
                ],
              ),
            ),
          ),
        ),
      ),
    );
  }
}

class _StatsSection extends StatelessWidget {
  final DashboardData data;
  final String Function(int) formatAmount;

  const _StatsSection({required this.data, required this.formatAmount});

  @override
  Widget build(BuildContext context) {
    final stats = [
      _StatItem(
        '待审核',
        '${data.pendingReview}',
        Icons.pending,
        const Color(0xFFFF6B6B),
      ),
      _StatItem(
        '用户数',
        '${data.usersTotal ?? '--'}',
        Icons.people,
        const Color(0xFF4ECDC4),
      ),
      _StatItem('服务器', '${data.vpsCount}', Icons.dns, const Color(0xFF45B7D1)),
      _StatItem(
        '钱包订单',
        '${data.walletOrdersTotal ?? '--'}',
        Icons.account_balance_wallet,
        const Color(0xFF96CEB4),
      ),
      _StatItem(
        '累计收入',
        formatAmount(data.revenueCents),
        Icons.trending_up,
        const Color(0xFFFFA94D),
      ),
    ];

    return Container(
      margin: const EdgeInsets.fromLTRB(12, 16, 12, 8),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          const Padding(
            padding: EdgeInsets.symmetric(horizontal: 4),
            child: Text(
              '数据概览',
              style: TextStyle(fontSize: 16, fontWeight: FontWeight.bold),
            ),
          ),
          const SizedBox(height: 12),
          SizedBox(
            height: 100,
            child: ListView.separated(
              scrollDirection: Axis.horizontal,
              itemCount: stats.length,
              padding: const EdgeInsets.symmetric(horizontal: 4),
              separatorBuilder: (_, i) => const SizedBox(width: 10),
              itemBuilder: (context, index) {
                final item = stats[index];
                return Container(
                  width: 110,
                  padding: const EdgeInsets.all(12),
                  decoration: BoxDecoration(
                    gradient: LinearGradient(
                      begin: Alignment.topLeft,
                      end: Alignment.bottomRight,
                      colors: [item.color, item.color.withValues(alpha: 0.85)],
                    ),
                    borderRadius: BorderRadius.circular(12),
                    boxShadow: [
                      BoxShadow(
                        color: item.color.withValues(alpha: 0.3),
                        blurRadius: 8,
                        offset: const Offset(0, 2),
                      ),
                    ],
                  ),
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    mainAxisAlignment: MainAxisAlignment.spaceBetween,
                    children: [
                      Container(
                        padding: const EdgeInsets.all(6),
                        decoration: BoxDecoration(
                          color: Colors.white.withValues(alpha: 0.2),
                          borderRadius: BorderRadius.circular(8),
                        ),
                        child: Icon(item.icon, color: Colors.white, size: 16),
                      ),
                      Column(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                          Text(
                            item.value,
                            style: const TextStyle(
                              fontSize: 22,
                              fontWeight: FontWeight.bold,
                              color: Colors.white,
                            ),
                          ),
                          Text(
                            item.label,
                            style: const TextStyle(
                              fontSize: 11,
                              color: Colors.white,
                              fontWeight: FontWeight.w500,
                            ),
                          ),
                        ],
                      ),
                    ],
                  ),
                );
              },
            ),
          ),
        ],
      ),
    );
  }
}

class _QuickEntrySection extends StatelessWidget {
  const _QuickEntrySection();

  @override
  Widget build(BuildContext context) {
    final entries = [
      _EntryItem('订单管理', Icons.receipt_long, const OrdersScreen()),
      _EntryItem(
        '钱包订单',
        Icons.account_balance_wallet,
        const WalletOrdersScreen(),
      ),
      _EntryItem('用户管理', Icons.people, const UsersScreen()),
      _EntryItem('服务器', Icons.dns, const ServersScreen()),
      _EntryItem('工单管理', Icons.support_agent, const TicketsScreen()),
      _EntryItem('操作日志', Icons.history, const AuditLogsScreen()),
      _EntryItem('定时任务', Icons.schedule, const ScheduledTasksScreen()),
      _EntryItem('API密钥', Icons.vpn_key, const ApiKeysScreen()),
    ];

    return Container(
      margin: const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
      padding: const EdgeInsets.all(16),
      decoration: BoxDecoration(
        color: Colors.white,
        borderRadius: BorderRadius.circular(12),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          const Text(
            '快捷入口',
            style: TextStyle(fontSize: 16, fontWeight: FontWeight.bold),
          ),
          const SizedBox(height: 16),
          GridView.count(
            shrinkWrap: true,
            physics: const NeverScrollableScrollPhysics(),
            crossAxisCount: 4,
            mainAxisSpacing: 20,
            crossAxisSpacing: 10,
            childAspectRatio: 1,
            children: entries.map((e) {
              return InkWell(
                onTap: () => Navigator.push(
                  context,
                  MaterialPageRoute(builder: (_) => e.screen),
                ),
                child: Column(
                  children: [
                    Container(
                      width: 44,
                      height: 44,
                      decoration: BoxDecoration(
                        color: const Color(0xFFF0F7F6),
                        borderRadius: BorderRadius.circular(12),
                      ),
                      child: Icon(
                        e.icon,
                        color: const Color(0xFF00BFA6),
                        size: 24,
                      ),
                    ),
                    const SizedBox(height: 6),
                    Text(
                      e.label,
                      style: const TextStyle(fontSize: 12),
                      maxLines: 1,
                      overflow: TextOverflow.ellipsis,
                    ),
                  ],
                ),
              );
            }).toList(),
          ),
        ],
      ),
    );
  }
}

class _ManagementSection extends StatelessWidget {
  const _ManagementSection();

  @override
  Widget build(BuildContext context) {
    final items = [
      _ManagementItem(
        '支付渠道',
        '启用/停用支付方式',
        Icons.payments,
        const PaymentProvidersScreen(),
      ),
      _ManagementItem(
        '系统设置',
        '配置键值管理',
        Icons.settings,
        const SettingsKvScreen(),
      ),
      _ManagementItem(
        '商品与计费',
        '区域/线路/套餐管理',
        Icons.category,
        const CatalogHubScreen(),
      ),
      _ManagementItem(
        '权限列表',
        '系统权限定义',
        Icons.security,
        const PermissionsScreen(),
      ),
    ];

    return Container(
      margin: const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
      padding: const EdgeInsets.all(16),
      decoration: BoxDecoration(
        color: Colors.white,
        borderRadius: BorderRadius.circular(12),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          const Text(
            '系统管理',
            style: TextStyle(fontSize: 16, fontWeight: FontWeight.bold),
          ),
          const SizedBox(height: 8),
          ...items.map((item) {
            return InkWell(
              onTap: () => Navigator.push(
                context,
                MaterialPageRoute(builder: (_) => item.screen),
              ),
              child: Padding(
                padding: const EdgeInsets.symmetric(vertical: 12),
                child: Row(
                  children: [
                    Container(
                      width: 36,
                      height: 36,
                      decoration: BoxDecoration(
                        color: const Color(0xFFF0F7F6),
                        borderRadius: BorderRadius.circular(8),
                      ),
                      child: Icon(
                        item.icon,
                        color: const Color(0xFF00BFA6),
                        size: 20,
                      ),
                    ),
                    const SizedBox(width: 12),
                    Expanded(
                      child: Column(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                          Text(
                            item.title,
                            style: const TextStyle(
                              fontSize: 14,
                              fontWeight: FontWeight.w500,
                            ),
                          ),
                          Text(
                            item.subtitle,
                            style: TextStyle(
                              fontSize: 12,
                              color: Colors.grey[600],
                            ),
                          ),
                        ],
                      ),
                    ),
                    Icon(Icons.chevron_right, color: Colors.grey[400]),
                  ],
                ),
              ),
            );
          }),
        ],
      ),
    );
  }
}

class _StatItem {
  final String label;
  final String value;
  final IconData icon;
  final Color color;

  const _StatItem(this.label, this.value, this.icon, this.color);
}

class _EntryItem {
  final String label;
  final IconData icon;
  final Widget screen;

  const _EntryItem(this.label, this.icon, this.screen);
}

class _ManagementItem {
  final String title;
  final String subtitle;
  final IconData icon;
  final Widget screen;

  const _ManagementItem(this.title, this.subtitle, this.icon, this.screen);
}

class DashboardData {
  final int totalOrders;
  final int pendingReview;
  final int revenueCents;
  final int vpsCount;
  final int expiringSoon;
  final int? usersTotal;
  final int? walletOrdersTotal;

  DashboardData({
    required this.totalOrders,
    required this.pendingReview,
    required this.revenueCents,
    required this.vpsCount,
    required this.expiringSoon,
    required this.usersTotal,
    required this.walletOrdersTotal,
  });
}

class _ErrorState extends StatelessWidget {
  final String message;
  final VoidCallback onRetry;

  const _ErrorState({required this.message, required this.onRetry});

  @override
  Widget build(BuildContext context) {
    return Center(
      child: Padding(
        padding: const EdgeInsets.all(24),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Text(message, textAlign: TextAlign.center),
            const SizedBox(height: 16),
            FilledButton(onPressed: onRetry, child: const Text('重试')),
          ],
        ),
      ),
    );
  }
}

class _EmptyState extends StatelessWidget {
  const _EmptyState();

  @override
  Widget build(BuildContext context) {
    return const Center(child: Text('暂无数据'));
  }
}
