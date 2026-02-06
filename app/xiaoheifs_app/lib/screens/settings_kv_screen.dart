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
    return items.where((item) => item.key.toLowerCase().contains(keyword)).toList();
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
          return const Scaffold(body: Center(child: CircularProgressIndicator()));
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
            actions: [
              TextButton(onPressed: _refresh, child: const Text('刷新')),
            ],
          ),
          body: ListView(
            padding: const EdgeInsets.all(16),
            children: [
              Card(
                child: Padding(
                  padding: const EdgeInsets.all(12),
                  child: Row(
                    children: [
                      Expanded(
                        child: TextField(
                          controller: _keywordController,
                          decoration: const InputDecoration(labelText: '搜索 Key'),
                        ),
                      ),
                      const SizedBox(width: 12),
                      FilledButton(onPressed: _refresh, child: const Text('搜索')),
                    ],
                  ),
                ),
              ),
              const SizedBox(height: 12),
              if (items.isEmpty)
                const Center(child: Text('暂无设置'))
              else
                ...items.map(
                  (item) => Card(
                    child: ListTile(
                      title: Text(item.key),
                      subtitle: Text(
                        item.value,
                        maxLines: 2,
                        overflow: TextOverflow.ellipsis,
                      ),
                      trailing: Text(item.updatedAt.isEmpty ? '' : _formatLocal(item.updatedAt)),
                      onTap: () => _edit(item),
                    ),
                  ),
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
          TextButton(onPressed: () => Navigator.pop(context, false), child: const Text('取消')),
          FilledButton(onPressed: () => Navigator.pop(context, true), child: const Text('保存')),
        ],
      ),
    );
    if (ok != true) return;
    final client = context.read<AppState>().apiClient;
    if (client == null) return;
    await client.patchJson('/admin/api/v1/settings', body: {
      'key': item.key,
      'value': controller.text,
    });
    _refresh();
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
