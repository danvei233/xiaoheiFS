import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:provider/provider.dart';

import '../app_state.dart';
import '../services/api_client.dart';
import '../services/avatar.dart';

class UserDetailScreen extends StatefulWidget {
  final int userId;

  const UserDetailScreen({super.key, required this.userId});

  @override
  State<UserDetailScreen> createState() => _UserDetailScreenState();
}

class _UserDetailScreenState extends State<UserDetailScreen> {
  ApiClient? _client;
  bool _loading = true;
  bool _busy = false;
  bool _changed = false;

  Map<String, dynamic> _user = {};
  Map<String, dynamic> _wallet = {};
  List<Map<String, dynamic>> _orders = [];
  List<Map<String, dynamic>> _walletTx = [];
  Map<String, dynamic>? _realname;

  String _realnameStatus = '';
  final _realnameReason = TextEditingController();

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    _client = context.read<AppState>().apiClient;
    if (_client != null) {
      _loadAll();
    }
  }

  @override
  void dispose() {
    _realnameReason.dispose();
    super.dispose();
  }

  Future<void> _loadAll() async {
    setState(() => _loading = true);
    try {
      final client = _client;
      if (client == null) return;
      final userResp = await client.getJson(
        '/admin/api/v1/users/${widget.userId}',
      );
      final walletResp = await client.getJson(
        '/admin/api/v1/wallets/${widget.userId}',
      );
      final ordersResp = await client.getJson(
        '/admin/api/v1/orders',
        query: {
          'user_id': widget.userId.toString(),
          'limit': '20',
          'offset': '0',
        },
      );
      final txResp = await client.getJson(
        '/admin/api/v1/wallets/${widget.userId}/transactions',
        query: {'limit': '20', 'offset': '0'},
      );
      final realnameResp = await client.getJson(
        '/admin/api/v1/realname/records',
        query: {
          'user_id': widget.userId.toString(),
          'limit': '1',
          'offset': '0',
        },
      );
      final realnameItems = (realnameResp['items'] as List<dynamic>? ?? [])
          .cast<Map<String, dynamic>>();

      setState(() {
        _user = _unwrapUser(userResp);
        _wallet = (walletResp['wallet'] as Map<String, dynamic>?) ?? {};
        _orders = (ordersResp['items'] as List<dynamic>? ?? [])
            .cast<Map<String, dynamic>>();
        _walletTx = (txResp['items'] as List<dynamic>? ?? [])
            .cast<Map<String, dynamic>>();
        _realname = realnameItems.isNotEmpty ? realnameItems.first : null;
        _realnameStatus = _realname?['status']?.toString() ?? '';
        _realnameReason.text = _realname?['reason']?.toString() ?? '';
        _loading = false;
      });
    } catch (e) {
      if (mounted) {
        setState(() => _loading = false);
        ScaffoldMessenger.of(
          context,
        ).showSnackBar(SnackBar(content: Text('加载失败：$e')));
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    if (_loading) {
      return const Scaffold(body: Center(child: CircularProgressIndicator()));
    }
    return WillPopScope(
      onWillPop: () async {
        Navigator.pop(context, _changed);
        return false;
      },
      child: Scaffold(
        appBar: AppBar(
          title: const Text('用户详情'),
          actions: [
            IconButton(icon: const Icon(Icons.refresh), onPressed: _loadAll),
          ],
        ),
        body: DefaultTabController(
          length: 4,
          child: Column(
            children: [
              Padding(
                padding: const EdgeInsets.fromLTRB(12, 6, 12, 4),
                child: Container(
                  decoration: BoxDecoration(
                    color: Theme.of(context).colorScheme.surface,
                    borderRadius: BorderRadius.circular(10),
                    border: Border.all(
                      color: Theme.of(context)
                          .colorScheme
                          .outlineVariant
                          .withOpacity(0.5),
                    ),
                  ),
                  child: TabBar(
                    labelColor: Theme.of(context).colorScheme.primary,
                    unselectedLabelColor:
                        Theme.of(context).colorScheme.onSurfaceVariant,
                    indicatorColor: Theme.of(context).colorScheme.primary,
                    indicatorSize: TabBarIndicatorSize.tab,
                    labelStyle: const TextStyle(
                      fontSize: 11,
                      fontWeight: FontWeight.w600,
                    ),
                    unselectedLabelStyle: const TextStyle(fontSize: 11),
                    labelPadding: const EdgeInsets.symmetric(vertical: 6),
                    tabs: const [
                      Tab(text: '概览'),
                      Tab(text: '实名'),
                      Tab(text: '订单'),
                      Tab(text: '钱包'),
                    ],
                  ),
                ),
              ),
              Expanded(
                child: TabBarView(
                  children: [
                    ListView(
                      padding: const EdgeInsets.fromLTRB(12, 6, 12, 16),
                      children: [
                        _UserHeader(
                          user: _user,
                          avatarUrl: _avatarUrl(
                            _user,
                            context.read<AppState>().apiClient?.baseUrl ?? '',
                          ),
                        ),
                        const SizedBox(height: 6),
                        _QuickStatsRow(
                          balance: _wallet['balance'],
                          frozen: _wallet['frozen'],
                          ordersCount: _orders.length,
                          txCount: _walletTx.length,
                        ),
                        const SizedBox(height: 6),
                        _UserInfoPanel(user: _user),
                        const SizedBox(height: 6),
                        Container(
                          padding: const EdgeInsets.all(6),
                          decoration: BoxDecoration(
                            color: Theme.of(context).colorScheme.surface,
                            borderRadius: BorderRadius.circular(10),
                            border: Border.all(
                              color: Theme.of(context)
                                  .colorScheme
                                  .outlineVariant
                                  .withOpacity(0.5),
                            ),
                          ),
                          child: _ActionBar(
                            busy: _busy,
                            onToggle: _toggleStatus,
                            onReset: _resetPassword,
                            onImpersonate: _impersonate,
                          ),
                        ),
                      ],
                    ),
                    ListView(
                      padding: const EdgeInsets.fromLTRB(12, 6, 12, 16),
                      children: [
                        const _SectionHeader(
                          title: '实名认证',
                          icon: Icons.verified_user,
                        ),
                        _RealnameCard(
                          status: _realnameStatus,
                          reasonController: _realnameReason,
                          busy: _busy,
                          hasRecord: _realname != null,
                          onStatusChanged: (value) =>
                              setState(() => _realnameStatus = value),
                          onSubmit: _updateRealname,
                        ),
                      ],
                    ),
                    ListView(
                      padding: const EdgeInsets.fromLTRB(12, 6, 12, 16),
                      children: [
                        _SectionHeader(
                          title: '订单记录',
                          icon: Icons.receipt_long,
                          count: _orders.length,
                        ),
                        if (_orders.isEmpty)
                          const _EmptyLine(text: '暂无订单')
                        else
                          ..._orders.map((order) => _OrderTile(order: order)),
                      ],
                    ),
                    ListView(
                      padding: const EdgeInsets.fromLTRB(12, 6, 12, 16),
                      children: [
                        const _SectionHeader(
                          title: '钱包余额',
                          icon: Icons.account_balance_wallet,
                        ),
                        _WalletSummaryCard(
                          balance: _wallet['balance'],
                          frozen: _wallet['frozen'],
                        ),
                        const SizedBox(height: 6),
                        _SectionHeader(
                          title: '钱包记录',
                          icon: Icons.history,
                          count: _walletTx.length,
                        ),
                        if (_walletTx.isEmpty)
                          const _EmptyLine(text: '暂无钱包记录')
                        else
                          ..._walletTx.map((tx) => _WalletTxTile(tx: tx)),
                      ],
                    ),
                  ],
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }

  Future<void> _toggleStatus() async {
    if (_busy) return;
    setState(() => _busy = true);
    try {
      final client = _client;
      if (client == null) return;
      final status = _user['status'] == 'active' ? 'blocked' : 'active';
      await client.patchJson(
        '/admin/api/v1/users/${widget.userId}/status',
        body: {'status': status},
      );
      _changed = true;
      await _loadAll();
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  Future<void> _resetPassword() async {
    if (_busy) return;
    final controller = TextEditingController();
    final ok = await showDialog<bool>(
      context: context,
      builder: (context) => AlertDialog(
        title: const Text('重置密码'),
        content: TextField(
          controller: controller,
          obscureText: true,
          decoration: const InputDecoration(labelText: '新密码'),
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(context, false),
            child: const Text('取消'),
          ),
          FilledButton(
            onPressed: () => Navigator.pop(context, true),
            child: const Text('确认'),
          ),
        ],
      ),
    );
    if (ok == true) {
      setState(() => _busy = true);
      try {
        final client = _client;
        if (client == null) return;
        await client.postJson(
          '/admin/api/v1/users/${widget.userId}/reset-password',
          body: {'password': controller.text},
        );
        ScaffoldMessenger.of(
          context,
        ).showSnackBar(const SnackBar(content: Text('已重置密码')));
      } finally {
        if (mounted) setState(() => _busy = false);
      }
    }
  }

  Future<void> _impersonate() async {
    if (_busy) return;
    final client = _client;
    if (client == null) return;
    final resp = await client.postJson(
      '/admin/api/v1/users/${widget.userId}/impersonate',
    );
    final token = resp['access_token'] as String?;
    if (token == null || token.isEmpty) {
      ScaffoldMessenger.of(
        context,
      ).showSnackBar(const SnackBar(content: Text('未获取到用户令牌')));
      return;
    }
    await Clipboard.setData(ClipboardData(text: token));
    if (mounted) {
      showDialog<void>(
        context: context,
        builder: (context) => AlertDialog(
          title: const Text('用户令牌'),
          content: Text('已复制到剪贴板：\n$token'),
          actions: [
            TextButton(
              onPressed: () => Navigator.pop(context),
              child: const Text('关闭'),
            ),
          ],
        ),
      );
    }
  }

  Future<void> _updateRealname() async {
    if (_busy || _realname == null) return;
    setState(() => _busy = true);
    try {
      final client = _client;
      if (client == null) return;
      await client.patchJson(
        '/admin/api/v1/users/${widget.userId}/realname-status',
        body: {
          'status': _realnameStatus,
          'reason': _realnameReason.text.trim(),
        },
      );
      _changed = true;
      await _loadAll();
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }
}

class _ActionBar extends StatelessWidget {
  final bool busy;
  final VoidCallback onToggle;
  final VoidCallback onReset;
  final VoidCallback onImpersonate;

  const _ActionBar({
    required this.busy,
    required this.onToggle,
    required this.onReset,
    required this.onImpersonate,
  });

  @override
  Widget build(BuildContext context) {
    return Wrap(
      spacing: 8,
      runSpacing: 8,
      children: [
        FilledButton.icon(
          style: FilledButton.styleFrom(
            padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 6),
            minimumSize: const Size(0, 30),
            tapTargetSize: MaterialTapTargetSize.shrinkWrap,
            textStyle: const TextStyle(fontSize: 11),
          ),
          onPressed: busy ? null : onToggle,
          icon: const Icon(Icons.power_settings_new_rounded),
          label: const Text('禁用/启用'),
        ),
        OutlinedButton.icon(
          style: OutlinedButton.styleFrom(
            padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 6),
            minimumSize: const Size(0, 30),
            tapTargetSize: MaterialTapTargetSize.shrinkWrap,
            textStyle: const TextStyle(fontSize: 11),
          ),
          onPressed: busy ? null : onReset,
          icon: const Icon(Icons.lock_reset_rounded),
          label: const Text('重置密码'),
        ),
        OutlinedButton.icon(
          style: OutlinedButton.styleFrom(
            padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 6),
            minimumSize: const Size(0, 30),
            tapTargetSize: MaterialTapTargetSize.shrinkWrap,
            textStyle: const TextStyle(fontSize: 11),
          ),
          onPressed: busy ? null : onImpersonate,
          icon: const Icon(Icons.login_rounded),
          label: const Text('以此用户登录'),
        ),
      ],
    );
  }
}

class _UserHeader extends StatelessWidget {
  final Map<String, dynamic> user;
  final String avatarUrl;

  const _UserHeader({required this.user, required this.avatarUrl});

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final colorScheme = theme.colorScheme;
    final status = (user['status'] ?? '').toString();
    final role = (user['role'] ?? '').toString();
    final username = (user['username'] ?? '-').toString();
    final email = (user['email'] ?? '').toString();
    final phone = (user['phone'] ?? '').toString();
    final qq = (user['qq'] ?? '').toString();

    return Container(
      padding: const EdgeInsets.all(6),
      decoration: BoxDecoration(
        gradient: LinearGradient(
          colors: [
            colorScheme.primary.withOpacity(0.08),
            colorScheme.primary.withOpacity(0.02),
          ],
          begin: Alignment.topLeft,
          end: Alignment.bottomRight,
        ),
        borderRadius: BorderRadius.circular(10),
        border: Border.all(color: colorScheme.outlineVariant.withOpacity(0.5)),
      ),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          _Avatar(url: avatarUrl, radius: 16),
          const SizedBox(width: 4),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Row(
                  children: [
                    Expanded(
                      child: Text(
                        username,
                        style: theme.textTheme.titleMedium?.copyWith(
                          fontWeight: FontWeight.w700,
                        ),
                        maxLines: 1,
                        overflow: TextOverflow.ellipsis,
                      ),
                    ),
                    _StatusPill(status: status),
                  ],
                ),
                const SizedBox(height: 4),
                Text(
                  'ID ${user['id'] ?? '-'} · ${role.isEmpty ? '-' : role}',
                  style: theme.textTheme.bodySmall?.copyWith(
                    color: colorScheme.onSurfaceVariant,
                  ),
                ),
                const SizedBox(height: 6),
                Wrap(
                  spacing: 8,
                  runSpacing: 6,
                  children: [
                    if (email.isNotEmpty)
                      _ContactChip(icon: Icons.email_outlined, text: email),
                    if (phone.isNotEmpty)
                      _ContactChip(icon: Icons.phone_outlined, text: phone),
                    if (qq.isNotEmpty)
                      _ContactChip(icon: Icons.chat_outlined, text: qq),
                  ],
                ),
              ],
            ),
          ),
        ],
      ),
    );
  }
}

class _StatusPill extends StatelessWidget {
  final String status;

  const _StatusPill({required this.status});

  @override
  Widget build(BuildContext context) {
    final colorScheme = Theme.of(context).colorScheme;
    final color = _statusColor(status, colorScheme);
    final label = _statusLabel(status);
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 2),
      decoration: BoxDecoration(
        color: color.withOpacity(0.12),
        borderRadius: BorderRadius.circular(999),
      ),
      child: Text(
        label,
        style: TextStyle(
          color: color,
          fontWeight: FontWeight.w600,
          fontSize: 10,
        ),
      ),
    );
  }
}

