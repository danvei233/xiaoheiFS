import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../app_state.dart';
import '../services/api_client.dart';

class RealnameRecordsScreen extends StatefulWidget {
  const RealnameRecordsScreen({super.key});

  @override
  State<RealnameRecordsScreen> createState() => _RealnameRecordsScreenState();
}

class _RealnameRecordsScreenState extends State<RealnameRecordsScreen> {
  ApiClient? _client;
  bool _loading = false;
  bool _submitting = false;
  String _statusFilter = '';
  final TextEditingController _userIdCtl = TextEditingController();

  int _page = 1;
  int _pageSize = 20;
  int _total = 0;
  List<_RealnameRecord> _items = [];

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    final client = context.read<AppState>().apiClient;
    if (client?.baseUrl != _client?.baseUrl ||
        client?.token != _client?.token ||
        client?.apiKey != _client?.apiKey) {
      _client = client;
      if (client != null) {
        _load(showSpinner: true);
      }
    }
  }

  @override
  void dispose() {
    _userIdCtl.dispose();
    super.dispose();
  }

  Future<void> _load({bool showSpinner = false}) async {
    final client = _client;
    if (client == null) return;
    if (showSpinner) {
      setState(() => _loading = true);
    } else {
      _loading = true;
    }
    try {
      final query = <String, String>{
        'limit': _pageSize.toString(),
        'offset': ((_page - 1) * _pageSize).toString(),
      };
      final userId = _userIdCtl.text.trim();
      if (userId.isNotEmpty) {
        query['user_id'] = userId;
      }
      final resp = await client.getJson(
        '/admin/api/v1/realname/records',
        query: query,
      );
      var rows = (resp['items'] as List<dynamic>? ?? [])
          .whereType<Map<String, dynamic>>()
          .map(_RealnameRecord.fromJson)
          .toList();
      if (_statusFilter.isNotEmpty) {
        rows = rows.where((e) => e.status == _statusFilter).toList();
      }
      if (!mounted) return;
      setState(() {
        _items = rows;
        _total = _asInt(resp['total']);
        _loading = false;
      });
    } catch (e) {
      if (!mounted) return;
      setState(() => _loading = false);
      ScaffoldMessenger.of(
        context,
      ).showSnackBar(SnackBar(content: Text('加载失败：$e')));
    }
  }

  int _asInt(dynamic value) {
    if (value is int) return value;
    if (value is double) return value.toInt();
    if (value is String) return int.tryParse(value) ?? 0;
    return 0;
  }

  Future<void> _updateStatus(
    _RealnameRecord item,
    String status, {
    String reason = '',
  }) async {
    final client = _client;
    if (client == null || _submitting) return;
    setState(() => _submitting = true);
    try {
      await client.patchJson(
        '/admin/api/v1/users/${item.userId}/realname-status',
        body: {'status': status, 'reason': reason},
      );
      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text(status == 'verified' ? '已通过认证' : '已拒绝认证')),
      );
      await _load();
    } on ApiException catch (e) {
      if (!mounted) return;
      ScaffoldMessenger.of(
        context,
      ).showSnackBar(SnackBar(content: Text('操作失败：${e.message}')));
    } catch (e) {
      if (!mounted) return;
      ScaffoldMessenger.of(
        context,
      ).showSnackBar(SnackBar(content: Text('操作失败：$e')));
    } finally {
      if (mounted) setState(() => _submitting = false);
    }
  }

  Future<void> _reject(_RealnameRecord item) async {
    final reasonCtl = TextEditingController();
    final ok = await showModalBottomSheet<bool>(
      context: context,
      isScrollControlled: true,
      builder: (context) {
        return Padding(
          padding: EdgeInsets.fromLTRB(
            16,
            16,
            16,
            MediaQuery.of(context).viewInsets.bottom + 16,
          ),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              const Text(
                '驳回原因',
                style: TextStyle(fontSize: 16, fontWeight: FontWeight.w700),
              ),
              const SizedBox(height: 10),
              TextField(
                controller: reasonCtl,
                maxLines: 3,
                decoration: const InputDecoration(hintText: '请输入驳回原因（可选）'),
              ),
              const SizedBox(height: 12),
              Row(
                children: [
                  Expanded(
                    child: OutlinedButton(
                      onPressed: () => Navigator.pop(context, false),
                      child: const Text('取消'),
                    ),
                  ),
                  const SizedBox(width: 8),
                  Expanded(
                    child: FilledButton(
                      onPressed: () => Navigator.pop(context, true),
                      child: const Text('确认驳回'),
                    ),
                  ),
                ],
              ),
            ],
          ),
        );
      },
    );
    if (ok == true) {
      await _updateStatus(item, 'rejected', reason: reasonCtl.text.trim());
    }
    reasonCtl.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final totalPages = (_total / _pageSize).ceil();
    return Scaffold(
      appBar: AppBar(
        title: const Text('实名认证记录'),
        actions: [
          IconButton(
            onPressed: _loading ? null : () => _load(showSpinner: true),
            icon: const Icon(Icons.refresh),
          ),
        ],
      ),
      body: Stack(
        children: [
          RefreshIndicator(
            onRefresh: _load,
            child: ListView(
              physics: const AlwaysScrollableScrollPhysics(),
              padding: const EdgeInsets.fromLTRB(12, 10, 12, 16),
              children: [
                Container(
                  padding: const EdgeInsets.all(10),
                  decoration: BoxDecoration(
                    color: Colors.white,
                    borderRadius: BorderRadius.circular(12),
                    border: Border.all(color: const Color(0xFFE5EAF2)),
                  ),
                  child: Column(
                    children: [
                      TextField(
                        controller: _userIdCtl,
                        keyboardType: TextInputType.number,
                        decoration: const InputDecoration(
                          prefixIcon: Icon(Icons.search),
                          hintText: '按用户 ID 过滤（可空）',
                        ),
                        onSubmitted: (_) {
                          _page = 1;
                          _load();
                        },
                      ),
                      const SizedBox(height: 8),
                      SizedBox(
                        height: 36,
                        child: ListView(
                          scrollDirection: Axis.horizontal,
                          children: [
                            _StatusChip(
                              label: '全部',
                              selected: _statusFilter.isEmpty,
                              onTap: () {
                                setState(() => _statusFilter = '');
                                _load();
                              },
                            ),
                            _StatusChip(
                              label: '待审核',
                              selected: _statusFilter == 'pending',
                              onTap: () {
                                setState(() => _statusFilter = 'pending');
                                _load();
                              },
                            ),
                            _StatusChip(
                              label: '已通过',
                              selected: _statusFilter == 'verified',
                              onTap: () {
                                setState(() => _statusFilter = 'verified');
                                _load();
                              },
                            ),
                            _StatusChip(
                              label: '已拒绝',
                              selected: _statusFilter == 'rejected',
                              onTap: () {
                                setState(() => _statusFilter = 'rejected');
                                _load();
                              },
                            ),
                          ],
                        ),
                      ),
                    ],
                  ),
                ),
                const SizedBox(height: 10),
                if (_items.isEmpty && !_loading)
                  const Center(
                    child: Padding(
                      padding: EdgeInsets.only(top: 40),
                      child: Text('暂无实名认证记录'),
                    ),
                  )
                else
                  ..._items.map(
                    (item) => _RecordCard(
                      item: item,
                      submitting: _submitting,
                      onApprove: () => _updateStatus(item, 'verified'),
                      onReject: () => _reject(item),
                    ),
                  ),
                const SizedBox(height: 8),
                Row(
                  children: [
                    Expanded(
                      child: Text(
                        '第 $_page / ${totalPages == 0 ? 1 : totalPages} 页 · 共 $_total 条',
                      ),
                    ),
                    OutlinedButton(
                      onPressed: _page > 1
                          ? () {
                              setState(() => _page -= 1);
                              _load();
                            }
                          : null,
                      child: const Text('上一页'),
                    ),
                    const SizedBox(width: 8),
                    OutlinedButton(
                      onPressed: _page * _pageSize < _total
                          ? () {
                              setState(() => _page += 1);
                              _load();
                            }
                          : null,
                      child: const Text('下一页'),
                    ),
                  ],
                ),
              ],
            ),
          ),
          if (_loading)
            const Positioned(
              left: 0,
              right: 0,
              top: 0,
              child: LinearProgressIndicator(minHeight: 2),
            ),
        ],
      ),
    );
  }
}

