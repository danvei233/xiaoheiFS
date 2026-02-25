import 'dart:async';
import 'dart:convert';
import 'dart:math' as math;

import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../app_state.dart';
import '../services/probe_api.dart';
import '../services/sse_client.dart';

class ProbeDetailScreen extends StatefulWidget {
  const ProbeDetailScreen({super.key, required this.probeId});

  final int probeId;

  @override
  State<ProbeDetailScreen> createState() => _ProbeDetailScreenState();
}

class _ProbeDetailScreenState extends State<ProbeDetailScreen> {
  ProbeApi? _api;
  bool _loading = false;
  String _error = '';
  ProbeNode? _probe;
  ProbeSla? _sla;
  DateTime? _lastRefreshAt;
  Timer? _refreshTimer;
  bool _refreshInFlight = false;
  bool _portsExpanded = false;

  final TextEditingController _keywordCtl = TextEditingController();
  final ScrollController _logScrollCtl = ScrollController();
  String _logSource = 'file:logs';
  int _logLines = 300;
  bool _logFollow = true;
  bool _autoScroll = true;
  bool _logLoading = false;
  bool _logRunning = false;
  final List<String> _logRows = [];
  SseConnection? _logConn;

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    final client = context.read<AppState>().apiClient;
    if (_api == null && client != null) {
      _api = ProbeApi(client);
      _refreshAll();
      _refreshTimer = Timer.periodic(const Duration(seconds: 5), (_) {
        _refreshAll();
      });
    }
  }

  @override
  void dispose() {
    _refreshTimer?.cancel();
    _stopLog();
    _keywordCtl.dispose();
    _logScrollCtl.dispose();
    super.dispose();
  }

  Future<void> _refreshAll({bool forceSnapshot = false}) async {
    if (_api == null || _refreshInFlight) return;
    _refreshInFlight = true;
    setState(() {
      _loading = true;
      _error = '';
    });
    try {
      final ts = DateTime.now().millisecondsSinceEpoch;
      final results = await Future.wait([
        _api!.getProbeDetail(
          widget.probeId,
          refreshSnapshot: forceSnapshot,
          timestamp: ts,
        ),
        _api!.getProbeSla(widget.probeId, days: 7, timestamp: ts),
      ]);
      if (!mounted) return;
      setState(() {
        _probe = (results[0] as ProbeDetailResult).probe;
        _sla = results[1] as ProbeSla?;
        _lastRefreshAt = DateTime.now();
        _portsExpanded = false;
      });
    } catch (e) {
      if (!mounted) return;
      setState(() => _error = e.toString());
      ScaffoldMessenger.of(
        context,
      ).showSnackBar(SnackBar(content: Text('刷新失败: $e')));
    } finally {
      _refreshInFlight = false;
      if (mounted) setState(() => _loading = false);
    }
  }

  Future<void> _startLog() async {
    if (_api == null || _probe == null) return;
    _stopLog();
    setState(() {
      _logLoading = true;
      _logRows.clear();
    });

    try {
      final session = await _api!.createLogSession(
        widget.probeId,
        source: _logSource,
        keyword: _keywordCtl.text.trim(),
        lines: _logLines,
        follow: _logFollow,
      );
      if (session.streamPath.isEmpty) {
        throw Exception('未返回日志流地址');
      }

      final appState = context.read<AppState>();
      final auth = appState.session?.token?.isNotEmpty == true
          ? appState.session!.token!
          : (appState.session?.apiKey ?? '');
      final headers = <String, String>{
        if (auth.isNotEmpty) 'Authorization': 'Bearer $auth',
      };
      final url = _toAbsoluteUrl(
        appState.session?.apiUrl ?? '',
        session.streamPath,
      );

      _logConn = SseClient.connect(
        url,
        headers: headers,
        onMessage: _onLogMessage,
        onError: (error) {
          if (!mounted) return;
          ScaffoldMessenger.of(
            context,
          ).showSnackBar(const SnackBar(content: Text('日志连接中断，请重试')));
          setState(() {
            _logRunning = false;
            _logLoading = false;
          });
        },
        onDone: () {
          if (!mounted) return;
          setState(() {
            _logRunning = false;
            _logLoading = false;
          });
        },
      );

      if (!mounted) return;
      setState(() {
        _logRunning = true;
        _logLoading = false;
      });
    } catch (e) {
      if (!mounted) return;
      setState(() {
        _logRunning = false;
        _logLoading = false;
      });
      ScaffoldMessenger.of(
        context,
      ).showSnackBar(SnackBar(content: Text('启动日志失败: $e')));
    }
  }

  void _onLogMessage(SseEvent event) {
    if (!mounted) return;
    final raw = event.data;
    if (raw.isEmpty) return;

    try {
      final parsed = jsonDecode(raw);
      if (parsed is Map<String, dynamic>) {
        final type = (parsed['type'] ?? '').toString();
        if (type == 'log_chunk' && parsed['data'] != null) {
          _appendLog(parsed['data'].toString());
          return;
        }
        if (type == 'log_end') {
          setState(() => _logRunning = false);
          return;
        }
      }
    } catch (_) {}

    _appendLog(raw);
  }

  void _appendLog(String text) {
    final normalized = _normalizeDotNetDate(text);
    final lines = normalized.replaceAll('\r\n', '\n').split('\n');
    setState(() {
      _logRows.addAll(lines);
      if (_logRows.length > 4000) {
        _logRows.removeRange(0, _logRows.length - 4000);
      }
    });
    if (_autoScroll) {
      WidgetsBinding.instance.addPostFrameCallback((_) {
        if (!_logScrollCtl.hasClients) return;
        _logScrollCtl.jumpTo(_logScrollCtl.position.maxScrollExtent);
      });
    }
  }

  void _stopLog() {
    _logConn?.close();
    _logConn = null;
    if (mounted) {
      setState(() {
        _logRunning = false;
        _logLoading = false;
      });
    }
  }

  String _toAbsoluteUrl(String baseUrl, String path) {
    if (path.startsWith('http://') || path.startsWith('https://')) return path;
    final base = baseUrl.endsWith('/')
        ? baseUrl.substring(0, baseUrl.length - 1)
        : baseUrl;
    final p = path.startsWith('/') ? path : '/$path';
    return '$base$p';
  }

  @override
  Widget build(BuildContext context) {
    final probe = _probe;
    final isNarrow = MediaQuery.of(context).size.width < 900;

    return Scaffold(
      appBar: AppBar(
        title: Text(probe?.name.isNotEmpty == true ? probe!.name : '探针详情'),
        actions: [
          IconButton(
            onPressed: _loading ? null : () => _refreshAll(forceSnapshot: true),
            icon: const Icon(Icons.refresh),
          ),
        ],
      ),
      body: probe == null && _loading
          ? const Center(child: CircularProgressIndicator())
          : RefreshIndicator(
              onRefresh: () => _refreshAll(forceSnapshot: true),
              child: ListView(
                physics: const AlwaysScrollableScrollPhysics(),
                padding: const EdgeInsets.fromLTRB(10, 8, 10, 14),
                children: [
                  _buildHeader(probe),
                  if (_error.isNotEmpty)
                    Padding(
                      padding: const EdgeInsets.only(bottom: 8),
                      child: Text(
                        '刷新失败：$_error',
                        style: TextStyle(
                          color: Theme.of(context).colorScheme.error,
                        ),
                      ),
                    ),
                  const SizedBox(height: 6),
                  _buildMetrics(probe, isNarrow),
                  const SizedBox(height: 6),
                  _buildResourceCards(probe, isNarrow),
                  const SizedBox(height: 6),
                  _buildSystemAndDisk(probe, isNarrow),
                  if (_ports(probe).isNotEmpty) ...[
                    const SizedBox(height: 6),
                    _buildPorts(probe),
                  ],
                  const SizedBox(height: 6),
                  _buildLogCard(),
                ],
              ),
            ),
    );
  }

  Widget _buildHeader(ProbeNode? probe) {
    final tags = probe?.tags ?? const <String>[];
    return Container(
      padding: const EdgeInsets.all(12),
      decoration: BoxDecoration(
        borderRadius: BorderRadius.circular(14),
        gradient: const LinearGradient(
          colors: [Color(0xFF0EA5E9), Color(0xFF2563EB)],
          begin: Alignment.topLeft,
          end: Alignment.bottomRight,
        ),
        boxShadow: const [
          BoxShadow(
            color: Color(0x260F172A),
            blurRadius: 16,
            offset: Offset(0, 8),
          ),
        ],
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              Container(
                width: 40,
                height: 40,
                decoration: BoxDecoration(
                  color: Colors.white24,
                  borderRadius: BorderRadius.circular(12),
                ),
                child: const Icon(Icons.sensors_rounded, color: Colors.white),
              ),
              const SizedBox(width: 8),
              Expanded(
                child: Text(
                  probe?.name.isNotEmpty == true ? probe!.name : '-',
                  style: Theme.of(context).textTheme.titleMedium?.copyWith(
                    fontWeight: FontWeight.w700,
                    color: Colors.white,
                  ),
                ),
              ),
              _StatusPill(status: probe?.status ?? ''),
            ],
          ),
          const SizedBox(height: 4),
          Text(
            'ID: ${probe?.id ?? '-'} · Agent: ${probe?.agentId.isNotEmpty == true ? probe!.agentId : '-'}',
            style: const TextStyle(color: Colors.white, fontSize: 12),
          ),
          const SizedBox(height: 2),
          Text(
            '上次刷新：${_formatDateTime(_lastRefreshAt?.toIso8601String() ?? '')}',
            style: const TextStyle(color: Colors.white70, fontSize: 11),
          ),
          if (tags.isNotEmpty) ...[
            const SizedBox(height: 6),
            Wrap(
              spacing: 6,
              runSpacing: 6,
              children: tags
                  .take(6)
                  .map(
                    (t) => Container(
                      padding: const EdgeInsets.symmetric(
                        horizontal: 8,
                        vertical: 3,
                      ),
                      decoration: BoxDecoration(
                        color: Colors.white24,
                        borderRadius: BorderRadius.circular(999),
                      ),
                      child: Text(
                        t,
                        style: const TextStyle(color: Colors.white, fontSize: 10.5),
                      ),
                    ),
                  )
                  .toList(),
            ),
          ],
        ],
      ),
    );
  }

  Widget _buildMetrics(ProbeNode? probe, bool isNarrow) {
    final items = [
      _MetricItem(
        '系统运行时长',
        _formatUptime(
          _system(probe)['uptime'] ?? _system(probe)['uptime_seconds'],
        ),
        '持续运行中',
        Icons.timer_outlined,
      ),
      _MetricItem(
        '7天 SLA',
        '${(_sla?.uptimePercent ?? 0).toStringAsFixed(2)}%',
        '在线 ${_sla?.onlineSeconds ?? 0} 秒',
        Icons.shield_outlined,
      ),
      _MetricItem(
        '最后心跳',
        _formatDateShort(probe?.lastHeartbeatAt ?? ''),
        _fromNow(probe?.lastHeartbeatAt ?? ''),
        Icons.favorite_border_rounded,
      ),
      _MetricItem(
        '最后快照',
        _formatDateShort(probe?.lastSnapshotAt ?? ''),
        _fromNow(probe?.lastSnapshotAt ?? ''),
        Icons.image_search_rounded,
      ),
    ];

    if (isNarrow) {
      return GridView.builder(
        shrinkWrap: true,
        physics: const NeverScrollableScrollPhysics(),
        gridDelegate: const SliverGridDelegateWithFixedCrossAxisCount(
          crossAxisCount: 2,
          mainAxisSpacing: 6,
          crossAxisSpacing: 6,
          mainAxisExtent: 100,
        ),
        itemCount: items.length,
        itemBuilder: (_, i) => _MetricCard(item: items[i]),
      );
    }

    return Row(
      children: List.generate(items.length, (i) {
        return Expanded(
          child: Padding(
            padding: EdgeInsets.only(right: i == items.length - 1 ? 0 : 8),
            child: _MetricCard(item: items[i]),
          ),
        );
      }),
    );
  }

  Widget _buildResourceCards(ProbeNode? probe, bool isNarrow) {
    final cpu = _num(_cpu(probe)['usage_percent']);
    final mem = _num(_memory(probe)['usage_percent']);
    final disks = _disks(probe).take(2).toList();

    final cards = <Widget>[
      _ResourceCard(
        title: 'CPU 使用率',
        percent: cpu,
        detail1: '型号：${_str(_cpu(probe)['model'])}',
        detail2: '核心：${_str(_cpu(probe)['cores'])}',
      ),
      _ResourceCard(
        title: '内存使用率',
        percent: mem,
        detail1: '总量：${_formatBytes(_memory(probe)['total'])}',
        detail2: '已用：${_formatBytes(_memory(probe)['used'])}',
      ),
      ...disks.map(
        (d) => _ResourceCard(
          title: '磁盘 ${_str(d['mount'])}',
          percent: _num(d['usage_percent']),
          detail1: '${_formatBytes(d['used'])} / ${_formatBytes(d['total'])}',
          detail2: _str(d['fs']),
        ),
      ),
    ];

    if (isNarrow) {
      return Column(
        children: cards
            .map(
              (w) =>
                  Padding(padding: const EdgeInsets.only(bottom: 8), child: w),
            )
            .toList(),
      );
    }

    return Row(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: cards
          .map(
            (w) => Expanded(
              child: Padding(
                padding: const EdgeInsets.only(right: 8),
                child: w,
              ),
            ),
          )
          .toList(),
    );
  }

  Widget _buildSystemAndDisk(ProbeNode? probe, bool isNarrow) {
    final systemCard = _Panel(
      title: '系统信息',
      child: Column(
        children: [
          _kv('主机名', _str(_system(probe)['hostname'])),
          _kv('平台', _str(_system(probe)['platform'])),
          _kv(
            '内核',
            _str(_system(probe)['kernel'] ?? _system(probe)['kernel_version']),
          ),
          _kv('OS 类型', probe?.osType ?? '-'),
        ],
      ),
    );

    final diskRows = _disks(probe);
    final diskCard = _Panel(
      title: '磁盘详情',
      child: SingleChildScrollView(
        scrollDirection: Axis.horizontal,
        child: DataTable(
          columns: const [
            DataColumn(label: Text('挂载点')),
            DataColumn(label: Text('文件系统')),
            DataColumn(label: Text('总量')),
            DataColumn(label: Text('使用率')),
          ],
          rows: diskRows
              .map(
                (d) => DataRow(
                  cells: [
                    DataCell(Text(_str(d['mount']))),
                    DataCell(Text(_str(d['fs']))),
                    DataCell(Text(_formatBytes(d['total']))),
                    DataCell(
                      Text('${_num(d['usage_percent']).toStringAsFixed(1)}%'),
                    ),
                  ],
                ),
              )
              .toList(),
        ),
      ),
    );

    if (isNarrow) {
      return Column(
        children: [systemCard, const SizedBox(height: 8), diskCard],
      );
    }

    return Row(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Expanded(child: systemCard),
        const SizedBox(width: 8),
        Expanded(child: diskCard),
      ],
    );
  }

  Widget _buildPorts(ProbeNode? probe) {
    final ports = _ports(probe);
    final isNarrow = MediaQuery.of(context).size.width < 760;
    const int defaultVisible = 8;
    final visible = _portsExpanded ? ports : ports.take(defaultVisible).toList();
    final hasMore = ports.length > defaultVisible;

    return _Panel(
      title: '端口监听',
      extra: Text('共 ${ports.length} 条'),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          if (isNarrow)
            ...visible.map((p) {
              final listening = p['listen'] == true;
              return Container(
                margin: const EdgeInsets.only(bottom: 8),
                padding: const EdgeInsets.all(8),
                decoration: BoxDecoration(
                  borderRadius: BorderRadius.circular(10),
                  border: Border.all(color: const Color(0xFFE2E8F0)),
                  color: const Color(0xFFFAFCFF),
                ),
                child: Row(
                  children: [
                    Container(
                      padding: const EdgeInsets.symmetric(
                        horizontal: 8,
                        vertical: 4,
                      ),
                      decoration: BoxDecoration(
                        color: listening
                            ? const Color(0xFFDCFCE7)
                            : const Color(0xFFFEE2E2),
                        borderRadius: BorderRadius.circular(999),
                      ),
                      child: Text(
                        '${p['port'] ?? '-'}',
                        style: TextStyle(
                          color: listening
                              ? const Color(0xFF166534)
                              : const Color(0xFF991B1B),
                          fontWeight: FontWeight.w700,
                        ),
                      ),
                    ),
                    const SizedBox(width: 6),
                    Expanded(
                      child: Column(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                          Text(
                            '${_str(p['proto'])} · ${listening ? '监听' : '未监听'}',
                            style: const TextStyle(
                              fontWeight: FontWeight.w600,
                              fontSize: 12.5,
                            ),
                          ),
                          Text(
                            _str(p['process_name']),
                            maxLines: 1,
                            overflow: TextOverflow.ellipsis,
                            style: const TextStyle(
                              fontSize: 11,
                              color: Colors.black54,
                            ),
                          ),
                        ],
                      ),
                    ),
                  ],
                ),
              );
            })
          else
            SingleChildScrollView(
              scrollDirection: Axis.horizontal,
              child: DataTable(
                columns: const [
                  DataColumn(label: Text('端口')),
                  DataColumn(label: Text('协议')),
                  DataColumn(label: Text('状态')),
                  DataColumn(label: Text('进程')),
                ],
                rows: visible
                    .map(
                      (p) => DataRow(
                        cells: [
                          DataCell(Text('${p['port'] ?? '-'}')),
                          DataCell(Text(_str(p['proto']))),
                          DataCell(Text((p['listen'] == true) ? '监听' : '未监听')),
                          DataCell(Text(_str(p['process_name']))),
                        ],
                      ),
                    )
                    .toList(),
              ),
            ),
          if (hasMore)
            Align(
              alignment: Alignment.centerLeft,
              child: TextButton.icon(
                onPressed: () =>
                    setState(() => _portsExpanded = !_portsExpanded),
                icon: Icon(
                  _portsExpanded
                      ? Icons.expand_less_rounded
                      : Icons.expand_more_rounded,
                ),
                label: Text(
                  _portsExpanded
                      ? '收起端口列表'
                      : '展开更多端口（默认显示 $defaultVisible 条）',
                ),
              ),
            ),
        ],
      ),
    );
  }

  Widget _buildLogCard() {
    final sourceOptions = const [
      ('文件日志（logs）', 'file:logs'),
      ('Linux Journal(system)', 'journal:system'),
      ('Linux Journal(pveproxy)', 'journal:pveproxy'),
      ('Windows 系统关键日志', 'eventlog:System:important'),
      ('Windows 系统全部日志', 'eventlog:System:full'),
      ('Windows 开关机/崩溃', 'eventlog:System:power'),
      ('Windows 应用关键日志', 'eventlog:Application:important'),
      ('Windows Hyper-V 关键日志', 'eventlog:Hyper-V-Worker:important'),
    ];

    return _Panel(
      title: '日志查看',
      extra: Text('状态：${_logRunning ? '运行中' : '已停止'} · ${_logRows.length} 行'),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          LayoutBuilder(
            builder: (context, constraints) {
              final narrow = constraints.maxWidth < 760;
              if (narrow) {
                return Column(
                  children: [
                    Row(
                      children: [
                        Expanded(
                          child: DropdownButtonFormField<String>(
                            value: _logSource,
                            decoration: const InputDecoration(
                              isDense: true,
                              labelText: '日志源',
                            ),
                            items: sourceOptions
                                .map(
                                  (e) => DropdownMenuItem(
                                    value: e.$2,
                                    child: Text(e.$1, overflow: TextOverflow.ellipsis),
                                  ),
                                )
                                .toList(),
                            onChanged: (value) =>
                                setState(() => _logSource = value ?? _logSource),
                          ),
                        ),
                        const SizedBox(width: 8),
                        SizedBox(
                          width: 120,
                          child: DropdownButtonFormField<int>(
                            value: _logLines,
                            decoration: const InputDecoration(
                              isDense: true,
                              labelText: '行数',
                            ),
                            items: const [50, 100, 300, 500, 1000, 2000]
                                .map((v) => DropdownMenuItem(value: v, child: Text('$v')))
                                .toList(),
                            onChanged: (value) =>
                                setState(() => _logLines = value ?? _logLines),
                          ),
                        ),
                      ],
                    ),
                    const SizedBox(height: 8),
                    TextField(
                      controller: _keywordCtl,
                      decoration: const InputDecoration(
                        isDense: true,
                        labelText: '关键字过滤',
                        prefixIcon: Icon(Icons.search),
                      ),
                    ),
                    const SizedBox(height: 6),
                    Row(
                      children: [
                        Expanded(
                          child: SwitchListTile(
                            dense: true,
                            contentPadding: EdgeInsets.zero,
                            value: _logFollow,
                            title: const Text('跟随'),
                            onChanged: (value) =>
                                setState(() => _logFollow = value),
                          ),
                        ),
                        Expanded(
                          child: SwitchListTile(
                            dense: true,
                            contentPadding: EdgeInsets.zero,
                            value: _autoScroll,
                            title: const Text('自动滚动'),
                            onChanged: (value) =>
                                setState(() => _autoScroll = value),
                          ),
                        ),
                      ],
                    ),
                    Wrap(
                      spacing: 8,
                      runSpacing: 8,
                      children: [
                        FilledButton.icon(
                          onPressed: _logLoading ? null : _startLog,
                          icon: Icon(
                            _logRunning ? Icons.pause_circle : Icons.play_circle,
                          ),
                          label: Text(_logRunning ? '重新开始' : '开始'),
                        ),
                        OutlinedButton.icon(
                          onPressed: _logRunning ? _stopLog : null,
                          icon: const Icon(Icons.stop_circle_outlined),
                          label: const Text('停止'),
                        ),
                        OutlinedButton.icon(
                          onPressed: () => setState(_logRows.clear),
                          icon: const Icon(Icons.clear_all),
                          label: const Text('清空'),
                        ),
                      ],
                    ),
                  ],
                );
              }
              return Wrap(
                spacing: 8,
                runSpacing: 8,
                children: [
                  SizedBox(
                    width: 260,
                    child: DropdownButtonFormField<String>(
                      value: _logSource,
                      decoration: const InputDecoration(
                        isDense: true,
                        labelText: '日志源',
                      ),
                      items: sourceOptions
                          .map(
                            (e) => DropdownMenuItem(value: e.$2, child: Text(e.$1)),
                          )
                          .toList(),
                      onChanged: (value) =>
                          setState(() => _logSource = value ?? _logSource),
                    ),
                  ),
                  SizedBox(
                    width: 180,
                    child: TextField(
                      controller: _keywordCtl,
                      decoration: const InputDecoration(
                        isDense: true,
                        labelText: '关键字过滤',
                      ),
                    ),
                  ),
                  SizedBox(
                    width: 120,
                    child: DropdownButtonFormField<int>(
                      value: _logLines,
                      decoration: const InputDecoration(
                        isDense: true,
                        labelText: '行数',
                      ),
                      items: const [50, 100, 300, 500, 1000, 2000]
                          .map((v) => DropdownMenuItem(value: v, child: Text('$v')))
                          .toList(),
                      onChanged: (value) =>
                          setState(() => _logLines = value ?? _logLines),
                    ),
                  ),
                  SizedBox(
                    width: 120,
                    child: SwitchListTile(
                      dense: true,
                      contentPadding: EdgeInsets.zero,
                      value: _logFollow,
                      title: const Text('跟随'),
                      onChanged: (value) => setState(() => _logFollow = value),
                    ),
                  ),
                  SizedBox(
                    width: 140,
                    child: SwitchListTile(
                      dense: true,
                      contentPadding: EdgeInsets.zero,
                      value: _autoScroll,
                      title: const Text('自动滚动'),
                      onChanged: (value) => setState(() => _autoScroll = value),
                    ),
                  ),
                  FilledButton.icon(
                    onPressed: _logLoading ? null : _startLog,
                    icon: Icon(
                      _logRunning ? Icons.pause_circle : Icons.play_circle,
                    ),
                    label: Text(_logRunning ? '重新开始' : '开始'),
                  ),
                  OutlinedButton.icon(
                    onPressed: _logRunning ? _stopLog : null,
                    icon: const Icon(Icons.stop_circle_outlined),
                    label: const Text('停止'),
                  ),
                  OutlinedButton.icon(
                    onPressed: () => setState(_logRows.clear),
                    icon: const Icon(Icons.clear_all),
                    label: const Text('清空'),
                  ),
                ],
              );
            },
          ),
          const SizedBox(height: 8),
          Container(
            width: double.infinity,
            decoration: BoxDecoration(
              gradient: const LinearGradient(
                colors: [Color(0xFF0B1220), Color(0xFF111827)],
                begin: Alignment.topCenter,
                end: Alignment.bottomCenter,
              ),
              borderRadius: BorderRadius.circular(8),
              border: Border.all(color: const Color(0xFF1E293B)),
            ),
            child: Column(
              children: [
                Container(
                  height: 30,
                  padding: const EdgeInsets.symmetric(horizontal: 10),
                  decoration: const BoxDecoration(
                    border: Border(
                      bottom: BorderSide(color: Color(0xFF1F2937)),
                    ),
                  ),
                  child: Row(
                    children: [
                      const _Dot(Color(0xFFF87171)),
                      const SizedBox(width: 5),
                      const _Dot(Color(0xFFFBBF24)),
                      const SizedBox(width: 5),
                      const _Dot(Color(0xFF34D399)),
                      const SizedBox(width: 10),
                      Text(
                        _logSource,
                        style: const TextStyle(
                          color: Color(0xFF94A3B8),
                          fontSize: 11,
                          fontFamily: 'monospace',
                        ),
                      ),
                      const Spacer(),
                      Text(
                        'lines: ${_logRows.length}',
                        style: const TextStyle(
                          color: Color(0xFF64748B),
                          fontSize: 11,
                          fontFamily: 'monospace',
                        ),
                      ),
                    ],
                  ),
                ),
                SizedBox(
                  height: 220,
                  child: _logRows.isEmpty
                      ? const Center(
                          child: Text(
                            '暂无日志输出，点击“开始”获取日志',
                            style: TextStyle(color: Color(0xFF94A3B8)),
                          ),
                        )
                      : ListView.builder(
                          controller: _logScrollCtl,
                          itemCount: _logRows.length,
                          itemBuilder: (context, index) {
                            final line = _logRows[index];
                            return Container(
                              color: index.isEven
                                  ? Colors.transparent
                                  : const Color(0x220F172A),
                              padding: const EdgeInsets.symmetric(
                                horizontal: 8,
                                vertical: 1.2,
                              ),
                              child: Row(
                                crossAxisAlignment: CrossAxisAlignment.start,
                                children: [
                                  SizedBox(
                                    width: 34,
                                    child: Text(
                                      '${index + 1}'.padLeft(4, ' '),
                                      style: const TextStyle(
                                        color: Color(0xFF64748B),
                                        fontFamily: 'monospace',
                                        fontSize: 10.5,
                                      ),
                                    ),
                                  ),
                                  Expanded(
                                    child: Text(
                                      line,
                                      style: TextStyle(
                                        color: _logColor(line),
                                        fontFamily: 'monospace',
                                        fontSize: 11,
                                        height: 1.25,
                                      ),
                                    ),
                                  ),
                                ],
                              ),
                            );
                          },
                        ),
                ),
              ],
            ),
          ),
        ],
      ),
    );
  }

  Map<String, dynamic> _system(ProbeNode? probe) =>
      probe?.snapshot.system ?? const {};
  Map<String, dynamic> _cpu(ProbeNode? probe) =>
      probe?.snapshot.cpu ?? const {};
  Map<String, dynamic> _memory(ProbeNode? probe) =>
      probe?.snapshot.memory ?? const {};
  List<Map<String, dynamic>> _disks(ProbeNode? probe) =>
      probe?.snapshot.disks ?? const [];
  List<Map<String, dynamic>> _ports(ProbeNode? probe) =>
      probe?.snapshot.ports ?? const [];

  Widget _kv(String k, String v) {
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 4),
      child: Row(
        children: [
          SizedBox(
            width: 84,
            child: Text(k, style: const TextStyle(color: Colors.black54)),
          ),
          Expanded(child: Text(v)),
        ],
      ),
    );
  }

  Color _logColor(String line) {
    final s = line.toLowerCase();
    if (s.contains('[error]') ||
        s.contains('[critical]') ||
        s.contains('[fail]')) {
      return const Color(0xFFF87171);
    }
    if (s.contains('[warning]') || s.contains('[warn]')) {
      return const Color(0xFFFBBF24);
    }
    if (s.contains('[info]')) {
      return const Color(0xFF60A5FA);
    }
    if (s.contains('[debug]')) {
      return const Color(0xFF9CA3AF);
    }
    return const Color(0xFFE2E8F0);
  }

  String _normalizeDotNetDate(String input) {
    final re = RegExp(r'/Date\\((\\d+)(?:[+-]\\d+)?\\)/');
    return input.replaceAllMapped(re, (m) {
      final ms = int.tryParse(m.group(1) ?? '');
      if (ms == null) return m.group(0) ?? '';
      return _formatDateTime(
        DateTime.fromMillisecondsSinceEpoch(ms).toIso8601String(),
      );
    });
  }

  String _formatDateTime(String raw) {
    if (raw.isEmpty) return '-';
    final dt = DateTime.tryParse(raw);
    if (dt == null) return raw;
    final l = dt.toLocal();
    return '${l.year}-${_pad2(l.month)}-${_pad2(l.day)} ${_pad2(l.hour)}:${_pad2(l.minute)}:${_pad2(l.second)}';
  }

  String _formatDateShort(String raw) {
    if (raw.isEmpty) return '-';
    final dt = DateTime.tryParse(raw);
    if (dt == null) return raw;
    final l = dt.toLocal();
    return '${_pad2(l.month)}-${_pad2(l.day)} ${_pad2(l.hour)}:${_pad2(l.minute)}';
  }

  String _fromNow(String raw) {
    if (raw.isEmpty) return '-';
    final dt = DateTime.tryParse(raw);
    if (dt == null) return '-';
    final diff = DateTime.now().difference(dt.toLocal());
    if (diff.inDays > 0) return '${diff.inDays}天前';
    if (diff.inHours > 0) return '${diff.inHours}小时前';
    if (diff.inMinutes > 0) return '${diff.inMinutes}分钟前';
    return '${diff.inSeconds}秒前';
  }

  String _formatUptime(dynamic value) {
    final sec = _num(value).toInt();
    if (sec <= 0) return '-';
    final day = sec ~/ 86400;
    final hour = (sec % 86400) ~/ 3600;
    final min = (sec % 3600) ~/ 60;
    if (day > 0) return '$day天 $hour小时';
    return '$hour小时 $min分';
  }

  String _formatBytes(dynamic value) {
    final n = _num(value);
    if (n <= 0) return '0 B';
    const units = ['B', 'KB', 'MB', 'GB', 'TB'];
    var size = n;
    var idx = 0;
    while (size >= 1024 && idx < units.length - 1) {
      size /= 1024;
      idx += 1;
    }
    return '${size.toStringAsFixed(idx == 0 ? 0 : 2)} ${units[idx]}';
  }

  double _num(dynamic value) {
    if (value is num) return value.toDouble();
    if (value is String) return double.tryParse(value) ?? 0;
    return 0;
  }

  String _str(dynamic value) {
    final text = value?.toString() ?? '';
    return text.isEmpty ? '-' : text;
  }

  String _pad2(int v) => v.toString().padLeft(2, '0');
}