class _ContactChip extends StatelessWidget {
  final IconData icon;
  final String text;

  const _ContactChip({required this.icon, required this.text});

  @override
  Widget build(BuildContext context) {
    final colorScheme = Theme.of(context).colorScheme;
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 2),
      decoration: BoxDecoration(
        color: colorScheme.surface,
        borderRadius: BorderRadius.circular(10),
        border: Border.all(color: colorScheme.outlineVariant.withOpacity(0.5)),
      ),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          Icon(icon, size: 12, color: colorScheme.onSurfaceVariant),
          const SizedBox(width: 4),
          Text(
            text,
            style: TextStyle(
              color: colorScheme.onSurfaceVariant,
              fontSize: 10,
              fontWeight: FontWeight.w500,
            ),
          ),
        ],
      ),
    );
  }
}

class _QuickStatsRow extends StatelessWidget {
  final dynamic balance;
  final dynamic frozen;
  final int ordersCount;
  final int txCount;

  const _QuickStatsRow({
    required this.balance,
    required this.frozen,
    required this.ordersCount,
    required this.txCount,
  });

  @override
  Widget build(BuildContext context) {
    return Row(
      children: [
        Expanded(
          child: _StatTile(
            label: 'Available',
            value: 'CNY ${_money(balance)}',
            icon: Icons.account_balance_wallet_outlined,
          ),
        ),
        const SizedBox(width: 4),
        Expanded(
          child: _StatTile(
            label: 'Frozen',
            value: 'CNY ${_money(frozen)}',
            icon: Icons.lock_outline,
          ),
        ),
        const SizedBox(width: 4),
        Expanded(
          child: _StatTile(
            label: 'Orders',
            value: ordersCount.toString(),
            icon: Icons.receipt_long_outlined,
          ),
        ),
        const SizedBox(width: 4),
        Expanded(
          child: _StatTile(
            label: 'Transactions',
            value: txCount.toString(),
            icon: Icons.history,
          ),
        ),
      ],
    );
  }
}

