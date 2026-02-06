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
          appBar: AppBar(
            title: const Text('钱包订单'),
            actions: [
              TextButton(onPressed: _reset, child: const Text('重置')),
              TextButton(onPressed: _refresh, child: const Text('刷新')),
            ],
          ),
          body: ListView(
            padding: const EdgeInsets.all(16),
            children: [
              Card(
                child: Padding(
                  padding: const EdgeInsets.all(12),
                  child: Column(
                    children: [
                      Row(
                        children: [
                          Expanded(
                            child: DropdownButtonFormField<String>(
                              value: _status.isEmpty ? null : _status,
                              items: const [
                                DropdownMenuItem(value: '', child: Text('全部')),
                                DropdownMenuItem(value: 'pending_review', child: Text('待审核')),
                                DropdownMenuItem(value: 'approved', child: Text('已通过')),
                                DropdownMenuItem(value: 'rejected', child: Text('已拒绝')),
                              ],
                              onChanged: (value) => _status = value ?? '',
                              decoration: const InputDecoration(labelText: '状态'),
                            ),
                          ),
                          const SizedBox(width: 12),
                          Expanded(
                            child: TextField(
                              controller: _userIdController,
                              keyboardType: TextInputType.number,
                              decoration: const InputDecoration(labelText: '用户ID'),
                            ),
                          ),
                        ],
                      ),
                      const SizedBox(height: 12),
                      Row(
                        children: [
                          Expanded(
                            child: FilledButton(
                              onPressed: _loading
                                  ? null
                                  : () {
                                      _page = 1;
                                      _refresh();
                                    },
                              child: const Text('筛选'),
                            ),
                          ),
                        ],
                      )
                    ],
                  ),
                ),
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
    return Card(
      child: ListTile(
        title: Text('用户 ${item.userId} · ${_typeText(item.type)}'),
        subtitle: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text('状态：${_statusText(item.status)} · ${_formatLocal(item.createdAt)}'),
            if (item.note.isNotEmpty) Text('备注：${item.note}'),
          ],
        ),
        trailing: Column(
          mainAxisAlignment: MainAxisAlignment.center,
          children: [
            Text(
              '${item.type == 'withdraw' ? '-' : '+'}¥${item.amount.toStringAsFixed(2)}',
              style: TextStyle(
                color: item.type == 'withdraw' ? const Color(0xFFD32F2F) : const Color(0xFF00A68C),
                fontWeight: FontWeight.w600,
              ),
            ),
            const SizedBox(height: 6),
            Container(
              padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
              decoration: BoxDecoration(
                color: statusColor.withOpacity(0.12),
                borderRadius: BorderRadius.circular(999),
              ),
              child: Text(
                _statusText(item.status),
                style: TextStyle(color: statusColor, fontSize: 12, fontWeight: FontWeight.w600),
              ),
            ),
          ],
        ),
        onTap: item.status == 'pending_review'
            ? () async {
                final action = await showModalBottomSheet<String>(
                  context: context,
                  builder: (context) => SafeArea(
                    child: Column(
                      mainAxisSize: MainAxisSize.min,
                      children: [
                        ListTile(title: const Text('通过'), onTap: () => Navigator.pop(context, 'approve')),
                        ListTile(title: const Text('拒绝'), onTap: () => Navigator.pop(context, 'reject')),
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
