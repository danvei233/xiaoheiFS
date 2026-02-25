import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:provider/provider.dart';

import '../app_state.dart';

class PluginsSettingsScreen extends StatefulWidget {
  const PluginsSettingsScreen({super.key});

  @override
  State<PluginsSettingsScreen> createState() => _PluginsSettingsScreenState();
}

class _PluginsSettingsScreenState extends State<PluginsSettingsScreen> {
  bool _loading = false;
  String _error = '';
  List<Map<String, dynamic>> _items = [];
  List<Map<String, dynamic>> _discovered = [];
  bool _discoverLoading = false;
  String _keyword = '';
  String _category = 'all';
  String _busyKey = '';

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    _load();
  }

  Future<void> _load() async {
    final client = context.read<AppState>().apiClient;
    if (client == null) return;
    setState(() {
      _loading = true;
      _error = '';
    });
    try {
      final resp = await client.getJson('/admin/api/v1/plugins');
      final rows = (resp['items'] as List<dynamic>? ?? [])
          .map((e) => _asMap(e))
          .where((e) => e.isNotEmpty)
          .toList();
      if (!mounted) return;
      setState(() => _items = rows);
    } catch (e) {
      if (!mounted) return;
      setState(() => _error = e.toString());
    } finally {
      if (mounted) setState(() => _loading = false);
    }
  }

  Future<void> _toggle(Map<String, dynamic> row, bool enable) async {
    final client = context.read<AppState>().apiClient;
    if (client == null) return;
    final cat = _asStr(row['category']);
    final pid = _asStr(row['plugin_id']);
    final iid = _asStr(row['instance_id']).isEmpty ? 'default' : _asStr(row['instance_id']);
    if (cat.isEmpty || pid.isEmpty) return;
    final key = '$cat/$pid/$iid';
    setState(() => _busyKey = key);
    try {
      await client.postJson('/admin/api/v1/plugins/$cat/$pid/$iid/${enable ? 'enable' : 'disable'}');
      row['enabled'] = enable;
      if (mounted) setState(() {});
    } finally {
      if (mounted) setState(() => _busyKey = '');
    }
  }

  Future<void> _deleteInstance(Map<String, dynamic> row) async {
    final client = context.read<AppState>().apiClient;
    if (client == null) return;
    final cat = _asStr(row['category']);
    final pid = _asStr(row['plugin_id']);
    final iid = _asStr(row['instance_id']).isEmpty ? 'default' : _asStr(row['instance_id']);
    if (cat.isEmpty || pid.isEmpty) return;
    await client.deleteJson('/admin/api/v1/plugins/$cat/$pid/$iid');
    await _load();
  }

  Future<void> _addInstance(Map<String, dynamic> row) async {
    final client = context.read<AppState>().apiClient;
    if (client == null) return;
    final cat = _asStr(row['category']);
    final pid = _asStr(row['plugin_id']);
    if (cat.isEmpty || pid.isEmpty) return;
    final ctl = TextEditingController();
    final ok = await showDialog<bool>(
      context: context,
      builder: (context) => AlertDialog(
        title: const Text('新增实例'),
        content: TextField(
          controller: ctl,
          decoration: const InputDecoration(labelText: 'instance_id（可留空自动生成）'),
        ),
        actions: [
          TextButton(onPressed: () => Navigator.pop(context, false), child: const Text('取消')),
          FilledButton(onPressed: () => Navigator.pop(context, true), child: const Text('创建')),
        ],
      ),
    );
    if (ok != true) return;
    await client.postJson('/admin/api/v1/plugins/$cat/$pid/instances', body: {
      'instance_id': ctl.text.trim(),
    });
    await _load();
  }

  Future<void> _openConfig(Map<String, dynamic> row) async {
    final client = context.read<AppState>().apiClient;
    if (client == null) return;
    final cat = _asStr(row['category']);
    final pid = _asStr(row['plugin_id']);
    final iid = _asStr(row['instance_id']).isEmpty ? 'default' : _asStr(row['instance_id']);
    if (cat.isEmpty || pid.isEmpty) return;

    Map<String, dynamic> schema = {};
    Map<String, dynamic> model = {};
    String rawJson = '{}';
    String error = '';
    try {
      final res = await Future.wait([
        client.getJson('/admin/api/v1/plugins/$cat/$pid/$iid/config/schema'),
        client.getJson('/admin/api/v1/plugins/$cat/$pid/$iid/config'),
      ]);
      final schemaText = _asStr(res[0]['json_schema']);
      final cfgText = _asStr(res[1]['config_json']);
      final parsedSchema = _tryDecodeObject(schemaText);
      final parsedCfg = _tryDecodeObject(cfgText);
      schema = parsedSchema ?? {};
      model = parsedCfg ?? {};
      rawJson = const JsonEncoder.withIndent('  ').convert(model);
    } catch (e) {
      error = e.toString();
    }

    final props = _asMap(schema['properties']);
    final required = _toList(schema['required'], const <String>[]);

    if (!mounted) return;
    await showDialog<void>(
      context: context,
      builder: (context) => StatefulBuilder(
        builder: (context, setModal) => AlertDialog(
          title: Text('配置 ${_asStr(row['name']).isEmpty ? pid : _asStr(row['name'])}'),
          content: SizedBox(
            width: 520,
            child: SingleChildScrollView(
              child: Column(
                mainAxisSize: MainAxisSize.min,
                children: [
                  if (error.isNotEmpty)
                    Padding(
                      padding: const EdgeInsets.only(bottom: 8),
                      child: Text('读取配置失败：$error'),
                    ),
                  ...props.entries.map((entry) {
                    final key = entry.key;
                    final def = _asMap(entry.value);
                    final type = _asStr(def['type']).toLowerCase();
                    final title = _asStr(def['title']).isEmpty ? key : _asStr(def['title']);
                    final enums = (def['enum'] as List<dynamic>? ?? []).map((e) => e.toString()).toList();
                    final must = required.contains(key);
                    if (type == 'boolean') {
                      final current = model[key] == true;
                      return SwitchListTile(
                        value: current,
                        onChanged: (v) => setModal(() => model[key] = v),
                        title: Text(must ? '$title *' : title),
                      );
                    }
                    if (enums.isNotEmpty) {
                      final current = _asStr(model[key]);
                      return Padding(
                        padding: const EdgeInsets.only(bottom: 8),
                        child: DropdownButtonFormField<String>(
                          value: current.isEmpty ? null : current,
                          decoration: InputDecoration(labelText: must ? '$title *' : title),
                          items: enums.map((v) => DropdownMenuItem(value: v, child: Text(v))).toList(),
                          onChanged: (v) => setModal(() => model[key] = v ?? ''),
                        ),
                      );
                    }
                    final ctl = TextEditingController(text: _asStr(model[key]));
                    return Padding(
                      padding: const EdgeInsets.only(bottom: 8),
                      child: TextField(
                        controller: ctl,
                        keyboardType: (type == 'integer' || type == 'number')
                            ? TextInputType.number
                            : TextInputType.text,
                        onChanged: (v) {
                          if (type == 'integer') {
                            model[key] = int.tryParse(v) ?? 0;
                            return;
                          }
                          if (type == 'number') {
                            model[key] = double.tryParse(v) ?? 0;
                            return;
                          }
                          model[key] = v;
                        },
                        decoration: InputDecoration(labelText: must ? '$title *' : title),
                      ),
                    );
                  }),
                  const SizedBox(height: 8),
                  TextField(
                    controller: TextEditingController(text: rawJson),
                    maxLines: 8,
                    onChanged: (v) => rawJson = v,
                    decoration: const InputDecoration(labelText: '原始 JSON（高级）'),
                  ),
                ],
              ),
            ),
          ),
          actions: [
            TextButton(onPressed: () => Navigator.pop(context), child: const Text('取消')),
            FilledButton(
              onPressed: () async {
                Map<String, dynamic> payload = model;
                final parsedRaw = _tryDecodeObject(rawJson);
                if (parsedRaw != null) payload = parsedRaw;
                await client.putJson('/admin/api/v1/plugins/$cat/$pid/$iid/config', body: {
                  'config_json': jsonEncode(payload),
                });
                if (context.mounted) Navigator.pop(context);
              },
              child: const Text('保存'),
            ),
          ],
        ),
      ),
    );
  }

  Future<void> _openDiscover() async {
    final client = context.read<AppState>().apiClient;
    if (client == null) return;
    setState(() => _discoverLoading = true);
    try {
      final res = await client.getJson('/admin/api/v1/plugins/discover');
      final rows = (res['items'] as List<dynamic>? ?? [])
          .map(_asMap)
          .where((e) => e.isNotEmpty)
          .toList();
      if (!mounted) return;
      setState(() => _discovered = rows);
    } finally {
      if (mounted) setState(() => _discoverLoading = false);
    }
    if (!mounted) return;
    await showModalBottomSheet<void>(
      context: context,
      isScrollControlled: true,
      builder: (context) => SafeArea(
        child: SizedBox(
          height: MediaQuery.of(context).size.height * 0.78,
          child: Column(
            children: [
              const ListTile(
                title: Text('从目录发现'),
                subtitle: Text('发现 ./plugins 下未导入数据库的插件'),
              ),
              Expanded(
                child: _discoverLoading
                    ? const Center(child: CircularProgressIndicator())
                    : _discovered.isEmpty
                        ? const Center(child: Text('未发现可导入插件'))
                        : ListView.builder(
                            itemCount: _discovered.length,
                            itemBuilder: (context, index) {
                              final row = _discovered[index];
                              final cat = _asStr(row['category']);
                              final pid = _asStr(row['plugin_id']);
                              return ListTile(
                                title: Text(_asStr(row['name']).isEmpty ? pid : _asStr(row['name'])),
                                subtitle: Text('$cat/$pid  签名:${_asStr(row['signature_status']).isEmpty ? '-' : _asStr(row['signature_status'])}'),
                                trailing: FilledButton(
                                  onPressed: () async {
                                    final client = context.read<AppState>().apiClient;
                                    if (client == null) return;
                                    await client.postJson('/admin/api/v1/plugins/$cat/$pid/import');
                                    if (!context.mounted) return;
                                    ScaffoldMessenger.of(context).showSnackBar(
                                      const SnackBar(content: Text('导入成功')),
                                    );
                                    Navigator.pop(context);
                                    await _load();
                                  },
                                  child: const Text('导入'),
                                ),
                              );
                            },
                          ),
              ),
            ],
          ),
        ),
      ),
    );
  }

  List<Map<String, dynamic>> _filtered() {
    final kw = _keyword.trim().toLowerCase();
    return _items.where((it) {
      final cat = _asStr(it['category']);
      if (_category != 'all' && cat != _category) return false;
      if (kw.isEmpty) return true;
      final hay = '${_asStr(it['name'])} ${_asStr(it['plugin_id'])} ${_asStr(it['instance_id'])} $cat'.toLowerCase();
      return hay.contains(kw);
    }).toList();
  }

  @override
  Widget build(BuildContext context) {
    final rows = _filtered();
    return Scaffold(
      appBar: AppBar(
        title: const Text('插件设置'),
        actions: [
          IconButton(
            onPressed: _loading ? null : _openDiscover,
            icon: const Icon(Icons.travel_explore),
            tooltip: '从目录发现',
          ),
          IconButton(
            onPressed: _loading ? null : _load,
            icon: const Icon(Icons.refresh),
          ),
        ],
      ),
      body: RefreshIndicator(
        onRefresh: _load,
        child: ListView(
          physics: const AlwaysScrollableScrollPhysics(),
          padding: const EdgeInsets.fromLTRB(12, 12, 12, 20),
          children: [
            Row(
              children: [
                Expanded(
                  child: TextField(
                    onChanged: (v) => setState(() => _keyword = v),
                    decoration: const InputDecoration(
                      isDense: true,
                      hintText: '搜索插件名 / plugin_id',
                      prefixIcon: Icon(Icons.search, size: 18),
                    ),
                  ),
                ),
                const SizedBox(width: 8),
                DropdownButton<String>(
                  value: _category,
                  items: const [
                    DropdownMenuItem(value: 'all', child: Text('全部')),
                    DropdownMenuItem(value: 'payment', child: Text('payment')),
                    DropdownMenuItem(value: 'sms', child: Text('sms')),
                    DropdownMenuItem(value: 'kyc', child: Text('kyc')),
                    DropdownMenuItem(value: 'automation', child: Text('automation')),
                  ],
                  onChanged: (v) => setState(() => _category = v ?? 'all'),
                ),
              ],
            ),
            const SizedBox(height: 10),
            if (_error.isNotEmpty)
              Padding(
                padding: const EdgeInsets.only(bottom: 8),
                child: Text('加载失败：$_error'),
              ),
            if (_loading && _items.isEmpty)
              const Padding(
                padding: EdgeInsets.only(top: 80),
                child: Center(child: CircularProgressIndicator()),
              )
            else if (_items.isEmpty)
              const Padding(
                padding: EdgeInsets.only(top: 80),
                child: Center(child: Text('暂无插件')),
              )
            else
              ...rows.map((row) {
                final enabled = row['enabled'] == true;
                final loaded = row['loaded'] == true;
                final cat = _asStr(row['category']);
                final pid = _asStr(row['plugin_id']);
                final iid = _asStr(row['instance_id']).isEmpty ? 'default' : _asStr(row['instance_id']);
                final key = '$cat/$pid/$iid';
                return Container(
                  margin: const EdgeInsets.only(bottom: 8),
                  decoration: BoxDecoration(
                    color: Colors.white,
                    borderRadius: BorderRadius.circular(12),
                    border: Border.all(color: const Color(0xFFE5EAF2)),
                  ),
                  child: Column(
                    children: [
                      ListTile(
                        title: Text(
                          _asStr(row['name']).isEmpty ? _asStr(row['plugin_id']) : _asStr(row['name']),
                          maxLines: 1,
                          overflow: TextOverflow.ellipsis,
                        ),
                        subtitle: Text(
                          '分类:$cat 版本:${_asStr(row['version'])}\n实例:$iid 加载:${loaded ? '是' : '否'}',
                        ),
                        isThreeLine: true,
                        trailing: Switch.adaptive(
                          value: enabled,
                          onChanged: _busyKey == key ? null : (v) => _toggle(row, v),
                        ),
                      ),
                      Padding(
                        padding: const EdgeInsets.fromLTRB(12, 0, 12, 10),
                        child: Row(
                          children: [
                            OutlinedButton(
                              onPressed: () => _openConfig(row),
                              child: const Text('配置'),
                            ),
                            const SizedBox(width: 8),
                            OutlinedButton(
                              onPressed: () => _addInstance(row),
                              child: const Text('新增实例'),
                            ),
                            const SizedBox(width: 8),
                            TextButton(
                              onPressed: () => _deleteInstance(row),
                              child: const Text('卸载', style: TextStyle(color: Color(0xFFD32F2F))),
                            ),
                          ],
                        ),
                      ),
                    ],
                  ),
                );
              }),
          ],
        ),
      ),
    );
  }
}

class DebugCenterScreen extends StatefulWidget {
  const DebugCenterScreen({super.key});

  @override
  State<DebugCenterScreen> createState() => _DebugCenterScreenState();
}

class _DebugCenterScreenState extends State<DebugCenterScreen> {
  bool _loading = false;
  bool _enabled = false;
  Map<String, dynamic> _logs = {};
  String _error = '';
  bool _savingRetention = false;
  final _automationRetentionCtl = TextEditingController(text: '30');
  final _auditRetentionCtl = TextEditingController(text: '30');
  final _syncRetentionCtl = TextEditingController(text: '30');
  final _taskRetentionCtl = TextEditingController(text: '30');
  final _probeEventRetentionCtl = TextEditingController(text: '30');
  final _probeSessionRetentionCtl = TextEditingController(text: '30');

  @override
  void dispose() {
    _automationRetentionCtl.dispose();
    _auditRetentionCtl.dispose();
    _syncRetentionCtl.dispose();
    _taskRetentionCtl.dispose();
    _probeEventRetentionCtl.dispose();
    _probeSessionRetentionCtl.dispose();
    super.dispose();
  }

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    _load();
  }

  Future<void> _load() async {
    final client = context.read<AppState>().apiClient;
    if (client == null) return;
    setState(() {
      _loading = true;
      _error = '';
    });
    try {
      final res = await Future.wait([
        client.getJson('/admin/api/v1/debug/status'),
        client.getJson('/admin/api/v1/debug/logs'),
        _loadSettingsMap(client),
      ]);
      final map = _asMap(res[2]);
      if (!mounted) return;
      setState(() {
        _enabled = res[0]['enabled'] == true;
        _logs = _asMap(res[1]);
        _automationRetentionCtl.text = '${_toInt(map['automation_log_retention_days'], 30)}';
        _auditRetentionCtl.text = '${_toInt(map['audit_log_retention_days'], 30)}';
        _syncRetentionCtl.text = '${_toInt(map['sync_log_retention_days'], 30)}';
        _taskRetentionCtl.text = '${_toInt(map['task_run_log_retention_days'], 30)}';
        _probeEventRetentionCtl.text = '${_toInt(map['probe_event_retention_days'], 30)}';
        _probeSessionRetentionCtl.text = '${_toInt(map['probe_session_retention_days'], 30)}';
      });
    } catch (e) {
      if (!mounted) return;
      setState(() => _error = e.toString());
    } finally {
      if (mounted) setState(() => _loading = false);
    }
  }

  Future<void> _toggle(bool v) async {
    final client = context.read<AppState>().apiClient;
    if (client == null) return;
    setState(() => _enabled = v);
    try {
      await client.patchJson('/admin/api/v1/debug/status', body: {'enabled': v});
    } catch (_) {
      if (!mounted) return;
      setState(() => _enabled = !v);
    }
  }

  Future<void> _saveRetention() async {
    final client = context.read<AppState>().apiClient;
    if (client == null) return;
    setState(() => _savingRetention = true);
    try {
      await _saveSettingsItems(client, [
        ('automation_log_retention_days', '${_toInt(_automationRetentionCtl.text, 30)}'),
        ('audit_log_retention_days', '${_toInt(_auditRetentionCtl.text, 30)}'),
        ('sync_log_retention_days', '${_toInt(_syncRetentionCtl.text, 30)}'),
        ('task_run_log_retention_days', '${_toInt(_taskRetentionCtl.text, 30)}'),
        ('probe_event_retention_days', '${_toInt(_probeEventRetentionCtl.text, 30)}'),
        ('probe_session_retention_days', '${_toInt(_probeSessionRetentionCtl.text, 30)}'),
      ]);
      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(const SnackBar(content: Text('日志保留策略已保存')));
    } finally {
      if (mounted) setState(() => _savingRetention = false);
    }
  }

  int _count(String group) {
    final g = _asMap(_logs[group]);
    final items = g['items'] as List<dynamic>?;
    return items?.length ?? 0;
  }

  List<Map<String, dynamic>> _itemsOf(String group) {
    final g = _asMap(_logs[group]);
    final items = (g['items'] as List<dynamic>? ?? []).map(_asMap).toList();
    return items;
  }

  Future<void> _openLogDetails(String title, String key, String apiType) async {
    final client = context.read<AppState>().apiClient;
    if (client == null) return;
    List<Map<String, dynamic>> items = _itemsOf(key);
    try {
      final res = await client.getJson('/admin/api/v1/debug/logs', query: {'types': apiType, 'limit': '100', 'offset': '0'});
      final target = _asMap(res[key]);
      items = (target['items'] as List<dynamic>? ?? []).map(_asMap).toList();
    } catch (_) {}
    if (!mounted) return;
    await showModalBottomSheet<void>(
      context: context,
      isScrollControlled: true,
      builder: (context) => SafeArea(
        child: SizedBox(
          height: MediaQuery.of(context).size.height * 0.8,
          child: Column(
            children: [
              ListTile(
                title: Text('$title 详情'),
                subtitle: Text('${items.length} 条'),
              ),
              const Divider(height: 1),
              Expanded(
                child: items.isEmpty
                    ? const Center(child: Text('暂无日志'))
                    : ListView.separated(
                        itemCount: items.length,
                        separatorBuilder: (_, __) => const Divider(height: 1),
                        itemBuilder: (context, index) {
                          final row = items[index];
                          final detail = _asMap(row['detail']);
                          final action = _asStr(row['action']);
                          final protocol = _asStr(detail['protocol']).isEmpty ? _asStr(row['protocol']) : _asStr(detail['protocol']);
                          final success = row['success'] == true || _toBool(row['success'], false);
                          final ts = _asStr(row['ts']).isEmpty ? _asStr(row['created_at']) : _asStr(row['ts']);
                          final msg = _asStr(row['message']).isEmpty ? jsonEncode(row) : _asStr(row['message']);
                          return ListTile(
                            dense: true,
                            title: Text(
                              apiType == 'automation'
                                  ? (action.isEmpty ? msg : action)
                                  : msg,
                              maxLines: 2,
                              overflow: TextOverflow.ellipsis,
                            ),
                            subtitle: Text(
                              apiType == 'automation'
                                  ? '${success ? '成功' : '失败'} · ${protocol.isEmpty ? '-' : protocol} · ${ts.isEmpty ? '-' : ts}'
                                  : (ts.isEmpty ? '-' : ts),
                            ),
                            trailing: apiType == 'automation'
                                ? const Icon(Icons.chevron_right_rounded)
                                : null,
                            onTap: () async {
                              if (apiType == 'automation') {
                                await _openAutomationLogDetail(row);
                                return;
                              }
                              await showDialog<void>(
                                context: context,
                                builder: (context) => AlertDialog(
                                  title: const Text('日志详情'),
                                  content: SingleChildScrollView(
                                    child: SelectableText(const JsonEncoder.withIndent('  ').convert(row)),
                                  ),
                                  actions: [
                                    TextButton(onPressed: () => Navigator.pop(context), child: const Text('关闭')),
                                  ],
                                ),
                              );
                            },
                          );
                        },
                      ),
              ),
            ],
          ),
        ),
      ),
    );
  }

  Future<void> _openAutomationLogDetail(Map<String, dynamic> row) async {
    final detail = _asMap(row['detail']);
    final request = _asMap(detail['request']);
    final response = _asMap(detail['response']);
    final protocol = _asStr(detail['protocol']).isEmpty ? _asStr(row['protocol']) : _asStr(detail['protocol']);
    final conn = _asStr(detail['connection']);
    final action = _asStr(row['action']);
    final orderId = _asStr(row['order_id']);
    final message = _asStr(row['message']);
    final bodyText = _prettyJson(_pick(response, const ['body', 'data', 'raw']));
    final reqBodyText = _prettyJson(_pick(request, const ['body', 'data', 'raw']));
    if (!mounted) return;
    await showDialog<void>(
      context: context,
      builder: (context) => AlertDialog(
        title: const Text('自动化日志详情'),
        content: SizedBox(
          width: 640,
          child: SingleChildScrollView(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text('动作: ${action.isEmpty ? '-' : action}'),
                Text('订单: ${orderId.isEmpty ? '-' : orderId}'),
                Text('协议: ${protocol.isEmpty ? '-' : protocol}'),
                Text('连接: ${conn.isEmpty ? '-' : conn}'),
                Text('结果: ${row['success'] == true ? '成功' : '失败'}'),
                if (message.isNotEmpty) ...[
                  const SizedBox(height: 6),
                  Text('消息: $message'),
                ],
                const SizedBox(height: 10),
                const Text('Request', style: TextStyle(fontWeight: FontWeight.w700)),
                const SizedBox(height: 4),
                SelectableText(
                  _prettyJson({
                    'method': request['method'],
                    'url': request['url'],
                    'headers': request['headers'],
                    'body': _decodeBest(request['body'], reqBodyText),
                  }),
                  style: const TextStyle(fontSize: 12, height: 1.35),
                ),
                const SizedBox(height: 10),
                const Text('Response', style: TextStyle(fontWeight: FontWeight.w700)),
                const SizedBox(height: 4),
                SelectableText(
                  _prettyJson({
                    'status': response['status'],
                    'duration_ms': response['duration_ms'],
                    'headers': response['headers'],
                    'body': _decodeBest(response['body'], bodyText),
                  }),
                  style: const TextStyle(fontSize: 12, height: 1.35),
                ),
              ],
            ),
          ),
        ),
        actions: [
          TextButton(
            onPressed: () async {
              final text = const JsonEncoder.withIndent('  ').convert(row);
              await Clipboard.setData(ClipboardData(text: text));
              if (!mounted) return;
              ScaffoldMessenger.of(context).showSnackBar(const SnackBar(content: Text('已复制日志详情')));
            },
            child: const Text('复制'),
          ),
          FilledButton(onPressed: () => Navigator.pop(context), child: const Text('关闭')),
        ],
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('调试中心')),
      body: RefreshIndicator(
        onRefresh: _load,
        child: ListView(
          physics: const AlwaysScrollableScrollPhysics(),
          padding: const EdgeInsets.fromLTRB(12, 12, 12, 20),
          children: [
            if (_error.isNotEmpty)
              Padding(
                padding: const EdgeInsets.only(bottom: 8),
                child: Text('加载失败：$_error'),
              ),
            Container(
              decoration: BoxDecoration(
                color: Colors.white,
                borderRadius: BorderRadius.circular(12),
                border: Border.all(color: const Color(0xFFE5EAF2)),
              ),
              child: SwitchListTile(
                title: const Text('调试模式'),
                subtitle: const Text('开启后记录更多调试日志'),
                value: _enabled,
                onChanged: _loading ? null : _toggle,
              ),
            ),
            const SizedBox(height: 10),
            Container(
              decoration: BoxDecoration(
                color: Colors.white,
                borderRadius: BorderRadius.circular(12),
                border: Border.all(color: const Color(0xFFE5EAF2)),
              ),
              padding: const EdgeInsets.all(12),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  const Text('日志保留策略(天)', style: TextStyle(fontWeight: FontWeight.w700)),
                  const SizedBox(height: 8),
                  Row(
                    children: [
                      Expanded(child: TextField(controller: _automationRetentionCtl, keyboardType: TextInputType.number, decoration: const InputDecoration(labelText: '自动化'))),
                      const SizedBox(width: 8),
                      Expanded(child: TextField(controller: _auditRetentionCtl, keyboardType: TextInputType.number, decoration: const InputDecoration(labelText: '审计'))),
                      const SizedBox(width: 8),
                      Expanded(child: TextField(controller: _syncRetentionCtl, keyboardType: TextInputType.number, decoration: const InputDecoration(labelText: '同步'))),
                    ],
                  ),
                  const SizedBox(height: 8),
                  Row(
                    children: [
                      Expanded(child: TextField(controller: _taskRetentionCtl, keyboardType: TextInputType.number, decoration: const InputDecoration(labelText: '计划任务'))),
                      const SizedBox(width: 8),
                      Expanded(child: TextField(controller: _probeEventRetentionCtl, keyboardType: TextInputType.number, decoration: const InputDecoration(labelText: '探针事件'))),
                      const SizedBox(width: 8),
                      Expanded(child: TextField(controller: _probeSessionRetentionCtl, keyboardType: TextInputType.number, decoration: const InputDecoration(labelText: '探针会话'))),
                    ],
                  ),
                  const SizedBox(height: 8),
                  Align(
                    alignment: Alignment.centerRight,
                    child: FilledButton(
                      onPressed: _savingRetention ? null : _saveRetention,
                      child: Text(_savingRetention ? '保存中...' : '保存策略'),
                    ),
                  ),
                ],
              ),
            ),
            const SizedBox(height: 10),
            _debugTile('审计日志', _count('audit_logs')),
            _debugTile('自动化日志', _count('automation_logs')),
            _debugTile('同步日志', _count('sync_logs')),
          ],
        ),
      ),
    );
  }

  Widget _debugTile(String title, int count) {
    return Container(
      margin: const EdgeInsets.only(bottom: 8),
      decoration: BoxDecoration(
        color: Colors.white,
        borderRadius: BorderRadius.circular(12),
        border: Border.all(color: const Color(0xFFE5EAF2)),
      ),
      child: ListTile(
        title: Text(title),
        trailing: Text('$count 条'),
        onTap: () {
          if (title == '审计日志') {
            _openLogDetails(title, 'audit_logs', 'audit');
            return;
          }
          if (title == '自动化日志') {
            _openLogDetails(title, 'automation_logs', 'automation');
            return;
          }
          _openLogDetails(title, 'sync_logs', 'sync');
        },
      ),
    );
  }
}