class _RecordCard extends StatelessWidget {
  final _RealnameRecord item;
  final bool submitting;
  final VoidCallback onApprove;
  final VoidCallback onReject;

  const _RecordCard({
    required this.item,
    required this.submitting,
    required this.onApprove,
    required this.onReject,
  });

  @override
  Widget build(BuildContext context) {
    final canReview = item.status != 'verified';
    return Card(
      margin: const EdgeInsets.only(bottom: 10),
      child: Padding(
        padding: const EdgeInsets.all(10),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                Expanded(
                  child: Text(
                    item.realName.isEmpty ? '-' : item.realName,
                    style: const TextStyle(
                      fontSize: 15,
                      fontWeight: FontWeight.w700,
                    ),
                  ),
                ),
                _StatusBadge(status: item.status),
              ],
            ),
            const SizedBox(height: 4),
            Text('用户 ID：${item.userId}'),
            Text('证件号：${item.idNumber.isEmpty ? '-' : item.idNumber}'),
            Text('认证来源：${item.provider.isEmpty ? '-' : item.provider}'),
            Text('申请时间：${_fmt(item.createdAt)}'),
            Text('通过时间：${_fmt(item.verifiedAt)}'),
            if (item.reason.isNotEmpty) Text('原因：${item.reason}'),
            const SizedBox(height: 8),
            Row(
              children: [
                Expanded(
                  child: FilledButton(
                    onPressed: (submitting || !canReview) ? null : onApprove,
                    child: const Text('通过'),
                  ),
                ),
                const SizedBox(width: 8),
                Expanded(
                  child: OutlinedButton(
                    onPressed: (submitting || !canReview) ? null : onReject,
                    child: const Text('驳回'),
                  ),
                ),
              ],
            ),
          ],
        ),
      ),
    );
  }

  String _fmt(String raw) {
    if (raw.isEmpty) return '-';
    final dt = DateTime.tryParse(raw);
    if (dt == null) return raw;
    final local = dt.toLocal();
    String p(int v) => v.toString().padLeft(2, '0');
    return '${local.year}-${p(local.month)}-${p(local.day)} ${p(local.hour)}:${p(local.minute)}';
  }
}

