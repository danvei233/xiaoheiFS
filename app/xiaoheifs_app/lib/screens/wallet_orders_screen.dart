import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../app_state.dart';

class WalletOrdersScreen extends StatefulWidget {
  const WalletOrdersScreen({super.key});

  @override
  State<WalletOrdersScreen> createState() => _WalletOrdersScreenState();
}

class _WalletOrdersScreenState extends State<WalletOrdersScreen> {
  Future<List<WalletOrderItem>>? _future;
  bool _loading = false;

  final _userIdController = TextEditingController();
  String _status = '';
  int _page = 1;
  int _pageSize = 20;
  int _total = 0;

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    final client = context.read<AppState>().apiClient;
    if (client != null) {
      _future = _load(client);
    }
  }

  @override
  void dispose() {
    _userIdController.dispose();
    super.dispose();
  }

  Future<List<WalletOrderItem>> _load(client) async {
    setState(() => _loading = true);
    try {
      final params = <String, String>{
        'limit': _pageSize.toString(),
        'offset': ((_page - 1) * _pageSize).toString(),
      };
      if (_status.isNotEmpty) params['status'] = _status;
      final userId = _userIdController.text.trim();
      if (userId.isNotEmpty && int.tryParse(userId) != null) {
        params['user_id'] = userId;
      }
      final resp = await client.getJson('/admin/api/v1/wallet/orders', query: params);
      final items = (resp['items'] as List<dynamic>? ?? [])
          .map((e) => WalletOrderItem.fromJson(e as Map<String, dynamic>))
          .toList();
      _total = (resp['total'] as int?) ?? items.length;
      return items;
    } finally {
      if (mounted) setState(() => _loading = false);
    }
  }

  void _refresh() {
    final client = context.read<AppState>().apiClient;
    if (client != null) setState(() => _future = _load(client));
  }

  void _reset() {
    _status = '';
    _userIdController.clear();
    _page = 1;
    _refresh();
  }

  @override
  Widget build(BuildContext context) {
    return FutureBuilder<List<WalletOrderItem>>(
      future: _future,
      builder: (context, snapshot) {
        if (snapshot.connectionState == ConnectionState.waiting) {
          return const Scaffold(body: Center(child: CircularProgressIndicator()));
        }
        if (snapshot.hasError) {
          return Scaffold(
            appBar: AppBar(title: const Text('钱包订单')),
            body: Center(child: Text('加载失败：$snapshot')),
          );
        }
        final items = snapshot.data ?? [];
        return Scaffold(
          appBar: AppBar(title: const Text('钱包订单')),
          body: ListView(
            padding: const EdgeInsets.fromLTRB(16, 16, 16, 24),
            children: [
              _WalletFilterCard(
                status: _status,
                userIdController: _userIdController,
                loading: _loading,
                onStatusChanged: (value) => _status = value ?? '',
                onSearch: () {
                  _page = 1;
                  _refresh();
                },
                onReset: _reset,
                onRefresh: _refresh,
              ),
              const SizedBox(height: 12),
              if (items.isEmpty)
                const Center(child: Text('暂无订单'))
              else
                ...items.map(
                  (item) => _WalletOrderTile(
                    item: item,
                    onApprove: _approve,
                    onReject: _reject,
                  ),
                ),
              const SizedBox(height: 12),
              _PaginationBar(
                page: _page,
                pageSize: _pageSize,
                total: _total,
                onPrev: _page > 1
                    ? () {
                        _page -= 1;
                        _refresh();
                      }
                    : null,
                onNext: _page * _pageSize < _total
                    ? () {
                        _page += 1;
                        _refresh();
                      }
                    : null,
              ),
            ],
          ),
        );
      },
    );
  }

  Future<void> _approve(WalletOrderItem item) async {
    final client = context.read<AppState>().apiClient;
    if (client == null) return;
    await client.postJson('/admin/api/v1/wallet/orders/${item.id}/approve');
    _refresh();
  }

  Future<void> _reject(WalletOrderItem item) async {
    final reasonCtl = TextEditingController();
    final ok = await showDialog<bool>(
      context: context,
      builder: (context) => AlertDialog(
        title: const Text('拒绝订单'),
        content: TextField(controller: reasonCtl, decoration: const InputDecoration(hintText: '拒绝原因')),
        actions: [
          TextButton(onPressed: () => Navigator.pop(context, false), child: const Text('取消')),
          FilledButton(onPressed: () => Navigator.pop(context, true), child: const Text('确认')),
        ],
      ),
    );
    if (ok != true) return;
    final client = context.read<AppState>().apiClient;
    if (client == null) return;
    await client.postJson('/admin/api/v1/wallet/orders/${item.id}/reject', body: {
      'reason': reasonCtl.text.trim(),
    });
    _refresh();
  }
}

