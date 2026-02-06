import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../app_state.dart';
import 'ticket_detail_screen.dart';

class TicketsScreen extends StatefulWidget {
  const TicketsScreen({super.key});

  @override
  State<TicketsScreen> createState() => _TicketsScreenState();
}

class _TicketsScreenState extends State<TicketsScreen> {
  bool _loading = false;
  List<TicketItem> _items = [];

  String _status = '';
  final _userIdController = TextEditingController();
  final _qController = TextEditingController();

  int _page = 1;
  int _pageSize = 20;
  int _total = 0;

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    _refresh(showSpinner: _items.isEmpty);
  }

  @override
  void dispose() {
    _userIdController.dispose();
    _qController.dispose();
    super.dispose();
  }

  Future<void> _refresh({bool showSpinner = false}) async {
    final client = context.read<AppState>().apiClient;
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
      if (_status.isNotEmpty) query['status'] = _status;
      final userId = _userIdController.text.trim();
      if (userId.isNotEmpty) query['user_id'] = userId;
      final q = _qController.text.trim();
      if (q.isNotEmpty) query['q'] = q;
      final resp = await client.getJson('/admin/api/v1/tickets', query: query);
      final items = (resp['items'] as List<dynamic>? ?? [])
          .map((e) => TicketItem.fromJson(e as Map<String, dynamic>))
          .toList();
      await _attachUsernames(client, items);
      _total = (resp['total'] as int?) ?? items.length;
      if (mounted) {
        setState(() {
          _items = items;
          _loading = false;
        });
      }
    } catch (e) {
      if (mounted) {
        setState(() => _loading = false);
        ScaffoldMessenger.of(context)
            .showSnackBar(SnackBar(content: Text('加载失败：$e')));
      }
    }
  }

  Future<void> _attachUsernames(client, List<TicketItem> items) async {
    final ids = items.map((e) => e.userId).toSet();
    for (final id in ids) {
      if (id <= 0) continue;
      try {
        final user = await client.getJson('/admin/api/v1/users/$id');
        final map = user['user'] is Map<String, dynamic>
            ? user['user'] as Map<String, dynamic>
            : (user['data'] is Map<String, dynamic> ? user['data'] as Map<String, dynamic> : user);
        final name = map['username'] as String?;
        if (name == null || name.isEmpty) continue;
        for (final item in items) {
          if (item.userId == id) item.username = name;
        }
      } catch (_) {
        continue;
      }
    }
  }

  void _resetFilters() {
    _status = '';
    _userIdController.clear();
    _qController.clear();
    _page = 1;
    _refresh();
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Stack(
      children: [
        ListView(
          padding: const EdgeInsets.all(16),
          children: [
            Row(
              children: [
                IconButton(
                  onPressed: () => Navigator.maybePop(context),
                  icon: const Icon(Icons.arrow_back),
                ),
                const SizedBox(width: 4),
                Text('工单管理', style: theme.textTheme.titleLarge?.copyWith(fontWeight: FontWeight.w700)),
              ],
            ),
            const SizedBox(height: 8),
            Row(
              mainAxisAlignment: MainAxisAlignment.spaceBetween,
              children: [
                Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text('处理用户提交的技术支持与咨询工单',
                        style: theme.textTheme.bodySmall
                            ?.copyWith(color: Colors.black54)),
                  ],
                ),
                OutlinedButton.icon(
                  onPressed: _loading ? null : () => _refresh(),
                  icon: const Icon(Icons.refresh),
                  label: const Text('刷新'),
                ),
              ],
            ),
            const SizedBox(height: 12),
            _FilterCard(
              status: _status,
              onStatusChanged: (value) {
                _status = value;
                _page = 1;
                _refresh();
              },
              userIdController: _userIdController,
              qController: _qController,
              onSearch: () {
                _page = 1;
                _refresh();
              },
              onReset: _resetFilters,
            ),
            const SizedBox(height: 12),
            if (_items.isEmpty && !_loading)
              const _EmptyState()
            else
              ..._items.map((item) => _TicketCard(item: item, onTap: _openDetail)),
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
        if (_loading)
          const Positioned(
            left: 0,
            right: 0,
            top: 0,
            child: LinearProgressIndicator(minHeight: 2),
          ),
      ],
    );
  }

  Future<void> _openDetail(TicketItem item) async {
    await Navigator.push<void>(
      context,
      MaterialPageRoute(builder: (_) => TicketDetailScreen(ticketId: item.id)),
    );
    _refresh();
  }
}

class _FilterCard extends StatelessWidget {
  final String status;
  final ValueChanged<String> onStatusChanged;
  final TextEditingController userIdController;
  final TextEditingController qController;
  final VoidCallback onSearch;
  final VoidCallback onReset;