class RealnameConfigScreen extends StatefulWidget {
  const RealnameConfigScreen({super.key});

  @override
  State<RealnameConfigScreen> createState() => _RealnameConfigScreenState();
}

class _RealnameConfigScreenState extends State<RealnameConfigScreen> {
  bool _loading = false;
  bool _enabled = false;
  String _provider = '';
  List<String> _blockActions = [];
  List<Map<String, dynamic>> _providers = [];

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    _load();
  }

  Future<void> _load() async {
    final client = context.read<AppState>().apiClient;
    if (client == null) return;
    setState(() => _loading = true);
    try {
      final res = await Future.wait([
        client.getJson('/admin/api/v1/realname/config'),
        client.getJson('/admin/api/v1/realname/providers'),
      ]);
      final cfg = _asMap(res[0]);
      final items = (res[1]['items'] as List<dynamic>? ?? [])
          .map(_asMap)
          .where((e) => e.isNotEmpty)
          .toList();
      if (!mounted) return;
      setState(() {
        _enabled = cfg['enabled'] == true;
        _provider = _asStr(cfg['provider']);
        _blockActions = (cfg['block_actions'] as List<dynamic>? ?? [])
            .map((e) => e.toString())
            .where((e) => e.isNotEmpty)
            .toList();
        _providers = items;
      });
    } finally {
      if (mounted) setState(() => _loading = false);
    }
  }

  Future<void> _save() async {
    final client = context.read<AppState>().apiClient;
    if (client == null) return;
    await client.patchJson(
      '/admin/api/v1/realname/config',
      body: {'enabled': _enabled, 'provider': _provider, 'block_actions': _blockActions},
    );
    if (!mounted) return;
    ScaffoldMessenger.of(context).showSnackBar(
      const SnackBar(content: Text('保存成功')),
    );
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('实名设置'),
        actions: [TextButton(onPressed: _loading ? null : _save, child: const Text('保存'))],
      ),
      body: ListView(
        padding: const EdgeInsets.fromLTRB(12, 12, 12, 20),
        children: [
          SwitchListTile(
            value: _enabled,
            onChanged: (v) => setState(() => _enabled = v),
            title: const Text('启用实名'),
          ),
          const SizedBox(height: 8),
          DropdownButtonFormField<String>(
            value: _provider.isEmpty ? null : _provider,
            decoration: const InputDecoration(labelText: '实名服务商'),
            items: _providers
                .map((e) => DropdownMenuItem(
                      value: _asStr(e['key']),
                      child: Text(
                        _asStr(e['name']).isEmpty
                            ? _asStr(e['key'])
                            : '${_asStr(e['name'])} (${_asStr(e['key'])})',
                      ),
                    ))
                .toList(),
            onChanged: (v) => setState(() => _provider = v ?? ''),
          ),
          const SizedBox(height: 8),
          const Align(
            alignment: Alignment.centerLeft,
            child: Text('限制的操作', style: TextStyle(fontWeight: FontWeight.w700)),
          ),
          const SizedBox(height: 6),
          Wrap(
            spacing: 8,
            runSpacing: 8,
            children: [
              FilterChip(
                label: const Text('购买 VPS'),
                selected: _blockActions.contains('purchase_vps'),
                onSelected: (v) => setState(() {
                  if (v) {
                    if (!_blockActions.contains('purchase_vps')) _blockActions.add('purchase_vps');
                  } else {
                    _blockActions.remove('purchase_vps');
                  }
                }),
              ),
              FilterChip(
                label: const Text('续费 VPS'),
                selected: _blockActions.contains('renew_vps'),
                onSelected: (v) => setState(() {
                  if (v) {
                    if (!_blockActions.contains('renew_vps')) _blockActions.add('renew_vps');
                  } else {
                    _blockActions.remove('renew_vps');
                  }
                }),
              ),
              FilterChip(
                label: const Text('扩容 VPS'),
                selected: _blockActions.contains('resize_vps'),
                onSelected: (v) => setState(() {
                  if (v) {
                    if (!_blockActions.contains('resize_vps')) _blockActions.add('resize_vps');
                  } else {
                    _blockActions.remove('resize_vps');
                  }
                }),
              ),
            ],
          ),
        ],
      ),
    );
  }
}

class SmsSettingsScreen extends StatefulWidget {
  const SmsSettingsScreen({super.key});

  @override
  State<SmsSettingsScreen> createState() => _SmsSettingsScreenState();
}

class _SmsSettingsScreenState extends State<SmsSettingsScreen> {
  bool _loading = false;
  bool _enabled = false;
  final _pluginCtl = TextEditingController();
  final _instanceCtl = TextEditingController();
  final _defaultTplCtl = TextEditingController();
  final _providerTplCtl = TextEditingController();
  List<Map<String, dynamic>> _templates = [];
  List<Map<String, dynamic>> _pluginItems = [];

  @override
  void dispose() {
    _pluginCtl.dispose();
    _instanceCtl.dispose();
    _defaultTplCtl.dispose();
    _providerTplCtl.dispose();
    super.dispose();
  }

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    _load();
  }

  Future<void> _load() async {
    final client = context.read<AppState>().apiClient;
    if (client == null) return;
    setState(() => _loading = true);
    try {
      final res = await Future.wait([
        client.getJson('/admin/api/v1/integrations/sms'),
        client.getJson('/admin/api/v1/sms-templates'),
        client.getJson('/admin/api/v1/plugins'),
      ]);
      final cfg = _asMap(res[0]);
      final tpls = (res[1]['items'] as List<dynamic>? ?? [])
          .map(_asMap)
          .where((e) => e.isNotEmpty)
          .toList();
      final plugins = (res[2]['items'] as List<dynamic>? ?? [])
          .map(_asMap)
          .where((e) => e.isNotEmpty)
          .toList();
      if (!mounted) return;
      setState(() {
        _enabled = cfg['enabled'] == true;
        _pluginCtl.text = _asStr(cfg['plugin_id']);
        _instanceCtl.text = _asStr(cfg['instance_id']);
        _defaultTplCtl.text = _asStr(cfg['default_template_id']);
        _providerTplCtl.text = _asStr(cfg['provider_template_id']);
        _templates = tpls;
        _pluginItems = plugins;
      });
    } finally {
      if (mounted) setState(() => _loading = false);
    }
  }

  Future<void> _saveConfig() async {
    final client = context.read<AppState>().apiClient;
    if (client == null) return;
    await client.patchJson('/admin/api/v1/integrations/sms', body: {
      'enabled': _enabled,
      'plugin_id': _pluginCtl.text.trim(),
      'instance_id': _instanceCtl.text.trim(),
      'default_template_id': _defaultTplCtl.text.trim(),
      'provider_template_id': _providerTplCtl.text.trim(),
    });
    if (!mounted) return;
    ScaffoldMessenger.of(context).showSnackBar(
      const SnackBar(content: Text('短信配置已保存')),
    );
    await _load();
  }

  Future<void> _editTemplate({Map<String, dynamic>? row}) async {
    final id = row?['id'];
    final nameCtl = TextEditingController(text: _asStr(row?['name']));
    final contentCtl = TextEditingController(text: _asStr(row?['content']));
    var enabled = row?['enabled'] == true;
    final action = await showDialog<String>(
      context: context,
      builder: (context) => StatefulBuilder(
        builder: (context, setModal) => AlertDialog(
          title: Text(id == null ? '新增短信模板' : '编辑短信模板'),
          content: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              TextField(controller: nameCtl, decoration: const InputDecoration(labelText: '模板名称')),
              const SizedBox(height: 8),
              TextField(controller: contentCtl, maxLines: 4, decoration: const InputDecoration(labelText: '模板内容')),
              const SizedBox(height: 8),
              SwitchListTile(
                value: enabled,
                onChanged: (v) => setModal(() => enabled = v),
                title: const Text('启用'),
              ),
            ],
          ),
          actions: [
            if (id != null)
              TextButton(
                onPressed: () => Navigator.pop(context, 'delete'),
                child: const Text('删除'),
              ),
            TextButton(onPressed: () => Navigator.pop(context, 'cancel'), child: const Text('取消')),
            FilledButton(onPressed: () => Navigator.pop(context, 'save'), child: const Text('保存')),
          ],
        ),
      ),
    );
    final client = context.read<AppState>().apiClient;
    if (client == null) return;
    if (action == 'delete' && id != null) {
      await client.deleteJson('/admin/api/v1/sms-templates/$id');
      await _load();
      return;
    }
    if (action != 'save') return;
    final payload = {
      if (id != null) 'id': id,
      'name': nameCtl.text.trim(),
      'content': contentCtl.text.trim(),
      'enabled': enabled,
    };
    if (id == null) {
      await client.postJson('/admin/api/v1/sms-templates', body: payload);
    } else {
      await client.patchJson('/admin/api/v1/sms-templates/$id', body: payload);
    }
    await _load();
  }

  @override
  Widget build(BuildContext context) {
    final smsBindings = _pluginItems
        .where((e) => _asStr(e['category']) == 'sms')
        .map((e) {
          final pid = _asStr(e['plugin_id']);
          final iid = _asStr(e['instance_id']).isEmpty ? 'default' : _asStr(e['instance_id']);
          final name = _asStr(e['name']).isEmpty ? pid : _asStr(e['name']);
          return (
            value: '$pid::$iid',
            label: '$name ($pid/$iid)',
          );
        })
        .where((e) => e.value.isNotEmpty)
        .toList();
    final selectedBindingRaw = _pluginCtl.text.trim().isEmpty
        ? null
        : '${_pluginCtl.text.trim()}::${_instanceCtl.text.trim().isEmpty ? 'default' : _instanceCtl.text.trim()}';
    final selectedBinding = selectedBindingRaw != null &&
            smsBindings.any((e) => e.value == selectedBindingRaw)
        ? selectedBindingRaw
        : null;
    final tplOptions = _templates
        .map((e) => (
              value: _asStr(e['id']),
              label: _asStr(e['name']).isEmpty ? _asStr(e['id']) : _asStr(e['name']),
            ))
        .where((e) => e.value.isNotEmpty)
        .toList();
    return Scaffold(
      appBar: AppBar(
        title: const Text('短信设置'),
        actions: [
          IconButton(onPressed: _loading ? null : _load, icon: const Icon(Icons.refresh)),
          TextButton(onPressed: _loading ? null : _saveConfig, child: const Text('保存')),
        ],
      ),
      body: ListView(
        padding: const EdgeInsets.fromLTRB(12, 12, 12, 20),
        children: [
          SwitchListTile(
            value: _enabled,
            onChanged: (v) => setState(() => _enabled = v),
            title: const Text('启用短信'),
          ),
          const SizedBox(height: 8),
          DropdownButtonFormField<String>(
            value: selectedBinding,
            decoration: const InputDecoration(labelText: '短信插件'),
            items: smsBindings
                .map((e) => DropdownMenuItem(value: e.value, child: Text(e.label)))
                .toList(),
            onChanged: (v) {
              if (v == null || v.isEmpty) {
                setState(() {
                  _pluginCtl.text = '';
                  _instanceCtl.text = '';
                });
                return;
              }
              final parts = v.split('::');
              setState(() {
                _pluginCtl.text = parts.isNotEmpty ? parts.first : '';
                _instanceCtl.text = parts.length > 1 ? parts[1] : 'default';
              });
            },
          ),
          const SizedBox(height: 8),
          TextField(controller: _instanceCtl, decoration: const InputDecoration(labelText: '实例 ID (自动可改)')),
          const SizedBox(height: 8),
          DropdownButtonFormField<String>(
            value: _defaultTplCtl.text.isEmpty ? null : _defaultTplCtl.text,
            decoration: const InputDecoration(labelText: '默认模板'),
            items: tplOptions
                .map((e) => DropdownMenuItem(value: e.value, child: Text(e.label)))
                .toList(),
            onChanged: (v) => setState(() => _defaultTplCtl.text = v ?? ''),
          ),
          const SizedBox(height: 8),
          DropdownButtonFormField<String>(
            value: _providerTplCtl.text.isEmpty ? null : _providerTplCtl.text,
            decoration: const InputDecoration(labelText: '供应商模板'),
            items: tplOptions
                .map((e) => DropdownMenuItem(value: e.value, child: Text(e.label)))
                .toList(),
            onChanged: (v) => setState(() => _providerTplCtl.text = v ?? ''),
          ),
          const SizedBox(height: 12),
          Row(
            children: [
              const Expanded(
                child: Text('短信模板', style: TextStyle(fontSize: 15, fontWeight: FontWeight.w700)),
              ),
              FilledButton.icon(
                onPressed: () => _editTemplate(),
                icon: const Icon(Icons.add, size: 16),
                label: const Text('新增'),
              ),
            ],
          ),
          const SizedBox(height: 8),
          ..._templates.map((row) => Container(
                margin: const EdgeInsets.only(bottom: 8),
                decoration: BoxDecoration(
                  color: Colors.white,
                  borderRadius: BorderRadius.circular(12),
                  border: Border.all(color: const Color(0xFFE5EAF2)),
                ),
                child: ListTile(
                  title: Text(_asStr(row['name'])),
                  subtitle: Text(
                    _asStr(row['content']),
                    maxLines: 2,
                    overflow: TextOverflow.ellipsis,
                  ),
                  trailing: Icon(
                    row['enabled'] == true ? Icons.check_circle : Icons.pause_circle,
                    color: row['enabled'] == true ? const Color(0xFF16A34A) : const Color(0xFF64748B),
                  ),
                  onTap: () => _editTemplate(row: row),
                ),
              )),
        ],
      ),
    );
  }
}

class SiteSettingsScreen extends StatefulWidget {
  const SiteSettingsScreen({super.key});

  @override
  State<SiteSettingsScreen> createState() => _SiteSettingsScreenState();
}

class _SiteSettingsScreenState extends State<SiteSettingsScreen> {
  bool _loading = false;
  final _name = TextEditingController();
  final _url = TextEditingController();
  final _logo = TextEditingController();
  final _favicon = TextEditingController();
  final _desc = TextEditingController();
  final _keywords = TextEditingController();
  final _company = TextEditingController();
  final _contactPhone = TextEditingController();
  final _contactEmail = TextEditingController();
  final _contactQq = TextEditingController();
  final _wechatQrcode = TextEditingController();
  final _icp = TextEditingController();
  final _psbe = TextEditingController();
  bool _maintenanceMode = false;
  final _maintenanceMessage = TextEditingController();
  final _stats = TextEditingController();