class _WalletOrderTile extends StatelessWidget {
  final WalletOrderItem item;
  final Future<void> Function(WalletOrderItem) onApprove;
  final Future<void> Function(WalletOrderItem) onReject;

  const _WalletOrderTile({
    required this.item,
    required this.onApprove,
    required this.onReject,
  });

  @override
  Widget build(BuildContext context) {
    final statusColor = _statusColor(item.status);
    final theme = Theme.of(context);
    final colorScheme = theme.colorScheme;
    return Material(
      color: Colors.transparent,
      child: InkWell(
        borderRadius: BorderRadius.circular(16),
        onTap: item.status == 'pending_review'
            ? () async {
                final action = await showModalBottomSheet<String>(
                  context: context,
                  builder: (context) => SafeArea(
                    child: Column(
                      mainAxisSize: MainAxisSize.min,
                      children: [
                        ListTile(
                          title: const Text('通过'),
                          onTap: () => Navigator.pop(context, 'approve'),
                        ),
                        ListTile(
                          title: const Text('拒绝'),
                          onTap: () => Navigator.pop(context, 'reject'),
                        ),
                      ],
                    ),
                  ),
                );
                if (action == 'approve') {
                  await onApprove(item);
                } else if (action == 'reject') {
                  await onReject(item);
                }
              }
            : null,
        child: Container(
          margin: const EdgeInsets.only(bottom: 12),
          padding: const EdgeInsets.all(12),
          decoration: BoxDecoration(
            color: colorScheme.surface,
            borderRadius: BorderRadius.circular(16),
            border:
                Border.all(color: colorScheme.outlineVariant.withOpacity(0.5)),
            boxShadow: [
              BoxShadow(
                color: colorScheme.shadow.withOpacity(0.05),
                blurRadius: 8,
                offset: const Offset(0, 2),
              ),
            ],
          ),
          child: Row(
            children: [
              Container(
                padding: const EdgeInsets.all(8),
                decoration: BoxDecoration(
                  color: statusColor.withOpacity(0.12),
                  borderRadius: BorderRadius.circular(10),
                ),
                child: Icon(
                  item.type == 'withdraw'
                      ? Icons.outbox_outlined
                      : Icons.account_balance_wallet_outlined,
                  color: statusColor,
                ),
              ),
              const SizedBox(width: 10),
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                      '用户 ${item.userId} · ${_typeText(item.type)}',
                      style: theme.textTheme.titleSmall?.copyWith(
                        fontWeight: FontWeight.w700,
                      ),
                    ),
                    const SizedBox(height: 4),
                    Text(
                      _formatLocal(item.createdAt),
                      style: theme.textTheme.labelSmall?.copyWith(
                        color: colorScheme.onSurfaceVariant,
                      ),
                    ),
                    if (item.note.isNotEmpty) ...[
                      const SizedBox(height: 2),
                      Text(
                        item.note,
                        style: theme.textTheme.bodySmall?.copyWith(
                          color: colorScheme.onSurfaceVariant,
                        ),
                      ),
                    ],
                  ],
                ),
              ),
              Column(
                crossAxisAlignment: CrossAxisAlignment.end,
                children: [
                  Text(
                    '${item.type == 'withdraw' ? '-' : '+'}¥${item.amount.toStringAsFixed(2)}',
                    style: TextStyle(
                      color: item.type == 'withdraw'
                          ? const Color(0xFFD32F2F)
                          : const Color(0xFF00A68C),
                      fontWeight: FontWeight.w700,
                    ),
                  ),
                  const SizedBox(height: 6),
                  Container(
                    padding:
                        const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
                    decoration: BoxDecoration(
                      color: statusColor.withOpacity(0.12),
                      borderRadius: BorderRadius.circular(999),
                    ),
                    child: Text(
                      _statusText(item.status),
                      style: TextStyle(
                        color: statusColor,
                        fontSize: 12,
                        fontWeight: FontWeight.w600,
                      ),
                    ),
                  ),
                ],
              ),
            ],
          ),
        ),
      ),
    );
  }
}