class _StatTile extends StatelessWidget {
  final String label;
  final String value;
  final IconData icon;

  const _StatTile({
    required this.label,
    required this.value,
    required this.icon,
  });

  @override
  Widget build(BuildContext context) {
    final colorScheme = Theme.of(context).colorScheme;
    return Container(
      padding: const EdgeInsets.all(6),
      decoration: BoxDecoration(
        color: colorScheme.surface,
        borderRadius: BorderRadius.circular(10),
        border: Border.all(color: colorScheme.outlineVariant.withOpacity(0.5)),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Icon(icon, size: 14, color: colorScheme.primary),
          const SizedBox(height: 6),
          Text(
            value,
            style: Theme.of(context).textTheme.titleSmall?.copyWith(
                  fontWeight: FontWeight.w700,
                ),
          ),
          const SizedBox(height: 2),
          Text(
            label,
            style: Theme.of(context).textTheme.labelSmall?.copyWith(
                  color: colorScheme.onSurfaceVariant,
                ),
          ),
        ],
      ),
    );
  }
}

class _UserInfoPanel extends StatelessWidget {
  final Map<String, dynamic> user;

  const _UserInfoPanel({required this.user});

  @override
  Widget build(BuildContext context) {
    final colorScheme = Theme.of(context).colorScheme;
    final items = <_InfoRow>[
      _InfoRow(label: 'Email', value: user['email']?.toString() ?? '-'),
      _InfoRow(label: 'Phone', value: user['phone']?.toString() ?? '-'),
      _InfoRow(label: 'QQ', value: user['qq']?.toString() ?? '-'),
      _InfoRow(
        label: 'Status',
        value: _statusLabel(user['status']?.toString() ?? ''),
      ),
      _InfoRow(label: 'Role', value: user['role']?.toString() ?? '-'),
      _InfoRow(
        label: 'Created',
        value: _formatLocal(user['created_at']?.toString() ?? ''),
        fullWidth: true,
      ),
    ];

    return Container(
      padding: const EdgeInsets.all(6),
      decoration: BoxDecoration(
        color: colorScheme.surface,
        borderRadius: BorderRadius.circular(10),
        border: Border.all(color: colorScheme.outlineVariant.withOpacity(0.5)),
      ),
      child: Wrap(
        spacing: 8,
        runSpacing: 6,
        children: items.map((item) {
          return SizedBox(
            width: item.fullWidth
                ? double.infinity
                : (MediaQuery.of(context).size.width - 16 * 2 - 12) / 2,
            child: item,
          );
        }).toList(),
      ),
    );
  }
}

