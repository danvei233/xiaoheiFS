import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../app_state.dart';

class PaymentProvidersScreen extends StatefulWidget {
  const PaymentProvidersScreen({super.key});

  @override
  State<PaymentProvidersScreen> createState() => _PaymentProvidersScreenState();
}

class _PaymentProvidersScreenState extends State<PaymentProvidersScreen> {
  Future<List<PaymentProviderItem>>? _future;

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    final client = context.read<AppState>().apiClient;
    if (client != null) {
      _future = _load(client);
    }
  }

  Future<List<PaymentProviderItem>> _load(client) async {
    final resp = await client.getJson('/admin/api/v1/payments/providers');
    return (resp['items'] as List<dynamic>? ?? [])
        .map((e) => PaymentProviderItem.fromJson(e as Map<String, dynamic>))
        .toList();
  }

  void _refresh() {
    final client = context.read<AppState>().apiClient;
    if (client != null) setState(() => _future = _load(client));
  }

  @override
  Widget build(BuildContext context) {
    return FutureBuilder<List<PaymentProviderItem>>(
      future: _future,
      builder: (context, snapshot) {
        if (snapshot.connectionState == ConnectionState.waiting) {
          return const Scaffold(body: Center(child: CircularProgressIndicator()));
        }
        if (snapshot.hasError) {
          return Scaffold(
            appBar: AppBar(title: const Text('支付渠道')),
            body: Center(child: Text('加载失败：$snapshot')),
          );
        }
        final items = snapshot.data ?? [];
        return Scaffold(
          appBar: AppBar(
            title: const Text('支付渠道'),
            actions: [
              IconButton(onPressed: _refresh, icon: const Icon(Icons.refresh)),
            ],
          ),
          body: ListView.builder(
            padding: const EdgeInsets.all(16),
            itemCount: items.isEmpty ? 1 : items.length,
            itemBuilder: (context, index) {
              if (items.isEmpty) return const Center(child: Text('暂无渠道'));
              final item = items[index];
              return Card(
                child: ListTile(
                  title: Text(item.name),
                  subtitle: Text(item.key),
                  trailing: Row(
                    mainAxisSize: MainAxisSize.min,
                    children: [
                      IconButton(
                        icon: const Icon(Icons.settings),
                        onPressed: () => _editConfig(item),
                      ),
                      Switch(
                        value: item.enabled,
                        onChanged: (v) => _toggle(item, v),
                      ),
                    ],
                  ),
                ),
              );
            },
          ),
        );
      },
    );
  }

  Future<void> _toggle(PaymentProviderItem item, bool enabled) async {
    final client = context.read<AppState>().apiClient;
    if (client == null) return;
    await client.patchJson(
      '/admin/api/v1/payments/providers/${item.key}',
      body: {'enabled': enabled, 'config_json': item.configJson ?? ''},
    );
    _refresh();
  }

  Future<void> _editConfig(PaymentProviderItem item) async {
    final controller = TextEditingController(text: item.configJson ?? '');
    bool enabled = item.enabled;
    final ok = await showDialog<bool>(
      context: context,
      builder: (context) => AlertDialog(
        title: Text('配置 ${item.name}'),
        content: StatefulBuilder(
          builder: (context, setModal) => Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              SwitchListTile(
                value: enabled,
                onChanged: (v) => setModal(() => enabled = v),
                title: const Text('启用'),
              ),
              TextField(
                controller: controller,
                maxLines: 8,
                decoration: const InputDecoration(labelText: '配置 JSON'),
              ),
            ],
          ),
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
    await client.patchJson(
      '/admin/api/v1/payments/providers/${item.key}',
      body: {'enabled': enabled, 'config_json': controller.text},
    );
    _refresh();
  }
}

class PaymentProviderItem {
  final String key;
  final String name;
  final bool enabled;
  final String? configJson;

  PaymentProviderItem({
    required this.key,
    required this.name,
    required this.enabled,
    required this.configJson,
  });

  factory PaymentProviderItem.fromJson(Map<String, dynamic> json) {
    return PaymentProviderItem(
      key: json['key'] as String? ?? '',
      name: json['name'] as String? ?? '',
      enabled: json['enabled'] as bool? ?? false,
      configJson: json['config_json'] as String?,
    );
  }
}