  @override
  void dispose() {
    _name.dispose();
    _url.dispose();
    _logo.dispose();
    _favicon.dispose();
    _desc.dispose();
    _keywords.dispose();
    _company.dispose();
    _contactPhone.dispose();
    _contactEmail.dispose();
    _contactQq.dispose();
    _wechatQrcode.dispose();
    _icp.dispose();
    _psbe.dispose();
    _maintenanceMessage.dispose();
    _stats.dispose();
    super.dispose();
  }

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    _load();
  }

  Future<void> _load() async {
    final client = context.read<AppState>().apiClient;
    if (client == null) return;
    setState(() => _loading = true);
    try {
      final map = await _loadSettingsMap(client);
      _name.text = _strVal(map['site_name']);
      _url.text = _strVal(map['site_url']);
      _logo.text = _strVal(map['logo_url']).isNotEmpty ? _strVal(map['logo_url']) : _strVal(map['site_logo']);
      _favicon.text = _strVal(map['favicon_url']).isNotEmpty ? _strVal(map['favicon_url']) : _strVal(map['site_favicon']);
      _desc.text = _strVal(map['site_description']);
      _keywords.text = _strVal(map['site_keywords']);
      _company.text = _strVal(map['company_name']);
      _contactPhone.text = _strVal(map['contact_phone']);
      _contactEmail.text = _strVal(map['contact_email']);
      _contactQq.text = _strVal(map['contact_qq']);
      _wechatQrcode.text = _strVal(map['wechat_qrcode']);
      _icp.text = _strVal(map['icp_number']).isNotEmpty ? _strVal(map['icp_number']) : _strVal(map['site_icp']);
      _psbe.text = _strVal(map['psbe_number']);
      _maintenanceMode = _toBool(map['maintenance_mode'], false);
      _maintenanceMessage.text = _strVal(map['maintenance_message']);
      _stats.text = _strVal(map['analytics_code']).isNotEmpty
          ? _strVal(map['analytics_code'])
          : _strVal(map['site_statistics_code']);
      if (mounted) setState(() {});
    } finally {
      if (mounted) setState(() => _loading = false);
    }
  }

  Future<void> _save() async {
    final client = context.read<AppState>().apiClient;
    if (client == null) return;
    await _saveSettingsItems(client, [
      ('site_name', _name.text.trim()),
      ('site_url', _url.text.trim()),
      ('logo_url', _logo.text.trim()),
      ('favicon_url', _favicon.text.trim()),
      ('site_description', _desc.text.trim()),
      ('site_keywords', _keywords.text.trim()),
      ('company_name', _company.text.trim()),
      ('contact_phone', _contactPhone.text.trim()),
      ('contact_email', _contactEmail.text.trim()),
      ('contact_qq', _contactQq.text.trim()),
      ('wechat_qrcode', _wechatQrcode.text.trim()),
      ('icp_number', _icp.text.trim()),
      ('psbe_number', _psbe.text.trim()),
      ('maintenance_mode', '$_maintenanceMode'),
      ('maintenance_message', _maintenanceMessage.text.trim()),
      ('analytics_code', _stats.text.trim()),
    ]);
    if (!mounted) return;
    ScaffoldMessenger.of(context).showSnackBar(const SnackBar(content: Text('站点设置已保存')));
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('站点设置'),
        actions: [
          IconButton(onPressed: _loading ? null : _load, icon: const Icon(Icons.refresh)),
          TextButton(onPressed: _loading ? null : _save, child: const Text('保存')),
        ],
      ),
      body: ListView(
        padding: const EdgeInsets.fromLTRB(12, 12, 12, 20),
        children: [
          TextField(controller: _name, decoration: const InputDecoration(labelText: '站点名称')),
          const SizedBox(height: 8),
          TextField(controller: _url, decoration: const InputDecoration(labelText: '站点 URL')),
          const SizedBox(height: 8),
          TextField(controller: _logo, decoration: const InputDecoration(labelText: 'Logo URL')),
          const SizedBox(height: 8),
          TextField(controller: _favicon, decoration: const InputDecoration(labelText: 'Favicon URL')),
          const SizedBox(height: 8),
          TextField(controller: _desc, maxLines: 3, decoration: const InputDecoration(labelText: '站点描述')),
          const SizedBox(height: 8),
          TextField(controller: _keywords, decoration: const InputDecoration(labelText: '关键词')),
          const SizedBox(height: 8),
          TextField(controller: _company, decoration: const InputDecoration(labelText: '公司名称')),
          const SizedBox(height: 8),
          TextField(controller: _contactPhone, decoration: const InputDecoration(labelText: '联系电话')),
          const SizedBox(height: 8),
          TextField(controller: _contactEmail, decoration: const InputDecoration(labelText: '联系邮箱')),
          const SizedBox(height: 8),
          TextField(controller: _contactQq, decoration: const InputDecoration(labelText: 'QQ 号码')),
          const SizedBox(height: 8),
          TextField(controller: _wechatQrcode, decoration: const InputDecoration(labelText: '微信二维码 URL')),
          const SizedBox(height: 8),
          TextField(controller: _icp, decoration: const InputDecoration(labelText: 'ICP备案号')),
          const SizedBox(height: 8),
          TextField(controller: _psbe, decoration: const InputDecoration(labelText: '公安备案号')),
          const SizedBox(height: 8),
          SwitchListTile(
            value: _maintenanceMode,
            onChanged: (v) => setState(() => _maintenanceMode = v),
            title: const Text('维护模式'),
          ),
          const SizedBox(height: 8),
          TextField(controller: _maintenanceMessage, maxLines: 2, decoration: const InputDecoration(labelText: '维护提示信息')),
          const SizedBox(height: 8),
          TextField(controller: _stats, maxLines: 3, decoration: const InputDecoration(labelText: '统计代码')),
        ],
      ),
    );
  }
}

class AuthSettingsScreen extends StatefulWidget {
  const AuthSettingsScreen({super.key});

  @override
  State<AuthSettingsScreen> createState() => _AuthSettingsScreenState();
}

class _AuthSettingsScreenState extends State<AuthSettingsScreen> {
  bool _loading = false;
  bool registerEnabled = true;
  bool registerEmailRequired = true;
  bool passwordUpper = false;
  bool passwordLower = false;
  bool passwordNumber = false;
  bool passwordSymbol = false;
  bool loginRateLimitEnabled = true;
  bool loginNotifyEnabled = true;
  bool loginNotifyFirst = true;
  bool loginNotifyIp = true;
  bool passwordResetEnabled = true;
  bool authEmailBindEnabled = true;
  bool authPhoneBindEnabled = true;
  bool authContactChangeNotifyOldEnabled = true;
  bool authBindRequirePasswordWhenNo2fa = false;
  bool authRebindRequirePasswordWhenNo2fa = true;
  bool auth2faEnabled = true;
  bool auth2faBindEnabled = true;
  bool auth2faRebindEnabled = true;
  final requiredFieldsCtl = TextEditingController(text: 'username,password');
  final verifyChannelsCtl = TextEditingController(text: 'email');
  final loginNotifyChannelsCtl = TextEditingController(text: 'email');
  final pwdResetChannelsCtl = TextEditingController(text: 'email');
  final minLenCtl = TextEditingController(text: '6');
  final verifyTtlCtl = TextEditingController(text: '600');
  final rateWindowCtl = TextEditingController(text: '300');
  final rateMaxCtl = TextEditingController(text: '5');
  final pwdResetTtlCtl = TextEditingController(text: '600');
  final bindTtlCtl = TextEditingController(text: '600');
  final smsLenCtl = TextEditingController(text: '6');
  final smsComplexCtl = TextEditingController(text: 'digits');
  final emailLenCtl = TextEditingController(text: '6');
  final emailComplexCtl = TextEditingController(text: 'alnum');

  @override
  void dispose() {
    requiredFieldsCtl.dispose();
    verifyChannelsCtl.dispose();
    loginNotifyChannelsCtl.dispose();
    pwdResetChannelsCtl.dispose();
    minLenCtl.dispose();
    verifyTtlCtl.dispose();
    rateWindowCtl.dispose();
    rateMaxCtl.dispose();
    pwdResetTtlCtl.dispose();
    bindTtlCtl.dispose();
    smsLenCtl.dispose();
    smsComplexCtl.dispose();
    emailLenCtl.dispose();
    emailComplexCtl.dispose();
    super.dispose();
  }

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    _load();
  }

  Future<void> _load() async {
    final client = context.read<AppState>().apiClient;
    if (client == null) return;
    setState(() => _loading = true);
    try {
      final map = await _loadSettingsMap(client);
      registerEnabled = _toBool(map['auth_register_enabled'], true);
      registerEmailRequired = _toBool(map['auth_register_email_required'], true);
      passwordUpper = _toBool(map['auth_password_require_upper'], false);
      passwordLower = _toBool(map['auth_password_require_lower'], false);
      passwordNumber = _toBool(map['auth_password_require_number'], false);
      passwordSymbol = _toBool(map['auth_password_require_symbol'], false);
      loginRateLimitEnabled = _toBool(map['auth_login_rate_limit_enabled'], true);
      loginNotifyEnabled = _toBool(map['auth_login_notify_enabled'], true);
      loginNotifyFirst = _toBool(map['auth_login_notify_on_first_login'], true);
      loginNotifyIp = _toBool(map['auth_login_notify_on_ip_change'], true);
      passwordResetEnabled = _toBool(map['auth_password_reset_enabled'], true);
      authEmailBindEnabled = _toBool(map['auth_email_bind_enabled'], true);
      authPhoneBindEnabled = _toBool(map['auth_phone_bind_enabled'], true);
      authContactChangeNotifyOldEnabled = _toBool(map['auth_contact_change_notify_old_enabled'], true);
      authBindRequirePasswordWhenNo2fa = _toBool(map['auth_bind_require_password_when_no_2fa'], false);
      authRebindRequirePasswordWhenNo2fa = _toBool(map['auth_rebind_require_password_when_no_2fa'], true);
      auth2faEnabled = _toBool(map['auth_2fa_enabled'], true);
      auth2faBindEnabled = _toBool(map['auth_2fa_bind_enabled'], true);
      auth2faRebindEnabled = _toBool(map['auth_2fa_rebind_enabled'], true);
      requiredFieldsCtl.text = _listToCsv(_toList(map['auth_register_required_fields'], ['username', 'password']));
      verifyChannelsCtl.text = _listToCsv(_toList(map['auth_register_verify_channels'], ['email']));
      loginNotifyChannelsCtl.text = _listToCsv(_toList(map['auth_login_notify_channels'], ['email']));
      pwdResetChannelsCtl.text = _listToCsv(_toList(map['auth_password_reset_channels'], ['email']));
      minLenCtl.text = '${_toInt(map['auth_password_min_len'], 6)}';
      verifyTtlCtl.text = '${_toInt(map['auth_register_verify_ttl_sec'], 600)}';
      rateWindowCtl.text = '${_toInt(map['auth_login_rate_limit_window_sec'], 300)}';
      rateMaxCtl.text = '${_toInt(map['auth_login_rate_limit_max_attempts'], 5)}';
      pwdResetTtlCtl.text = '${_toInt(map['auth_password_reset_verify_ttl_sec'], 600)}';
      bindTtlCtl.text = '${_toInt(map['auth_contact_bind_verify_ttl_sec'], 600)}';
      smsLenCtl.text = '${_toInt(map['auth_sms_code_len'], 6)}';
      smsComplexCtl.text = _strVal(map['auth_sms_code_complexity']).isEmpty ? 'digits' : _strVal(map['auth_sms_code_complexity']);
      emailLenCtl.text = '${_toInt(map['auth_email_code_len'], 6)}';
      emailComplexCtl.text = _strVal(map['auth_email_code_complexity']).isEmpty ? 'alnum' : _strVal(map['auth_email_code_complexity']);
      if (mounted) setState(() {});
    } finally {
      if (mounted) setState(() => _loading = false);
    }
  }

  Future<void> _save() async {
    final client = context.read<AppState>().apiClient;
    if (client == null) return;
    final required = _csvToList(requiredFieldsCtl.text, fallback: ['username', 'password']);
    if (!required.contains('username')) required.add('username');
    if (!required.contains('password')) required.add('password');
    required.remove('email');
    final verifyChannels = _csvToList(verifyChannelsCtl.text, fallback: ['email']);
    final registerVerifyType = verifyChannels.contains('email')
        ? 'email'
        : (verifyChannels.contains('sms') ? 'sms' : 'none');
    await _saveSettingsItems(client, [
      ('auth_register_enabled', '$registerEnabled'),
      ('auth_register_required_fields', _jsonList(required)),
      ('auth_register_email_required', '$registerEmailRequired'),
      ('auth_password_min_len', '${_toInt(minLenCtl.text, 6)}'),
      ('auth_password_require_upper', '$passwordUpper'),
      ('auth_password_require_lower', '$passwordLower'),
      ('auth_password_require_number', '$passwordNumber'),
      ('auth_password_require_symbol', '$passwordSymbol'),
      ('auth_register_verify_type', registerVerifyType),
      ('auth_register_verify_channels', _jsonList(verifyChannels)),
      ('auth_register_verify_ttl_sec', '${_toInt(verifyTtlCtl.text, 600)}'),
      ('auth_login_rate_limit_enabled', '$loginRateLimitEnabled'),
      ('auth_login_rate_limit_window_sec', '${_toInt(rateWindowCtl.text, 300)}'),
      ('auth_login_rate_limit_max_attempts', '${_toInt(rateMaxCtl.text, 5)}'),
      ('auth_login_notify_enabled', '$loginNotifyEnabled'),
      ('auth_login_notify_channels', _jsonList(_csvToList(loginNotifyChannelsCtl.text, fallback: ['email']))),
      ('auth_login_notify_on_first_login', '$loginNotifyFirst'),
      ('auth_login_notify_on_ip_change', '$loginNotifyIp'),
      ('auth_password_reset_enabled', '$passwordResetEnabled'),
      ('auth_password_reset_channels', _jsonList(_csvToList(pwdResetChannelsCtl.text, fallback: ['email']))),
      ('auth_password_reset_verify_ttl_sec', '${_toInt(pwdResetTtlCtl.text, 600)}'),
      ('auth_email_bind_enabled', '$authEmailBindEnabled'),
      ('auth_phone_bind_enabled', '$authPhoneBindEnabled'),
      ('auth_contact_change_notify_old_enabled', '$authContactChangeNotifyOldEnabled'),
      ('auth_contact_bind_verify_ttl_sec', '${_toInt(bindTtlCtl.text, 600)}'),
      ('auth_bind_require_password_when_no_2fa', '$authBindRequirePasswordWhenNo2fa'),
      ('auth_rebind_require_password_when_no_2fa', '$authRebindRequirePasswordWhenNo2fa'),
      ('auth_sms_code_len', '${_toInt(smsLenCtl.text, 6)}'),
      ('auth_sms_code_complexity', smsComplexCtl.text.trim()),
      ('auth_email_code_len', '${_toInt(emailLenCtl.text, 6)}'),
      ('auth_email_code_complexity', emailComplexCtl.text.trim()),
      ('auth_2fa_enabled', '$auth2faEnabled'),
      ('auth_2fa_bind_enabled', '$auth2faBindEnabled'),
      ('auth_2fa_rebind_enabled', '$auth2faRebindEnabled'),
    ]);
    if (!mounted) return;
    ScaffoldMessenger.of(context).showSnackBar(const SnackBar(content: Text('注册与登录设置已保存')));
  }

  List<String> _csvValues(TextEditingController ctl) {
    return ctl.text
        .split(',')
        .map((e) => e.trim().toLowerCase())
        .where((e) => e.isNotEmpty)
        .toList();
  }

  void _toggleCsvValue(TextEditingController ctl, String value, bool selected) {
    final set = _csvValues(ctl).toSet();
    if (selected) {
      set.add(value);
    } else {
      set.remove(value);
    }
    ctl.text = set.join(',');
    setState(() {});
  }

  @override
  Widget build(BuildContext context) {
    final requiredSet = _csvValues(requiredFieldsCtl).toSet();
    requiredSet.add('username');
    requiredSet.add('password');
    requiredSet.remove('email');
    final verifyChannelsSet = _csvValues(verifyChannelsCtl).toSet();
    final loginNotifyChannelsSet = _csvValues(loginNotifyChannelsCtl).toSet();
    final pwdResetChannelsSet = _csvValues(pwdResetChannelsCtl).toSet();
    return Scaffold(
      appBar: AppBar(
        title: const Text('注册设置'),
        actions: [
          IconButton(onPressed: _loading ? null : _load, icon: const Icon(Icons.refresh)),
          TextButton(onPressed: _loading ? null : _save, child: const Text('保存')),
        ],
      ),
      body: ListView(
        padding: const EdgeInsets.fromLTRB(12, 12, 12, 20),
        children: [
          SwitchListTile(value: registerEnabled, onChanged: (v) => setState(() => registerEnabled = v), title: const Text('开启注册')),
          const Align(alignment: Alignment.centerLeft, child: Text('必填字段', style: TextStyle(fontWeight: FontWeight.w700))),
          const SizedBox(height: 6),
          Wrap(spacing: 8, children: [
            FilterChip(label: const Text('用户名'), selected: true, onSelected: null),
            FilterChip(label: const Text('密码'), selected: true, onSelected: null),
            FilterChip(
              label: const Text('手机号'),
              selected: requiredSet.contains('phone'),
              onSelected: (v) => _toggleCsvValue(requiredFieldsCtl, 'phone', v),
            ),
            FilterChip(
              label: const Text('QQ'),
              selected: requiredSet.contains('qq'),
              onSelected: (v) => _toggleCsvValue(requiredFieldsCtl, 'qq', v),
            ),
          ]),
          const SizedBox(height: 8),
          SwitchListTile(value: registerEmailRequired, onChanged: (v) => setState(() => registerEmailRequired = v), title: const Text('邮箱必填')),
          const SizedBox(height: 8),
          TextField(controller: minLenCtl, keyboardType: TextInputType.number, decoration: const InputDecoration(labelText: '密码最小长度')),
          const SizedBox(height: 6),
          Wrap(spacing: 8, children: [
            FilterChip(label: const Text('大写'), selected: passwordUpper, onSelected: (v) => setState(() => passwordUpper = v)),
            FilterChip(label: const Text('小写'), selected: passwordLower, onSelected: (v) => setState(() => passwordLower = v)),
            FilterChip(label: const Text('数字'), selected: passwordNumber, onSelected: (v) => setState(() => passwordNumber = v)),
            FilterChip(label: const Text('符号'), selected: passwordSymbol, onSelected: (v) => setState(() => passwordSymbol = v)),
          ]),
          const SizedBox(height: 8),
          const Align(alignment: Alignment.centerLeft, child: Text('注册验证渠道', style: TextStyle(fontWeight: FontWeight.w700))),
          const SizedBox(height: 6),
          Wrap(spacing: 8, children: [
            FilterChip(
              label: const Text('邮箱'),
              selected: verifyChannelsSet.contains('email'),
              onSelected: (v) => _toggleCsvValue(verifyChannelsCtl, 'email', v),
            ),
            FilterChip(
              label: const Text('短信'),
              selected: verifyChannelsSet.contains('sms'),
              onSelected: (v) => _toggleCsvValue(verifyChannelsCtl, 'sms', v),
            ),
          ]),
          const SizedBox(height: 8),
          TextField(controller: verifyTtlCtl, keyboardType: TextInputType.number, decoration: const InputDecoration(labelText: '注册验证码有效期(秒)')),
          const SizedBox(height: 8),
          SwitchListTile(value: loginRateLimitEnabled, onChanged: (v) => setState(() => loginRateLimitEnabled = v), title: const Text('启用登录频率限制')),
          Row(children: [
            Expanded(child: TextField(controller: rateWindowCtl, keyboardType: TextInputType.number, decoration: const InputDecoration(labelText: '窗口秒数'))),
            const SizedBox(width: 8),
            Expanded(child: TextField(controller: rateMaxCtl, keyboardType: TextInputType.number, decoration: const InputDecoration(labelText: '最大次数'))),
          ]),
          const SizedBox(height: 8),
          SwitchListTile(value: loginNotifyEnabled, onChanged: (v) => setState(() => loginNotifyEnabled = v), title: const Text('登录提醒开关')),
          Row(children: [
            Expanded(child: SwitchListTile(value: loginNotifyFirst, onChanged: (v) => setState(() => loginNotifyFirst = v), title: const Text('首次登录'))),
            Expanded(child: SwitchListTile(value: loginNotifyIp, onChanged: (v) => setState(() => loginNotifyIp = v), title: const Text('IP变化'))),
          ]),
          const Align(alignment: Alignment.centerLeft, child: Text('登录提醒渠道', style: TextStyle(fontWeight: FontWeight.w700))),
          const SizedBox(height: 6),
          Wrap(spacing: 8, children: [
            FilterChip(
              label: const Text('邮箱'),
              selected: loginNotifyChannelsSet.contains('email'),
              onSelected: (v) => _toggleCsvValue(loginNotifyChannelsCtl, 'email', v),
            ),
            FilterChip(
              label: const Text('短信'),
              selected: loginNotifyChannelsSet.contains('sms'),
              onSelected: (v) => _toggleCsvValue(loginNotifyChannelsCtl, 'sms', v),
            ),
          ]),
          const SizedBox(height: 8),
          SwitchListTile(value: passwordResetEnabled, onChanged: (v) => setState(() => passwordResetEnabled = v), title: const Text('找回密码开关')),
          const Align(alignment: Alignment.centerLeft, child: Text('找回密码渠道', style: TextStyle(fontWeight: FontWeight.w700))),
          const SizedBox(height: 6),
          Wrap(spacing: 8, children: [
            FilterChip(
              label: const Text('邮箱'),
              selected: pwdResetChannelsSet.contains('email'),
              onSelected: (v) => _toggleCsvValue(pwdResetChannelsCtl, 'email', v),
            ),
            FilterChip(
              label: const Text('短信'),
              selected: pwdResetChannelsSet.contains('sms'),
              onSelected: (v) => _toggleCsvValue(pwdResetChannelsCtl, 'sms', v),
            ),
          ]),
          const SizedBox(height: 8),
          TextField(controller: pwdResetTtlCtl, keyboardType: TextInputType.number, decoration: const InputDecoration(labelText: '找回验证码有效期(秒)')),
          const SizedBox(height: 8),
          SwitchListTile(value: authEmailBindEnabled, onChanged: (v) => setState(() => authEmailBindEnabled = v), title: const Text('邮箱绑定功能')),
          SwitchListTile(value: authPhoneBindEnabled, onChanged: (v) => setState(() => authPhoneBindEnabled = v), title: const Text('手机号绑定功能')),
          SwitchListTile(
            value: authContactChangeNotifyOldEnabled,
            onChanged: (v) => setState(() => authContactChangeNotifyOldEnabled = v),
            title: const Text('换绑后通知旧联系方式'),
          ),
          TextField(controller: bindTtlCtl, keyboardType: TextInputType.number, decoration: const InputDecoration(labelText: '绑定验证码有效期(秒)')),
          const SizedBox(height: 8),
          SwitchListTile(
            value: authBindRequirePasswordWhenNo2fa,
            onChanged: (v) => setState(() => authBindRequirePasswordWhenNo2fa = v),
            title: const Text('未开2FA时首次绑定需密码'),
          ),
          SwitchListTile(
            value: authRebindRequirePasswordWhenNo2fa,
            onChanged: (v) => setState(() => authRebindRequirePasswordWhenNo2fa = v),
            title: const Text('未开2FA时换绑需密码'),
          ),
          const SizedBox(height: 8),
          Row(children: [
            Expanded(child: TextField(controller: smsLenCtl, keyboardType: TextInputType.number, decoration: const InputDecoration(labelText: '短信码长度'))),
            const SizedBox(width: 8),
            Expanded(
              child: DropdownButtonFormField<String>(
                value: smsComplexCtl.text.isEmpty ? 'digits' : smsComplexCtl.text,
                decoration: const InputDecoration(labelText: '短信复杂度'),
                items: const [
                  DropdownMenuItem(value: 'digits', child: Text('纯数字')),
                  DropdownMenuItem(value: 'letters', child: Text('纯字母(大写)')),
                  DropdownMenuItem(value: 'alnum', child: Text('字母+数字')),
                ],
                onChanged: (v) => setState(() => smsComplexCtl.text = v ?? 'digits'),
              ),
            ),
          ]),
          const SizedBox(height: 8),
          Row(children: [
            Expanded(child: TextField(controller: emailLenCtl, keyboardType: TextInputType.number, decoration: const InputDecoration(labelText: '邮箱码长度'))),
            const SizedBox(width: 8),
            Expanded(
              child: DropdownButtonFormField<String>(
                value: emailComplexCtl.text.isEmpty ? 'alnum' : emailComplexCtl.text,
                decoration: const InputDecoration(labelText: '邮箱复杂度'),
                items: const [
                  DropdownMenuItem(value: 'digits', child: Text('纯数字')),
                  DropdownMenuItem(value: 'letters', child: Text('纯字母(大写)')),
                  DropdownMenuItem(value: 'alnum', child: Text('字母+数字')),
                ],
                onChanged: (v) => setState(() => emailComplexCtl.text = v ?? 'alnum'),
              ),
            ),
          ]),
          const SizedBox(height: 8),
          SwitchListTile(value: auth2faEnabled, onChanged: (v) => setState(() => auth2faEnabled = v), title: const Text('2FA 总开关')),
          Row(children: [
            Expanded(child: SwitchListTile(value: auth2faBindEnabled, onChanged: (v) => setState(() => auth2faBindEnabled = v), title: const Text('2FA 绑定'))),
            Expanded(child: SwitchListTile(value: auth2faRebindEnabled, onChanged: (v) => setState(() => auth2faRebindEnabled = v), title: const Text('2FA 换绑'))),
          ]),
        ],
      ),
    );
  }
}