class _StatusBadge extends StatelessWidget {
  final String status;

  const _StatusBadge({required this.status});

  @override
  Widget build(BuildContext context) {
    final (text, color) = switch (status) {
      'verified' => ('已通过', const Color(0xFF2E7D32)),
      'rejected' => ('已拒绝', const Color(0xFFD32F2F)),
      'pending' => ('待审核', const Color(0xFFEF6C00)),
      _ => (status.isEmpty ? '未知' : status, Colors.black54),
    };
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
      decoration: BoxDecoration(
        color: color.withOpacity(0.12),
        borderRadius: BorderRadius.circular(999),
      ),
      child: Text(
        text,
        style: TextStyle(
          color: color,
          fontWeight: FontWeight.w600,
          fontSize: 12,
        ),
      ),
    );
  }
}

class _StatusChip extends StatelessWidget {
  final String label;
  final bool selected;
  final VoidCallback onTap;

  const _StatusChip({
    required this.label,
    required this.selected,
    required this.onTap,
  });

  @override
  Widget build(BuildContext context) {
    final c = Theme.of(context).colorScheme;
    return Padding(
      padding: const EdgeInsets.only(right: 8),
      child: InkWell(
        borderRadius: BorderRadius.circular(999),
        onTap: onTap,
        child: Container(
          padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
          decoration: BoxDecoration(
            borderRadius: BorderRadius.circular(999),
            color: selected ? c.primaryContainer : Colors.white,
            border: Border.all(
              color: selected ? c.primary : const Color(0xFFD7DFEB),
            ),
          ),
          child: Text(
            label,
            style: TextStyle(
              fontSize: 12,
              color: selected ? c.primary : c.onSurface,
            ),
          ),
        ),
      ),
    );
  }
}

class _RealnameRecord {
  final int id;
  final int userId;
  final String realName;
  final String idNumber;
  final String status;
  final String provider;
  final String reason;
  final String createdAt;
  final String verifiedAt;

  const _RealnameRecord({
    required this.id,
    required this.userId,
    required this.realName,
    required this.idNumber,
    required this.status,
    required this.provider,
    required this.reason,
    required this.createdAt,
    required this.verifiedAt,
  });

  factory _RealnameRecord.fromJson(Map<String, dynamic> json) {
    return _RealnameRecord(
      id: _toInt(json['id']),
      userId: _toInt(json['user_id']),
      realName: (json['real_name'] ?? '').toString(),
      idNumber: (json['id_number'] ?? '').toString(),
      status: (json['status'] ?? '').toString(),
      provider: (json['provider'] ?? '').toString(),
      reason: (json['reason'] ?? '').toString(),
      createdAt: (json['created_at'] ?? '').toString(),
      verifiedAt: (json['verified_at'] ?? '').toString(),
    );
  }

  static int _toInt(dynamic value) {
    if (value is int) return value;
    if (value is double) return value.toInt();
    if (value is String) return int.tryParse(value) ?? 0;
    return 0;
  }
}