class _WalletFilterCard extends StatelessWidget {
  final String status;
  final TextEditingController userIdController;
  final bool loading;
  final ValueChanged<String?> onStatusChanged;
  final VoidCallback onSearch;
  final VoidCallback onReset;
  final VoidCallback onRefresh;

  const _WalletFilterCard({
    required this.status,
    required this.userIdController,
    required this.loading,
    required this.onStatusChanged,
    required this.onSearch,
    required this.onReset,
    required this.onRefresh,
  });

  @override
  Widget build(BuildContext context) {
    final colorScheme = Theme.of(context).colorScheme;
    return Container(
      padding: const EdgeInsets.all(12),
      decoration: BoxDecoration(
        color: colorScheme.surface,
        borderRadius: BorderRadius.circular(16),
        border: Border.all(color: colorScheme.outlineVariant.withOpacity(0.5)),
        boxShadow: [
          BoxShadow(
            color: colorScheme.shadow.withOpacity(0.05),
            blurRadius: 8,
            offset: const Offset(0, 2),
          ),
        ],
      ),
      child: Column(
        children: [
          SizedBox(
            height: 40,
            child: ListView(
              scrollDirection: Axis.horizontal,
              children: [
                _StatusFilterChip(
                  label: '全部',
                  selected: status.isEmpty,
                  onTap: () => onStatusChanged(''),
                ),
                const SizedBox(width: 8),
                _StatusFilterChip(
                  label: '待审核',
                  selected: status == 'pending_review',
                  onTap: () => onStatusChanged('pending_review'),
                ),
                const SizedBox(width: 8),
                _StatusFilterChip(
                  label: '已通过',
                  selected: status == 'approved',
                  onTap: () => onStatusChanged('approved'),
                ),
                const SizedBox(width: 8),
                _StatusFilterChip(
                  label: '已拒绝',
                  selected: status == 'rejected',
                  onTap: () => onStatusChanged('rejected'),
                ),
              ],
            ),
          ),
          const SizedBox(height: 10),
          Row(
            children: [
              Expanded(
                child: TextField(
                  controller: userIdController,
                  keyboardType: TextInputType.number,
                  decoration: const InputDecoration(
                    hintText: '用户 ID',
                    prefixIcon: Icon(Icons.person_outline),
                  ),
                ),
              ),
              const SizedBox(width: 10),
              FilledButton.icon(
                onPressed: loading ? null : onSearch,
                icon: const Icon(Icons.search_rounded),
                label: const Text('搜索'),
              ),
            ],
          ),
          const SizedBox(height: 10),
          Row(
            children: [
              Expanded(
                child: OutlinedButton.icon(
                  onPressed: onReset,
                  icon: const Icon(Icons.restart_alt_rounded),
                  label: const Text('重置'),
                ),
              ),
              const SizedBox(width: 10),
              Expanded(
                child: OutlinedButton.icon(
                  onPressed: onRefresh,
                  icon: const Icon(Icons.refresh_rounded),
                  label: const Text('刷新'),
                ),
              ),
            ],
          ),
        ],
      ),
    );
  }
}

class _StatusFilterChip extends StatelessWidget {
  final String label;
  final bool selected;
  final VoidCallback onTap;