  const _FilterCard({
    required this.status,
    required this.onStatusChanged,
    required this.userIdController,
    required this.qController,
    required this.onSearch,
    required this.onReset,
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
                      DropdownMenuItem(value: '', child: Text('全部')),
                      DropdownMenuItem(value: 'open', child: Text('待处理')),
                      DropdownMenuItem(value: 'waiting_user', child: Text('等待用户')),
                      DropdownMenuItem(value: 'waiting_admin', child: Text('处理中')),
                      DropdownMenuItem(value: 'closed', child: Text('已关闭')),
                    ],
                    onChanged: (value) => onStatusChanged(value ?? ''),
                    decoration: const InputDecoration(labelText: '状态'),
                  ),
                ),
                const SizedBox(width: 12),
                Expanded(
                  child: TextField(
                    controller: userIdController,
                    keyboardType: TextInputType.number,
                    decoration: const InputDecoration(labelText: '用户 ID'),
                  ),
                ),
              ],
            ),
            const SizedBox(height: 12),
            TextField(
              controller: qController,
              decoration: const InputDecoration(labelText: '关键词（标题/内容）'),
            ),
            const SizedBox(height: 12),
            Row(
              children: [
                Expanded(child: FilledButton(onPressed: onSearch, child: const Text('筛选'))),
                const SizedBox(width: 12),
                OutlinedButton(onPressed: onReset, child: const Text('重置')),
              ],
            )
          ],
        ),
      ),
    );
  }
}

class _TicketCard extends StatelessWidget {
  final TicketItem item;
  final Future<void> Function(TicketItem) onTap;

  const _TicketCard({required this.item, required this.onTap});

  @override
  Widget build(BuildContext context) {
    final statusMeta = _ticketStatusMeta(item.status);
    return Card(
      child: ListTile(
        leading: CircleAvatar(
          backgroundColor: statusMeta.color.withOpacity(0.12),
          child: Icon(statusMeta.icon, color: statusMeta.color),
        ),
        title: Text(item.subject.isEmpty ? '工单 #${item.id}' : item.subject),
        subtitle: Text('用户 ${item.username ?? item.userId} · ${_formatLocal(item.createdAt)}'),
        trailing: _StatusTag(label: statusMeta.label, color: statusMeta.color),
        onTap: () => onTap(item),
      ),
    );
  }
}

class TicketItem {
  final int id;
  final int userId;
  final String subject;
  final String status;
  final String createdAt;
  final String updatedAt;
  String? username;

  TicketItem({
    required this.id,
    required this.userId,
    required this.subject,
    required this.status,
    required this.createdAt,
    required this.updatedAt,
    this.username,
  });

  factory TicketItem.fromJson(Map<String, dynamic> json) {
    return TicketItem(
      id: _asInt(json['id']),
      userId: _asInt(json['user_id']),
      subject: json['subject'] as String? ?? '',
      status: json['status'] as String? ?? '',
      createdAt: json['created_at']?.toString() ?? '',
      updatedAt: json['updated_at']?.toString() ?? '',
    );
  }
}

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
    return LayoutBuilder(
      builder: (context, constraints) {
        return Wrap(
          alignment: WrapAlignment.spaceBetween,
          runSpacing: 8,
          spacing: 12,
          children: [
            ConstrainedBox(
              constraints: BoxConstraints(maxWidth: constraints.maxWidth),
              child: Text(
                '第 $page / $totalPages 页 · 共 $total 条',
                style: Theme.of(context).textTheme.bodySmall,
                overflow: TextOverflow.ellipsis,
              ),
            ),
            Wrap(
              spacing: 8,
              children: [
                OutlinedButton(onPressed: onPrev, child: const Text('上一页')),
                OutlinedButton(onPressed: onNext, child: const Text('下一页')),
              ],
            ),
          ],
        );
      },
    );
  }
}

class _StatusTag extends StatelessWidget {
  final String label;
  final Color color;

  const _StatusTag({required this.label, required this.color});

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
      decoration: BoxDecoration(
        color: color.withOpacity(0.12),
        borderRadius: BorderRadius.circular(999),
      ),
      child: Text(
        label,
        style: TextStyle(color: color, fontSize: 12, fontWeight: FontWeight.w600),
      ),
    );
  }
}

class _StatusMeta {
  final String label;
  final IconData icon;
  final Color color;

  const _StatusMeta(this.label, this.icon, this.color);
}

_StatusMeta _ticketStatusMeta(String status) {
  switch (status) {
    case 'open':
      return const _StatusMeta('待处理', Icons.report, Color(0xFF1E88E5));
    case 'waiting_user':
      return const _StatusMeta('等待用户', Icons.hourglass_bottom, Color(0xFFEF6C00));
    case 'waiting_admin':
      return const _StatusMeta('处理中', Icons.support_agent, Color(0xFF7B1FA2));
    case 'closed':
      return const _StatusMeta('已关闭', Icons.check_circle, Color(0xFF00A68C));
    default:
      return _StatusMeta(status.isEmpty ? '未知' : status, Icons.info, const Color(0xFF546E7A));
  }
}

String _formatLocal(String raw) {
  if (raw.isEmpty) return '-';
  final dt = DateTime.tryParse(raw);
  if (dt == null) return raw;
  final local = dt.toLocal();
  return '${local.year}-${_pad2(local.month)}-${_pad2(local.day)} ${_pad2(local.hour)}:${_pad2(local.minute)}';
}

String _pad2(int v) => v.toString().padLeft(2, '0');

int _asInt(dynamic value) {
  if (value is int) return value;
  if (value is num) return value.toInt();
  if (value is String) return int.tryParse(value) ?? 0;
  return 0;
}

class _EmptyState extends StatelessWidget {
  const _EmptyState();

  @override
  Widget build(BuildContext context) {
    return const Center(
      child: Padding(
        padding: EdgeInsets.all(32),
        child: Text('暂无工单'),
      ),
    );
  }
}