class _InfoRow extends StatelessWidget {
  final String label;
  final String value;
  final bool fullWidth;

  const _InfoRow({
    required this.label,
    required this.value,
    this.fullWidth = false,
  });

  @override
  Widget build(BuildContext context) {
    final colorScheme = Theme.of(context).colorScheme;
    return Row(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        SizedBox(
          width: 52,
          child: Text(
            label,
            style: Theme.of(context).textTheme.bodySmall?.copyWith(
                  color: colorScheme.onSurfaceVariant,
                ),
          ),
        ),
        const SizedBox(width: 4),
        Expanded(
          child: Text(
            value,
            style: Theme.of(context).textTheme.bodyMedium?.copyWith(
                  fontWeight: FontWeight.w600,
                ),
          ),
        ),
      ],
    );
  }
}

class _WalletSummaryCard extends StatelessWidget {
  final dynamic balance;
  final dynamic frozen;

  const _WalletSummaryCard({required this.balance, required this.frozen});

  @override
  Widget build(BuildContext context) {
    final colorScheme = Theme.of(context).colorScheme;
    return Container(
      padding: const EdgeInsets.all(6),
      decoration: BoxDecoration(
        color: colorScheme.surface,
        borderRadius: BorderRadius.circular(10),
        border: Border.all(color: colorScheme.outlineVariant.withOpacity(0.5)),
      ),
      child: Row(
        children: [
          Expanded(
            child: _MoneyTile(
              label: '可用余额',
              value: _money(balance),
              color: const Color(0xFF00A68C),
            ),
          ),
          const SizedBox(width: 4),
          Expanded(
            child: _MoneyTile(
              label: '冻结余额',
              value: _money(frozen),
              color: const Color(0xFFEF6C00),
            ),
          ),
        ],
      ),
    );
  }
}