  const _StatusFilterChip({
    required this.label,
    required this.selected,
    required this.onTap,
  });

  @override
  Widget build(BuildContext context) {
    final colorScheme = Theme.of(context).colorScheme;
    final bgColor = selected
        ? colorScheme.primaryContainer.withOpacity(0.7)
        : colorScheme.surface;
    final borderColor = selected
        ? colorScheme.primary
        : colorScheme.outlineVariant.withOpacity(0.7);
    final textColor =
        selected ? colorScheme.primary : colorScheme.onSurface;

    return Material(
      color: Colors.transparent,
      child: InkWell(
        onTap: onTap,
        borderRadius: BorderRadius.circular(12),
        child: Container(
          padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
          decoration: BoxDecoration(
            color: bgColor,
            borderRadius: BorderRadius.circular(12),
            border: Border.all(color: borderColor),
          ),
          child: Row(
            mainAxisSize: MainAxisSize.min,
            children: [
              if (selected) ...[
                Icon(Icons.check_rounded, size: 16, color: textColor),
                const SizedBox(width: 6),
              ],
              Text(
                label,
                style: TextStyle(
                  color: textColor,
                  fontWeight: FontWeight.w600,
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}

class WalletOrderItem {
  final int id;
  final int userId;
  final String type;
  final String status;
  final double amount;
  final String note;
  final String createdAt;

  WalletOrderItem({
    required this.id,
    required this.userId,
    required this.type,
    required this.status,
    required this.amount,
    required this.note,
    required this.createdAt,
  });

  factory WalletOrderItem.fromJson(Map<String, dynamic> json) {
    return WalletOrderItem(
      id: json['id'] as int? ?? 0,
      userId: json['user_id'] as int? ?? 0,
      type: json['type'] as String? ?? '',
      status: json['status'] as String? ?? '',
      amount: (json['amount'] as num?)?.toDouble() ?? 0,
      note: json['note'] as String? ?? '',
      createdAt: json['created_at']?.toString() ?? '',
    );
  }
}

String _typeText(String type) {
  switch (type) {
    case 'recharge':
      return '充值';
    case 'withdraw':
      return '提现';
    case 'refund':
      return '退款';
    default:
      return type;
  }
}

String _statusText(String status) {
  switch (status) {
    case 'pending_review':
      return '待审核';
    case 'approved':
      return '已通过';
    case 'rejected':
      return '已拒绝';
    default:
      return status;
  }
}

Color _statusColor(String status) {
  switch (status) {
    case 'pending_review':
      return const Color(0xFFEF6C00);
    case 'approved':
      return const Color(0xFF00A68C);
    case 'rejected':
      return const Color(0xFFD32F2F);
    default:
      return Colors.black54;
  }
}

String _formatLocal(String raw) {
  if (raw.isEmpty) return '';
  final dt = DateTime.tryParse(raw);
  if (dt == null) return raw;
  final local = dt.toLocal();
  return '${local.year}-${_pad2(local.month)}-${_pad2(local.day)} ${_pad2(local.hour)}:${_pad2(local.minute)}';
}

String _pad2(int v) => v.toString().padLeft(2, '0');

class _PaginationBar extends StatelessWidget {
  final int page;
  final int pageSize;
  final int total;
  final VoidCallback? onPrev;
  final VoidCallback? onNext;

  const _PaginationBar({
    required this.page,
    required this.pageSize,
    required this.total,
    this.onPrev,
    this.onNext,
  });

  @override
  Widget build(BuildContext context) {
    final totalPages = (total / pageSize).ceil();
    return Row(
      mainAxisAlignment: MainAxisAlignment.spaceBetween,
      children: [
        Text('第 $page / $totalPages 页 · 共 $total 条'),
        Row(
          children: [
            OutlinedButton(onPressed: onPrev, child: const Text('上一页')),
            const SizedBox(width: 8),
            OutlinedButton(onPressed: onNext, child: const Text('下一页')),
          ],
        )
      ],
    );
  }
}
