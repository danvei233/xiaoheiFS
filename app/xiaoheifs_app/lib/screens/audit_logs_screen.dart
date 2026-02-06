import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../app_state.dart';
import '../services/api_client.dart';

class AuditLogsScreen extends StatefulWidget {
  const AuditLogsScreen({super.key});

  @override
  State<AuditLogsScreen> createState() => _AuditLogsScreenState();
}

class _AuditLogsScreenState extends State<AuditLogsScreen> {
  Future<List<AuditLogItem>>? _future;
  ApiClient? _client;

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    final client = context.read<AppState>().apiClient;
    if (client?.baseUrl != _client?.baseUrl ||
        client?.apiKey != _client?.apiKey ||
        client?.token != _client?.token) {
      _client = client;
      if (client != null) {
        _future = _load(client);
      }
    }
  }

  Future<List<AuditLogItem>> _load(ApiClient client) async {
    final resp = await client.getJson('/admin/api/v1/audit-logs');
    final items = (resp['items'] as List<dynamic>? ?? [])
        .map((e) => AuditLogItem.fromJson(e as Map<String, dynamic>))
        .toList();
    return items;
  }

  @override
  Widget build(BuildContext context) {
    return FutureBuilder<List<AuditLogItem>>(
      future: _future,
      builder: (context, snapshot) {
        if (snapshot.connectionState == ConnectionState.waiting) {
          return const Scaffold(
            body: Center(child: CircularProgressIndicator()),
          );
        }
        if (snapshot.hasError) {
          return Scaffold(
            appBar: AppBar(title: const Text('操作日志')),
            body: _ErrorState(
              message: '加载日志失败，请检查 API Key 权限。',
              onRetry: () {
                final client = context.read<AppState>().apiClient;
                if (client != null) {
                  setState(() {
                    _future = _load(client);
                  });
                }
              },
            ),
          );
        }
        final items = snapshot.data ?? [];
        return Scaffold(
          appBar: AppBar(title: const Text('操作日志')),
          body: RefreshIndicator(
            onRefresh: () async {
              final client = context.read<AppState>().apiClient;
              if (client != null) {
                setState(() {
                  _future = _load(client);
                });
              }
              await _future;
            },
            child: ListView.builder(
              padding: const EdgeInsets.all(16),
              itemCount: items.isEmpty ? 1 : items.length,
              itemBuilder: (context, index) {
                if (items.isEmpty) {
                  return const _EmptyState();
                }
                final item = items[index];
                return Card(
                  child: ListTile(
                    leading: const Icon(Icons.history),
                    title: Text(item.action),
                    subtitle: Text('${item.targetType} · ${item.targetId}'),
                    trailing: Text(item.createdAt),
                  ),
                );
              },
            ),
          ),
        );
      },
    );
  }
}

class AuditLogItem {
  final int id;
  final String action;
  final String targetType;
  final String targetId;
  final String createdAt;

  AuditLogItem({
    required this.id,
    required this.action,
    required this.targetType,
    required this.targetId,
    required this.createdAt,
  });

  factory AuditLogItem.fromJson(Map<String, dynamic> json) {
    return AuditLogItem(
      id: json['id'] as int? ?? 0,
      action: json['action'] as String? ?? '未知操作',
      targetType: json['target_type'] as String? ?? '',
      targetId: json['target_id'] as String? ?? '',
      createdAt: json['created_at'] as String? ?? '',
    );
  }
}

class _ErrorState extends StatelessWidget {
  final String message;
  final VoidCallback onRetry;

  const _ErrorState({required this.message, required this.onRetry});

  @override
  Widget build(BuildContext context) {
    return Center(
      child: Padding(
        padding: const EdgeInsets.all(24),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Text(message, textAlign: TextAlign.center),
            const SizedBox(height: 16),
            FilledButton(onPressed: onRetry, child: const Text('重试')),
          ],
        ),
      ),
    );
  }
}

class _EmptyState extends StatelessWidget {
  const _EmptyState();

  @override
  Widget build(BuildContext context) {
    return const Padding(
      padding: EdgeInsets.all(24),
      child: Center(child: Text('暂无日志')),
    );
  }
}