class CaptchaSettingsScreen extends StatefulWidget {
  const CaptchaSettingsScreen({super.key});

  @override
  State<CaptchaSettingsScreen> createState() => _CaptchaSettingsScreenState();
}

class _CaptchaSettingsScreenState extends State<CaptchaSettingsScreen> {
  bool _loading = false;
  bool registerEnabled = true;
  bool loginEnabled = false;
  String provider = 'image';
  final lenCtl = TextEditingController(text: '5');
  final complexityCtl = TextEditingController(text: 'alnum');
  final geeIdCtl = TextEditingController();
  final geeKeyCtl = TextEditingController();
  final geeApiCtl = TextEditingController(text: 'https://gcaptcha4.geetest.com');

  @override
  void dispose() {
    lenCtl.dispose();
    complexityCtl.dispose();
    geeIdCtl.dispose();
    geeKeyCtl.dispose();
    geeApiCtl.dispose();
    super.dispose();
  }

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    _load();
  }

  Future<void> _load() async {
    final client = context.read<AppState>().apiClient;
    if (client == null) return;
    setState(() => _loading = true);
    try {
      final map = await _loadSettingsMap(client);
      registerEnabled = _toBool(map['auth_register_captcha_enabled'], true);
      loginEnabled = _toBool(map['auth_login_captcha_enabled'], false);
      final p = _strVal(map['auth_captcha_provider']).toLowerCase();
      provider = p == 'geetest' ? 'geetest' : 'image';
      lenCtl.text = '${_toInt(map['auth_captcha_code_len'], 5)}';
      complexityCtl.text = _strVal(map['auth_captcha_code_complexity']).isEmpty ? 'alnum' : _strVal(map['auth_captcha_code_complexity']);
      geeIdCtl.text = _strVal(map['auth_geetest_captcha_id']);
      geeKeyCtl.text = _strVal(map['auth_geetest_captcha_key']);
      geeApiCtl.text = _strVal(map['auth_geetest_api_server']).isEmpty ? 'https://gcaptcha4.geetest.com' : _strVal(map['auth_geetest_api_server']);
      if (mounted) setState(() {});
    } finally {
      if (mounted) setState(() => _loading = false);
    }
  }

  Future<void> _save() async {
    final client = context.read<AppState>().apiClient;
    if (client == null) return;
    await _saveSettingsItems(client, [
      ('auth_register_captcha_enabled', '$registerEnabled'),
      ('auth_login_captcha_enabled', '$loginEnabled'),
      ('auth_captcha_provider', provider),
      ('auth_captcha_code_len', '${_toInt(lenCtl.text, 5)}'),
      ('auth_captcha_code_complexity', complexityCtl.text.trim()),
      ('auth_geetest_captcha_id', geeIdCtl.text.trim()),
      ('auth_geetest_captcha_key', geeKeyCtl.text.trim()),
      ('auth_geetest_api_server', geeApiCtl.text.trim()),
    ]);
    if (!mounted) return;
    ScaffoldMessenger.of(context).showSnackBar(const SnackBar(content: Text('验证码设置已保存')));
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('验证码设置'),
        actions: [
          IconButton(onPressed: _loading ? null : _load, icon: const Icon(Icons.refresh)),
          TextButton(onPressed: _loading ? null : _save, child: const Text('保存')),
        ],
      ),
      body: ListView(
        padding: const EdgeInsets.fromLTRB(12, 12, 12, 20),
        children: [
          SwitchListTile(value: registerEnabled, onChanged: (v) => setState(() => registerEnabled = v), title: const Text('注册启用验证码')),
          SwitchListTile(value: loginEnabled, onChanged: (v) => setState(() => loginEnabled = v), title: const Text('登录启用验证码')),
          const SizedBox(height: 8),
          DropdownButtonFormField<String>(
            value: provider,
            decoration: const InputDecoration(labelText: '验证码方案'),
            items: const [
              DropdownMenuItem(value: 'image', child: Text('图形验证码')),
              DropdownMenuItem(value: 'geetest', child: Text('极验 GeeTest')),
            ],
            onChanged: (v) => setState(() => provider = v ?? 'image'),
          ),
          const SizedBox(height: 8),
          Row(children: [
            Expanded(child: TextField(controller: lenCtl, keyboardType: TextInputType.number, decoration: const InputDecoration(labelText: '图形码长度'))),
            const SizedBox(width: 8),
            Expanded(child: TextField(controller: complexityCtl, decoration: const InputDecoration(labelText: '图形码复杂度'))),
          ]),
          const SizedBox(height: 8),
          TextField(controller: geeIdCtl, decoration: const InputDecoration(labelText: 'GeeTest Captcha ID')),
          const SizedBox(height: 8),
          TextField(controller: geeKeyCtl, decoration: const InputDecoration(labelText: 'GeeTest Captcha Key')),
          const SizedBox(height: 8),
          TextField(controller: geeApiCtl, decoration: const InputDecoration(labelText: 'GeeTest API Server')),
        ],
      ),
    );
  }
}

class LifecycleSettingsScreen extends StatefulWidget {
  const LifecycleSettingsScreen({super.key});

  @override
  State<LifecycleSettingsScreen> createState() => _LifecycleSettingsScreenState();
}

class _LifecycleSettingsScreenState extends State<LifecycleSettingsScreen> {
  bool _loading = false;
  bool emailExpireEnabled = false;
  bool autoDeleteEnabled = false;
  bool emergencyRenewEnabled = true;
  final expireReminderDaysCtl = TextEditingController(text: '7');
  final autoDeleteDaysCtl = TextEditingController(text: '7');
  final renewWindowDaysCtl = TextEditingController(text: '7');
  final renewDaysCtl = TextEditingController(text: '1');
  final renewIntervalHoursCtl = TextEditingController(text: '720');

  @override
  void dispose() {
    expireReminderDaysCtl.dispose();
    autoDeleteDaysCtl.dispose();
    renewWindowDaysCtl.dispose();
    renewDaysCtl.dispose();
    renewIntervalHoursCtl.dispose();
    super.dispose();
  }

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    _load();
  }

  Future<void> _load() async {
    final client = context.read<AppState>().apiClient;
    if (client == null) return;
    setState(() => _loading = true);
    try {
      final map = await _loadSettingsMap(client);
      emailExpireEnabled = _toBool(map['email_expire_enabled'], false);
      autoDeleteEnabled = _toBool(map['auto_delete_enabled'], false);
      emergencyRenewEnabled = _toBool(map['emergency_renew_enabled'], true);
      expireReminderDaysCtl.text = '${_toInt(map['expire_reminder_days'], 7)}';
      autoDeleteDaysCtl.text = '${_toInt(map['auto_delete_days'], 7)}';
      renewWindowDaysCtl.text = '${_toInt(map['emergency_renew_window_days'], 7)}';
      renewDaysCtl.text = '${_toInt(map['emergency_renew_days'], 1)}';
      renewIntervalHoursCtl.text = '${_toInt(map['emergency_renew_interval_hours'], 720)}';
      if (mounted) setState(() {});
    } finally {
      if (mounted) setState(() => _loading = false);
    }
  }

  Future<void> _save() async {
    final client = context.read<AppState>().apiClient;
    if (client == null) return;
    await _saveSettingsItems(client, [
      ('email_expire_enabled', '$emailExpireEnabled'),
      ('expire_reminder_days', '${_toInt(expireReminderDaysCtl.text, 7)}'),
      ('auto_delete_enabled', '$autoDeleteEnabled'),
      ('auto_delete_days', '${_toInt(autoDeleteDaysCtl.text, 7)}'),
      ('emergency_renew_enabled', '$emergencyRenewEnabled'),
      ('emergency_renew_window_days', '${_toInt(renewWindowDaysCtl.text, 7)}'),
      ('emergency_renew_days', '${_toInt(renewDaysCtl.text, 1)}'),
      ('emergency_renew_interval_hours', '${_toInt(renewIntervalHoursCtl.text, 720)}'),
    ]);
    if (!mounted) return;
    ScaffoldMessenger.of(context).showSnackBar(const SnackBar(content: Text('生命周期设置已保存')));
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('生命周期设置'),
        actions: [
          IconButton(onPressed: _loading ? null : _load, icon: const Icon(Icons.refresh)),
          TextButton(onPressed: _loading ? null : _save, child: const Text('保存')),
        ],
      ),
      body: ListView(
        padding: const EdgeInsets.fromLTRB(12, 12, 12, 20),
        children: [
          const Text('到期提醒', style: TextStyle(fontWeight: FontWeight.w700)),
          SwitchListTile(value: emailExpireEnabled, onChanged: (v) => setState(() => emailExpireEnabled = v), title: const Text('邮件提醒')),
          TextField(controller: expireReminderDaysCtl, keyboardType: TextInputType.number, decoration: const InputDecoration(labelText: '提前提醒天数')),
          const SizedBox(height: 12),
          const Text('VPS 到期删除策略', style: TextStyle(fontWeight: FontWeight.w700)),
          SwitchListTile(value: autoDeleteEnabled, onChanged: (v) => setState(() => autoDeleteEnabled = v), title: const Text('自动删除')),
          TextField(controller: autoDeleteDaysCtl, keyboardType: TextInputType.number, decoration: const InputDecoration(labelText: '到期后自动删除天数')),
          const SizedBox(height: 12),
          const Text('紧急续费', style: TextStyle(fontWeight: FontWeight.w700)),
          SwitchListTile(value: emergencyRenewEnabled, onChanged: (v) => setState(() => emergencyRenewEnabled = v), title: const Text('允许紧急续费')),
          TextField(controller: renewWindowDaysCtl, keyboardType: TextInputType.number, decoration: const InputDecoration(labelText: '续费窗口天数')),
          const SizedBox(height: 8),
          TextField(controller: renewDaysCtl, keyboardType: TextInputType.number, decoration: const InputDecoration(labelText: '每次续费延长天数')),
          const SizedBox(height: 8),
          TextField(controller: renewIntervalHoursCtl, keyboardType: TextInputType.number, decoration: const InputDecoration(labelText: '续费间隔(小时)')),
        ],
      ),
    );
  }
}

class AdminSettingsAdvancedScreen extends StatefulWidget {
  const AdminSettingsAdvancedScreen({super.key});

  @override
  State<AdminSettingsAdvancedScreen> createState() => _AdminSettingsAdvancedScreenState();
}

class _AdminSettingsAdvancedScreenState extends State<AdminSettingsAdvancedScreen> {
  bool _loading = false;
  List<Map<String, dynamic>> _admins = [];
  List<Map<String, dynamic>> _groups = [];
  String _keyword = '';

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    _load();
  }

  Future<void> _load() async {
    final client = context.read<AppState>().apiClient;
    if (client == null) return;
    setState(() => _loading = true);
    try {
      final res = await Future.wait([
        client.getJson('/admin/api/v1/admins', query: {'limit': '200', 'offset': '0', 'status': 'all'}),
        client.getJson('/admin/api/v1/permission-groups'),
      ]);
      if (!mounted) return;
      setState(() {
        _admins = (res[0]['items'] as List<dynamic>? ?? []).map(_asMap).toList();
        _groups = (res[1]['items'] as List<dynamic>? ?? []).map(_asMap).toList();
      });
    } finally {
      if (mounted) setState(() => _loading = false);
    }
  }

  String _groupName(dynamic id) {
    final sid = _asStr(id);
    for (final g in _groups) {
      if (_pickStr(g, const ['id', 'ID']) == sid) {
        final name = _pickStr(g, const ['name', 'Name']);
        return name.isEmpty ? '-' : name;
      }
    }
    return '-';
  }

  Future<void> _editAdmin({Map<String, dynamic>? row}) async {
    final id = row == null ? null : _pick(row, const ['id', 'ID']);
    final nameCtl = TextEditingController(text: row == null ? '' : _pickStr(row, const ['username', 'Username']));
    final emailCtl = TextEditingController(text: row == null ? '' : _pickStr(row, const ['email', 'Email']));
    final qqCtl = TextEditingController(text: row == null ? '' : _pickStr(row, const ['qq', 'QQ']));
    final pwdCtl = TextEditingController();
    String groupId = row == null ? '' : _pickStr(row, const ['permission_group_id', 'PermissionGroupID']);
    final ok = await showDialog<String>(
      context: context,
      builder: (context) => StatefulBuilder(
        builder: (context, setModal) => AlertDialog(
          title: Text(id == null ? '创建管理员' : '编辑管理员'),
          content: SingleChildScrollView(
            child: Column(
              mainAxisSize: MainAxisSize.min,
              children: [
                TextField(controller: nameCtl, decoration: const InputDecoration(labelText: '用户名')),
                const SizedBox(height: 8),
                TextField(controller: emailCtl, decoration: const InputDecoration(labelText: '邮箱')),
                const SizedBox(height: 8),
                TextField(controller: qqCtl, decoration: const InputDecoration(labelText: 'QQ')),
                if (id == null) ...[
                  const SizedBox(height: 8),
                  TextField(controller: pwdCtl, decoration: const InputDecoration(labelText: '密码')),
                ],
                const SizedBox(height: 8),
                DropdownButtonFormField<String>(
                  value: _groups.any((g) => _pickStr(g, const ['id', 'ID']) == groupId) ? groupId : null,
                  decoration: const InputDecoration(labelText: '权限组'),
                  items: _groups
                      .map((g) => DropdownMenuItem(
                            value: _pickStr(g, const ['id', 'ID']),
                            child: Text(_pickStr(g, const ['name', 'Name']).isEmpty ? _pickStr(g, const ['id', 'ID']) : _pickStr(g, const ['name', 'Name'])),
                          ))
                      .toList(),
                  onChanged: (v) => setModal(() => groupId = v ?? ''),
                ),
              ],
            ),
          ),
          actions: [
            if (id != null)
              TextButton(onPressed: () => Navigator.pop(context, 'delete'), child: const Text('删除')),
            TextButton(onPressed: () => Navigator.pop(context, 'cancel'), child: const Text('取消')),
            FilledButton(onPressed: () => Navigator.pop(context, 'save'), child: const Text('保存')),
          ],
        ),
      ),
    );
    final client = context.read<AppState>().apiClient;
    if (client == null) return;
    if (ok == 'delete' && id != null) {
      await client.deleteJson('/admin/api/v1/admins/$id');
      await _load();
      return;
    }
    if (ok != 'save') return;
    if (groupId.isEmpty) {
      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(const SnackBar(content: Text('请选择权限组')));
      return;
    }
    final payload = {
      'username': nameCtl.text.trim(),
      'email': emailCtl.text.trim(),
      'qq': qqCtl.text.trim(),
      'permission_group_id': _toInt(groupId, 0),
      if (id == null) 'password': pwdCtl.text.trim(),
    };
    if (id == null) {
      await client.postJson('/admin/api/v1/admins', body: payload);
    } else {
      await client.patchJson('/admin/api/v1/admins/$id', body: payload);
    }
    await _load();
  }

  Future<void> _toggleStatus(Map<String, dynamic> row) async {
    final client = context.read<AppState>().apiClient;
    if (client == null) return;
    final status = _pickStr(row, const ['status', 'Status']);
    final next = status == 'active' ? 'disabled' : 'active';
    final id = _pick(row, const ['id', 'ID']);
    if (id == null) return;
    final ok = await showDialog<bool>(
      context: context,
      builder: (context) => AlertDialog(
        title: Text(next == 'disabled' ? '确认禁用' : '确认启用'),
        content: Text('确定要${next == 'disabled' ? '禁用' : '启用'}管理员 ${_pickStr(row, const ['username', 'Username'])} 吗？'),
        actions: [
          TextButton(onPressed: () => Navigator.pop(context, false), child: const Text('取消')),
          FilledButton(onPressed: () => Navigator.pop(context, true), child: const Text('确认')),
        ],
      ),
    );
    if (ok != true) return;
    await client.patchJson('/admin/api/v1/admins/$id/status', body: {'status': next});
    await _load();
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('管理员列表'),
        actions: [
          IconButton(onPressed: _loading ? null : _load, icon: const Icon(Icons.refresh)),
          IconButton(onPressed: _loading ? null : () => _editAdmin(), icon: const Icon(Icons.add)),
        ],
      ),
      body: RefreshIndicator(
        onRefresh: _load,
        child: ListView(
          physics: const AlwaysScrollableScrollPhysics(),
          padding: const EdgeInsets.fromLTRB(12, 12, 12, 20),
          children: [
            TextField(
              onChanged: (v) => setState(() => _keyword = v),
              decoration: const InputDecoration(
                hintText: '搜索用户名/邮箱/QQ',
                prefixIcon: Icon(Icons.search, size: 18),
                isDense: true,
              ),
            ),
            const SizedBox(height: 10),
            if (_loading && _admins.isEmpty)
              const Padding(
                padding: EdgeInsets.only(top: 80),
                child: Center(child: CircularProgressIndicator()),
              )
            else if (_admins.isEmpty)
              const Padding(
                padding: EdgeInsets.only(top: 80),
                child: Center(child: Text('暂无管理员')),
              )
            else
              ..._admins.where((row) {
                final kw = _keyword.trim().toLowerCase();
                if (kw.isEmpty) return true;
                final hay = '${_pickStr(row, const ['username', 'Username'])} ${_pickStr(row, const ['email', 'Email'])} ${_pickStr(row, const ['qq', 'QQ'])}'.toLowerCase();
                return hay.contains(kw);
              }).map((row) {
                final active = _pickStr(row, const ['status', 'Status']) == 'active';
                final qq = _pickStr(row, const ['qq', 'QQ']);
                final avatar = qq.isEmpty ? '' : 'https://q1.qlogo.cn/g?b=qq&nk=$qq&s=100';
                return Container(
                  margin: const EdgeInsets.only(bottom: 8),
                  decoration: BoxDecoration(
                    color: Colors.white,
                    borderRadius: BorderRadius.circular(12),
                    border: Border.all(color: const Color(0xFFE5EAF2)),
                  ),
                  child: ListTile(
                    leading: CircleAvatar(
                      backgroundColor: const Color(0xFFEFF6FF),
                      foregroundImage: avatar.isEmpty ? null : NetworkImage(avatar),
                      child: avatar.isEmpty
                          ? Text(_pickStr(row, const ['username', 'Username']).isEmpty ? 'A' : _pickStr(row, const ['username', 'Username']).characters.first)
                          : null,
                    ),
                    title: Text(_pickStr(row, const ['username', 'Username'])),
                    subtitle: Text('邮箱:${_pickStr(row, const ['email', 'Email'])}\n权限组:${_groupName(_pick(row, const ['permission_group_id', 'PermissionGroupID']))}'),
                    isThreeLine: true,
                    trailing: Column(
                      mainAxisAlignment: MainAxisAlignment.center,
                      children: [
                        Container(
                          padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 2),
                          decoration: BoxDecoration(
                            color: active ? const Color(0xFFE8F5E9) : const Color(0xFFF1F5F9),
                            borderRadius: BorderRadius.circular(999),
                          ),
                          child: Text(active ? '启用' : '禁用', style: const TextStyle(fontSize: 11)),
                        ),
                        const SizedBox(height: 3),
                        GestureDetector(
                          onTap: () => _toggleStatus(row),
                          child: Text(active ? '禁用' : '启用', style: const TextStyle(fontSize: 12, color: Color(0xFF1E88E5))),
                        ),
                      ],
                    ),
                    onTap: () => _editAdmin(row: row),
                  ),
                );
              }),
          ],
        ),
      ),
    );
  }
}

class PermissionGroupSettingsAdvancedScreen extends StatefulWidget {
  const PermissionGroupSettingsAdvancedScreen({super.key});