class _MetricItem {
  final String title;
  final String value;
  final String subtitle;
  final IconData icon;

  const _MetricItem(this.title, this.value, this.subtitle, this.icon);
}

class _MetricCard extends StatelessWidget {
  const _MetricCard({required this.item});

  final _MetricItem item;

  @override
  Widget build(BuildContext context) {
    return Container(
      margin: const EdgeInsets.only(bottom: 0),
      padding: const EdgeInsets.all(10),
      decoration: BoxDecoration(
        borderRadius: BorderRadius.circular(14),
        color: Colors.white,
        border: Border.all(color: const Color(0xFFE2E8F0)),
        boxShadow: const [
          BoxShadow(
            color: Color(0x120F172A),
            blurRadius: 10,
            offset: Offset(0, 4),
          ),
        ],
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Container(
            width: 30,
            height: 30,
            decoration: BoxDecoration(
              color: const Color(0xFFEAF4FF),
              borderRadius: BorderRadius.circular(9),
            ),
            child: Icon(item.icon, size: 16, color: const Color(0xFF1E88E5)),
          ),
          const SizedBox(height: 6),
          Text(
            item.title,
            style: const TextStyle(fontSize: 11, color: Colors.black54),
          ),
          const SizedBox(height: 4),
          Text(
            item.value,
            style: const TextStyle(fontSize: 17, fontWeight: FontWeight.w700),
          ),
          const SizedBox(height: 2),
          Text(
            item.subtitle,
            style: const TextStyle(fontSize: 11, color: Colors.black54),
          ),
        ],
      ),
    );
  }
}