class _MoneyTile extends StatelessWidget {
  final String label;
  final String value;
  final Color color;

  const _MoneyTile({
    required this.label,
    required this.value,
    required this.color,
  });

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Container(
      padding: const EdgeInsets.all(6),
      decoration: BoxDecoration(
        color: color.withOpacity(0.1),
        borderRadius: BorderRadius.circular(10),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(
            '￥$value',
            style: theme.textTheme.bodyLarge?.copyWith(
              color: color,
              fontWeight: FontWeight.w700,
              fontSize: 12,
            ),
          ),
          const SizedBox(height: 4),
          Text(
            label,
            style: theme.textTheme.bodySmall?.copyWith(
              color: color.withOpacity(0.8),
            ),
          ),
        ],
      ),
    );
  }
}

class _SectionHeader extends StatelessWidget {
  final String title;
  final IconData icon;
  final int? count;

  const _SectionHeader({
    required this.title,
    required this.icon,
    this.count,
  });

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final colorScheme = theme.colorScheme;
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 6),
      child: Row(
        children: [
          Container(
            padding: const EdgeInsets.all(6),
            decoration: BoxDecoration(
              color: colorScheme.primaryContainer.withOpacity(0.5),
              borderRadius: BorderRadius.circular(10),
            ),
            child: Icon(icon, size: 14, color: colorScheme.primary),
          ),
          const SizedBox(width: 4),
          Text(
            title,
            style: theme.textTheme.bodyLarge?.copyWith(
              fontWeight: FontWeight.w700,
            ),
          ),
          if (count != null) ...[
            const SizedBox(width: 4),
            Container(
              padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 2),
              decoration: BoxDecoration(
                color: colorScheme.primaryContainer.withOpacity(0.6),
                borderRadius: BorderRadius.circular(999),
              ),
              child: Text(
                count.toString(),
                style: theme.textTheme.labelSmall?.copyWith(
                  color: colorScheme.primary,
                  fontWeight: FontWeight.w700,
                ),
              ),
            ),
          ],
        ],
      ),
    );
  }
}