  @override
  State<PermissionGroupSettingsAdvancedScreen> createState() => _PermissionGroupSettingsAdvancedScreenState();
}

class _PermissionGroupSettingsAdvancedScreenState extends State<PermissionGroupSettingsAdvancedScreen> {
  bool _loading = false;
  List<Map<String, dynamic>> _groups = [];
  List<Map<String, dynamic>> _permissions = [];
  String _keyword = '';

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    _load();
  }

  List<String> _permListOf(Map<String, dynamic> row) {
    final v = _pick(row, const ['permissions', 'Permissions', 'permissions_json', 'PermissionsJSON']);
    if (v is List) return v.map((e) => e.toString()).toList();
    final s = _asStr(v);
    try {
      final decoded = jsonDecode(s);
      if (decoded is List) {
        return decoded.map((e) => e.toString()).where((e) => e.trim().isNotEmpty).toList();
      }
    } catch (_) {}
    return [];
  }

  Future<void> _load() async {
    final client = context.read<AppState>().apiClient;
    if (client == null) return;
    setState(() => _loading = true);
    try {
      final res = await Future.wait([
        client.getJson('/admin/api/v1/permission-groups'),
        client.getJson('/admin/api/v1/permissions/list'),
      ]);
      if (!mounted) return;
      setState(() {
        _groups = (res[0]['items'] as List<dynamic>? ?? []).map(_asMap).toList();
        _permissions = (res[1]['items'] as List<dynamic>? ?? []).map(_asMap).toList();
      });
    } finally {
      if (mounted) setState(() => _loading = false);
    }
  }

  Future<void> _editGroup({Map<String, dynamic>? row}) async {
    final id = row == null ? null : _pick(row, const ['id', 'ID']);
    final nameCtl = TextEditingController(text: row == null ? '' : _pickStr(row, const ['name', 'Name']));
    final descCtl = TextEditingController(text: row == null ? '' : _pickStr(row, const ['description', 'Description']));
    final selected = _permListOf(row ?? {}).toSet();
    String permissionKeyword = '';
    final ok = await showDialog<String>(
      context: context,
      builder: (context) => StatefulBuilder(
        builder: (context, setModal) {
          final grouped = <String, List<Map<String, dynamic>>>{};
          final kw = permissionKeyword.trim().toLowerCase();
          for (final p in _permissions) {
            final code = _pickStr(p, const ['code', 'Code']);
            final friendly = _pickStr(p, const ['friendly_name', 'FriendlyName']).isNotEmpty
                ? _pickStr(p, const ['friendly_name', 'FriendlyName'])
                : (_pickStr(p, const ['name', 'Name']).isNotEmpty ? _pickStr(p, const ['name', 'Name']) : code);
            final cat = _pickStr(p, const ['category', 'Category']).isEmpty ? '其他' : _pickStr(p, const ['category', 'Category']);
            final hay = '$code $friendly $cat'.toLowerCase();
            if (kw.isNotEmpty && !hay.contains(kw)) continue;
            grouped.putIfAbsent(cat, () => []).add(p);
          }
          return AlertDialog(
            title: Text(id == null ? '创建权限组' : '编辑权限组'),
            content: SizedBox(
              width: 560,
              child: SingleChildScrollView(
                child: Column(
                  mainAxisSize: MainAxisSize.min,
                  children: [
                    TextField(controller: nameCtl, decoration: const InputDecoration(labelText: '名称')),
                    const SizedBox(height: 8),
                    TextField(controller: descCtl, maxLines: 2, decoration: const InputDecoration(labelText: '描述')),
                    const SizedBox(height: 8),
                    TextField(
                      onChanged: (v) => setModal(() => permissionKeyword = v),
                      decoration: const InputDecoration(
                        labelText: '搜索权限',
                        prefixIcon: Icon(Icons.search, size: 18),
                      ),
                    ),
                    const SizedBox(height: 8),
                    Row(
                      children: [
                        OutlinedButton(
                          onPressed: () => setModal(() {
                            selected
                              ..clear()
                              ..addAll(_permissions.map((e) => _pickStr(e, const ['code', 'Code'])).where((e) => e.isNotEmpty));
                          }),
                          child: const Text('全选'),
                        ),
                        const SizedBox(width: 8),
                        OutlinedButton(
                          onPressed: () => setModal(selected.clear),
                          child: const Text('清空'),
                        ),
                      ],
                    ),
                    const SizedBox(height: 8),
                    ...grouped.entries.map((entry) => ExpansionTile(
                          tilePadding: EdgeInsets.zero,
                          title: Text(entry.key),
                          children: entry.value.map((p) {
                            final code = _pickStr(p, const ['code', 'Code']);
                            final name = _pickStr(p, const ['friendly_name', 'FriendlyName']).isNotEmpty
                                ? _pickStr(p, const ['friendly_name', 'FriendlyName'])
                                : (_pickStr(p, const ['name', 'Name']).isNotEmpty ? _pickStr(p, const ['name', 'Name']) : code);
                            return CheckboxListTile(
                              dense: true,
                              contentPadding: EdgeInsets.zero,
                              value: selected.contains(code),
                              title: Text('$name (${_pickStr(p, const ['code', 'Code'])})'),
                              onChanged: (v) => setModal(() {
                                if (v == true) {
                                  selected.add(code);
                                } else {
                                  selected.remove(code);
                                }
                              }),
                            );
                          }).toList(),
                        )),
                  ],
                ),
              ),
            ),
            actions: [
              if (id != null)
                TextButton(onPressed: () => Navigator.pop(context, 'delete'), child: const Text('删除')),
              TextButton(onPressed: () => Navigator.pop(context, 'cancel'), child: const Text('取消')),
              FilledButton(onPressed: () => Navigator.pop(context, 'save'), child: const Text('保存')),
            ],
          );
        },
      ),
    );
    final client = context.read<AppState>().apiClient;
    if (client == null) return;
    if (ok == 'delete' && id != null) {
      await client.deleteJson('/admin/api/v1/permission-groups/$id');
      await _load();
      return;
    }
    if (ok != 'save') return;
    if (nameCtl.text.trim().isEmpty) {
      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(const SnackBar(content: Text('请输入权限组名称')));
      return;
    }
    if (selected.isEmpty) {
      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(const SnackBar(content: Text('请至少选择一个权限')));
      return;
    }
    final payload = {
      'name': nameCtl.text.trim(),
      'description': descCtl.text.trim(),
      'permissions': selected.toList(),
    };
    if (id == null) {
      await client.postJson('/admin/api/v1/permission-groups', body: payload);
    } else {
      await client.patchJson('/admin/api/v1/permission-groups/$id', body: payload);
    }
    await _load();
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('权限组设置'),
        actions: [
          IconButton(onPressed: _loading ? null : _load, icon: const Icon(Icons.refresh)),
          IconButton(onPressed: _loading ? null : () => _editGroup(), icon: const Icon(Icons.add)),
        ],
      ),
      body: RefreshIndicator(
        onRefresh: _load,
        child: ListView(
          physics: const AlwaysScrollableScrollPhysics(),
          padding: const EdgeInsets.fromLTRB(12, 12, 12, 20),
          children: [
            TextField(
              onChanged: (v) => setState(() => _keyword = v),
              decoration: const InputDecoration(
                hintText: '搜索权限组',
                prefixIcon: Icon(Icons.search, size: 18),
                isDense: true,
              ),
            ),
            const SizedBox(height: 10),
            if (_loading && _groups.isEmpty)
              const Padding(
                padding: EdgeInsets.only(top: 80),
                child: Center(child: CircularProgressIndicator()),
              )
            else if (_groups.isEmpty)
              const Padding(
                padding: EdgeInsets.only(top: 80),
                child: Center(child: Text('暂无权限组')),
              )
            else
              ..._groups.where((row) {
                final kw = _keyword.trim().toLowerCase();
                if (kw.isEmpty) return true;
                final hay = '${_pickStr(row, const ['name', 'Name'])} ${_pickStr(row, const ['description', 'Description'])}'.toLowerCase();
                return hay.contains(kw);
              }).map((row) {
                final count = _permListOf(row).length;
                return Container(
                  margin: const EdgeInsets.only(bottom: 8),
                  decoration: BoxDecoration(
                    color: Colors.white,
                    borderRadius: BorderRadius.circular(12),
                    border: Border.all(color: const Color(0xFFE5EAF2)),
                  ),
                  child: ListTile(
                    title: Text(_pickStr(row, const ['name', 'Name'])),
                    subtitle: Text('${_pickStr(row, const ['description', 'Description'])}\n权限 $count 项'),
                    isThreeLine: true,
                    onTap: () => _editGroup(row: row),
                  ),
                );
              }),
          ],
        ),
      ),
    );
  }
}

class UserTierSettingsAdvancedScreen extends StatefulWidget {
  const UserTierSettingsAdvancedScreen({super.key});

  @override
  State<UserTierSettingsAdvancedScreen> createState() => _UserTierSettingsAdvancedScreenState();
}

