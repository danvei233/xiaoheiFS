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
        body: ListView(
          padding: const EdgeInsets.all(16),
          children: [
            _InfoCard(
              title: '用户信息',
              leading: _Avatar(
                url: _avatarUrl(
                  _user,
                  context.read<AppState>().apiClient?.baseUrl ?? '',
                ),
                radius: 22,
              ),
              lines: [
                '用户ID：${_user['id'] ?? '-'}',
                '用户名：${_user['username'] ?? '-'}',
                '邮箱：${_user['email'] ?? '-'}',
                '手机号：${_user['phone'] ?? '-'}',
                'QQ：${_user['qq'] ?? '-'}',
                '状态：${_user['status'] ?? '-'}',
                '角色：${_user['role'] ?? '-'}',
                '创建时间：${_formatLocal(_user['created_at']?.toString() ?? '')}',
              ],
            ),
            const SizedBox(height: 12),
            _ActionBar(
              busy: _busy,
              onToggle: _toggleStatus,
              onReset: _resetPassword,
              onImpersonate: _impersonate,
            ),
            const SizedBox(height: 16),
            _SectionTitle(title: '实名认证'),
            _RealnameCard(
              status: _realnameStatus,
              reasonController: _realnameReason,
              busy: _busy,
              hasRecord: _realname != null,
              onStatusChanged: (value) =>
                  setState(() => _realnameStatus = value),
              onSubmit: _updateRealname,
            ),
            const SizedBox(height: 16),
            _SectionTitle(title: '钱包余额'),
            _InfoCard(
              title: '余额',
              lines: [
                '可用余额：￥${_money(_wallet['balance'])}',
                '冻结余额：￥${_money(_wallet['frozen'])}',
              ],
            ),
            const SizedBox(height: 16),
            _SectionTitle(title: '订单记录'),
            if (_orders.isEmpty)
              const _EmptyLine(text: '暂无订单')
            else
              ..._orders.map((order) => _OrderTile(order: order)),
            const SizedBox(height: 16),
            _SectionTitle(title: '钱包记录'),
            if (_walletTx.isEmpty)
              const _EmptyLine(text: '暂无钱包记录')
            else
              ..._walletTx.map((tx) => _WalletTxTile(tx: tx)),
          ],
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
        FilledButton(
          onPressed: busy ? null : onToggle,
          child: const Text('禁用/启用'),
        ),
        OutlinedButton(
          onPressed: busy ? null : onReset,
          child: const Text('重置密码'),
        ),
        OutlinedButton(
          onPressed: busy ? null : onImpersonate,
          child: const Text('以此用户登录'),
        ),
      ],
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
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(12),
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
                const SizedBox(width: 12),
                Expanded(
                  child: TextField(
                    controller: reasonController,
                    decoration: const InputDecoration(labelText: '审核备注（可选）'),
                    enabled: hasRecord,
                  ),
                ),
              ],
            ),
            const SizedBox(height: 12),
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
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                if (leading != null) ...[leading!, const SizedBox(width: 12)],
                Text(title, style: Theme.of(context).textTheme.titleMedium),
              ],
            ),
            const SizedBox(height: 8),
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

class _SectionTitle extends StatelessWidget {
  final String title;

  const _SectionTitle({required this.title});

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 6),
      child: Text(title, style: Theme.of(context).textTheme.titleMedium),
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
    final amount = order['total_amount'];
    return Card(
      child: ListTile(
        leading: const Icon(Icons.receipt_long),
        title: Text('订单号 ${order['order_no'] ?? order['id'] ?? '-'}'),
        subtitle: Text(
          '状态 ${order['status'] ?? '-'}\n${_formatLocal(order['created_at']?.toString() ?? '')}',
        ),
        trailing: Text('¥${_money(amount)}'),
      ),
    );
  }
}

class _WalletTxTile extends StatelessWidget {
  final Map<String, dynamic> tx;

  const _WalletTxTile({required this.tx});

  @override
  Widget build(BuildContext context) {
    return Card(
      child: ListTile(
        leading: const Icon(Icons.account_balance_wallet),
        title: Text(tx['type']?.toString() ?? '-'),
        subtitle: Text(
          '${tx['note'] ?? '-'}\n${_formatLocal(tx['created_at']?.toString() ?? '')}',
        ),
        trailing: Text('¥${_money(tx['amount'])}'),
      ),
    );
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
