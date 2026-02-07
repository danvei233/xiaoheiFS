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
    return Material(
      child: Stack(
        children: [
          ListView(
            padding: const EdgeInsets.fromLTRB(12, 12, 12, 16),
            children: [
            Row(
              mainAxisAlignment: MainAxisAlignment.spaceBetween,
              children: [
                Text(
                  '工单管理',
                  style: theme.textTheme.titleMedium?.copyWith(
                        fontWeight: FontWeight.w700,
                        fontSize: 16,
                      ),
                ),
                OutlinedButton.icon(
                  onPressed: _loading ? null : () => _refresh(),
                  style: OutlinedButton.styleFrom(
                    padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 6),
                    minimumSize: const Size(0, 30),
                    tapTargetSize: MaterialTapTargetSize.shrinkWrap,
                    textStyle: const TextStyle(fontSize: 12),
                  ),
                  icon: const Icon(Icons.refresh, size: 16),
                  label: const Text('刷新'),
                ),
              ],
            ),
            const SizedBox(height: 6),
            Text(
              '处理用户提交的技术支持与咨询工单',
              style: theme.textTheme.bodySmall
                  ?.copyWith(color: theme.colorScheme.onSurfaceVariant),
            ),
            const SizedBox(height: 8),
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
            const SizedBox(height: 10),
            if (_items.isEmpty && !_loading)
              const _EmptyState()
            else
              ..._items.map((item) => _TicketCard(item: item, onTap: _openDetail)),
            const SizedBox(height: 10),
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
      ),
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
    final theme = Theme.of(context);
    final colorScheme = theme.colorScheme;
    return Container(
      padding: const EdgeInsets.all(8),
      decoration: BoxDecoration(
        color: colorScheme.surface,
        borderRadius: BorderRadius.circular(12),
        border: Border.all(color: colorScheme.outlineVariant.withOpacity(0.5)),
        boxShadow: [
          BoxShadow(
            color: colorScheme.shadow.withOpacity(0.05),
            blurRadius: 6,
            offset: const Offset(0, 1),
          ),
        ],
      ),
      child: Column(
        children: [
          Row(
            children: [
              Expanded(
                child: TextField(
                  controller: qController,
                  decoration: const InputDecoration(
                    hintText: '关键词（标题/内容）',
                    prefixIcon: Icon(Icons.search, size: 16),
                    isDense: true,
                    contentPadding: EdgeInsets.symmetric(horizontal: 10, vertical: 10),
                  ),
                ),
              ),
              const SizedBox(width: 8),
              FilledButton.icon(
                onPressed: onSearch,
                style: FilledButton.styleFrom(
                  padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 6),
                  minimumSize: const Size(0, 30),
                  tapTargetSize: MaterialTapTargetSize.shrinkWrap,
                  textStyle: const TextStyle(fontSize: 12),
                ),
                icon: const Icon(Icons.search_rounded, size: 16),
                label: const Text('搜索'),
              ),
            ],
          ),
          const SizedBox(height: 8),
          SizedBox(
            height: 32,
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
                  label: '待处理',
                  selected: status == 'open',
                  onTap: () => onStatusChanged('open'),
                ),
                const SizedBox(width: 8),
                _StatusFilterChip(
                  label: '等待用户',
                  selected: status == 'waiting_user',
                  onTap: () => onStatusChanged('waiting_user'),
                ),
                const SizedBox(width: 8),
                _StatusFilterChip(
                  label: '处理中',
                  selected: status == 'waiting_admin',
                  onTap: () => onStatusChanged('waiting_admin'),
                ),
                const SizedBox(width: 8),
                _StatusFilterChip(
                  label: '已关闭',
                  selected: status == 'closed',
                  onTap: () => onStatusChanged('closed'),
                ),
              ],
            ),
          ),
          const SizedBox(height: 8),
          Row(
            children: [
              Expanded(
                child: TextField(
                  controller: userIdController,
                  keyboardType: TextInputType.number,
                  decoration: const InputDecoration(
                    hintText: '用户 ID',
                    prefixIcon: Icon(Icons.person_outline, size: 16),
                    isDense: true,
                    contentPadding: EdgeInsets.symmetric(horizontal: 10, vertical: 10),
                  ),
                ),
              ),
              const SizedBox(width: 8),
              OutlinedButton.icon(
                onPressed: onReset,
                style: OutlinedButton.styleFrom(
                  padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 6),
                  minimumSize: const Size(0, 30),
                  tapTargetSize: MaterialTapTargetSize.shrinkWrap,
                  textStyle: const TextStyle(fontSize: 12),
                ),
                icon: const Icon(Icons.restart_alt_rounded, size: 16),
                label: const Text('重置'),
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
        borderRadius: BorderRadius.circular(10),
        child: Container(
          padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 6),
          decoration: BoxDecoration(
            color: bgColor,
            borderRadius: BorderRadius.circular(10),
            border: Border.all(color: borderColor),
          ),
          child: Row(
            mainAxisSize: MainAxisSize.min,
            children: [
              if (selected) ...[
                Icon(Icons.check_rounded, size: 14, color: textColor),
                const SizedBox(width: 4),
              ],
              Text(
                label,
                style: TextStyle(
                  color: textColor,
                  fontWeight: FontWeight.w600,
                  fontSize: 12,
                ),
              ),
            ],
          ),
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
    final theme = Theme.of(context);
    final colorScheme = theme.colorScheme;
    return Material(
      color: Colors.transparent,
      child: InkWell(
        borderRadius: BorderRadius.circular(16),
        onTap: () => onTap(item),
        child: Container(
          margin: const EdgeInsets.only(bottom: 8),
          padding: const EdgeInsets.all(10),
          decoration: BoxDecoration(
            color: colorScheme.surface,
            borderRadius: BorderRadius.circular(12),
            border: Border.all(color: colorScheme.outlineVariant.withOpacity(0.5)),
            boxShadow: [
              BoxShadow(
                color: colorScheme.shadow.withOpacity(0.05),
                blurRadius: 6,
                offset: const Offset(0, 1),
              ),
            ],
          ),
          child: Row(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              CircleAvatar(
                radius: 16,
                backgroundColor: statusMeta.color.withOpacity(0.12),
                child: Icon(statusMeta.icon, color: statusMeta.color, size: 16),
              ),
              const SizedBox(width: 8),
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                      item.subject.isEmpty ? '工单 #${item.id}' : item.subject,
                      style: theme.textTheme.titleSmall?.copyWith(
                        fontWeight: FontWeight.w700,
                        fontSize: 13,
                      ),
                      maxLines: 1,
                      overflow: TextOverflow.ellipsis,
                    ),
                    const SizedBox(height: 3),
                    Text(
                      '用户 ${item.username ?? item.userId}',
                      style: theme.textTheme.bodySmall?.copyWith(
                        color: colorScheme.onSurfaceVariant,
                      ),
                    ),
                    const SizedBox(height: 2),
                    Text(
                      _formatLocal(item.createdAt),
                      style: theme.textTheme.labelSmall?.copyWith(
                        color: colorScheme.onSurfaceVariant,
                        fontSize: 11,
                      ),
                    ),
                  ],
                ),
              ),
              _StatusTag(label: statusMeta.label, color: statusMeta.color),
            ],
          ),
        ),
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
                OutlinedButton(
                  onPressed: onPrev,
                  style: OutlinedButton.styleFrom(
                    padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 6),
                    minimumSize: const Size(0, 30),
                    tapTargetSize: MaterialTapTargetSize.shrinkWrap,
                    textStyle: const TextStyle(fontSize: 12),
                  ),
                  child: const Text('上一页'),
                ),
                OutlinedButton(
                  onPressed: onNext,
                  style: OutlinedButton.styleFrom(
                    padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 6),
                    minimumSize: const Size(0, 30),
                    tapTargetSize: MaterialTapTargetSize.shrinkWrap,
                    textStyle: const TextStyle(fontSize: 12),
                  ),
                  child: const Text('下一页'),
                ),
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
      padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 3),
      decoration: BoxDecoration(
        color: color.withOpacity(0.12),
        borderRadius: BorderRadius.circular(999),
      ),
      child: Text(
        label,
        style: TextStyle(color: color, fontSize: 11, fontWeight: FontWeight.w600),
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
        padding: EdgeInsets.all(20),
        child: Text('暂无工单', style: TextStyle(fontSize: 12)),
      ),
    );
  }
}
