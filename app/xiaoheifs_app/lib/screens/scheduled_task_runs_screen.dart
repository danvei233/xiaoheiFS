import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../app_state.dart';

class ScheduledTaskRunsScreen extends StatefulWidget {
  final String taskKey;
  final String title;

  const ScheduledTaskRunsScreen({
    super.key,
    required this.taskKey,
    required this.title,
  });

  @override
  State<ScheduledTaskRunsScreen> createState() =>
      _ScheduledTaskRunsScreenState();
}

class _ScheduledTaskRunsScreenState extends State<ScheduledTaskRunsScreen> {
  bool _loading = false;
  int _page = 1;
  int _pageSize = 20;
  int _total = 0;
  List<Map<String, dynamic>> _items = [];

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    _load(showSpinner: true);
  }

  Future<void> _load({bool showSpinner = false}) async {
    final client = context.read<AppState>().apiClient;
    if (client == null) return;
    if (showSpinner) {
      setState(() => _loading = true);
    } else {
      _loading = true;
    }
    try {
      final resp = await client.getJson(
        '/admin/api/v1/scheduled-tasks/${widget.taskKey}/runs',
        query: {
          'limit': _pageSize.toString(),
          'offset': ((_page - 1) * _pageSize).toString(),
        },
      );
      if (!mounted) return;
      setState(() {
        _items = (resp['items'] as List<dynamic>? ?? [])
            .whereType<Map<String, dynamic>>()
            .toList();
        _total = _toInt(resp['total'], _items.length);
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

  int _toInt(dynamic value, int fallback) {
    if (value is int) return value;
    if (value is double) return value.toInt();
    if (value is String) return int.tryParse(value) ?? fallback;
    return fallback;
  }

  @override
  Widget build(BuildContext context) {
    final totalPages = (_total / _pageSize).ceil();
    return Scaffold(
      appBar: AppBar(
        title: Text('${widget.title} · 运行记录'),
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
              padding: const EdgeInsets.fromLTRB(12, 12, 12, 16),
              children: [
                Container(
                  margin: const EdgeInsets.only(bottom: 12),
                  padding: const EdgeInsets.all(12),
                  decoration: BoxDecoration(
                    gradient: const LinearGradient(
                      colors: [Color(0xFF1E88E5), Color(0xFF42A5F5)],
                      begin: Alignment.topLeft,
                      end: Alignment.bottomRight,
                    ),
                    borderRadius: BorderRadius.circular(14),
                  ),
                  child: const Row(
                    children: [
                      Icon(Icons.history_toggle_off, color: Colors.white),
                      SizedBox(width: 8),
                      Expanded(
                        child: Text(
                          '下拉可刷新，支持分页查看任务执行历史',
                          style: TextStyle(
                            color: Colors.white,
                            fontWeight: FontWeight.w600,
                          ),
                        ),
                      ),
                    ],
                  ),
                ),
                if (_items.isEmpty && !_loading)
                  const Padding(
                    padding: EdgeInsets.only(top: 40),
                    child: Center(child: Text('暂无运行记录')),
                  )
                else
                  ..._items.map((item) => _RunCard(item: item)),
                const SizedBox(height: 8),
                Text(
                  '第 $_page / ${totalPages == 0 ? 1 : totalPages} 页 · 共 $_total 条',
                ),
                const SizedBox(height: 8),
                Wrap(
                  spacing: 8,
                  runSpacing: 8,
                  children: [
                    OutlinedButton(
                      onPressed: _page > 1
                          ? () {
                              setState(() => _page -= 1);
                              _load();
                            }
                          : null,
                      child: const Text('上一页'),
                    ),
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

class _RunCard extends StatelessWidget {
  final Map<String, dynamic> item;

  const _RunCard({required this.item});

  @override
  Widget build(BuildContext context) {
    final status = (item['status'] ?? item['result'] ?? '').toString();
    final color = status.toLowerCase().contains('fail')
        ? const Color(0xFFD32F2F)
        : (status.toLowerCase().contains('success')
              ? const Color(0xFF2E7D32)
              : const Color(0xFFEF6C00));
    final started = _readTime(item, const [
      'run_at',
      'started_at',
      'start_at',
      'created_at',
    ]);
    final ended = _readTime(item, const [
      'finished_at',
      'end_at',
      'updated_at',
    ]);
    final duration = (item['duration_ms'] ?? item['duration'] ?? '').toString();
    final detail = (item['error'] ?? item['message'] ?? item['detail'] ?? '')
        .toString();

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
                    '记录 #${item['id'] ?? '-'}',
                    style: const TextStyle(fontWeight: FontWeight.w700),
                  ),
                ),
                Container(
                  padding: const EdgeInsets.symmetric(
                    horizontal: 8,
                    vertical: 4,
                  ),
                  decoration: BoxDecoration(
                    color: color.withOpacity(0.12),
                    borderRadius: BorderRadius.circular(999),
                  ),
                  child: Text(
                    status.isEmpty ? '未知' : status,
                    style: TextStyle(
                      color: color,
                      fontSize: 12,
                      fontWeight: FontWeight.w600,
                    ),
                  ),
                ),
              ],
            ),
            const SizedBox(height: 4),
            Text('开始：${_fmt(started)}'),
            Text('结束：${_fmt(ended)}'),
            if (duration.isNotEmpty) Text('耗时：$duration'),
            if (detail.isNotEmpty) ...[
              const SizedBox(height: 4),
              Text('详情：$detail'),
            ],
          ],
        ),
      ),
    );
  }

  String _readTime(Map<String, dynamic> map, List<String> keys) {
    for (final k in keys) {
      final v = map[k];
      if (v != null && v.toString().trim().isNotEmpty) return v.toString();
    }
    return '';
  }

  String _fmt(String raw) {
    if (raw.isEmpty) return '-';
    final dt = DateTime.tryParse(raw);
    if (dt == null) return raw;
    final local = dt.toLocal();
    String p(int v) => v.toString().padLeft(2, '0');
    return '${local.year}-${p(local.month)}-${p(local.day)} ${p(local.hour)}:${p(local.minute)}:${p(local.second)}';
  }
}