class _RealnameCard extends StatelessWidget {
  final String status;
  final TextEditingController reasonController;
  final bool busy;
  final bool hasRecord;
  final ValueChanged<String> onStatusChanged;
  final VoidCallback onSubmit;

  const _RealnameCard({
    required this.status,
    required this.reasonController,
    required this.busy,
    required this.hasRecord,
    required this.onStatusChanged,
    required this.onSubmit,
  });

  @override
  Widget build(BuildContext context) {
    final colorScheme = Theme.of(context).colorScheme;
    return Container(
      padding: const EdgeInsets.all(6),
      decoration: BoxDecoration(
        color: colorScheme.surface,
        borderRadius: BorderRadius.circular(10),
        border: Border.all(color: colorScheme.outlineVariant.withOpacity(0.5)),
      ),
      child: Column(
        children: [
          Row(
            children: [
              Expanded(
                child: DropdownButtonFormField<String>(
                  value: status.isEmpty ? null : status,
                  items: const [
                    DropdownMenuItem(value: 'pending', child: Text('待审核')),
                    DropdownMenuItem(value: 'verified', child: Text('已通过')),
                    DropdownMenuItem(value: 'failed', child: Text('未通过')),
                  ],
                  onChanged: hasRecord
                      ? (value) => onStatusChanged(value ?? '')
                      : null,
                  decoration: const InputDecoration(labelText: '实名状态'),
                ),
              ),
              const SizedBox(width: 4),
              Expanded(
                child: TextField(
                  controller: reasonController,
                  decoration: const InputDecoration(labelText: '审核备注（可选）'),
                  enabled: hasRecord,
                ),
              ),
            ],
          ),
          const SizedBox(height: 6),
          Align(
            alignment: Alignment.centerRight,
            child: FilledButton(
              onPressed: hasRecord && !busy ? onSubmit : null,
              child: const Text('更新实名状态'),
            ),
          ),
          if (!hasRecord)
            const Padding(
              padding: EdgeInsets.only(top: 8),
              child: Text('暂无实名认证记录，无法修改状态'),
            ),
        ],
      ),
    );
  }
}

