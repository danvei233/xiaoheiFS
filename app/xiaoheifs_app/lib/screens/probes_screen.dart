import 'dart:async';

import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../app_state.dart';
import '../services/probe_api.dart';
import 'probe_detail_screen.dart';

class ProbesScreen extends StatefulWidget {
  const ProbesScreen({super.key});

  @override
  State<ProbesScreen> createState() => _ProbesScreenState();
}

class _ProbesScreenState extends State<ProbesScreen> {
  ApiClientHolder? _holder;
  ProbeApi? _api;
  bool _loading = false;
  bool _creating = false;
  bool _listInFlight = false;
  String _refreshError = '';
  DateTime? _lastRefreshAt;
  Timer? _refreshTimer;

  final TextEditingController _keywordCtl = TextEditingController();
  String _status = '';
  int _page = 1;
  int _pageSize = 20;
  int _total = 0;
  List<ProbeNode> _rows = [];
  final Map<int, double?> _slaMap = {};

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    final client = context.read<AppState>().apiClient;
    if (client == null) return;
    final next = ApiClientHolder(client.baseUrl, client.token, client.apiKey);
    if (_holder != next) {
      _holder = next;
      _api = ProbeApi(client);
      _fetchList();
      _startAutoRefresh();
    }
  }

  @override
  void dispose() {
    _refreshTimer?.cancel();
    _keywordCtl.dispose();
    super.dispose();
  }

  void _startAutoRefresh() {
    _refreshTimer?.cancel();
    _refreshTimer = Timer.periodic(const Duration(seconds: 10), (_) {
      _fetchList();
    });
  }

  Future<void> _fetchList() async {
    if (_api == null || _listInFlight) return;
    _listInFlight = true;
    setState(() => _loading = true);
    try {
      final ts = DateTime.now().millisecondsSinceEpoch;
      final resp = await _api!.listProbes(
        keyword: _keywordCtl.text.trim(),
        status: _status,
        limit: _pageSize,
        offset: (_page - 1) * _pageSize,
        timestamp: ts,
      );
      final slaMap = await _loadSlaForRows(resp.items, ts);
      if (!mounted) return;
      setState(() {
        _rows = resp.items;
        _total = resp.total;
        _slaMap
          ..clear()
          ..addAll(slaMap);
        _lastRefreshAt = DateTime.now();
        _refreshError = '';
      });
    } catch (e) {
      if (!mounted) return;
      setState(() {
        _refreshError = e.toString();
      });
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text('刷新失败: $e')),
      );
    } finally {
      _listInFlight = false;
      if (mounted) setState(() => _loading = false);
    }
  }

  Future<Map<int, double?>> _loadSlaForRows(List<ProbeNode> rows, int ts) async {
    if (_api == null) return {};
    final result = <int, double?>{};
    await Future.wait(rows.map((row) async {
      try {
        final sla = await _api!.getProbeSla(row.id, days: 7, timestamp: ts);
        result[row.id] = sla?.uptimePercent;
      } catch (_) {
        result[row.id] = null;
      }
    }));
    return result;
  }

  Future<void> _openCreateDialog() async {
    final nameCtl = TextEditingController();
    final agentCtl = TextEditingController();
    final tagsCtl = TextEditingController();
    String osType = 'linux';

    final ok = await showDialog<bool>(
      context: context,
      builder: (context) {
        return StatefulBuilder(
          builder: (context, setModalState) {
            return AlertDialog(
              title: const Text('新增探针'),
              content: SingleChildScrollView(
                child: Column(
                  mainAxisSize: MainAxisSize.min,
                  children: [
                    TextField(
                      controller: nameCtl,
                      decoration: const InputDecoration(labelText: '探针名称'),
                    ),
                    TextField(
                      controller: agentCtl,
                      decoration: const InputDecoration(labelText: 'Agent ID'),
                    ),
                    const SizedBox(height: 8),
                    DropdownButtonFormField<String>(
                      value: osType,
                      decoration: const InputDecoration(labelText: 'OS 类型'),
                      items: const [
                        DropdownMenuItem(value: 'linux', child: Text('Linux')),
                        DropdownMenuItem(value: 'windows', child: Text('Windows')),
                      ],
                      onChanged: (value) => setModalState(() => osType = value ?? 'linux'),
                    ),
                    TextField(
                      controller: tagsCtl,
                      decoration: const InputDecoration(
                        labelText: '标签(逗号分隔)',
                        hintText: 'region:hkg,role:edge',
                      ),
                    ),
                  ],
                ),
              ),
              actions: [
                TextButton(onPressed: () => Navigator.pop(context, false), child: const Text('取消')),
                FilledButton(onPressed: () => Navigator.pop(context, true), child: const Text('创建')),
              ],
            );
          },
        );
      },
    );

    if (ok == true) {
      if (_api == null) return;
      final agentId = agentCtl.text.trim();
      if (agentId.isEmpty) {
        ScaffoldMessenger.of(context).showSnackBar(const SnackBar(content: Text('请输入 Agent ID')));
        return;
      }
      setState(() => _creating = true);
      try {
        final tags = tagsCtl.text
            .split(',')
            .map((e) => e.trim())
            .where((e) => e.isNotEmpty)
            .toList();
        final result = await _api!.createProbe(
          name: nameCtl.text.trim(),
          agentId: agentId,
          osType: osType,
          tags: tags,
        );
        if (!mounted) return;
        ScaffoldMessenger.of(context).showSnackBar(const SnackBar(content: Text('创建成功')));
        _fetchList();
        final token = result.enrollToken ?? '';
        if (token.isNotEmpty) {
          _showTokenDialog(token, title: '一次性注册码');
        }
      } catch (e) {
        if (!mounted) return;
        ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text('创建失败: $e')));
      } finally {
        if (mounted) setState(() => _creating = false);
      }
    }

    nameCtl.dispose();
    agentCtl.dispose();
    tagsCtl.dispose();
  }

  Future<void> _resetEnroll(ProbeNode row) async {
    if (_api == null) return;
    try {
      final token = await _api!.resetEnrollToken(row.id);
      if (!mounted) return;
      if (token == null || token.isEmpty) {
        ScaffoldMessenger.of(context).showSnackBar(const SnackBar(content: Text('未获取到注册码')));
        return;
      }
      _showTokenDialog(token, title: '一次性注册码');
      ScaffoldMessenger.of(context).showSnackBar(const SnackBar(content: Text('已重置注册码')));
    } catch (e) {
      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text('重置失败: $e')));
    }
  }

  void _showTokenDialog(String token, {required String title}) {
    showDialog<void>(
      context: context,
      builder: (context) => AlertDialog(
        title: Text(title),
        content: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            const Text('仅展示一次，请尽快配置到探针端。'),
            const SizedBox(height: 12),
            SelectableText(token),
          ],
        ),
        actions: [
          TextButton(onPressed: () => Navigator.pop(context), child: const Text('关闭')),
        ],
      ),
    );
  }

  void _openDetail(ProbeNode row) {
    Navigator.push(
      context,
      MaterialPageRoute(builder: (_) => ProbeDetailScreen(probeId: row.id)),
    );
  }

  @override
  Widget build(BuildContext context) {
    return LayoutBuilder(
      builder: (context, constraints) {
        final isWide = constraints.maxWidth >= 900;
        return Scaffold(
          appBar: AppBar(
            title: const Text('探针监控'),
            actions: [
              IconButton(
                tooltip: '刷新',
                onPressed: _loading ? null : _fetchList,
                icon: const Icon(Icons.refresh),
              ),
              Padding(
                padding: const EdgeInsets.only(right: 8),
                child: FilledButton.icon(
                  onPressed: _creating ? null : _openCreateDialog,
                  icon: const Icon(Icons.add),
                  label: const Text('新增探针'),
                ),
              ),
            ],
          ),
          body: Column(
            children: [
              _buildHeaderInfo(),
              _buildFilters(),
              Expanded(
                child: RefreshIndicator(
                  onRefresh: _fetchList,
                  child: _rows.isEmpty && !_loading
                      ? ListView(
                          children: const [
                            SizedBox(height: 120),
                            Center(child: Text('暂无探针')),
                          ],
                        )
                      : isWide
                          ? _buildDesktopTable()
                          : _buildMobileCards(),
                ),
              ),
              _buildPagination(),
            ],
          ),
        );
      },
    );
  }

  Widget _buildHeaderInfo() {
    final refreshText = _lastRefreshAt == null ? '-' : _formatTime(_lastRefreshAt!.toIso8601String());
    return Container(
      width: double.infinity,
      margin: const EdgeInsets.fromLTRB(12, 8, 12, 6),
      padding: const EdgeInsets.all(10),
      decoration: BoxDecoration(
        borderRadius: BorderRadius.circular(10),
        color: Theme.of(context).colorScheme.surface,
        border: Border.all(color: Theme.of(context).colorScheme.outlineVariant.withOpacity(0.5)),
      ),
      child: Text(
        _refreshError.isEmpty ? '上次刷新：$refreshText' : '上次刷新：$refreshText，刷新失败：$_refreshError',
        style: Theme.of(context).textTheme.bodySmall,
      ),
    );
  }

  Widget _buildFilters() {
    return Container(
      margin: const EdgeInsets.fromLTRB(12, 0, 12, 8),
      child: Wrap(
        spacing: 8,
        runSpacing: 8,
        crossAxisAlignment: WrapCrossAlignment.center,
        children: [
          SizedBox(
            width: 240,
            child: TextField(
              controller: _keywordCtl,
              decoration: const InputDecoration(
                isDense: true,
                hintText: '按名称/AgentID搜索',
                prefixIcon: Icon(Icons.search),
              ),
            ),
          ),
          SizedBox(
            width: 140,
            child: DropdownButtonFormField<String>(
              value: _status.isEmpty ? null : _status,
              decoration: const InputDecoration(isDense: true, hintText: '状态'),
              items: const [
                DropdownMenuItem(value: 'online', child: Text('在线')),
                DropdownMenuItem(value: 'offline', child: Text('离线')),
              ],
              onChanged: (value) => setState(() => _status = value ?? ''),
            ),
          ),
          FilledButton(
            onPressed: () {
              setState(() => _page = 1);
              _fetchList();
            },
            child: const Text('查询'),
          ),
        ],
      ),
    );
  }

  Widget _buildDesktopTable() {
    return ListView(
      padding: const EdgeInsets.fromLTRB(12, 0, 12, 8),
      children: [
        SingleChildScrollView(
          scrollDirection: Axis.horizontal,
          child: DataTable(
            columns: const [
              DataColumn(label: Text('ID')),
              DataColumn(label: Text('名称')),
              DataColumn(label: Text('Agent ID')),
              DataColumn(label: Text('状态')),
              DataColumn(label: Text('OS')),
              DataColumn(label: Text('CPU')),
              DataColumn(label: Text('内存')),
              DataColumn(label: Text('标签')),
              DataColumn(label: Text('最后心跳')),
              DataColumn(label: Text('7天SLA')),
              DataColumn(label: Text('操作')),
            ],
            rows: _rows.map((row) {
              return DataRow(cells: [
                DataCell(Text('${row.id}')),
                DataCell(Text(row.name.isEmpty ? '-' : row.name)),
                DataCell(Text(row.agentId.isEmpty ? '-' : row.agentId)),
                DataCell(_StatusChip(status: row.status)),
                DataCell(Text(row.osType.isEmpty ? '-' : row.osType)),
                DataCell(_UsageBar(value: _usage(row, 'cpu'))),
                DataCell(_UsageBar(value: _usage(row, 'mem'))),
                DataCell(Text(row.tags.isEmpty ? '-' : row.tags.join(', '))),
                DataCell(Text(_formatTime(row.lastHeartbeatAt))),
                DataCell(Text(_formatSla(row.id))),
                DataCell(Row(
                  children: [
                    TextButton(onPressed: () => _openDetail(row), child: const Text('详情')),
                    TextButton(onPressed: () => _resetEnroll(row), child: const Text('重置注册码')),
                  ],
                )),
              ]);
            }).toList(),
          ),
        ),
      ],
    );
  }

  Widget _buildMobileCards() {
    return ListView.separated(
      padding: const EdgeInsets.fromLTRB(12, 0, 12, 8),
      itemCount: _rows.length,
      separatorBuilder: (_, __) => const SizedBox(height: 8),
      itemBuilder: (context, index) {
        final row = _rows[index];
        return Container(
          padding: const EdgeInsets.all(12),
          decoration: BoxDecoration(
            color: Theme.of(context).colorScheme.surface,
            borderRadius: BorderRadius.circular(10),
            border: Border.all(color: Theme.of(context).colorScheme.outlineVariant.withOpacity(0.5)),
          ),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Row(
                children: [
                  Expanded(
                    child: Text(
                      row.name.isEmpty ? row.agentId : row.name,
                      style: const TextStyle(fontWeight: FontWeight.w700),
                    ),
                  ),
                  _StatusChip(status: row.status),
                ],
              ),
              const SizedBox(height: 4),
              Text('Agent: ${row.agentId.isEmpty ? '-' : row.agentId}'),
              Text('OS: ${row.osType.isEmpty ? '-' : row.osType} · SLA: ${_formatSla(row.id)}'),
              Text('心跳: ${_formatTime(row.lastHeartbeatAt)}'),
              if (row.tags.isNotEmpty) Text('标签: ${row.tags.join(', ')}'),
              const SizedBox(height: 8),
              _UsageLine(label: 'CPU', value: _usage(row, 'cpu')),
              _UsageLine(label: '内存', value: _usage(row, 'mem')),
              const SizedBox(height: 8),
              Wrap(
                spacing: 8,
                runSpacing: 8,
                children: [
                  SizedBox(
                    height: 40,
                    child: OutlinedButton(onPressed: () => _openDetail(row), child: const Text('详情')),
                  ),
                  SizedBox(
                    height: 40,
                    child: OutlinedButton(onPressed: () => _resetEnroll(row), child: const Text('重置注册码')),
                  ),
                ],
              ),
            ],
          ),
        );
      },
    );
  }

  Widget _buildPagination() {
    final totalPages = (_total / _pageSize).ceil();
    return Padding(
      padding: const EdgeInsets.fromLTRB(12, 4, 12, 10),
      child: Row(
        children: [
          Expanded(
            child: Text('第 $_page / ${totalPages == 0 ? 1 : totalPages} 页 · 共 $_total 条'),
          ),
          DropdownButton<int>(
            value: _pageSize,
            items: const [10, 20, 50]
                .map((e) => DropdownMenuItem(value: e, child: Text('$e/页')))
                .toList(),
            onChanged: (value) {
              if (value == null) return;
              setState(() {
                _pageSize = value;
                _page = 1;
              });
              _fetchList();
            },
          ),
          const SizedBox(width: 8),
          OutlinedButton(
            onPressed: _page > 1 && !_loading
                ? () {
                    setState(() => _page -= 1);
                    _fetchList();
                  }
                : null,
            child: const Text('上一页'),
          ),
          const SizedBox(width: 6),
          OutlinedButton(
            onPressed: _page * _pageSize < _total && !_loading
                ? () {
                    setState(() => _page += 1);
                    _fetchList();
                  }
                : null,
            child: const Text('下一页'),
          ),
        ],
      ),
    );
  }

  String _formatSla(int id) {
    final sla = _slaMap[id];
    if (sla == null) return '-';
    return '${sla.toStringAsFixed(2)}%';
  }

  double? _usage(ProbeNode row, String kind) {
    dynamic value;
    if (kind == 'cpu') {
      value = row.snapshot.cpu['usage_percent'] ?? row.snapshot.raw['cpu_usage_percent'];
    } else {
      value = row.snapshot.memory['usage_percent'] ?? row.snapshot.raw['mem_usage_percent'];
    }
    final parsed = _asDouble(value);
    if (parsed == null) return null;
    if (parsed < 0) return 0;
    if (parsed > 100) return 100;
    return parsed;
  }

  String _formatTime(String raw) {
    if (raw.isEmpty) return '-';
    final dt = DateTime.tryParse(raw);
    if (dt == null) return raw;
    final local = dt.toLocal();
    return '${local.year}-${_pad2(local.month)}-${_pad2(local.day)} ${_pad2(local.hour)}:${_pad2(local.minute)}:${_pad2(local.second)}';
  }

  String _pad2(int value) => value.toString().padLeft(2, '0');

  double? _asDouble(dynamic value) {
    if (value is num) return value.toDouble();
    if (value is String) return double.tryParse(value);
    return null;
  }
}