class _UserTierSettingsAdvancedScreenState extends State<UserTierSettingsAdvancedScreen> {
  bool _loading = false;
  List<Map<String, dynamic>> _groups = [];
  int? _selectedGroupId;
  List<Map<String, dynamic>> _discountRules = [];
  List<Map<String, dynamic>> _autoRules = [];
  List<Map<String, dynamic>> _goodsTypes = [];
  List<Map<String, dynamic>> _regions = [];
  List<Map<String, dynamic>> _planGroups = [];
  List<Map<String, dynamic>> _packages = [];

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    _load();
  }

  Future<void> _load() async {
    final client = context.read<AppState>().apiClient;
    if (client == null) return;
    setState(() => _loading = true);
    try {
      final res = await Future.wait([
        client.getJson('/admin/api/v1/user-tiers'),
        client.getJson('/admin/api/v1/goods-types'),
        client.getJson('/admin/api/v1/regions'),
        client.getJson('/admin/api/v1/plan-groups'),
        client.getJson('/admin/api/v1/packages'),
      ]);
      final rows = (res[0]['items'] as List<dynamic>? ?? []).map(_asMap).where((e) => e.isNotEmpty).toList();
      final goodsTypes = (res[1]['items'] as List<dynamic>? ?? []).map(_asMap).toList();
      final regions = (res[2]['items'] as List<dynamic>? ?? []).map(_asMap).toList();
      final planGroups = (res[3]['items'] as List<dynamic>? ?? []).map(_asMap).toList();
      final packages = (res[4]['items'] as List<dynamic>? ?? []).map(_asMap).toList();
      final selected = _selectedGroupId ?? (rows.isEmpty ? null : _pickInt(rows.first, const ['id', 'ID'], 0));
      if (!mounted) return;
      setState(() {
        _groups = rows;
        _goodsTypes = goodsTypes;
        _regions = regions;
        _planGroups = planGroups;
        _packages = packages;
        _selectedGroupId = selected == 0 ? null : selected;
      });
      await _loadRules();
    } finally {
      if (mounted) setState(() => _loading = false);
    }
  }

  Future<void> _loadRules() async {
    final gid = _selectedGroupId;
    final client = context.read<AppState>().apiClient;
    if (client == null || gid == null || gid <= 0) {
      if (mounted) {
        setState(() {
          _discountRules = [];
          _autoRules = [];
        });
      }
      return;
    }
    final res = await Future.wait([
      client.getJson('/admin/api/v1/user-tiers/$gid/discount-rules'),
      client.getJson('/admin/api/v1/user-tiers/$gid/auto-rules'),
    ]);
    if (!mounted) return;
    setState(() {
      _discountRules = (res[0]['items'] as List<dynamic>? ?? []).map(_asMap).toList();
      _autoRules = (res[1]['items'] as List<dynamic>? ?? []).map(_asMap).toList();
    });
  }

  Future<void> _rebuild({int? id}) async {
    final client = context.read<AppState>().apiClient;
    if (client == null) return;
    await client.postJson(id == null ? '/admin/api/v1/user-tiers/rebuild' : '/admin/api/v1/user-tiers/$id/rebuild');
    if (!mounted) return;
    ScaffoldMessenger.of(context).showSnackBar(
      SnackBar(content: Text(id == null ? '已触发全量缓存重建' : '已触发分组缓存重建')),
    );
  }

  Future<void> _editGroup({Map<String, dynamic>? row}) async {
    final id = row == null ? null : _pick(row, const ['id', 'ID']);
    final nameCtl = TextEditingController(text: row == null ? '' : _pickStr(row, const ['name', 'Name']));
    final colorCtl = TextEditingController(
      text: row == null
          ? '#1677ff'
          : (_pickStr(row, const ['color', 'Color']).isEmpty ? '#1677ff' : _pickStr(row, const ['color', 'Color'])),
    );
    String icon = row == null ? 'badge' : (_pickStr(row, const ['icon', 'Icon']).isEmpty ? 'badge' : _pickStr(row, const ['icon', 'Icon']));
    final priorityCtl = TextEditingController(text: '${row == null ? 10 : _pickInt(row, const ['priority', 'Priority'], 10)}');
    var autoApprove = row != null && _pickBool(row, const ['auto_approve_enabled', 'AutoApproveEnabled'], false);
    final isDefaultGroup = row != null && _pickBool(row, const ['is_default', 'IsDefault'], false);
    final ok = await showDialog<String>(
      context: context,
      builder: (context) => StatefulBuilder(
        builder: (context, setModal) => AlertDialog(
          title: Text(id == null ? '创建用户组' : '编辑用户组'),
          content: SingleChildScrollView(
            child: Column(
              mainAxisSize: MainAxisSize.min,
              children: [
                TextField(controller: nameCtl, decoration: const InputDecoration(labelText: '名称')),
                const SizedBox(height: 8),
                TextField(controller: colorCtl, decoration: const InputDecoration(labelText: '颜色(HEX)')),
                const SizedBox(height: 8),
                DropdownButtonFormField<String>(
                  value: icon,
                  decoration: const InputDecoration(labelText: '图标'),
                  items: const [
                    DropdownMenuItem(value: 'badge', child: Text('认证徽章')),
                    DropdownMenuItem(value: 'star', child: Text('星标')),
                    DropdownMenuItem(value: 'crown', child: Text('皇冠')),
                    DropdownMenuItem(value: 'rocket', child: Text('火箭')),
                    DropdownMenuItem(value: 'trophy', child: Text('奖杯')),
                    DropdownMenuItem(value: 'fire', child: Text('火焰')),
                    DropdownMenuItem(value: 'thunder', child: Text('闪电')),
                    DropdownMenuItem(value: 'gift', child: Text('礼物')),
                    DropdownMenuItem(value: 'heart', child: Text('爱心')),
                  ],
                  onChanged: (v) => setModal(() => icon = v ?? 'badge'),
                ),
                const SizedBox(height: 8),
                TextField(
                  controller: priorityCtl,
                  keyboardType: TextInputType.number,
                  decoration: const InputDecoration(labelText: '优先级'),
                ),
                SwitchListTile(
                  value: autoApprove,
                  onChanged: (v) => setModal(() => autoApprove = v),
                  title: const Text('自动审批开关'),
                ),
              ],
            ),
          ),
          actions: [
            if (id != null && !isDefaultGroup)
              TextButton(onPressed: () => Navigator.pop(context, 'delete'), child: const Text('删除')),
            TextButton(onPressed: () => Navigator.pop(context, 'cancel'), child: const Text('取消')),
            FilledButton(onPressed: () => Navigator.pop(context, 'save'), child: const Text('保存')),
          ],
        ),
      ),
    );
    final client = context.read<AppState>().apiClient;
    if (client == null) return;
    if (ok == 'delete' && id != null) {
      await client.deleteJson('/admin/api/v1/user-tiers/$id');
      await _load();
      return;
    }
    if (ok != 'save') return;
    final payload = {
      'name': nameCtl.text.trim(),
      'color': colorCtl.text.trim(),
      'icon': icon,
      'priority': _toInt(priorityCtl.text, 10),
      'auto_approve_enabled': autoApprove,
      if (id != null) 'is_default': isDefaultGroup,
    };
    if (id == null) {
      await client.postJson('/admin/api/v1/user-tiers', body: payload);
    } else {
      await client.patchJson('/admin/api/v1/user-tiers/$id', body: payload);
    }
    await _load();
  }

  List<DropdownMenuItem<int>> _idOptions(List<Map<String, dynamic>> items, {required String label}) {
    return items
        .map((e) {
          final id = _toInt(e['id'], 0);
          if (id <= 0) return null;
          final name = _asStr(e['name']).isEmpty ? '#$id' : _asStr(e['name']);
          return DropdownMenuItem<int>(value: id, child: Text('$label$name'));
        })
        .whereType<DropdownMenuItem<int>>()
        .toList();
  }

  bool _needGoodsType(String scope) => const ['goods_type', 'goods_type_region', 'plan_group', 'package', 'addon_config'].contains(scope);
  bool _needRegion(String scope) => const ['goods_type_region', 'plan_group'].contains(scope);
  bool _needPlanGroup(String scope) => const ['plan_group', 'package', 'addon_config'].contains(scope);
  bool _needPackage(String scope) => scope == 'package';
  String _normalizeScope(String scope) => scope == 'goods_type_region_plan_group' ? 'plan_group' : scope;

  Future<void> _editDiscountRule({Map<String, dynamic>? row}) async {
    final gid = _selectedGroupId;
    final client = context.read<AppState>().apiClient;
    if (client == null || gid == null || gid <= 0) return;
    final ruleId = row?['id'];
    final rowScope = row == null ? '' : _asStr(_pick(row, const ['scope', 'Scope']));
    String scope = _normalizeScope(rowScope.isEmpty ? 'all' : rowScope);
    int goodsTypeId = row == null ? 0 : _pickInt(row, const ['goods_type_id', 'GoodsTypeID'], 0);
    int regionId = row == null ? 0 : _pickInt(row, const ['region_id', 'RegionID'], 0);
    int planGroupId = row == null ? 0 : _pickInt(row, const ['plan_group_id', 'PlanGroupID'], 0);
    int packageId = row == null ? 0 : _pickInt(row, const ['package_id', 'PackageID'], 0);
    final discountCtl = TextEditingController(text: '${row == null ? 1000 : _pickInt(row, const ['discount_permille', 'DiscountPermille'], 1000)}');
    final fixedCtl = TextEditingController(text: row == null ? '' : _pickStr(row, const ['fixed_price', 'FixedPrice']));
    final coreCtl = TextEditingController(text: '${row == null ? 1000 : _pickInt(row, const ['add_core_permille', 'AddCorePermille'], 1000)}');
    final memCtl = TextEditingController(text: '${row == null ? 1000 : _pickInt(row, const ['add_mem_permille', 'AddMemPermille'], 1000)}');
    final diskCtl = TextEditingController(text: '${row == null ? 1000 : _pickInt(row, const ['add_disk_permille', 'AddDiskPermille'], 1000)}');
    final bwCtl = TextEditingController(text: '${row == null ? 1000 : _pickInt(row, const ['add_bw_permille', 'AddBwPermille'], 1000)}');
    final action = await showDialog<String>(
      context: context,
      builder: (context) => StatefulBuilder(
        builder: (context, setModal) => AlertDialog(
          title: Text(ruleId == null ? '新增优惠规则' : '编辑优惠规则'),
          content: SingleChildScrollView(
            child: Column(
              mainAxisSize: MainAxisSize.min,
              children: [
                DropdownButtonFormField<String>(
                  value: scope,
                  decoration: const InputDecoration(labelText: '对象范围'),
                  items: const [
                    DropdownMenuItem(value: 'all', child: Text('全部(不含附加项)')),
                    DropdownMenuItem(value: 'all_addons', child: Text('全部附加项')),
                    DropdownMenuItem(value: 'goods_type', child: Text('类型')),
                    DropdownMenuItem(value: 'goods_type_region', child: Text('类型-地区')),
                    DropdownMenuItem(value: 'plan_group', child: Text('类型-地区-线路')),
                    DropdownMenuItem(value: 'package', child: Text('套餐')),
                    DropdownMenuItem(value: 'addon_config', child: Text('附加项配置')),
                  ],
                  onChanged: (v) => setModal(() {
                    scope = v ?? 'all';
                    if (!_needGoodsType(scope)) goodsTypeId = 0;
                    if (!_needRegion(scope)) regionId = 0;
                    if (!_needPlanGroup(scope)) planGroupId = 0;
                    if (!_needPackage(scope)) packageId = 0;
                  }),
                ),
                if (_needGoodsType(scope)) ...[
                  const SizedBox(height: 8),
                  DropdownButtonFormField<int>(
                    value: goodsTypeId > 0 ? goodsTypeId : null,
                    decoration: const InputDecoration(labelText: '类型'),
                    items: _goodsTypes
                        .map((e) {
                          final id = _pickInt(e, const ['id', 'ID'], 0);
                          if (id <= 0) return null;
                          final name = _pickStr(e, const ['name', 'Name']);
                          return DropdownMenuItem<int>(value: id, child: Text(name.isEmpty ? '#$id' : name));
                        })
                        .whereType<DropdownMenuItem<int>>()
                        .toList(),
                    onChanged: (v) => setModal(() => goodsTypeId = v ?? 0),
                  ),
                ],
                if (_needRegion(scope)) ...[
                  const SizedBox(height: 8),
                  DropdownButtonFormField<int>(
                    value: regionId > 0 ? regionId : null,
                    decoration: const InputDecoration(labelText: '地区'),
                    items: _regions
                        .where((e) => goodsTypeId == 0 || _pickInt(e, const ['goods_type_id', 'GoodsTypeID'], 0) == goodsTypeId)
                        .map((e) {
                          final id = _pickInt(e, const ['id', 'ID'], 0);
                          if (id <= 0) return null;
                          final name = _pickStr(e, const ['name', 'Name']);
                          return DropdownMenuItem<int>(value: id, child: Text(name.isEmpty ? '#$id' : name));
                        })
                        .whereType<DropdownMenuItem<int>>()
                        .toList(),
                    onChanged: (v) => setModal(() => regionId = v ?? 0),
                  ),
                ],
                if (_needPlanGroup(scope)) ...[
                  const SizedBox(height: 8),
                  DropdownButtonFormField<int>(
                    value: planGroupId > 0 ? planGroupId : null,
                    decoration: const InputDecoration(labelText: '线路'),
                    items: _planGroups.where((e) {
                        final gt = _pickInt(e, const ['goods_type_id', 'GoodsTypeID'], 0);
                        final rg = _pickInt(e, const ['region_id', 'RegionID'], 0);
                        if (goodsTypeId > 0 && gt != goodsTypeId) return false;
                        if (regionId > 0 && rg != regionId) return false;
                        return true;
                      }).map((e) {
                        final id = _pickInt(e, const ['id', 'ID'], 0);
                        if (id <= 0) return null;
                        final name = _pickStr(e, const ['name', 'Name']);
                        return DropdownMenuItem<int>(value: id, child: Text(name.isEmpty ? '#$id' : name));
                      }).whereType<DropdownMenuItem<int>>().toList(),
                    onChanged: (v) => setModal(() => planGroupId = v ?? 0),
                  ),
                ],
                if (_needPackage(scope)) ...[
                  const SizedBox(height: 8),
                  DropdownButtonFormField<int>(
                    value: packageId > 0 ? packageId : null,
                    decoration: const InputDecoration(labelText: '套餐'),
                    items: _packages.where((e) {
                        final gt = _pickInt(e, const ['goods_type_id', 'GoodsTypeID'], 0);
                        final pg = _pickInt(e, const ['plan_group_id', 'PlanGroupID'], 0);
                        if (goodsTypeId > 0 && gt != goodsTypeId) return false;
                        if (planGroupId > 0 && pg != planGroupId) return false;
                        return true;
                      }).map((e) {
                        final id = _pickInt(e, const ['id', 'ID'], 0);
                        if (id <= 0) return null;
                        final name = _pickStr(e, const ['name', 'Name']);
                        return DropdownMenuItem<int>(value: id, child: Text(name.isEmpty ? '#$id' : name));
                      }).whereType<DropdownMenuItem<int>>().toList(),
                    onChanged: (v) => setModal(() => packageId = v ?? 0),
                  ),
                ],
                const SizedBox(height: 8),
                Row(
                  children: [
                    Expanded(child: TextField(controller: discountCtl, keyboardType: TextInputType.number, decoration: const InputDecoration(labelText: '折扣(‰)'))),
                    const SizedBox(width: 8),
                    Expanded(child: TextField(controller: fixedCtl, keyboardType: TextInputType.number, decoration: const InputDecoration(labelText: '固定价格(分)'))),
                  ],
                ),
                const SizedBox(height: 8),
                Row(
                  children: [
                    Expanded(child: TextField(controller: coreCtl, keyboardType: TextInputType.number, decoration: const InputDecoration(labelText: '核心附加(‰)'))),
                    const SizedBox(width: 8),
                    Expanded(child: TextField(controller: memCtl, keyboardType: TextInputType.number, decoration: const InputDecoration(labelText: '内存附加(‰)'))),
                  ],
                ),
                const SizedBox(height: 8),
                Row(
                  children: [
                    Expanded(child: TextField(controller: diskCtl, keyboardType: TextInputType.number, decoration: const InputDecoration(labelText: '磁盘附加(‰)'))),
                    const SizedBox(width: 8),
                    Expanded(child: TextField(controller: bwCtl, keyboardType: TextInputType.number, decoration: const InputDecoration(labelText: '带宽附加(‰)'))),
                  ],
                ),
              ],
            ),
          ),
          actions: [
            if (ruleId != null) TextButton(onPressed: () => Navigator.pop(context, 'delete'), child: const Text('删除')),
            TextButton(onPressed: () => Navigator.pop(context, 'cancel'), child: const Text('取消')),
            FilledButton(onPressed: () => Navigator.pop(context, 'save'), child: const Text('保存')),
          ],
        ),
      ),
    );
    if (action == 'delete' && ruleId != null) {
      await client.deleteJson('/admin/api/v1/user-tiers/$gid/discount-rules/$ruleId');
      await _loadRules();
      return;
    }
    if (action != 'save') return;
    if (_needGoodsType(scope) && goodsTypeId <= 0) return;
    if (_needRegion(scope) && regionId <= 0) return;
    if (_needPlanGroup(scope) && planGroupId <= 0) return;
    if (_needPackage(scope) && packageId <= 0) return;
    final payload = {
      'scope': _normalizeScope(scope),
      'goods_type_id': _needGoodsType(scope) ? goodsTypeId : 0,
      'region_id': _needRegion(scope) ? regionId : 0,
      'plan_group_id': _needPlanGroup(scope) ? planGroupId : 0,
      'package_id': _needPackage(scope) ? packageId : 0,
      'discount_permille': _toInt(discountCtl.text, 1000),
      'fixed_price': _toInt(fixedCtl.text, 0),
      'add_core_permille': _toInt(coreCtl.text, 1000),
      'add_mem_permille': _toInt(memCtl.text, 1000),
      'add_disk_permille': _toInt(diskCtl.text, 1000),
      'add_bw_permille': _toInt(bwCtl.text, 1000),
    };
    if (ruleId == null) {
      await client.postJson('/admin/api/v1/user-tiers/$gid/discount-rules', body: payload);
    } else {
      await client.patchJson('/admin/api/v1/user-tiers/$gid/discount-rules/$ruleId', body: payload);
    }
    await _loadRules();
  }

  List<Map<String, dynamic>> _parseConditions(dynamic raw) {
    final out = <Map<String, dynamic>>[];
    if (raw is List) {
      for (final e in raw) {
        out.add(_asMap(e));
      }
      return out;
    }
    final s = _asStr(raw);
    if (s.isEmpty) return out;
    try {
      final decoded = jsonDecode(s);
      if (decoded is List) {
        for (final e in decoded) {
          out.add(_asMap(e));
        }
      }
    } catch (_) {}
    return out;
  }

  String _conditionsLabel(dynamic raw) {
    final cond = _parseConditions(raw);
    if (cond.isEmpty) return '任意';
    return cond.map((e) => '${_asStr(e['metric'])} ${_asStr(e['operator'])} ${_asStr(e['value'])}').join(' AND ');
  }

  Future<void> _editAutoRule({Map<String, dynamic>? row}) async {
    final gid = _selectedGroupId;
    final client = context.read<AppState>().apiClient;
    if (client == null || gid == null || gid <= 0) return;
    final ruleId = row == null ? null : _pick(row, const ['id', 'ID']);
    final durationCtl = TextEditingController(text: '${row == null ? -1 : _pickInt(row, const ['duration_days', 'DurationDays'], -1)}');
    final sortCtl = TextEditingController(text: '${row == null ? 10 : _pickInt(row, const ['sort_order', 'SortOrder'], 10)}');
    final cond = row == null ? <Map<String, dynamic>>[] : _parseConditions(_pick(row, const ['conditions_json', 'ConditionsJSON']));
    final conditions = cond.isEmpty ? <Map<String, dynamic>>[{'metric': 'register_months', 'operator': 'gt', 'value': 0}] : cond;
    final action = await showDialog<String>(
      context: context,
      builder: (context) => StatefulBuilder(
        builder: (context, setModal) => AlertDialog(
          title: Text(ruleId == null ? '新增自动审批条件' : '编辑自动审批条件'),
          content: SizedBox(
            width: 520,
            child: SingleChildScrollView(
              child: Column(
                mainAxisSize: MainAxisSize.min,
                children: [
                  Row(
                    children: [
                      Expanded(child: TextField(controller: durationCtl, keyboardType: TextInputType.number, decoration: const InputDecoration(labelText: '时长(天,-1无限)'))),
                      const SizedBox(width: 8),
                      Expanded(child: TextField(controller: sortCtl, keyboardType: TextInputType.number, decoration: const InputDecoration(labelText: '排序'))),
                    ],
                  ),
                  const SizedBox(height: 10),
                  ...List.generate(conditions.length, (idx) {
                    final item = conditions[idx];
                    return Padding(
                      padding: const EdgeInsets.only(bottom: 8),
                      child: Row(
                        children: [
                          Expanded(
                            child: DropdownButtonFormField<String>(
                              value: _asStr(item['metric']).isEmpty ? 'register_months' : _asStr(item['metric']),
                              decoration: const InputDecoration(labelText: '条件'),
                              items: const [
                                DropdownMenuItem(value: 'register_months', child: Text('注册时长(月)')),
                                DropdownMenuItem(value: 'wallet_balance', child: Text('钱包余额(元)')),
                              ],
                              onChanged: (v) => setModal(() => item['metric'] = v ?? 'register_months'),
                            ),
                          ),
                          const SizedBox(width: 8),
                          SizedBox(
                            width: 110,
                            child: DropdownButtonFormField<String>(
                              value: _asStr(item['operator']).isEmpty ? 'gt' : _asStr(item['operator']),
                              decoration: const InputDecoration(labelText: '算符'),
                              items: const [
                                DropdownMenuItem(value: 'gt', child: Text('大于')),
                                DropdownMenuItem(value: 'lt', child: Text('小于')),
                                DropdownMenuItem(value: 'eq', child: Text('等于')),
                              ],
                              onChanged: (v) => setModal(() => item['operator'] = v ?? 'gt'),
                            ),
                          ),
                          const SizedBox(width: 8),
                          SizedBox(
                            width: 90,
                            child: TextField(
                              controller: TextEditingController(text: '${_toInt(item['value'], 0)}'),
                              keyboardType: TextInputType.number,
                              decoration: const InputDecoration(labelText: '目标'),
                              onChanged: (v) => item['value'] = _toInt(v, 0),
                            ),
                          ),
                          IconButton(
                            onPressed: () => setModal(() {
                              if (conditions.length > 1) conditions.removeAt(idx);
                            }),
                            icon: const Icon(Icons.delete_outline),
                          ),
                        ],
                      ),
                    );
                  }),
                  Align(
                    alignment: Alignment.centerLeft,
                    child: OutlinedButton.icon(
                      onPressed: () => setModal(() => conditions.add({'metric': 'register_months', 'operator': 'gt', 'value': 0})),
                      icon: const Icon(Icons.add, size: 16),
                      label: const Text('添加条件'),
                    ),
                  ),
                ],
              ),
            ),
          ),
          actions: [
            if (ruleId != null) TextButton(onPressed: () => Navigator.pop(context, 'delete'), child: const Text('删除')),
            TextButton(onPressed: () => Navigator.pop(context, 'cancel'), child: const Text('取消')),
            FilledButton(onPressed: () => Navigator.pop(context, 'save'), child: const Text('保存')),
          ],
        ),
      ),
    );
    if (action == 'delete' && ruleId != null) {
      await client.deleteJson('/admin/api/v1/user-tiers/$gid/auto-rules/$ruleId');
      await _loadRules();
      return;
    }
    if (action != 'save') return;
    final payload = {
      'duration_days': _toInt(durationCtl.text, -1),
      'sort_order': _toInt(sortCtl.text, 10),
      'conditions_json': jsonEncode(conditions),
    };
    if (ruleId == null) {
      await client.postJson('/admin/api/v1/user-tiers/$gid/auto-rules', body: payload);
    } else {
      await client.patchJson('/admin/api/v1/user-tiers/$gid/auto-rules/$ruleId', body: payload);
    }
    await _loadRules();
  }

  @override
  Widget build(BuildContext context) {
    final selected = _groups.where((e) => _pickInt(e, const ['id', 'ID'], 0) == _selectedGroupId).toList();
    final selectedGroup = selected.isEmpty ? null : selected.first;
    return Scaffold(
      appBar: AppBar(
        title: const Text('用户等级设置'),
        actions: [
          IconButton(onPressed: _loading ? null : _load, icon: const Icon(Icons.refresh)),
          IconButton(onPressed: _loading ? null : () => _rebuild(), icon: const Icon(Icons.build_outlined)),
          IconButton(onPressed: _loading ? null : () => _editGroup(), icon: const Icon(Icons.add)),
        ],
      ),
      body: RefreshIndicator(
        onRefresh: _load,
        child: ListView(
          physics: const AlwaysScrollableScrollPhysics(),
          padding: const EdgeInsets.fromLTRB(12, 12, 12, 20),
          children: [
            if (_loading && _groups.isEmpty)
              const Padding(
                padding: EdgeInsets.only(top: 80),
                child: Center(child: CircularProgressIndicator()),
              )
            else ...[
              const Text('用户组', style: TextStyle(fontWeight: FontWeight.w700)),
              const SizedBox(height: 8),
              ..._groups.map((row) {
                final gid = _pickInt(row, const ['id', 'ID'], 0);
                final isSelected = gid == _selectedGroupId;
                return Container(
                  margin: const EdgeInsets.only(bottom: 8),
                  decoration: BoxDecoration(
                    color: isSelected ? const Color(0xFFEFF6FF) : Colors.white,
                    borderRadius: BorderRadius.circular(12),
                    border: Border.all(color: const Color(0xFFE5EAF2)),
                  ),
                  child: ListTile(
                    title: Text(_pickStr(row, const ['name', 'Name'])),
                    subtitle: Text('图标:${_pickStr(row, const ['icon', 'Icon'])}  优先级:${_pickInt(row, const ['priority', 'Priority'], 0)}'),
                    trailing: Wrap(
                      spacing: 4,
                      children: [
                        if (row['is_default'] == true || row['IsDefault'] == true)
                          const Chip(label: Text('默认组'), visualDensity: VisualDensity.compact),
                        IconButton(icon: const Icon(Icons.sync), onPressed: () => _rebuild(id: gid)),
                      ],
                    ),
                    onTap: () async {
                      setState(() => _selectedGroupId = gid);
                      await _loadRules();
                    },
                    onLongPress: () => _editGroup(row: row),
                  ),
                );
              }),
              const SizedBox(height: 10),
              if (selectedGroup != null) ...[
                Row(
                  children: [
                  Expanded(child: Text('优惠规则 - ${_pickStr(selectedGroup, const ['name', 'Name'])}', style: const TextStyle(fontWeight: FontWeight.w700))),
                    FilledButton(
                      onPressed: (selectedGroup['is_default'] == true || selectedGroup['IsDefault'] == true)
                          ? null
                          : () => _editDiscountRule(),
                      child: const Text('新增'),
                    ),
                  ],
                ),
                const SizedBox(height: 8),
                if (_discountRules.isEmpty)
                  const Text('暂无优惠规则')
                else
                  ..._discountRules.map((row) => Container(
                        margin: const EdgeInsets.only(bottom: 8),
                        decoration: BoxDecoration(
                          color: Colors.white,
                          borderRadius: BorderRadius.circular(12),
                          border: Border.all(color: const Color(0xFFE5EAF2)),
                        ),
                        child: ListTile(
                          title: Text(_asStr(row['scope'])),
                          subtitle: Text('折扣:${_pickInt(row, const ['discount_permille', 'DiscountPermille'], 0) / 10}%  GT:${_pickInt(row, const ['goods_type_id', 'GoodsTypeID'], 0)} RG:${_pickInt(row, const ['region_id', 'RegionID'], 0)} PG:${_pickInt(row, const ['plan_group_id', 'PlanGroupID'], 0)} PKG:${_pickInt(row, const ['package_id', 'PackageID'], 0)}'),
                          onTap: (selectedGroup['is_default'] == true || selectedGroup['IsDefault'] == true)
                              ? null
                              : () => _editDiscountRule(row: row),
                        ),
                      )),
                const SizedBox(height: 10),
                Row(
                  children: [
                    Expanded(child: Text('自动审批规则 - ${_pickStr(selectedGroup, const ['name', 'Name'])}', style: const TextStyle(fontWeight: FontWeight.w700))),
                    FilledButton(
                      onPressed: (selectedGroup['is_default'] == true || selectedGroup['IsDefault'] == true)
                          ? null
                          : () => _editAutoRule(),
                      child: const Text('新增'),
                    ),
                  ],
                ),
                const SizedBox(height: 8),
                if (_autoRules.isEmpty)
                  const Text('暂无自动审批规则')
                else
                  ..._autoRules.map((row) => Container(
                        margin: const EdgeInsets.only(bottom: 8),
                        decoration: BoxDecoration(
                          color: Colors.white,
                          borderRadius: BorderRadius.circular(12),
                          border: Border.all(color: const Color(0xFFE5EAF2)),
                        ),
                        child: ListTile(
                          title: Text('排序:${_pickInt(row, const ['sort_order', 'SortOrder'], 0)}  时长:${_pickInt(row, const ['duration_days', 'DurationDays'], -1)}天'),
                          subtitle: Text(_conditionsLabel(_pick(row, const ['conditions_json', 'ConditionsJSON']))),
                          onTap: (selectedGroup['is_default'] == true || selectedGroup['IsDefault'] == true)
                              ? null
                              : () => _editAutoRule(row: row),
                        ),
                      )),
              ],
            ],
          ],
        ),
      ),
    );
  }
}

class CouponSettingsAdvancedScreen extends StatefulWidget {
  const CouponSettingsAdvancedScreen({super.key});

  @override
  State<CouponSettingsAdvancedScreen> createState() => _CouponSettingsAdvancedScreenState();
}