class _InfoCard extends StatelessWidget {
  final String title;
  final List<String> lines;
  final Widget? leading;

  const _InfoCard({required this.title, required this.lines, this.leading});

  @override
  Widget build(BuildContext context) {
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(6),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                if (leading != null) ...[leading!, const SizedBox(width: 12)],
                Text(title, style: Theme.of(context).textTheme.titleMedium),
              ],
            ),
            const SizedBox(height: 6),
            ...lines.map(
              (line) => Padding(
                padding: const EdgeInsets.symmetric(vertical: 2),
                child: Text(line),
              ),
            ),
          ],
        ),
      ),
    );
  }
}

class _Avatar extends StatelessWidget {
  final String url;
  final double radius;

  const _Avatar({required this.url, required this.radius});

  @override
  Widget build(BuildContext context) {
    final size = radius * 2;
    if (url.isEmpty) {
      return CircleAvatar(radius: radius, child: const Icon(Icons.person));
    }
    final session = context.read<AppState>().session;
    final headers = avatarHeaders(
      token: session?.token,
      apiKey: session?.apiKey,
    );
    return CircleAvatar(
      radius: radius,
      backgroundColor: Theme.of(context).colorScheme.surface,
      child: ClipOval(
        child: Image.network(
          url,
          width: size,
          height: size,
          fit: BoxFit.cover,
          headers: headers.isEmpty ? null : headers,
          errorBuilder: (context, error, stack) {
            return SizedBox(
              width: size,
              height: size,
              child: const Icon(Icons.person),
            );
          },
        ),
      ),
    );
  }
}

class _EmptyLine extends StatelessWidget {
  final String text;

  const _EmptyLine({required this.text});

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 6),
      child: Text(text, style: Theme.of(context).textTheme.bodySmall),
    );
  }
}

class _OrderTile extends StatelessWidget {
  final Map<String, dynamic> order;

  const _OrderTile({required this.order});

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final colorScheme = theme.colorScheme;
    final amount = order['total_amount'];
    return Container(
      margin: const EdgeInsets.only(bottom: 6),
      padding: const EdgeInsets.all(6),
      decoration: BoxDecoration(
        color: colorScheme.surface,
        borderRadius: BorderRadius.circular(10),
        border: Border.all(color: colorScheme.outlineVariant.withOpacity(0.5)),
      ),
      child: Row(
        children: [
          Container(
            padding: const EdgeInsets.all(6),
            decoration: BoxDecoration(
              color: colorScheme.primary.withOpacity(0.1),
              borderRadius: BorderRadius.circular(10),
            ),
            child: Icon(Icons.receipt_long, color: colorScheme.primary, size: 16),
          ),
          const SizedBox(width: 4),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(
                  '订单号 ${order['order_no'] ?? order['id'] ?? '-'}',
                  style: theme.textTheme.bodyLarge?.copyWith(
                    fontWeight: FontWeight.w700,
                  ),
                ),
                const SizedBox(height: 4),
                Text(
                  '状态：${order['status'] ?? '-'}',
                  style: theme.textTheme.bodySmall?.copyWith(
                    color: colorScheme.onSurfaceVariant,
                  ),
                ),
                const SizedBox(height: 2),
                Text(
                  _formatLocal(order['created_at']?.toString() ?? ''),
                  style: theme.textTheme.labelSmall?.copyWith(
                    color: colorScheme.onSurfaceVariant,
                  ),
                ),
              ],
            ),
          ),
          Text(
            '￥${_money(amount)}',
            style: theme.textTheme.bodyLarge?.copyWith(
              color: colorScheme.primary,
              fontWeight: FontWeight.w700,
            ),
          ),
        ],
      ),
    );
  }
}