class _ResourceCard extends StatelessWidget {
  const _ResourceCard({
    required this.title,
    required this.percent,
    required this.detail1,
    required this.detail2,
  });

  final String title;
  final double percent;
  final String detail1;
  final String detail2;

  @override
  Widget build(BuildContext context) {
    final v = percent.clamp(0, 100).toDouble();
    final color = v < 60
        ? const Color(0xFF2E7D32)
        : v < 85
        ? const Color(0xFFEF6C00)
        : const Color(0xFFD32F2F);

    return Container(
      padding: const EdgeInsets.fromLTRB(10, 8, 10, 8),
      decoration: BoxDecoration(
        borderRadius: BorderRadius.circular(14),
        color: Colors.white,
        border: Border.all(color: const Color(0xFFE2E8F0)),
        boxShadow: const [
          BoxShadow(
            color: Color(0x120F172A),
            blurRadius: 10,
            offset: Offset(0, 4),
          ),
        ],
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              SizedBox(
                width: 98,
                height: 64,
                child: _UsageGauge(value: v, color: color),
              ),
              const SizedBox(width: 6),
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                      title,
                      style: const TextStyle(
                        fontWeight: FontWeight.w700,
                        fontSize: 12.5,
                      ),
                      maxLines: 1,
                      overflow: TextOverflow.ellipsis,
                    ),
                    const SizedBox(height: 4),
                    Wrap(
                      spacing: 6,
                      runSpacing: 6,
                      children: [
                        _DensePill(text: '${v.toStringAsFixed(1)}%', color: color),
                        _DensePill(text: detail1, color: const Color(0xFF475569)),
                        _DensePill(text: detail2, color: const Color(0xFF64748B)),
                      ],
                    ),
                  ],
                ),
              ),
            ],
          ),
        ],
      ),
    );
  }
}

