import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../app_state.dart';

enum FieldType { text, number, boolValue }

class FieldDef {
  final String keyName;
  final String label;
  final FieldType type;
  final bool numberIsInt;
  final String? hint;

  const FieldDef({
    required this.keyName,
    required this.label,
    required this.type,
    this.numberIsInt = false,
    this.hint,
  });
}

class SimpleCrudScreen extends StatefulWidget {
  final String title;
  final String listPath;
  final String createPath;
  final String Function(Map<String, dynamic> item) updatePath;
  final String Function(Map<String, dynamic> item) deletePath;
  final List<FieldDef> fields;
  final String Function(Map<String, dynamic>) titleBuilder;
  final String Function(Map<String, dynamic>) subtitleBuilder;
  final Future<Map<String, dynamic>> Function(dynamic client)? lookupLoader;
  final Map<String, dynamic> Function(
    Map<String, dynamic> item,
    Map<String, dynamic> lookups,
  )?
  enrichItem;

  const SimpleCrudScreen({
    super.key,
    required this.title,
    required this.listPath,
    required this.createPath,
    required this.updatePath,
    required this.deletePath,
    required this.fields,
    required this.titleBuilder,
    required this.subtitleBuilder,
    this.lookupLoader,
    this.enrichItem,
  });

  @override
  State<SimpleCrudScreen> createState() => _SimpleCrudScreenState();
}

