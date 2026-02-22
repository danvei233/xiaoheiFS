import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../app_state.dart';
import 'scheduled_task_runs_screen.dart';

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
          return const Scaffold(
            body: Center(child: CircularProgressIndicator()),
          );
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
          body: RefreshIndicator(
            onRefresh: () async => _refresh(),
            child: ListView.builder(
              physics: const AlwaysScrollableScrollPhysics(),
              padding: const EdgeInsets.fromLTRB(16, 16, 16, 24),
              itemCount: items.isEmpty ? 2 : items.length + 1,
              itemBuilder: (context, index) {
                if (index == 0) {
                  return Container(
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
                        Icon(Icons.schedule_outlined, color: Colors.white),
                        SizedBox(width: 8),
                        Expanded(
                          child: Text(
                            '点击任务卡片可编辑策略，支持查看运行记录',
                            style: TextStyle(
                              color: Colors.white,
                              fontWeight: FontWeight.w600,
                            ),
                          ),
                        ),
                      ],
                    ),
                  );
                }
                if (items.isEmpty) {
                  return const Padding(
                    padding: EdgeInsets.only(top: 30),
                    child: Center(child: Text('暂无任务')),
                  );
                }
                final item = items[index - 1];
                final theme = Theme.of(context);
                final colorScheme = theme.colorScheme;
                final statusColor = item.enabled
                    ? const Color(0xFF00A68C)
                    : const Color(0xFF546E7A);
                return Material(
                  color: Colors.transparent,
                  child: InkWell(
                    borderRadius: BorderRadius.circular(16),
                    onTap: () => _openConfig(item),
                    child: Container(
                      margin: const EdgeInsets.only(bottom: 12),
                      padding: const EdgeInsets.all(12),
                      decoration: BoxDecoration(
                        color: colorScheme.surface,
                        borderRadius: BorderRadius.circular(16),
                        border: Border.all(
                          color: colorScheme.outlineVariant.withOpacity(0.5),
                        ),
                        boxShadow: [
                          BoxShadow(
                            color: colorScheme.shadow.withOpacity(0.05),
                            blurRadius: 8,
                            offset: const Offset(0, 2),
                          ),
                        ],
                      ),
                      child: Column(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                          Row(
                            children: [
                              Container(
                                padding: const EdgeInsets.all(8),
                                decoration: BoxDecoration(
                                  color: statusColor.withOpacity(0.12),
                                  borderRadius: BorderRadius.circular(10),
                                ),
                                child: Icon(Icons.schedule, color: statusColor),
                              ),
                              const SizedBox(width: 10),
                              Expanded(
                                child: Column(
                                  crossAxisAlignment: CrossAxisAlignment.start,
                                  children: [
                                    Text(
                                      item.name,
                                      style: theme.textTheme.titleSmall
                                          ?.copyWith(
                                            fontWeight: FontWeight.w700,
                                          ),
                                    ),
                                    const SizedBox(height: 4),
                                    Text(
                                      item.description,
                                      style: theme.textTheme.bodySmall
                                          ?.copyWith(
                                            color: colorScheme.onSurfaceVariant,
                                          ),
                                    ),
                                  ],
                                ),
                              ),
                              Switch(
                                value: item.enabled,
                                onChanged: _busy
                                    ? null
                                    : (value) => _toggle(item.key, value),
                              ),
                            ],
                          ),
                          const SizedBox(height: 8),
                          Wrap(
                            spacing: 8,
                            runSpacing: 6,
                            children: [
                              _InfoPill(
                                label: item.strategy,
                                color: colorScheme.primary,
                              ),
                              _InfoPill(
                                label: '上次 ${_formatLocal(item.lastRunAt)}',
                                color: colorScheme.onSurfaceVariant,
                                outlined: true,
                              ),
                              _InfoPill(
                                label: '下次 ${_formatLocal(item.nextRunAt)}',
                                color: colorScheme.onSurfaceVariant,
                                outlined: true,
                              ),
                            ],
                          ),
                          if (item.lastStatus.isNotEmpty) ...[
                            const SizedBox(height: 8),
                            Text(
                              '状态：${item.lastStatus}${item.lastError.isNotEmpty ? ' · ${item.lastError}' : ''}',
                              style: theme.textTheme.bodySmall?.copyWith(
                                color:
                                    item.lastStatus.toLowerCase().contains(
                                      'fail',
                                    )
                                    ? const Color(0xFFD32F2F)
                                    : colorScheme.onSurfaceVariant,
                              ),
                            ),
                          ],
                          const SizedBox(height: 8),
                          Row(
                            children: [
                              OutlinedButton.icon(
                                onPressed: () {
                                  Navigator.push(
                                    context,
                                    MaterialPageRoute(
                                      builder: (_) => ScheduledTaskRunsScreen(
                                        taskKey: item.key,
                                        title: item.name,
                                      ),
                                    ),
                                  );
                                },
                                icon: const Icon(Icons.history, size: 16),
                                label: const Text('运行记录'),
                              ),
                            ],
                          ),
                        ],
                      ),
                    ),
                  ),
                );
              },
            ),
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
    final intervalCtl = TextEditingController(
      text: item.intervalSec?.toString() ?? '3600',
    );
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
          TextButton(
            onPressed: () => Navigator.pop(context, false),
            child: const Text('取消'),
          ),
          FilledButton(
            onPressed: () => Navigator.pop(context, true),
            child: const Text('保存'),
          ),
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
      await client.patchJson(
        '/admin/api/v1/scheduled-tasks/${item.key}',
        body: payload,
      );
      _refresh();
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }
}

class _InfoPill extends StatelessWidget {
  final String label;
  final Color color;
  final bool outlined;

  const _InfoPill({
    required this.label,
    required this.color,
    this.outlined = false,
  });

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
      decoration: BoxDecoration(
        color: outlined ? Colors.transparent : color.withOpacity(0.12),
        borderRadius: BorderRadius.circular(999),
        border: outlined ? Border.all(color: color.withOpacity(0.4)) : null,
      ),
      child: Text(
        label,
        style: TextStyle(
          color: outlined ? color : color,
          fontSize: 12,
          fontWeight: FontWeight.w600,
        ),
      ),
    );
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
