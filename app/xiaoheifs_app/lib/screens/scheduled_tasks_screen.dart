import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../app_state.dart';

class ScheduledTasksScreen extends StatefulWidget {
  const ScheduledTasksScreen({super.key});

  @override
  State<ScheduledTasksScreen> createState() => _ScheduledTasksScreenState();
}

class _ScheduledTasksScreenState extends State<ScheduledTasksScreen> {
  Future<List<TaskItem>>? _future;
  bool _busy = false;

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    final client = context.read<AppState>().apiClient;
    if (client != null) {
      _future = _load(client);
    }
  }

  Future<List<TaskItem>> _load(client) async {
    final resp = await client.getJson('/admin/api/v1/scheduled-tasks');
    return (resp['items'] as List<dynamic>? ?? [])
        .map((e) => TaskItem.fromJson(e as Map<String, dynamic>))
        .toList();
  }

  void _refresh() {
    final client = context.read<AppState>().apiClient;
    if (client != null) setState(() => _future = _load(client));
  }

  @override
  Widget build(BuildContext context) {
    return FutureBuilder<List<TaskItem>>(
      future: _future,
      builder: (context, snapshot) {
        if (snapshot.connectionState == ConnectionState.waiting) {
          return const Scaffold(body: Center(child: CircularProgressIndicator()));
        }
        if (snapshot.hasError) {
          return Scaffold(
            appBar: AppBar(title: const Text('定时任务')),
            body: Center(child: Text('加载失败：$snapshot')),
          );
        }
        final items = snapshot.data ?? [];
        return Scaffold(
          appBar: AppBar(
            title: const Text('定时任务'),
            actions: [
              IconButton(onPressed: _refresh, icon: const Icon(Icons.refresh)),
            ],
          ),
          body: ListView.builder(
            padding: const EdgeInsets.all(16),
            itemCount: items.isEmpty ? 1 : items.length,
            itemBuilder: (context, index) {
              if (items.isEmpty) {
                return const Center(child: Text('暂无任务'));
              }
              final item = items[index];
              return Card(
                child: ListTile(
                  title: Text(item.name),
                  subtitle: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(item.description),
                      const SizedBox(height: 4),
                      Text('策略：${item.strategy}'),
                      Text('上次：${_formatLocal(item.lastRunAt)}'),
                      Text('下次：${_formatLocal(item.nextRunAt)}'),
                      if (item.lastStatus.isNotEmpty)
                        Text('状态：${item.lastStatus}${item.lastError.isNotEmpty ? ' · ${item.lastError}' : ''}'),
                    ],
                  ),
                  trailing: Switch(
                    value: item.enabled,
                    onChanged: _busy ? null : (value) => _toggle(item.key, value),
                  ),
                  onTap: () => _openConfig(item),
                ),
              );
            },
          ),
        );
      },
    );
  }

  Future<void> _toggle(String key, bool enabled) async {
    setState(() => _busy = true);
    try {
      final client = context.read<AppState>().apiClient;
      if (client == null) return;
      await client.patchJson(
        '/admin/api/v1/scheduled-tasks/$key',
        body: {'enabled': enabled},
      );
      _refresh();
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  Future<void> _openConfig(TaskItem item) async {
    final intervalCtl = TextEditingController(text: item.intervalSec?.toString() ?? '3600');
    final dailyCtl = TextEditingController(text: item.dailyAt ?? '00:00');
    String strategy = item.strategy;
    bool enabled = item.enabled;
    final ok = await showDialog<bool>(
      context: context,
      builder: (context) => AlertDialog(
        title: Text('配置 ${item.name}'),
        content: StatefulBuilder(
          builder: (context, setModal) {
            return Column(
              mainAxisSize: MainAxisSize.min,
              children: [
                SwitchListTile(
                  value: enabled,
                  onChanged: (v) => setModal(() => enabled = v),
                  title: const Text('启用'),
                ),
                DropdownButtonFormField<String>(
                  value: strategy,
                  items: const [
                    DropdownMenuItem(value: 'interval', child: Text('间隔执行')),
                    DropdownMenuItem(value: 'daily', child: Text('每日执行')),
                  ],
                  onChanged: (v) => setModal(() => strategy = v ?? 'interval'),
                  decoration: const InputDecoration(labelText: '策略'),
                ),
                const SizedBox(height: 8),
                if (strategy == 'interval')
                  TextField(
                    controller: intervalCtl,
                    keyboardType: TextInputType.number,
                    decoration: const InputDecoration(labelText: '间隔(秒)'),
                  ),
                if (strategy == 'daily')
                  TextField(
                    controller: dailyCtl,
                    decoration: const InputDecoration(labelText: '每日时间(HH:mm)'),
                  ),
              ],
            );
          },
        ),
        actions: [
          TextButton(onPressed: () => Navigator.pop(context, false), child: const Text('取消')),
          FilledButton(onPressed: () => Navigator.pop(context, true), child: const Text('保存')),
        ],
      ),
    );
    if (ok != true) return;
    setState(() => _busy = true);
    try {
      final client = context.read<AppState>().apiClient;
      if (client == null) return;
      final payload = <String, dynamic>{
        'enabled': enabled,
        'strategy': strategy,
      };
      if (strategy == 'interval') {
        payload['interval_sec'] = int.tryParse(intervalCtl.text.trim()) ?? 3600;
      } else {
        payload['daily_at'] = dailyCtl.text.trim();
      }
      await client.patchJson('/admin/api/v1/scheduled-tasks/${item.key}', body: payload);
      _refresh();
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }
}

class TaskItem {
  final String key;
  final String name;
  final String description;
  final String strategy;
  final bool enabled;
  final int? intervalSec;
  final String? dailyAt;
  final String lastRunAt;
  final String nextRunAt;
  final String lastStatus;
  final String lastError;

  TaskItem({
    required this.key,
    required this.name,
    required this.description,
    required this.strategy,
    required this.enabled,
    this.intervalSec,
    this.dailyAt,
    required this.lastRunAt,
    required this.nextRunAt,
    required this.lastStatus,
    required this.lastError,
  });

  factory TaskItem.fromJson(Map<String, dynamic> json) {
    return TaskItem(
      key: json['key'] as String? ?? '',
      name: json['name'] as String? ?? '',
      description: json['description'] as String? ?? '',
      strategy: json['strategy'] as String? ?? '',
      enabled: json['enabled'] as bool? ?? false,
      intervalSec: json['interval_sec'] as int?,
      dailyAt: json['daily_at'] as String?,
      lastRunAt: json['last_run_at']?.toString() ?? '',
      nextRunAt: json['next_run_at']?.toString() ?? '',
      lastStatus: json['last_status']?.toString() ?? '',
      lastError: json['last_error']?.toString() ?? '',
    );
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
