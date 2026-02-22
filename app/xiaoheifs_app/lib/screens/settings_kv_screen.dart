import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../app_state.dart';

class SettingsKvScreen extends StatefulWidget {
  const SettingsKvScreen({super.key});

  @override
  State<SettingsKvScreen> createState() => _SettingsKvScreenState();
}

class _SettingsKvScreenState extends State<SettingsKvScreen> {
  Future<List<SettingItem>>? _future;
  final _keywordController = TextEditingController();
  bool _busy = false;

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
    _keywordController.dispose();
    super.dispose();
  }

  Future<List<SettingItem>> _load(client) async {
    final resp = await client.getJson('/admin/api/v1/settings');
    final items = (resp['items'] as List<dynamic>? ?? [])
        .map((e) => SettingItem.fromJson(e as Map<String, dynamic>))
        .toList();
    final keyword = _keywordController.text.trim().toLowerCase();
    if (keyword.isEmpty) return items;
    return items
        .where((item) => item.key.toLowerCase().contains(keyword))
        .toList();
  }

  void _refresh() {
    final client = context.read<AppState>().apiClient;
    if (client != null) setState(() => _future = _load(client));
  }

  @override
  Widget build(BuildContext context) {
    return FutureBuilder<List<SettingItem>>(
      future: _future,
      builder: (context, snapshot) {
        if (snapshot.connectionState == ConnectionState.waiting) {
          return const Scaffold(
            body: Center(child: CircularProgressIndicator()),
          );
        }
        if (snapshot.hasError) {
          return Scaffold(
            appBar: AppBar(title: const Text('系统设置')),
            body: Center(child: Text('加载失败：$snapshot')),
          );
        }
        final items = snapshot.data ?? [];
        return Scaffold(
          appBar: AppBar(
            title: const Text('系统设置'),
            actions: [TextButton(onPressed: _refresh, child: const Text('刷新'))],
          ),
          body: Stack(
            children: [
              RefreshIndicator(
                onRefresh: () async => _refresh(),
                child: ListView(
                  physics: const AlwaysScrollableScrollPhysics(),
                  padding: const EdgeInsets.fromLTRB(16, 16, 16, 24),
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
                          Icon(
                            Icons.settings_applications,
                            color: Colors.white,
                          ),
                          SizedBox(width: 8),
                          Expanded(
                            child: Text(
                              '管理系统配置项，点击条目可直接编辑',
                              style: TextStyle(
                                color: Colors.white,
                                fontWeight: FontWeight.w600,
                              ),
                            ),
                          ),
                        ],
                      ),
                    ),
                    Card(
                      child: Padding(
                        padding: const EdgeInsets.all(12),
                        child: Column(
                          children: [
                            TextField(
                              controller: _keywordController,
                              decoration: const InputDecoration(
                                labelText: '搜索 Key',
                              ),
                            ),
                            const SizedBox(height: 10),
                            Row(
                              children: [
                                Expanded(
                                  child: OutlinedButton(
                                    onPressed: _refresh,
                                    child: const Text('应用过滤'),
                                  ),
                                ),
                              ],
                            ),
                          ],
                        ),
                      ),
                    ),
                    const SizedBox(height: 12),
                    if (items.isEmpty)
                      const Padding(
                        padding: EdgeInsets.only(top: 28),
                        child: Center(child: Text('暂无设置')),
                      )
                    else
                      ...items.map(
                        (item) => Container(
                          margin: const EdgeInsets.only(bottom: 10),
                          padding: const EdgeInsets.all(12),
                          decoration: BoxDecoration(
                            color: Colors.white,
                            borderRadius: BorderRadius.circular(14),
                            border: Border.all(color: const Color(0xFFE2E8F0)),
                          ),
                          child: InkWell(
                            onTap: () => _edit(item),
                            borderRadius: BorderRadius.circular(10),
                            child: Column(
                              crossAxisAlignment: CrossAxisAlignment.start,
                              children: [
                                Row(
                                  children: [
                                    Expanded(
                                      child: Text(
                                        item.key,
                                        style: const TextStyle(
                                          fontWeight: FontWeight.w700,
                                        ),
                                      ),
                                    ),
                                    Text(
                                      item.updatedAt.isEmpty
                                          ? '-'
                                          : _formatLocal(item.updatedAt),
                                      style: TextStyle(
                                        fontSize: 12,
                                        color: Colors.grey.shade700,
                                      ),
                                    ),
                                  ],
                                ),
                                const SizedBox(height: 6),
                                Text(
                                  item.value.isEmpty ? '-' : item.value,
                                  maxLines: 3,
                                  overflow: TextOverflow.ellipsis,
                                  style: TextStyle(
                                    fontSize: 13,
                                    color: Colors.grey.shade800,
                                  ),
                                ),
                              ],
                            ),
                          ),
                        ),
                      ),
                  ],
                ),
              ),
              if (_busy)
                const Positioned(
                  left: 0,
                  right: 0,
                  top: 0,
                  child: LinearProgressIndicator(minHeight: 2),
                ),
            ],
          ),
        );
      },
    );
  }

  Future<void> _edit(SettingItem item) async {
    final controller = TextEditingController(text: item.value);
    final ok = await showDialog<bool>(
      context: context,
      builder: (context) => AlertDialog(
        title: Text('编辑 ${item.key}'),
        content: TextField(controller: controller, maxLines: 6),
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
    final client = context.read<AppState>().apiClient;
    if (client == null) return;
    setState(() => _busy = true);
    try {
      await client.patchJson(
        '/admin/api/v1/settings',
        body: {'key': item.key, 'value': controller.text},
      );
      _refresh();
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }
}

class SettingItem {
  final String key;
  final String value;
  final String updatedAt;

  SettingItem({
    required this.key,
    required this.value,
    required this.updatedAt,
  });

  factory SettingItem.fromJson(Map<String, dynamic> json) {
    return SettingItem(
      key: json['key'] as String? ?? '',
      value: json['value'] as String? ?? '',
      updatedAt: json['updated_at']?.toString() ?? '',
    );
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