class _CouponSettingsAdvancedScreenState extends State<CouponSettingsAdvancedScreen> {
  bool _loading = false;
  List<Map<String, dynamic>> _groups = [];
  List<Map<String, dynamic>> _coupons = [];
  List<Map<String, dynamic>> _goodsTypes = [];
  List<Map<String, dynamic>> _regions = [];
  List<Map<String, dynamic>> _planGroups = [];
  List<Map<String, dynamic>> _packages = [];

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    _load();
  }

  Future<void> _load() async {
    final client = context.read<AppState>().apiClient;
    if (client == null) return;
    setState(() => _loading = true);
    try {
      final res = await Future.wait([
        client.getJson('/admin/api/v1/coupon-groups'),
        client.getJson('/admin/api/v1/coupons'),
        client.getJson('/admin/api/v1/goods-types'),
        client.getJson('/admin/api/v1/regions'),
        client.getJson('/admin/api/v1/plan-groups'),
        client.getJson('/admin/api/v1/packages'),
      ]);
      if (!mounted) return;
      setState(() {
        _groups = (res[0]['items'] as List<dynamic>? ?? []).map(_asMap).toList();
        _coupons = (res[1]['items'] as List<dynamic>? ?? []).map(_asMap).toList();
        _goodsTypes = (res[2]['items'] as List<dynamic>? ?? []).map(_asMap).toList();
        _regions = (res[3]['items'] as List<dynamic>? ?? []).map(_asMap).toList();
        _planGroups = (res[4]['items'] as List<dynamic>? ?? []).map(_asMap).toList();
        _packages = (res[5]['items'] as List<dynamic>? ?? []).map(_asMap).toList();
      });
    } finally {
      if (mounted) setState(() => _loading = false);
    }
  }

  String _groupName(dynamic id) {
    final sid = _asStr(id);
    for (final row in _groups) {
      if (_pickStr(row, const ['id', 'ID']) == sid) {
        return '${_pickStr(row, const ['name', 'Name'])}(#${_pickStr(row, const ['id', 'ID'])})';
      }
    }
    return '-';
  }

  Map<String, dynamic> _emptyRule() => {
        'scope': 'all',
        'goods_type_id': 0,
        'region_id': 0,
        'plan_group_id': 0,
        'package_id': 0,
        'addon_core_enabled': false,
        'addon_mem_enabled': false,
        'addon_disk_enabled': false,
        'addon_bw_enabled': false,
      };

  bool _needGoodsType(String scope) => const ['goods_type', 'goods_type_region', 'plan_group', 'package', 'addon_config'].contains(scope);
  bool _needRegion(String scope) => const ['goods_type_region', 'plan_group'].contains(scope);
  bool _needPlanGroup(String scope) => const ['plan_group', 'package', 'addon_config'].contains(scope);
  bool _needPackage(String scope) => scope == 'package';

  List<Map<String, dynamic>> _rulesOf(dynamic raw) {
    if (raw is List) return raw.map(_asMap).toList();
    final text = _asStr(raw);
    if (text.isEmpty) return [_emptyRule()];
    try {
      final decoded = jsonDecode(text);
      if (decoded is List) return decoded.map(_asMap).toList();
    } catch (_) {}
    return [_emptyRule()];
  }

  String _normalizeCouponScope(String scope) => scope == 'goods_type_region_plan_group' ? 'plan_group' : scope;

  Future<void> _editGroup({Map<String, dynamic>? row}) async {
    final id = row == null ? null : _pick(row, const ['id', 'ID']);
    final nameCtl = TextEditingController(text: row == null ? '' : _pickStr(row, const ['name', 'Name']));
    final rules = _rulesOf(row == null ? null : _pick(row, const ['rules', 'Rules']));
    final ok = await showDialog<String>(
      context: context,
      builder: (context) => StatefulBuilder(
        builder: (context, setModal) => AlertDialog(
          title: Text(id == null ? '新增商品组' : '编辑商品组'),
          content: SizedBox(
            width: 560,
            child: SingleChildScrollView(
              child: Column(
                mainAxisSize: MainAxisSize.min,
                children: [
                  TextField(controller: nameCtl, decoration: const InputDecoration(labelText: '名称')),
                  const SizedBox(height: 8),
                  Align(
                    alignment: Alignment.centerRight,
                    child: OutlinedButton.icon(
                      onPressed: () => setModal(() => rules.add(_emptyRule())),
                      icon: const Icon(Icons.add, size: 16),
                      label: const Text('新增策略'),
                    ),
                  ),
                  ...List.generate(rules.length, (idx) {
                    final rule = rules[idx];
                    var scope = _normalizeCouponScope(_asStr(rule['scope']).isEmpty ? 'all' : _asStr(rule['scope']));
                    int goodsTypeId = _toInt(rule['goods_type_id'], 0);
                    int regionId = _toInt(rule['region_id'], 0);
                    int planGroupId = _toInt(rule['plan_group_id'], 0);
                    int packageId = _toInt(rule['package_id'], 0);
                    bool addonCore = rule['addon_core_enabled'] == true;
                    bool addonMem = rule['addon_mem_enabled'] == true;
                    bool addonDisk = rule['addon_disk_enabled'] == true;
                    bool addonBw = rule['addon_bw_enabled'] == true;
                    return Container(
                      margin: const EdgeInsets.only(bottom: 10),
                      padding: const EdgeInsets.all(10),
                      decoration: BoxDecoration(
                        border: Border.all(color: const Color(0xFFE2E8F0)),
                        borderRadius: BorderRadius.circular(10),
                      ),
                      child: Column(
                        children: [
                          Row(
                            children: [
                              Text('策略 ${idx + 1}', style: const TextStyle(fontWeight: FontWeight.w700)),
                              const Spacer(),
                              IconButton(
                                onPressed: () => setModal(() {
                                  if (rules.length > 1) rules.removeAt(idx);
                                }),
                                icon: const Icon(Icons.delete_outline),
                              ),
                            ],
                          ),
                          DropdownButtonFormField<String>(
                            value: scope,
                            decoration: const InputDecoration(labelText: '范围'),
                            items: const [
                              DropdownMenuItem(value: 'all', child: Text('全部(不含附加)')),
                              DropdownMenuItem(value: 'all_addons', child: Text('全部附加项')),
                              DropdownMenuItem(value: 'goods_type', child: Text('类型')),
                              DropdownMenuItem(value: 'goods_type_region', child: Text('类型-地区')),
                              DropdownMenuItem(value: 'plan_group', child: Text('类型-地区-线路')),
                              DropdownMenuItem(value: 'package', child: Text('类型-地区-线路-套餐')),
                              DropdownMenuItem(value: 'addon_config', child: Text('附加项配置')),
                            ],
                            onChanged: (v) => setModal(() {
                              scope = v ?? 'all';
                              if (!_needGoodsType(scope)) goodsTypeId = 0;
                              if (!_needRegion(scope)) regionId = 0;
                              if (!_needPlanGroup(scope)) planGroupId = 0;
                              if (!_needPackage(scope)) packageId = 0;
                              rule['scope'] = scope;
                              rule['goods_type_id'] = goodsTypeId;
                              rule['region_id'] = regionId;
                              rule['plan_group_id'] = planGroupId;
                              rule['package_id'] = packageId;
                            }),
                          ),
                          if (_needGoodsType(scope)) ...[
                            const SizedBox(height: 8),
                            DropdownButtonFormField<int>(
                              value: goodsTypeId > 0 ? goodsTypeId : null,
                              decoration: const InputDecoration(labelText: '类型'),
                              items: _goodsTypes
                                  .map((e) {
                                    final id = _pickInt(e, const ['id', 'ID'], 0);
                                    if (id <= 0) return null;
                                    final name = _pickStr(e, const ['name', 'Name']);
                                    return DropdownMenuItem(value: id, child: Text(name.isEmpty ? '#$id' : name));
                                  })
                                  .whereType<DropdownMenuItem<int>>()
                                  .toList(),
                              onChanged: (v) => setModal(() {
                                goodsTypeId = v ?? 0;
                                rule['goods_type_id'] = goodsTypeId;
                              }),
                            ),
                          ],
                          if (_needRegion(scope)) ...[
                            const SizedBox(height: 8),
                            DropdownButtonFormField<int>(
                              value: regionId > 0 ? regionId : null,
                              decoration: const InputDecoration(labelText: '地区'),
                              items: _regions
                                  .where((e) => goodsTypeId == 0 || _pickInt(e, const ['goods_type_id', 'GoodsTypeID'], 0) == goodsTypeId)
                                  .map((e) {
                                    final id = _pickInt(e, const ['id', 'ID'], 0);
                                    if (id <= 0) return null;
                                    final name = _pickStr(e, const ['name', 'Name']);
                                    return DropdownMenuItem(value: id, child: Text(name.isEmpty ? '#$id' : name));
                                  })
                                  .whereType<DropdownMenuItem<int>>()
                                  .toList(),
                              onChanged: (v) => setModal(() {
                                regionId = v ?? 0;
                                rule['region_id'] = regionId;
                              }),
                            ),
                          ],
                          if (_needPlanGroup(scope)) ...[
                            const SizedBox(height: 8),
                            DropdownButtonFormField<int>(
                              value: planGroupId > 0 ? planGroupId : null,
                              decoration: const InputDecoration(labelText: '线路'),
                              items: _planGroups
                                  .where((e) => goodsTypeId == 0 || _pickInt(e, const ['goods_type_id', 'GoodsTypeID'], 0) == goodsTypeId)
                                  .where((e) => regionId == 0 || _pickInt(e, const ['region_id', 'RegionID'], 0) == regionId)
                                  .map((e) {
                                    final id = _pickInt(e, const ['id', 'ID'], 0);
                                    if (id <= 0) return null;
                                    final name = _pickStr(e, const ['name', 'Name']);
                                    return DropdownMenuItem(value: id, child: Text(name.isEmpty ? '#$id' : name));
                                  })
                                  .whereType<DropdownMenuItem<int>>()
                                  .toList(),
                              onChanged: (v) => setModal(() {
                                planGroupId = v ?? 0;
                                rule['plan_group_id'] = planGroupId;
                              }),
                            ),
                          ],
                          if (_needPackage(scope)) ...[
                            const SizedBox(height: 8),
                            DropdownButtonFormField<int>(
                              value: packageId > 0 ? packageId : null,
                              decoration: const InputDecoration(labelText: '套餐'),
                              items: _packages
                                  .where((e) => planGroupId == 0 || _pickInt(e, const ['plan_group_id', 'PlanGroupID'], 0) == planGroupId)
                                  .map((e) {
                                    final id = _pickInt(e, const ['id', 'ID'], 0);
                                    if (id <= 0) return null;
                                    final name = _pickStr(e, const ['name', 'Name']);
                                    return DropdownMenuItem(value: id, child: Text(name.isEmpty ? '#$id' : name));
                                  })
                                  .whereType<DropdownMenuItem<int>>()
                                  .toList(),
                              onChanged: (v) => setModal(() {
                                packageId = v ?? 0;
                                rule['package_id'] = packageId;
                              }),
                            ),
                          ],
                          if (scope == 'addon_config') ...[
                            const SizedBox(height: 8),
                            Row(
                              children: [
                                Expanded(
                                  child: CheckboxListTile(
                                    dense: true,
                                    value: addonCore,
                                    onChanged: (v) => setModal(() {
                                      addonCore = v == true;
                                      rule['addon_core_enabled'] = addonCore;
                                    }),
                                    title: const Text('附加CPU'),
                                  ),
                                ),
                                Expanded(
                                  child: CheckboxListTile(
                                    dense: true,
                                    value: addonMem,
                                    onChanged: (v) => setModal(() {
                                      addonMem = v == true;
                                      rule['addon_mem_enabled'] = addonMem;
                                    }),
                                    title: const Text('附加内存'),
                                  ),
                                ),
                              ],
                            ),
                            Row(
                              children: [
                                Expanded(
                                  child: CheckboxListTile(
                                    dense: true,
                                    value: addonDisk,
                                    onChanged: (v) => setModal(() {
                                      addonDisk = v == true;
                                      rule['addon_disk_enabled'] = addonDisk;
                                    }),
                                    title: const Text('附加磁盘'),
                                  ),
                                ),
                                Expanded(
                                  child: CheckboxListTile(
                                    dense: true,
                                    value: addonBw,
                                    onChanged: (v) => setModal(() {
                                      addonBw = v == true;
                                      rule['addon_bw_enabled'] = addonBw;
                                    }),
                                    title: const Text('附加带宽'),
                                  ),
                                ),
                              ],
                            ),
                          ],
                        ],
                      ),
                    );
                  }),
                ],
              ),
            ),
          ),
          actions: [
            if (id != null)
              TextButton(onPressed: () => Navigator.pop(context, 'delete'), child: const Text('删除')),
            TextButton(onPressed: () => Navigator.pop(context, 'cancel'), child: const Text('取消')),
            FilledButton(onPressed: () => Navigator.pop(context, 'save'), child: const Text('保存')),
          ],
        ),
      ),
    );
    final client = context.read<AppState>().apiClient;
    if (client == null) return;
    if (ok == 'delete' && id != null) {
      await client.deleteJson('/admin/api/v1/coupon-groups/$id');
      await _load();
      return;
    }
    if (ok != 'save') return;
    final payload = {
      'name': nameCtl.text.trim(),
      'rules': rules.map((e) {
        final scope = _normalizeCouponScope(_asStr(e['scope']).isEmpty ? 'all' : _asStr(e['scope']));
        return {
          'scope': scope,
          'goods_type_id': _needGoodsType(scope) ? _toInt(e['goods_type_id'], 0) : 0,
          'region_id': _needRegion(scope) ? _toInt(e['region_id'], 0) : 0,
          'plan_group_id': _needPlanGroup(scope) ? _toInt(e['plan_group_id'], 0) : 0,
          'package_id': _needPackage(scope) ? _toInt(e['package_id'], 0) : 0,
          'addon_core_enabled': scope == 'addon_config' ? e['addon_core_enabled'] == true : false,
          'addon_mem_enabled': scope == 'addon_config' ? e['addon_mem_enabled'] == true : false,
          'addon_disk_enabled': scope == 'addon_config' ? e['addon_disk_enabled'] == true : false,
          'addon_bw_enabled': scope == 'addon_config' ? e['addon_bw_enabled'] == true : false,
        };
      }).toList(),
    };
    if (id == null) {
      await client.postJson('/admin/api/v1/coupon-groups', body: payload);
    } else {
      await client.patchJson('/admin/api/v1/coupon-groups/$id', body: payload);
    }
    await _load();
  }

  Future<void> _editCoupon({Map<String, dynamic>? row}) async {
    final id = row == null ? null : _pick(row, const ['id', 'ID']);
    final codeCtl = TextEditingController(text: row == null ? '' : _pickStr(row, const ['code', 'Code']));
    final discountCtl = TextEditingController(text: '${row == null ? 900 : _pickInt(row, const ['discount_permille', 'DiscountPermille'], 900)}');
    final totalCtl = TextEditingController(text: '${row == null ? -1 : _pickInt(row, const ['total_limit', 'TotalLimit'], -1)}');
    final perCtl = TextEditingController(text: '${row == null ? -1 : _pickInt(row, const ['per_user_limit', 'PerUserLimit'], -1)}');
    String groupId = row == null ? '' : _asStr(_pick(row, const ['product_group_id', 'ProductGroupID']));
    bool newOnly = row != null && _pickBool(row, const ['new_user_only', 'NewUserOnly'], false);
    bool active = row == null ? true : _pickBool(row, const ['active', 'Active'], true);
    final ok = await showDialog<String>(
      context: context,
      builder: (context) => StatefulBuilder(
        builder: (context, setModal) => AlertDialog(
          title: Text(id == null ? '新增优惠码' : '编辑优惠码'),
          content: SingleChildScrollView(
            child: Column(
              mainAxisSize: MainAxisSize.min,
              children: [
                TextField(controller: codeCtl, decoration: const InputDecoration(labelText: '优惠码')),
                const SizedBox(height: 8),
                TextField(controller: discountCtl, keyboardType: TextInputType.number, decoration: const InputDecoration(labelText: '折扣(‰)')),
                const SizedBox(height: 8),
                DropdownButtonFormField<String>(
                  value: _groups.any((g) => _pickStr(g, const ['id', 'ID']) == groupId) ? groupId : null,
                  decoration: const InputDecoration(labelText: '商品组'),
                  items: _groups
                      .map((g) => DropdownMenuItem(value: _pickStr(g, const ['id', 'ID']), child: Text('${_pickStr(g, const ['name', 'Name'])}(#${_pickStr(g, const ['id', 'ID'])})')))
                      .toList(),
                  onChanged: (v) => setModal(() => groupId = v ?? ''),
                ),
                const SizedBox(height: 8),
                Row(
                  children: [
                    Expanded(child: TextField(controller: totalCtl, keyboardType: TextInputType.number, decoration: const InputDecoration(labelText: '总次数'))),
                    const SizedBox(width: 8),
                    Expanded(child: TextField(controller: perCtl, keyboardType: TextInputType.number, decoration: const InputDecoration(labelText: '单用户次数'))),
                  ],
                ),
                SwitchListTile(value: newOnly, onChanged: (v) => setModal(() => newOnly = v), title: const Text('仅新用户')),
                SwitchListTile(value: active, onChanged: (v) => setModal(() => active = v), title: const Text('启用')),
              ],
            ),
          ),
          actions: [
            if (id != null)
              TextButton(onPressed: () => Navigator.pop(context, 'delete'), child: const Text('删除')),
            TextButton(onPressed: () => Navigator.pop(context, 'cancel'), child: const Text('取消')),
            FilledButton(onPressed: () => Navigator.pop(context, 'save'), child: const Text('保存')),
          ],
        ),
      ),
    );
    final client = context.read<AppState>().apiClient;
    if (client == null) return;
    if (ok == 'delete' && id != null) {
      await client.deleteJson('/admin/api/v1/coupons/$id');
      await _load();
      return;
    }
    if (ok != 'save') return;
    final payload = {
      'code': codeCtl.text.trim(),
      'discount_permille': _toInt(discountCtl.text, 900),
      'product_group_id': _toInt(groupId, 0),
      'total_limit': _toInt(totalCtl.text, -1),
      'per_user_limit': _toInt(perCtl.text, -1),
      'new_user_only': newOnly,
      'active': active,
    };
    if (id == null) {
      await client.postJson('/admin/api/v1/coupons', body: payload);
    } else {
      await client.patchJson('/admin/api/v1/coupons/$id', body: payload);
    }
    await _load();
  }

  Future<void> _batchGenerate() async {
    final prefixCtl = TextEditingController(text: 'CP');
    final countCtl = TextEditingController(text: '20');
    final lenCtl = TextEditingController(text: '8');
    final discountCtl = TextEditingController(text: '900');
    String groupId = _groups.isEmpty ? '' : _pickStr(_groups.first, const ['id', 'ID']);
    final ok = await showDialog<bool>(
      context: context,
      builder: (context) => StatefulBuilder(
        builder: (context, setModal) => AlertDialog(
          title: const Text('批量生成优惠码'),
          content: SingleChildScrollView(
            child: Column(
              mainAxisSize: MainAxisSize.min,
              children: [
                TextField(controller: prefixCtl, decoration: const InputDecoration(labelText: '前缀')),
                const SizedBox(height: 8),
                Row(
                  children: [
                    Expanded(child: TextField(controller: countCtl, keyboardType: TextInputType.number, decoration: const InputDecoration(labelText: '数量'))),
                    const SizedBox(width: 8),
                    Expanded(child: TextField(controller: lenCtl, keyboardType: TextInputType.number, decoration: const InputDecoration(labelText: '随机长度'))),
                  ],
                ),
                const SizedBox(height: 8),
                TextField(controller: discountCtl, keyboardType: TextInputType.number, decoration: const InputDecoration(labelText: '折扣(‰)')),
                const SizedBox(height: 8),
                DropdownButtonFormField<String>(
                  value: _groups.any((g) => _pickStr(g, const ['id', 'ID']) == groupId) ? groupId : null,
                  decoration: const InputDecoration(labelText: '商品组'),
                  items: _groups
                      .map((g) => DropdownMenuItem(value: _pickStr(g, const ['id', 'ID']), child: Text('${_pickStr(g, const ['name', 'Name'])}(#${_pickStr(g, const ['id', 'ID'])})')))
                      .toList(),
                  onChanged: (v) => setModal(() => groupId = v ?? ''),
                ),
              ],
            ),
          ),
          actions: [
            TextButton(onPressed: () => Navigator.pop(context, false), child: const Text('取消')),
            FilledButton(onPressed: () => Navigator.pop(context, true), child: const Text('生成')),
          ],
        ),
      ),
    );
    if (ok != true) return;
    final client = context.read<AppState>().apiClient;
    if (client == null) return;
    await client.postJson('/admin/api/v1/coupons/batch-generate', body: {
      'prefix': prefixCtl.text.trim(),
      'count': _toInt(countCtl.text, 20),
      'length': _toInt(lenCtl.text, 8),
      'discount_permille': _toInt(discountCtl.text, 900),
      'product_group_id': _toInt(groupId, 0),
      'total_limit': -1,
      'per_user_limit': -1,
      'new_user_only': false,
      'active': true,
    });
    await _load();
  }

  @override
  Widget build(BuildContext context) {
    return DefaultTabController(
      length: 2,
      child: Scaffold(
        appBar: AppBar(
          title: const Text('优惠码设置'),
          bottom: const TabBar(tabs: [Tab(text: '商品组'), Tab(text: '优惠码')]),
          actions: [
            IconButton(onPressed: _loading ? null : _load, icon: const Icon(Icons.refresh)),
          ],
        ),
        body: TabBarView(
          children: [
            RefreshIndicator(
              onRefresh: _load,
              child: ListView(
                physics: const AlwaysScrollableScrollPhysics(),
                padding: const EdgeInsets.fromLTRB(12, 12, 12, 20),
                children: [
                  Row(
                    children: [
                      const Expanded(child: Text('商品组', style: TextStyle(fontWeight: FontWeight.w700))),
                      FilledButton(onPressed: () => _editGroup(), child: const Text('新增商品组')),
                    ],
                  ),
                  const SizedBox(height: 8),
                  ..._groups.map((row) => Container(
                        margin: const EdgeInsets.only(bottom: 8),
                        decoration: BoxDecoration(
                          color: Colors.white,
                          borderRadius: BorderRadius.circular(12),
                          border: Border.all(color: const Color(0xFFE5EAF2)),
                        ),
                        child: ListTile(
                          title: Text('${_pickStr(row, const ['name', 'Name'])} (#${_pickStr(row, const ['id', 'ID'])})'),
                          subtitle: Text('规则数：${_rulesOf(row['rules']).length}'),
                          onTap: () => _editGroup(row: row),
                        ),
                      )),
                ],
              ),
            ),
            RefreshIndicator(
              onRefresh: _load,
              child: ListView(
                physics: const AlwaysScrollableScrollPhysics(),
                padding: const EdgeInsets.fromLTRB(12, 12, 12, 20),
                children: [
                  Row(
                    children: [
                      const Expanded(child: Text('优惠码', style: TextStyle(fontWeight: FontWeight.w700))),
                      OutlinedButton(onPressed: _batchGenerate, child: const Text('批量生成')),
                      const SizedBox(width: 8),
                      FilledButton(onPressed: () => _editCoupon(), child: const Text('新增优惠码')),
                    ],
                  ),
                  const SizedBox(height: 8),
                  ..._coupons.map((row) => Container(
                        margin: const EdgeInsets.only(bottom: 8),
                        decoration: BoxDecoration(
                          color: Colors.white,
                          borderRadius: BorderRadius.circular(12),
                          border: Border.all(color: const Color(0xFFE5EAF2)),
                        ),
                        child: ListTile(
                          title: Text(_asStr(row['code'])),
                          subtitle: Text('折扣:${_toInt(row['discount_permille'], 0) / 10}%  商品组:${_groupName(row['product_group_id'])}'),
                          trailing: Icon(row['active'] == true ? Icons.check_circle : Icons.pause_circle),
                          onTap: () => _editCoupon(row: row),
                        ),
                      )),
                ],
              ),
            ),
          ],
        ),
      ),
    );
  }
}

class EmailSettingsAdvancedScreen extends StatefulWidget {
  const EmailSettingsAdvancedScreen({super.key});

  @override
  State<EmailSettingsAdvancedScreen> createState() => _EmailSettingsAdvancedScreenState();
}

class _EmailSettingsAdvancedScreenState extends State<EmailSettingsAdvancedScreen> {
  bool _loading = false;
  bool smtpEnabled = false;
  bool emailEnabled = false;
  bool emailExpireEnabled = false;
  final hostCtl = TextEditingController();
  final portCtl = TextEditingController();
  final userCtl = TextEditingController();
  final passCtl = TextEditingController();
  final fromCtl = TextEditingController();
  final expireDaysCtl = TextEditingController(text: '7');
  final smtpTestToCtl = TextEditingController();
  List<Map<String, dynamic>> _templates = [];