class _DensePill extends StatelessWidget {
  final String text;
  final Color color;
  const _DensePill({required this.text, required this.color});

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 2.5),
      decoration: BoxDecoration(
        color: color.withOpacity(0.10),
        borderRadius: BorderRadius.circular(999),
      ),
      constraints: const BoxConstraints(minHeight: 20),
      child: Text(
        text,
        style: TextStyle(
          color: color,
          fontSize: 10.5,
          fontWeight: FontWeight.w600,
        ),
        maxLines: 1,
        overflow: TextOverflow.ellipsis,
      ),
    );
  }
}

class _UsageGauge extends StatelessWidget {
  final double value;
  final Color color;
  const _UsageGauge({required this.value, required this.color});

  @override
  Widget build(BuildContext context) {
    return CustomPaint(
      painter: _UsageGaugePainter(value: value, color: color),
      child: const SizedBox.expand(),
    );
  }
}

class _UsageGaugePainter extends CustomPainter {
  final double value;
  final Color color;
  const _UsageGaugePainter({required this.value, required this.color});

  @override
  void paint(Canvas canvas, Size size) {
    final v = value.clamp(0, 100).toDouble();
    final center = Offset(size.width / 2, size.height * 0.9);
    final radius = math.min(size.width * 0.42, size.height * 0.9);
    const start = math.pi;
    const totalSweep = math.pi;
    final sweep = totalSweep * (v / 100);

    final base = Paint()
      ..style = PaintingStyle.stroke
      ..strokeWidth = 8
      ..strokeCap = StrokeCap.round
      ..color = const Color(0xFFE2E8F0);
    canvas.drawArc(
      Rect.fromCircle(center: center, radius: radius),
      start,
      totalSweep,
      false,
      base,
    );

    final active = Paint()
      ..style = PaintingStyle.stroke
      ..strokeWidth = 8
      ..strokeCap = StrokeCap.round
      ..shader = LinearGradient(
        colors: [color.withOpacity(0.7), color],
        begin: Alignment.centerLeft,
        end: Alignment.centerRight,
      ).createShader(
        Rect.fromCircle(center: center, radius: radius),
      );
    canvas.drawArc(
      Rect.fromCircle(center: center, radius: radius),
      start,
      sweep,
      false,
      active,
    );

    final tickPaint = Paint()
      ..color = const Color(0xFFCBD5E1)
      ..strokeWidth = 1.3;
    for (var i = 0; i <= 10; i++) {
      final a = start + totalSweep * (i / 10);
      final p1 = Offset(
        center.dx + math.cos(a) * (radius - 2),
        center.dy + math.sin(a) * (radius - 2),
      );
      final p2 = Offset(
        center.dx + math.cos(a) * (radius - 8),
        center.dy + math.sin(a) * (radius - 8),
      );
      canvas.drawLine(p1, p2, tickPaint);
    }

    final needleAngle = start + sweep;
    final needleEnd = Offset(
      center.dx + math.cos(needleAngle) * (radius - 14),
      center.dy + math.sin(needleAngle) * (radius - 14),
    );
    final needle = Paint()
      ..color = const Color(0xFF334155)
      ..strokeWidth = 2.2
      ..strokeCap = StrokeCap.round;
    canvas.drawLine(center, needleEnd, needle);
    canvas.drawCircle(center, 3.5, Paint()..color = const Color(0xFF334155));

    final tp = TextPainter(
      text: TextSpan(
        text: '${v.toStringAsFixed(1)}%',
        style: TextStyle(
          color: color,
          fontSize: 12,
          fontWeight: FontWeight.w800,
        ),
      ),
      textDirection: TextDirection.ltr,
      maxLines: 1,
    )..layout(minWidth: 0, maxWidth: size.width);
    tp.paint(
      canvas,
      Offset((size.width - tp.width) / 2, size.height * 0.50),
    );
  }

