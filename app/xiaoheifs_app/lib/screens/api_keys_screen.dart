import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:provider/provider.dart';

import '../app_state.dart';

class ApiKeysScreen extends StatefulWidget {
  const ApiKeysScreen({super.key});

  @override
  State<ApiKeysScreen> createState() => _ApiKeysScreenState();
}

class _ApiKeysScreenState extends State<ApiKeysScreen> {
  Future<List<ApiKeyItem>>? _future;
  bool _loading = false;

  int _page = 1;
  int _pageSize = 20;
  int _total = 0;
  String _status = '';
  final _keywordController = TextEditingController();

  List<PermissionGroupItem> _permissionGroups = const [];

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    final client = context.read<AppState>().apiClient;
    if (client != null) {
      _future = _load(client);
      _loadPermissionGroups(client);
    }
  }

  @override
  void dispose() {
    _keywordController.dispose();
    super.dispose();
  }

  Future<void> _loadPermissionGroups(client) async {
    try {
      final resp = await client.getJson('/admin/api/v1/permission-groups');
      final items = (resp['items'] as List<dynamic>? ?? [])
          .map((e) => PermissionGroupItem.fromJson(e as Map<String, dynamic>))
          .toList();
      if (mounted) {
        setState(() => _permissionGroups = items);
      }
    } catch (_) {}
  }

  String _permissionGroupName(int? id) {
    if (id == null) return '-';
    for (final g in _permissionGroups) {
      if (g.id == id) return g.name.isEmpty ? '-' : g.name;
    }
    return '-';
  }

  Future<List<ApiKeyItem>> _load(client) async {
    setState(() => _loading = true);
    try {
      final resp = await client.getJson('/admin/api/v1/api-keys', query: {
        'limit': _pageSize.toString(),
        'offset': ((_page - 1) * _pageSize).toString(),
      });
      final items = (resp['items'] as List<dynamic>? ?? [])
          .map((e) => ApiKeyItem.fromJson(e as Map<String, dynamic>))
          .toList();
      _total = (resp['total'] as int?) ?? items.length;

      final keyword = _keywordController.text.trim();
      final filtered = items.where((item) {
        if (_status.isNotEmpty && item.status != _status) return false;
        if (keyword.isNotEmpty && !item.keyHash.contains(keyword)) return false;
        return true;
      }).toList();
      return filtered;
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
    _keywordController.clear();
    _page = 1;
    _refresh();
  }

  @override
  Widget build(BuildContext context) {
    return FutureBuilder<List<ApiKeyItem>>(
      future: _future,
      builder: (context, snapshot) {
        if (snapshot.connectionState == ConnectionState.waiting) {
          return const Scaffold(body: Center(child: CircularProgressIndicator()));
        }
        if (snapshot.hasError) {
          return Scaffold(
            appBar: AppBar(title: const Text('API Keys')),
            body: Center(child: Text('加载失败：$snapshot')),
          );
        }
        final items = snapshot.data ?? [];
        return Scaffold(
          appBar: AppBar(
            title: const Text('API Keys'),
            actions: [
              TextButton(onPressed: _reset, child: const Text('重置')),
              TextButton(onPressed: _refresh, child: const Text('刷新')),
              IconButton(onPressed: _createKey, icon: const Icon(Icons.add)),
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
                                DropdownMenuItem(value: 'active', child: Text('启用')),
                                DropdownMenuItem(value: 'disabled', child: Text('禁用')),
                              ],
                              onChanged: (value) => _status = value ?? '',
                              decoration: const InputDecoration(labelText: '状态'),
                            ),
                          ),
                          const SizedBox(width: 12),
                          Expanded(
                            child: TextField(
                              controller: _keywordController,
                              decoration: const InputDecoration(labelText: 'Key Hash'),
                            ),
                          ),
                        ],
                      ),
                      const SizedBox(height: 12),
                      Row(
                        children: [
                          Expanded(
                            child: FilledButton(
                              onPressed: () {
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
                const Center(child: Text('暂无 Key'))
              else
                ...items.map(
                  (item) => _ApiKeyTile(
                    item: item,
                    permissionGroupName: _permissionGroupName(item.permissionGroupId),
                    onCopy: () => _copy(item.keyHash),
                    onToggle: () => _toggle(item),
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

  Future<void> _createKey() async {
    final nameCtl = TextEditingController();
    int? permissionGroupId;
    final ok = await showDialog<bool>(
      context: context,
      builder: (context) => AlertDialog(
        title: const Text('创建 API Key'),
        content: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            TextField(controller: nameCtl, decoration: const InputDecoration(labelText: '名称')),
            DropdownButtonFormField<int>(
              value: permissionGroupId,
              items: _permissionGroups
                  .map((g) => DropdownMenuItem<int>(
                        value: g.id,
                        child: Text(g.name.isEmpty ? g.id.toString() : g.name),
                      ))
                  .toList(),
              onChanged: (v) => permissionGroupId = v,
              decoration: const InputDecoration(labelText: '权限组'),
            ),
          ],
        ),
        actions: [
          TextButton(onPressed: () => Navigator.pop(context, false), child: const Text('取消')),
          FilledButton(onPressed: () => Navigator.pop(context, true), child: const Text('创建')),
        ],
      ),
    );
    if (ok != true) return;
    if (_permissionGroups.isNotEmpty && permissionGroupId == null) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          const SnackBar(content: Text('请先选择权限组')),
        );
      }
      return;
    }
    final client = context.read<AppState>().apiClient;
    if (client == null) return;
    final resp = await client.postJson('/admin/api/v1/api-keys', body: {
      'name': nameCtl.text.trim(),
      'permission_group_id': permissionGroupId,
      'scopes': [],
    });
    final apiKey = resp['api_key'] as String?;
    if (context.mounted && apiKey != null) {
      await showDialog<void>(
        context: context,
        builder: (context) => AlertDialog(
          title: const Text('API Key'),
          content: SelectableText(apiKey),
          actions: [
            TextButton(
              onPressed: () async {
                await Clipboard.setData(ClipboardData(text: apiKey));
                if (context.mounted) {
                  ScaffoldMessenger.of(context).showSnackBar(
                    const SnackBar(content: Text('已复制')),
                  );
                }
              },
              child: const Text('复制'),
            ),
            FilledButton(onPressed: () => Navigator.pop(context), child: const Text('确定')),
          ],
        ),
      );
    }
    _refresh();
  }

  Future<void> _toggle(ApiKeyItem item) async {
    final next = item.status == 'active' ? 'disabled' : 'active';
    final ok = await showDialog<bool>(
      context: context,
      builder: (context) => AlertDialog(
        title: const Text('切换状态'),
        content: Text('确认将 ${item.name} 设为 $next ?'),
        actions: [
          TextButton(onPressed: () => Navigator.pop(context, false), child: const Text('取消')),
          FilledButton(onPressed: () => Navigator.pop(context, true), child: const Text('确认')),
        ],
      ),
    );
    if (ok != true) return;
    final client = context.read<AppState>().apiClient;
    if (client == null) return;
    await client.patchJson(
      '/admin/api/v1/api-keys/${item.id}',
      body: {'status': next},
    );
    _refresh();
  }

  Future<void> _copy(String text) async {
    await Clipboard.setData(ClipboardData(text: text));
    if (context.mounted) {
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(content: Text('已复制')),
      );
    }
  }
}

class _ApiKeyTile extends StatelessWidget {
  final ApiKeyItem item;
  final String permissionGroupName;
  final VoidCallback onCopy;
  final VoidCallback onToggle;

  const _ApiKeyTile({
    required this.item,
    required this.permissionGroupName,
    required this.onCopy,
    required this.onToggle,
  });

  @override
  Widget build(BuildContext context) {
    final statusColor = item.status == 'active' ? const Color(0xFF00A68C) : const Color(0xFF546E7A);
    return Card(
      child: ListTile(
        title: Text(item.name.isEmpty ? 'API Key #${item.id}' : item.name),
        subtitle: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text('权限组：$permissionGroupName'),
            const SizedBox(height: 4),
            Text('Key Hash：${item.keyHash}'),
          ],
        ),
        trailing: Column(
          mainAxisAlignment: MainAxisAlignment.center,
          children: [
            Container(
              padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
              decoration: BoxDecoration(
                color: statusColor.withOpacity(0.12),
                borderRadius: BorderRadius.circular(999),
              ),
              child: Text(
                item.status,
                style: TextStyle(color: statusColor, fontSize: 12, fontWeight: FontWeight.w600),
              ),
            ),
            const SizedBox(height: 6),
            Row(
              mainAxisSize: MainAxisSize.min,
              children: [
                IconButton(onPressed: onCopy, icon: const Icon(Icons.copy, size: 18)),
                IconButton(onPressed: onToggle, icon: const Icon(Icons.sync_alt, size: 18)),
              ],
            ),
          ],
        ),
      ),
    );
  }
}

class ApiKeyItem {
  final int id;
  final String name;
  final String keyHash;
  final String status;
  final int? permissionGroupId;
  final String createdAt;

  ApiKeyItem({
    required this.id,
    required this.name,
    required this.keyHash,
    required this.status,
    required this.permissionGroupId,
    required this.createdAt,
  });

  factory ApiKeyItem.fromJson(Map<String, dynamic> json) {
    return ApiKeyItem(
      id: json['id'] as int? ?? 0,
      name: json['name'] as String? ?? '',
      keyHash: json['key_hash'] as String? ?? '',
      status: json['status'] as String? ?? '',
      permissionGroupId: json['permission_group_id'] as int?,
      createdAt: json['created_at']?.toString() ?? '',
    );
  }
}

class PermissionGroupItem {
  final int id;
  final String name;

  PermissionGroupItem({required this.id, required this.name});

  factory PermissionGroupItem.fromJson(Map<String, dynamic> json) {
    return PermissionGroupItem(
      id: json['id'] as int? ?? 0,
      name: json['name'] as String? ?? '',
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