  @override
  void dispose() {
    hostCtl.dispose();
    portCtl.dispose();
    userCtl.dispose();
    passCtl.dispose();
    fromCtl.dispose();
    expireDaysCtl.dispose();
    smtpTestToCtl.dispose();
    super.dispose();
  }

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    _load();
  }

  Future<void> _load() async {
    final client = context.read<AppState>().apiClient;
    if (client == null) return;
    setState(() => _loading = true);
    try {
      final res = await Future.wait([
        client.getJson('/admin/api/v1/settings'),
        client.getJson('/admin/api/v1/integrations/smtp'),
        client.getJson('/admin/api/v1/email-templates'),
      ]);
      final map = <String, dynamic>{};
      for (final raw in (res[0]['items'] as List<dynamic>? ?? [])) {
        final row = _asMap(raw);
        final key = _asStr(row['key']);
        if (key.isEmpty) continue;
        map[key] = row['value_json'] ?? row['value'] ?? '';
      }
      final smtp = _asMap(res[1]);
      final tpls = (res[2]['items'] as List<dynamic>? ?? []).map(_asMap).toList();
      if (!mounted) return;
      setState(() {
        hostCtl.text = _asStr(smtp['host']).isEmpty ? _asStr(map['smtp_host']) : _asStr(smtp['host']);
        portCtl.text = _asStr(smtp['port']).isEmpty ? _asStr(map['smtp_port']) : _asStr(smtp['port']);
        userCtl.text = _asStr(smtp['user']).isEmpty ? _asStr(map['smtp_user']) : _asStr(smtp['user']);
        passCtl.text = _asStr(smtp['pass']).isEmpty ? _asStr(map['smtp_pass']) : _asStr(smtp['pass']);
        fromCtl.text = _asStr(smtp['from']).isEmpty ? _asStr(map['smtp_from']) : _asStr(smtp['from']);
        smtpEnabled = smtp['enabled'] == true || _toBool(map['smtp_enabled'], false);
        emailEnabled = _toBool(map['email_enabled'], false);
        emailExpireEnabled = _toBool(map['email_expire_enabled'], false);
        expireDaysCtl.text = '${_toInt(map['expire_reminder_days'], 7)}';
        _templates = tpls;
      });
    } finally {
      if (mounted) setState(() => _loading = false);
    }
  }

  Future<void> _save() async {
    final client = context.read<AppState>().apiClient;
    if (client == null) return;
    await client.patchJson('/admin/api/v1/integrations/smtp', body: {
      'host': hostCtl.text.trim(),
      'port': portCtl.text.trim(),
      'user': userCtl.text.trim(),
      'pass': passCtl.text.trim(),
      'from': fromCtl.text.trim(),
      'enabled': smtpEnabled,
    });
    await _saveSettingsItems(client, [
      ('email_enabled', '$emailEnabled'),
      ('email_expire_enabled', '$emailExpireEnabled'),
      ('expire_reminder_days', '${_toInt(expireDaysCtl.text, 7)}'),
    ]);
    if (!mounted) return;
    ScaffoldMessenger.of(context).showSnackBar(const SnackBar(content: Text('邮件配置已保存')));
    await _load();
  }

  Future<void> _sendTest() async {
    final to = smtpTestToCtl.text.trim();
    if (to.isEmpty) return;
    final client = context.read<AppState>().apiClient;
    if (client == null) return;
    await client.postJson('/admin/api/v1/integrations/smtp/test', body: {
      'to': to,
      'subject': 'SMTP Test',
      'body': '如果您收到此邮件，则说明邮箱成功配置。',
      'html': false,
    });
    if (!mounted) return;
    ScaffoldMessenger.of(context).showSnackBar(const SnackBar(content: Text('测试邮件已发送')));
  }

  Future<void> _editTemplate({Map<String, dynamic>? row}) async {
    final id = row?['id'];
    final nameCtl = TextEditingController(text: _asStr(row?['name']));
    final subjectCtl = TextEditingController(text: _asStr(row?['subject']));
    final bodyCtl = TextEditingController(text: _asStr(row?['body']).isEmpty ? _asStr(row?['content']) : _asStr(row?['body']));
    var enabled = row == null ? true : row['enabled'] == true;
    final action = await showDialog<String>(
      context: context,
      builder: (context) => StatefulBuilder(
        builder: (context, setModal) => AlertDialog(
          title: Text(id == null ? '新增邮件模板' : '编辑邮件模板'),
          content: SingleChildScrollView(
            child: Column(
              mainAxisSize: MainAxisSize.min,
              children: [
                TextField(controller: nameCtl, decoration: const InputDecoration(labelText: '名称')),
                const SizedBox(height: 8),
                TextField(controller: subjectCtl, decoration: const InputDecoration(labelText: '主题')),
                const SizedBox(height: 8),
                TextField(controller: bodyCtl, maxLines: 8, decoration: const InputDecoration(labelText: '内容(支持HTML)')),
                SwitchListTile(value: enabled, onChanged: (v) => setModal(() => enabled = v), title: const Text('启用')),
              ],
            ),
          ),
          actions: [
            if (id != null)
              TextButton(onPressed: () => Navigator.pop(context, 'delete'), child: const Text('删除')),
            TextButton(onPressed: () => Navigator.pop(context, 'cancel'), child: const Text('取消')),
            FilledButton(onPressed: () => Navigator.pop(context, 'save'), child: const Text('保存')),
          ],
        ),
      ),
    );
    final client = context.read<AppState>().apiClient;
    if (client == null) return;
    if (action == 'delete' && id != null) {
      await client.deleteJson('/admin/api/v1/email-templates/$id');
      await _load();
      return;
    }
    if (action != 'save') return;
    final payload = {
      if (id != null) 'id': id,
      'name': nameCtl.text.trim(),
      'subject': subjectCtl.text.trim(),
      'body': bodyCtl.text,
      'enabled': enabled,
    };
    if (id == null) {
      await client.postJson('/admin/api/v1/email-templates', body: payload);
    } else {
      await client.patchJson('/admin/api/v1/email-templates/$id', body: payload);
    }
    await _load();
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('邮箱模板设置'),
        actions: [
          IconButton(onPressed: _loading ? null : _load, icon: const Icon(Icons.refresh)),
          TextButton(onPressed: _loading ? null : _save, child: const Text('保存')),
        ],
      ),
      body: ListView(
        padding: const EdgeInsets.fromLTRB(12, 12, 12, 20),
        children: [
          TextField(controller: hostCtl, decoration: const InputDecoration(labelText: 'SMTP Host')),
          const SizedBox(height: 8),
          TextField(controller: portCtl, decoration: const InputDecoration(labelText: 'SMTP Port')),
          const SizedBox(height: 8),
          TextField(controller: userCtl, decoration: const InputDecoration(labelText: 'SMTP User')),
          const SizedBox(height: 8),
          TextField(controller: passCtl, decoration: const InputDecoration(labelText: 'SMTP Pass')),
          const SizedBox(height: 8),
          TextField(controller: fromCtl, decoration: const InputDecoration(labelText: 'SMTP From')),
          SwitchListTile(value: smtpEnabled, onChanged: (v) => setState(() => smtpEnabled = v), title: const Text('启用SMTP')),
          SwitchListTile(value: emailEnabled, onChanged: (v) => setState(() => emailEnabled = v), title: const Text('启用邮件')),
          SwitchListTile(value: emailExpireEnabled, onChanged: (v) => setState(() => emailExpireEnabled = v), title: const Text('到期提醒邮件')),
          TextField(controller: expireDaysCtl, keyboardType: TextInputType.number, decoration: const InputDecoration(labelText: '到期提醒天数')),
          const SizedBox(height: 8),
          Row(
            children: [
              Expanded(child: TextField(controller: smtpTestToCtl, decoration: const InputDecoration(labelText: 'SMTP测试收件人'))),
              const SizedBox(width: 8),
              FilledButton(onPressed: _sendTest, child: const Text('测试发送')),
            ],
          ),
          const SizedBox(height: 16),
          Row(
            children: [
              const Expanded(child: Text('邮件模板', style: TextStyle(fontWeight: FontWeight.w700, fontSize: 15))),
              FilledButton.icon(
                onPressed: () => _editTemplate(),
                icon: const Icon(Icons.add, size: 16),
                label: const Text('新增'),
              ),
            ],
          ),
          const SizedBox(height: 8),
          ..._templates.map((row) => Container(
                margin: const EdgeInsets.only(bottom: 8),
                decoration: BoxDecoration(
                  color: Colors.white,
                  borderRadius: BorderRadius.circular(12),
                  border: Border.all(color: const Color(0xFFE5EAF2)),
                ),
                child: ListTile(
                  title: Text(_asStr(row['name'])),
                  subtitle: Text(_asStr(row['subject'])),
                  trailing: Icon(row['enabled'] == true ? Icons.check_circle : Icons.pause_circle),
                  onTap: () => _editTemplate(row: row),
                ),
              )),
        ],
      ),
    );
  }
}

class AdminProfileSettingsScreen extends StatefulWidget {
  const AdminProfileSettingsScreen({super.key});

  @override
  State<AdminProfileSettingsScreen> createState() => _AdminProfileSettingsScreenState();
}

class _AdminProfileSettingsScreenState extends State<AdminProfileSettingsScreen> {
  bool _loading = false;
  Map<String, dynamic> _profile = {};
  List<Map<String, dynamic>> _permissionGroups = [];
  List<Map<String, dynamic>> _permissions = [];

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    _load();
  }

  Future<void> _load() async {
    final client = context.read<AppState>().apiClient;
    if (client == null) return;
    setState(() => _loading = true);
    try {
      final res = await Future.wait([
        client.getJson('/admin/api/v1/profile'),
        client.getJson('/admin/api/v1/permission-groups'),
        client.getJson('/admin/api/v1/permissions/list'),
      ]);
      if (!mounted) return;
      setState(() {
        _profile = _asMap(res[0]);
        _permissionGroups = (res[1]['items'] as List<dynamic>? ?? []).map(_asMap).toList();
        _permissions = (res[2]['items'] as List<dynamic>? ?? []).map(_asMap).toList();
      });
    } finally {
      if (mounted) setState(() => _loading = false);
    }
  }

  String _groupName(dynamic id) {
    final sid = _asStr(id);
    for (final g in _permissionGroups) {
      if (_pickStr(g, const ['id', 'ID']) == sid) {
        return _pickStr(g, const ['name', 'Name']);
      }
    }
    return '-';
  }

  String _permName(String code) {
    for (final p in _permissions) {
      if (_pickStr(p, const ['code', 'Code']) == code) {
        final name = _pickStr(p, const ['friendly_name', 'FriendlyName']);
        if (name.isNotEmpty) return name;
        final fallback = _pickStr(p, const ['name', 'Name']);
        return fallback.isEmpty ? code : fallback;
      }
    }
    return code;
  }

  Future<void> _editProfile() async {
    final emailCtl = TextEditingController(text: _pickStr(_profile, const ['email', 'Email']));
    final qqCtl = TextEditingController(text: _pickStr(_profile, const ['qq', 'QQ']));
    final ok = await showDialog<bool>(
      context: context,
      builder: (context) => AlertDialog(
        title: const Text('编辑资料'),
        content: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            TextField(controller: emailCtl, decoration: const InputDecoration(labelText: '邮箱')),
            const SizedBox(height: 8),
            TextField(controller: qqCtl, decoration: const InputDecoration(labelText: 'QQ')),
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
    await client.patchJson('/admin/api/v1/profile', body: {
      'email': emailCtl.text.trim(),
      'qq': qqCtl.text.trim(),
    });
    if (!mounted) return;
    await context.read<AppState>().updateProfile(email: emailCtl.text.trim());
    await _load();
    if (!mounted) return;
    ScaffoldMessenger.of(context).showSnackBar(const SnackBar(content: Text('资料已更新')));
  }

  Future<void> _changePassword() async {
    final oldCtl = TextEditingController();
    final newCtl = TextEditingController();
    final confirmCtl = TextEditingController();
    final ok = await showDialog<bool>(
      context: context,
      builder: (context) => AlertDialog(
        title: const Text('修改密码'),
        content: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            TextField(controller: oldCtl, obscureText: true, decoration: const InputDecoration(labelText: '当前密码')),
            const SizedBox(height: 8),
            TextField(controller: newCtl, obscureText: true, decoration: const InputDecoration(labelText: '新密码')),
            const SizedBox(height: 8),
            TextField(controller: confirmCtl, obscureText: true, decoration: const InputDecoration(labelText: '确认新密码')),
          ],
        ),
        actions: [
          TextButton(onPressed: () => Navigator.pop(context, false), child: const Text('取消')),
          FilledButton(onPressed: () => Navigator.pop(context, true), child: const Text('提交')),
        ],
      ),
    );
    if (ok != true) return;
    if (newCtl.text != confirmCtl.text) {
      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(const SnackBar(content: Text('两次输入的新密码不一致')));
      return;
    }
    final client = context.read<AppState>().apiClient;
    if (client == null) return;
    await client.postJson('/admin/api/v1/profile/change-password', body: {
      'old_password': oldCtl.text,
      'new_password': newCtl.text,
    });
    if (!mounted) return;
    ScaffoldMessenger.of(context).showSnackBar(const SnackBar(content: Text('密码修改成功')));
  }

  @override
  Widget build(BuildContext context) {
    final qq = _pickStr(_profile, const ['qq', 'QQ']);
    final avatar = qq.isEmpty ? '' : 'https://q1.qlogo.cn/g?b=qq&nk=$qq&s=100';
    final permissions = _pick(_profile, const ['permissions', 'Permissions']) as List<dynamic>? ?? const [];
    return Scaffold(
      appBar: AppBar(
        title: const Text('个人资料设置'),
        actions: [
          IconButton(onPressed: _loading ? null : _load, icon: const Icon(Icons.refresh)),
          IconButton(onPressed: _loading ? null : _editProfile, icon: const Icon(Icons.edit_outlined)),
        ],
      ),
      body: RefreshIndicator(
        onRefresh: _load,
        child: ListView(
          physics: const AlwaysScrollableScrollPhysics(),
          padding: const EdgeInsets.fromLTRB(12, 12, 12, 20),
          children: [
            Container(
              padding: const EdgeInsets.all(12),
              decoration: BoxDecoration(
                color: Colors.white,
                borderRadius: BorderRadius.circular(12),
                border: Border.all(color: const Color(0xFFE5EAF2)),
              ),
              child: Row(
                children: [
                  CircleAvatar(
                    radius: 24,
                    foregroundImage: avatar.isEmpty ? null : NetworkImage(avatar),
                    child: avatar.isEmpty
                        ? Text(_pickStr(_profile, const ['username', 'Username']).isEmpty ? 'A' : _pickStr(_profile, const ['username', 'Username']).characters.first)
                        : null,
                  ),
                  const SizedBox(width: 10),
                  Expanded(
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        Text(_pickStr(_profile, const ['username', 'Username']), style: const TextStyle(fontWeight: FontWeight.w700)),
                        const SizedBox(height: 3),
                        Text(_pickStr(_profile, const ['email', 'Email']).isEmpty ? '-' : _pickStr(_profile, const ['email', 'Email'])),
                        Text('权限组: ${_groupName(_pick(_profile, const ['permission_group_id', 'PermissionGroupID']))}'),
                      ],
                    ),
                  ),
                ],
              ),
            ),
            const SizedBox(height: 10),
            Container(
              decoration: BoxDecoration(
                color: Colors.white,
                borderRadius: BorderRadius.circular(12),
                border: Border.all(color: const Color(0xFFE5EAF2)),
              ),
              child: ListTile(
                leading: const Icon(Icons.lock_outline),
                title: const Text('密码安全'),
                subtitle: const Text('修改管理员登录密码'),
                trailing: FilledButton(onPressed: _changePassword, child: const Text('修改')),
              ),
            ),
            const SizedBox(height: 10),
            Container(
              padding: const EdgeInsets.all(12),
              decoration: BoxDecoration(
                color: Colors.white,
                borderRadius: BorderRadius.circular(12),
                border: Border.all(color: const Color(0xFFE5EAF2)),
              ),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  const Text('权限列表', style: TextStyle(fontWeight: FontWeight.w700)),
                  const SizedBox(height: 8),
                  if (permissions.isEmpty)
                    const Text('暂无权限')
                  else
                    Wrap(
                      spacing: 8,
                      runSpacing: 8,
                      children: permissions.map((e) {
                        final code = e.toString();
                        return Chip(label: Text(_permName(code)));
                      }).toList(),
                    ),
                ],
              ),
            ),
          ],
        ),
      ),
    );
  }
}

class CmsUploadsSimpleScreen extends StatefulWidget {
  const CmsUploadsSimpleScreen({super.key});

  @override
  State<CmsUploadsSimpleScreen> createState() => _CmsUploadsSimpleScreenState();
}

class _CmsUploadsSimpleScreenState extends State<CmsUploadsSimpleScreen> {
  bool _loading = false;
  List<Map<String, dynamic>> _items = [];

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    _load();
  }

  Future<void> _load() async {
    final client = context.read<AppState>().apiClient;
    if (client == null) return;
    setState(() => _loading = true);
    try {
      final res = await client.getJson('/admin/api/v1/uploads', query: {'limit': '100', 'offset': '0'});
      if (!mounted) return;
      setState(() => _items = (res['items'] as List<dynamic>? ?? []).map(_asMap).toList());
    } finally {
      if (mounted) setState(() => _loading = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('上传中心'),
        actions: [IconButton(onPressed: _loading ? null : _load, icon: const Icon(Icons.refresh))],
      ),
      body: RefreshIndicator(
        onRefresh: _load,
        child: ListView(
          physics: const AlwaysScrollableScrollPhysics(),
          padding: const EdgeInsets.fromLTRB(12, 12, 12, 20),
          children: [
            const Text('媒体文件列表（简化版）', style: TextStyle(fontWeight: FontWeight.w700)),
            const SizedBox(height: 8),
            if (_loading && _items.isEmpty)
              const Padding(
                padding: EdgeInsets.only(top: 80),
                child: Center(child: CircularProgressIndicator()),
              )
            else if (_items.isEmpty)
              const Padding(
                padding: EdgeInsets.only(top: 80),
                child: Center(child: Text('暂无上传文件')),
              )
            else
              ..._items.map((row) {
                final name = _pickStr(row, const ['name', 'Name']).isEmpty ? _pickStr(row, const ['filename', 'Filename']) : _pickStr(row, const ['name', 'Name']);
                final url = _pickStr(row, const ['url', 'URL', 'path', 'Path']);
                final size = _pickStr(row, const ['size', 'Size']);
                final createdAt = _pickStr(row, const ['created_at', 'CreatedAt']);
                return Container(
                  margin: const EdgeInsets.only(bottom: 8),
                  decoration: BoxDecoration(
                    color: Colors.white,
                    borderRadius: BorderRadius.circular(12),
                    border: Border.all(color: const Color(0xFFE5EAF2)),
                  ),
                  child: ListTile(
                    title: Text(name.isEmpty ? '-' : name),
                    subtitle: Text('大小:${size.isEmpty ? '-' : size}  时间:${createdAt.isEmpty ? '-' : createdAt}\n$url'),
                    isThreeLine: true,
                    onLongPress: () async {
                      if (url.isEmpty) return;
                      await Clipboard.setData(ClipboardData(text: url));
                      if (!mounted) return;
                      ScaffoldMessenger.of(context).showSnackBar(const SnackBar(content: Text('已复制链接')));
                    },
                  ),
                );
              }),
          ],
        ),
      ),
    );
  }
}

Map<String, dynamic> _asMap(dynamic v) {
  if (v is Map<String, dynamic>) return v;
  if (v is Map) return v.map((k, val) => MapEntry(k.toString(), val));
  return <String, dynamic>{};
}

String _asStr(dynamic v) {
  final text = v?.toString() ?? '';
  return text.trim();
}

dynamic _pick(Map<String, dynamic> row, List<String> keys) {
  for (final k in keys) {
    if (row.containsKey(k) && row[k] != null) return row[k];
  }
  return null;
}

int _pickInt(Map<String, dynamic> row, List<String> keys, int fallback) {
  return _toInt(_pick(row, keys), fallback);
}

String _pickStr(Map<String, dynamic> row, List<String> keys) {
  return _asStr(_pick(row, keys));
}

bool _pickBool(Map<String, dynamic> row, List<String> keys, bool fallback) {
  return _toBool(_pick(row, keys), fallback);
}

dynamic _decodeBest(dynamic fallback, String pretty) {
  final text = pretty.trim();
  if (text.isEmpty) return fallback;
  try {
    return jsonDecode(text);
  } catch (_) {
    return fallback ?? text;
  }
}

String _prettyJson(dynamic value) {
  if (value == null) return '';
  if (value is String) {
    final raw = value.trim();
    if (raw.isEmpty) return '';
    try {
      final parsed = jsonDecode(raw);
      return const JsonEncoder.withIndent('  ').convert(parsed);
    } catch (_) {
      return raw;
    }
  }
  try {
    return const JsonEncoder.withIndent('  ').convert(value);
  } catch (_) {
    return _asStr(value);
  }
}

List<dynamic> _decodeJsonList(String text) {
  final raw = text.trim();
  if (raw.isEmpty) return [];
  try {
    final parsed = jsonDecode(raw);
    if (parsed is List) return parsed;
  } catch (_) {}
  return [];
}

Future<Map<String, dynamic>> _loadSettingsMap(dynamic client) async {
  final res = await client.getJson('/admin/api/v1/settings');
  final items = (res['items'] as List<dynamic>? ?? []);
  final map = <String, dynamic>{};
  for (final raw in items) {
    final row = _asMap(raw);
    final key = _asStr(row['key']);
    if (key.isEmpty) continue;
    map[key] = row['value_json'] ?? row['value'] ?? '';
  }
  return map;
}

Future<void> _saveSettingsItems(dynamic client, List<(String, String)> entries) async {
  final items = entries
      .map((e) => {'key': e.$1, 'value': e.$2})
      .toList();
  await client.patchJson('/admin/api/v1/settings', body: {'items': items});
}

Map<String, dynamic>? _tryDecodeObject(String raw) {
  if (raw.trim().isEmpty) return <String, dynamic>{};
  try {
    final v = jsonDecode(raw);
    if (v is Map<String, dynamic>) return v;
    if (v is Map) return v.map((k, val) => MapEntry(k.toString(), val));
    return <String, dynamic>{};
  } catch (_) {
    return null;
  }
}

bool _toBool(dynamic value, bool fallback) {
  final s = _strVal(value).toLowerCase();
  if (s.isEmpty) return fallback;
  if (s == 'true' || s == '1' || s == 'yes') return true;
  if (s == 'false' || s == '0' || s == 'no') return false;
  return fallback;
}

int _toInt(dynamic value, int fallback) {
  final s = _strVal(value).trim();
  if (s.isEmpty) return fallback;
  final n = int.tryParse(s);
  if (n != null) return n;
  final first = s.split(',').map((e) => e.trim()).firstWhere((e) => e.isNotEmpty, orElse: () => '');
  return int.tryParse(first) ?? fallback;
}

String _strVal(dynamic v) => v?.toString() ?? '';

List<String> _toList(dynamic value, List<String> fallback) {
  if (value == null) return fallback;
  final s = value.toString().trim();
  if (s.isEmpty) return fallback;
  if (s.startsWith('[') && s.endsWith(']')) {
    final body = s.substring(1, s.length - 1);
    final out = body
        .split(',')
        .map((e) => e.replaceAll('"', '').replaceAll("'", '').trim())
        .where((e) => e.isNotEmpty)
        .toList();
    return out.isEmpty ? fallback : out;
  }
  final split = s.split(',').map((e) => e.trim()).where((e) => e.isNotEmpty).toList();
  return split.isEmpty ? fallback : split;
}

List<String> _csvToList(String text, {required List<String> fallback}) {
  final list = text
      .split(',')
      .map((e) => e.trim().toLowerCase())
      .where((e) => e.isNotEmpty)
      .toList();
  return list.isEmpty ? fallback : list;
}

String _listToCsv(List<String> list) => list.join(',');

String _jsonList(List<String> list) => '[${list.map((e) => '"$e"').join(',')}]';
