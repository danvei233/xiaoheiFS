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
  bool _busy = false;

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
          return const Scaffold(
            body: Center(child: CircularProgressIndicator()),
          );
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
          body: Stack(
            children: [
              RefreshIndicator(
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
                            Icon(
                              Icons.account_balance_wallet_outlined,
                              color: Colors.white,
                            ),
                            SizedBox(width: 8),
                            Expanded(
                              child: Text(
                                '在这里统一管理支付渠道状态与配置 JSON',
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
                        padding: EdgeInsets.only(top: 40),
                        child: Center(child: Text('暂无渠道')),
                      );
                    }
                    final item = items[index - 1];
                    final statusColor = item.enabled
                        ? const Color(0xFF00A68C)
                        : const Color(0xFF546E7A);
                    return Container(
                      margin: const EdgeInsets.only(bottom: 12),
                      padding: const EdgeInsets.all(12),
                      decoration: BoxDecoration(
                        color: Colors.white,
                        borderRadius: BorderRadius.circular(14),
                        border: Border.all(color: const Color(0xFFE2E8F0)),
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
                                child: Icon(
                                  Icons.payments_outlined,
                                  color: statusColor,
                                ),
                              ),
                              const SizedBox(width: 10),
                              Expanded(
                                child: Column(
                                  crossAxisAlignment: CrossAxisAlignment.start,
                                  children: [
                                    Text(
                                      item.name,
                                      style: const TextStyle(
                                        fontWeight: FontWeight.w700,
                                      ),
                                    ),
                                    const SizedBox(height: 2),
                                    Text(
                                      item.key,
                                      style: TextStyle(
                                        fontSize: 12,
                                        color: Colors.grey.shade700,
                                      ),
                                    ),
                                  ],
                                ),
                              ),
                              Container(
                                padding: const EdgeInsets.symmetric(
                                  horizontal: 8,
                                  vertical: 4,
                                ),
                                decoration: BoxDecoration(
                                  color: statusColor.withOpacity(0.12),
                                  borderRadius: BorderRadius.circular(999),
                                ),
                                child: Text(
                                  item.enabled ? '已启用' : '已停用',
                                  style: TextStyle(
                                    color: statusColor,
                                    fontSize: 12,
                                    fontWeight: FontWeight.w600,
                                  ),
                                ),
                              ),
                            ],
                          ),
                          const SizedBox(height: 10),
                          Wrap(
                            spacing: 8,
                            runSpacing: 8,
                            children: [
                              OutlinedButton.icon(
                                onPressed: () => _editConfig(item),
                                icon: const Icon(Icons.settings, size: 16),
                                label: const Text('配置'),
                              ),
                              Switch(
                                value: item.enabled,
                                onChanged: _busy
                                    ? null
                                    : (v) => _toggle(item, v),
                              ),
                            ],
                          ),
                        ],
                      ),
                    );
                  },
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

  Future<void> _toggle(PaymentProviderItem item, bool enabled) async {
    final client = context.read<AppState>().apiClient;
    if (client == null) return;
    setState(() => _busy = true);
    try {
      await client.patchJson(
        '/admin/api/v1/payments/providers/${item.key}',
        body: {'enabled': enabled, 'config_json': item.configJson ?? ''},
      );
      _refresh();
    } finally {
      if (mounted) setState(() => _busy = false);
    }
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