  @override
  bool shouldRepaint(covariant _UsageGaugePainter oldDelegate) {
    return oldDelegate.value != value || oldDelegate.color != color;
  }
}

class _Panel extends StatelessWidget {
  const _Panel({required this.title, required this.child, this.extra});

  final String title;
  final Widget child;
  final Widget? extra;

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.all(10),
      decoration: BoxDecoration(
        borderRadius: BorderRadius.circular(14),
        color: Colors.white,
        border: Border.all(color: const Color(0xFFE2E8F0)),
        boxShadow: const [
          BoxShadow(
            color: Color(0x120F172A),
            blurRadius: 10,
            offset: Offset(0, 4),
          ),
        ],
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              Expanded(
                child: Text(
                  title,
                  style: const TextStyle(
                    fontWeight: FontWeight.w700,
                    fontSize: 14,
                  ),
                ),
              ),
              if (extra != null) extra!,
            ],
          ),
          const SizedBox(height: 6),
          child,
        ],
      ),
    );
  }
}

class _StatusPill extends StatelessWidget {
  const _StatusPill({required this.status});

  final String status;

  @override
  Widget build(BuildContext context) {
    final online = status.toLowerCase() == 'online';
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
      decoration: BoxDecoration(
        color: Colors.white.withOpacity(0.2),
        borderRadius: BorderRadius.circular(999),
        border: Border.all(color: Colors.white24),
      ),
      child: Text(
        online ? '在线' : '离线',
        style: const TextStyle(
          color: Colors.white,
          fontWeight: FontWeight.w600,
        ),
      ),
    );
  }
}

class _Dot extends StatelessWidget {
  final Color color;
  const _Dot(this.color);
  @override
  Widget build(BuildContext context) {
    return Container(
      width: 7,
      height: 7,
      decoration: BoxDecoration(color: color, borderRadius: BorderRadius.circular(8)),
    );
  }
}
