import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../app_state.dart';

class PermissionsScreen extends StatefulWidget {
  const PermissionsScreen({super.key});

  @override
  State<PermissionsScreen> createState() => _PermissionsScreenState();
}

class _PermissionsScreenState extends State<PermissionsScreen> {
  Future<List<PermissionItem>>? _future;
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

  Future<List<PermissionItem>> _load(client) async {
    final resp = await client.getJson('/admin/api/v1/permissions/list');
    final items = (resp['items'] as List<dynamic>? ?? [])
        .map((e) => PermissionItem.fromJson(e as Map<String, dynamic>))
        .toList();
    final keyword = _keywordController.text.trim().toLowerCase();
    if (keyword.isEmpty) return items;
    return items.where((item) {
      return [
        item.code,
        item.name,
        item.friendlyName,
        item.category,
        item.parentCode,
      ].any((v) => v.toLowerCase().contains(keyword));
    }).toList();
  }

  void _refresh() {
    final client = context.read<AppState>().apiClient;
    if (client != null) setState(() => _future = _load(client));
  }

  Future<void> _sync() async {
    final client = context.read<AppState>().apiClient;
    if (client == null) return;
    await client.postJson('/admin/api/v1/permissions/sync');
    _refresh();
  }

  @override
  Widget build(BuildContext context) {
    return FutureBuilder<List<PermissionItem>>(
      future: _future,
      builder: (context, snapshot) {
        if (snapshot.connectionState == ConnectionState.waiting) {
          return const Scaffold(body: Center(child: CircularProgressIndicator()));
        }
        if (snapshot.hasError) {
          return Scaffold(
            appBar: AppBar(title: const Text('权限列表')),
            body: Center(child: Text('加载失败：$snapshot')),
          );
        }
        final items = snapshot.data ?? [];
        return Scaffold(
          appBar: AppBar(
            title: const Text('权限列表'),
            actions: [
              TextButton(onPressed: _sync, child: const Text('同步')),
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
                          decoration: const InputDecoration(labelText: '搜索'),
                        ),
                      ),
                      const SizedBox(width: 12),
                      FilledButton(onPressed: _refresh, child: const Text('应用')),
                    ],
                  ),
                ),
              ),
              const SizedBox(height: 12),
              if (items.isEmpty)
                const Center(child: Text('暂无权限'))
              else
                ...items.map(
                  (item) => Card(
                    child: ListTile(
                      title: Text(item.label),
                      subtitle: Text('${item.code} · ${item.category}'),
                      trailing: const Icon(Icons.edit),
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

  Future<void> _edit(PermissionItem item) async {
    final nameCtl = TextEditingController(text: item.name);
    final friendlyCtl = TextEditingController(text: item.friendlyName);
    final categoryCtl = TextEditingController(text: item.category);
    final parentCtl = TextEditingController(text: item.parentCode);
    final sortCtl = TextEditingController(text: item.sortOrder.toString());
    final ok = await showDialog<bool>(
      context: context,
      builder: (context) => AlertDialog(
        title: Text('编辑 ${item.code}'),
        content: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            TextField(controller: nameCtl, decoration: const InputDecoration(labelText: '名称')),
            TextField(controller: friendlyCtl, decoration: const InputDecoration(labelText: '显示名')),
            TextField(controller: categoryCtl, decoration: const InputDecoration(labelText: '分类')),
            TextField(controller: parentCtl, decoration: const InputDecoration(labelText: '父级编码')),
            TextField(
              controller: sortCtl,
              keyboardType: TextInputType.number,
              decoration: const InputDecoration(labelText: '排序'),
            ),
          ],
        ),
        actions: [
          TextButton(onPressed: () => Navigator.pop(context, false), child: const Text('取消')),
          FilledButton(onPressed: () => Navigator.pop(context, true), child: const Text('保存')),
        ],
      ),
    );
    if (ok != true) return;
    final client = context.read<AppState>().apiClient;
    if (client == null) return;
    await client.patchJson('/admin/api/v1/permissions/${item.code}', body: {
      'name': nameCtl.text.trim(),
      'friendly_name': friendlyCtl.text.trim(),
      'category': categoryCtl.text.trim(),
      'parent_code': parentCtl.text.trim(),
      'sort_order': int.tryParse(sortCtl.text.trim()) ?? 0,
    });
    _refresh();
  }
}

class PermissionItem {
  final String code;
  final String name;
  final String friendlyName;
  final String category;
  final String parentCode;
  final int sortOrder;

  PermissionItem({
    required this.code,
    required this.name,
    required this.friendlyName,
    required this.category,
    required this.parentCode,
    required this.sortOrder,
  });

  String get label => friendlyName.isNotEmpty ? friendlyName : (name.isNotEmpty ? name : code);

  factory PermissionItem.fromJson(Map<String, dynamic> json) {
    return PermissionItem(
      code: json['code'] as String? ?? '',
      name: json['name'] as String? ?? '',
      friendlyName: json['friendly_name'] as String? ?? '',
      category: json['category'] as String? ?? '',
      parentCode: json['parent_code'] as String? ?? '',
      sortOrder: json['sort_order'] as int? ?? 0,
    );
  }
}