class _StatusChip extends StatelessWidget {
  const _StatusChip({required this.status});

  final String status;

  @override
  Widget build(BuildContext context) {
    final online = status.toLowerCase() == 'online';
    final color = online ? const Color(0xFF2E7D32) : const Color(0xFF546E7A);
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 3),
      decoration: BoxDecoration(
        color: color.withOpacity(0.12),
        borderRadius: BorderRadius.circular(999),
      ),
      child: Text(
        online ? '在线' : '离线',
        style: TextStyle(color: color, fontSize: 12, fontWeight: FontWeight.w600),
      ),
    );
  }
}

class _UsageBar extends StatelessWidget {
  const _UsageBar({required this.value});

  final double? value;

  @override
  Widget build(BuildContext context) {
    if (value == null) return const Text('-');
    final v = value!;
    final color = v < 60
        ? const Color(0xFF2E7D32)
        : v < 85
            ? const Color(0xFFEF6C00)
            : const Color(0xFFD32F2F);
    return SizedBox(
      width: 120,
      child: Row(
        children: [
          Expanded(
            child: LinearProgressIndicator(
              value: v / 100,
              color: color,
              backgroundColor: color.withOpacity(0.15),
              minHeight: 8,
            ),
          ),
          const SizedBox(width: 6),
          Text('${v.toStringAsFixed(1)}%'),
        ],
      ),
    );
  }
}

class _UsageLine extends StatelessWidget {
  const _UsageLine({required this.label, required this.value});

  final String label;
  final double? value;

  @override
  Widget build(BuildContext context) {
    return Row(
      children: [
        SizedBox(width: 44, child: Text(label)),
        Expanded(child: _UsageBar(value: value)),
      ],
    );
  }
}

class ApiClientHolder {
  final String baseUrl;
  final String? token;
  final String? apiKey;

  const ApiClientHolder(this.baseUrl, this.token, this.apiKey);

  @override
  bool operator ==(Object other) {
    return other is ApiClientHolder &&
        other.baseUrl == baseUrl &&
        other.token == token &&
        other.apiKey == apiKey;
  }

  @override
  int get hashCode => Object.hash(baseUrl, token, apiKey);
}