class _SimpleCrudScreenState extends State<SimpleCrudScreen> {
  Future<List<Map<String, dynamic>>>? _future;
  Map<String, dynamic> _lookups = {};

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    final client = context.read<AppState>().apiClient;
    if (client != null) {
      _future = _load(client);
    }
  }

  Future<List<Map<String, dynamic>>> _load(client) async {
    if (widget.lookupLoader != null) {
      try {
        _lookups = await widget.lookupLoader!(client);
      } catch (_) {
        _lookups = {};
      }
    }
    final resp = await client.getJson(widget.listPath);
    final items = (resp['items'] as List<dynamic>? ?? [])
        .map((e) => Map<String, dynamic>.from(e as Map))
        .toList();
    if (widget.enrichItem == null) {
      return items;
    }
    return items.map((item) => widget.enrichItem!(item, _lookups)).toList();
  }

  @override
  Widget build(BuildContext context) {
    return FutureBuilder<List<Map<String, dynamic>>>(
      future: _future,
      builder: (context, snapshot) {
        if (snapshot.connectionState == ConnectionState.waiting) {
          return const Scaffold(
            body: Center(child: CircularProgressIndicator()),
          );
        }
        if (snapshot.hasError) {
          return Scaffold(
            appBar: AppBar(title: Text(widget.title)),
            body: Center(child: Text('加载失败：$snapshot')),
          );
        }
        final items = snapshot.data ?? [];
        return Scaffold(
          appBar: AppBar(
            title: Text(widget.title),
            actions: [
              IconButton(
                icon: const Icon(Icons.refresh),
                onPressed: () {
                  final client = context.read<AppState>().apiClient;
                  if (client != null) {
                    setState(() => _future = _load(client));
                  }
                },
              ),
              IconButton(
                icon: const Icon(Icons.add),
                onPressed: () => _openEditor(),
              ),
            ],
          ),
          body: RefreshIndicator(
            onRefresh: () async {
              final client = context.read<AppState>().apiClient;
              if (client != null) {
                setState(() => _future = _load(client));
              }
              await _future;
            },
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
                    child: Row(
                      children: [
                        const Icon(Icons.tune, color: Colors.white),
                        const SizedBox(width: 8),
                        Expanded(
                          child: Text(
                            '${widget.title} 管理：支持下拉刷新与点击编辑',
                            style: const TextStyle(
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
                    child: Center(child: Text('暂无数据')),
                  );
                }
                final item = items[index - 1];
                return Container(
                  margin: const EdgeInsets.only(bottom: 10),
                  padding: const EdgeInsets.all(12),
                  decoration: BoxDecoration(
                    color: Colors.white,
                    borderRadius: BorderRadius.circular(14),
                    border: Border.all(color: const Color(0xFFE2E8F0)),
                  ),
                  child: InkWell(
                    onTap: () => _openEditor(item: item),
                    borderRadius: BorderRadius.circular(10),
                    child: Row(
                      children: [
                        Container(
                          padding: const EdgeInsets.all(8),
                          decoration: BoxDecoration(
                            color: const Color(0xFF1E88E5).withOpacity(0.12),
                            borderRadius: BorderRadius.circular(10),
                          ),
                          child: const Icon(
                            Icons.view_list_outlined,
                            color: Color(0xFF1E88E5),
                          ),
                        ),
                        const SizedBox(width: 10),
                        Expanded(
                          child: Column(
                            crossAxisAlignment: CrossAxisAlignment.start,
                            children: [
                              Text(
                                widget.titleBuilder(item),
                                style: const TextStyle(
                                  fontWeight: FontWeight.w700,
                                ),
                              ),
                              const SizedBox(height: 4),
                              Text(
                                widget.subtitleBuilder(item),
                                maxLines: 2,
                                overflow: TextOverflow.ellipsis,
                                style: const TextStyle(fontSize: 12),
                              ),
                            ],
                          ),
                        ),
                        const Icon(Icons.edit_outlined, size: 18),
                      ],
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

  Future<void> _openEditor({Map<String, dynamic>? item}) async {
    final controllers = <String, TextEditingController>{};
    final boolValues = <String, bool>{};
    for (final field in widget.fields) {
      final value = item?[field.keyName];
      if (field.type == FieldType.boolValue) {
        boolValues[field.keyName] = value == true;
      } else {
        controllers[field.keyName] = TextEditingController(
          text: value?.toString() ?? '',
        );
      }
    }
    final isEdit = item != null;
    final ok = await showDialog<bool>(
      context: context,
      builder: (context) => StatefulBuilder(
        builder: (context, setModal) => AlertDialog(
          title: Text(isEdit ? '编辑' : '新增'),
          content: SingleChildScrollView(
            child: Column(
              mainAxisSize: MainAxisSize.min,
              children: widget.fields.map((field) {
                if (field.type == FieldType.boolValue) {
                  return SwitchListTile(
                    contentPadding: EdgeInsets.zero,
                    value: boolValues[field.keyName] ?? false,
                    onChanged: (v) =>
                        setModal(() => boolValues[field.keyName] = v),
                    title: Text(field.label),
                  );
                }
                return Padding(
                  padding: const EdgeInsets.only(bottom: 12),
                  child: TextField(
                    controller: controllers[field.keyName],
                    keyboardType: field.type == FieldType.number
                        ? TextInputType.number
                        : TextInputType.text,
                    decoration: InputDecoration(
                      labelText: field.label,
                      hintText: field.hint,
                    ),
                  ),
                );
              }).toList(),
            ),
          ),
          actions: [
            if (isEdit)
              TextButton(
                onPressed: () => _delete(item!),
                child: const Text('删除'),
              ),
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
      ),
    );
    if (ok != true) return;
    final payload = <String, dynamic>{};
    for (final field in widget.fields) {
      if (field.type == FieldType.boolValue) {
        payload[field.keyName] = boolValues[field.keyName] ?? false;
        continue;
      }
      final text = controllers[field.keyName]?.text.trim() ?? '';
      if (field.type == FieldType.number) {
        payload[field.keyName] = field.numberIsInt
            ? int.tryParse(text) ?? 0
            : double.tryParse(text) ?? 0;
      } else {
        payload[field.keyName] = text;
      }
    }
    final client = context.read<AppState>().apiClient;
    if (client == null) return;
    if (isEdit) {
      await client.patchJson(widget.updatePath(item!), body: payload);
    } else {
      await client.postJson(widget.createPath, body: payload);
    }
    setState(() {
      _future = _load(client);
    });
  }

  Future<void> _delete(Map<String, dynamic> item) async {
    final client = context.read<AppState>().apiClient;
    if (client == null) return;
    await client.deleteJson(widget.deletePath(item));
    if (mounted) {
      setState(() {
        _future = _load(client);
      });
    }
    if (context.mounted) {
      Navigator.pop(context, false);
    }
  }
}