class _WalletTxTile extends StatelessWidget {
  final Map<String, dynamic> tx;

  const _WalletTxTile({required this.tx});

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final colorScheme = theme.colorScheme;
    return Container(
      margin: const EdgeInsets.only(bottom: 6),
      padding: const EdgeInsets.all(6),
      decoration: BoxDecoration(
        color: colorScheme.surface,
        borderRadius: BorderRadius.circular(10),
        border: Border.all(color: colorScheme.outlineVariant.withOpacity(0.5)),
      ),
      child: Row(
        children: [
          Container(
            padding: const EdgeInsets.all(6),
            decoration: BoxDecoration(
              color: colorScheme.secondary.withOpacity(0.1),
              borderRadius: BorderRadius.circular(10),
            ),
            child: Icon(
              Icons.account_balance_wallet,
              color: colorScheme.secondary,
              size: 16,
            ),
          ),
          const SizedBox(width: 4),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(
                  tx['type']?.toString() ?? '-',
                  style: theme.textTheme.bodyLarge?.copyWith(
                    fontWeight: FontWeight.w700,
                  ),
                ),
                const SizedBox(height: 4),
                Text(
                  tx['note']?.toString() ?? '-',
                  style: theme.textTheme.bodySmall?.copyWith(
                    color: colorScheme.onSurfaceVariant,
                  ),
                ),
                const SizedBox(height: 2),
                Text(
                  _formatLocal(tx['created_at']?.toString() ?? ''),
                  style: theme.textTheme.labelSmall?.copyWith(
                    color: colorScheme.onSurfaceVariant,
                  ),
                ),
              ],
            ),
          ),
          Text(
            '￥${_money(tx['amount'])}',
            style: theme.textTheme.bodyLarge?.copyWith(
              color: colorScheme.primary,
              fontWeight: FontWeight.w700,
            ),
          ),
        ],
      ),
    );
  }
}

Color _statusColor(String status, ColorScheme scheme) {
  switch (status) {
    case 'active':
      return const Color(0xFF00A68C);
    case 'pending':
      return const Color(0xFFEF6C00);
    case 'blocked':
      return const Color(0xFFD32F2F);
    default:
      return scheme.outline;
  }
}

String _statusLabel(String status) {
  switch (status) {
    case 'active':
      return '正常';
    case 'pending':
      return '待审核';
    case 'blocked':
      return '已禁用';
    default:
      return status.isEmpty ? '未知' : status;
  }
}

String _formatLocal(String raw) {
  if (raw.isEmpty) return '';
  final dt = DateTime.tryParse(raw);
  if (dt == null) return raw;
  final local = dt.toLocal();
  return '${local.year}-${_pad2(local.month)}-${_pad2(local.day)} ${_pad2(local.hour)}:${_pad2(local.minute)}:${_pad2(local.second)}';
}

String _pad2(int v) => v.toString().padLeft(2, '0');

String _money(dynamic value) {
  if (value is num) return value.toStringAsFixed(2);
  return value?.toString() ?? '0.00';
}

String _avatarUrl(Map<String, dynamic> user, String baseUrl) {
  return resolveAvatarUrl(
    baseUrl: baseUrl,
    qq: user['qq']?.toString(),
    avatarUrl: user['avatar_url']?.toString() ?? user['avatar']?.toString(),
  );
}

Map<String, dynamic> _unwrapUser(Map<String, dynamic> raw) {
  if (raw['user'] is Map<String, dynamic>) {
    return raw['user'] as Map<String, dynamic>;
  }
  if (raw['data'] is Map<String, dynamic>) {
    return raw['data'] as Map<String, dynamic>;
  }
  return raw;
}
