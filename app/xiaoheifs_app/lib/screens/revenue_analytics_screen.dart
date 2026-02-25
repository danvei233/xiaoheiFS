import 'dart:math' as math;
import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../app_state.dart';
import '../services/api_client.dart';
import '../services/avatar.dart';

class RevenueAnalyticsScreen extends StatefulWidget {
  const RevenueAnalyticsScreen({super.key});
  @override
  State<RevenueAnalyticsScreen> createState() => _RevenueAnalyticsScreenState();
}

class _RevenueAnalyticsScreenState extends State<RevenueAnalyticsScreen> {
  ApiClient? _c;
  bool _loading = false;
  bool _rankLoading = false;
  bool _filterExpanded = false;
  final _uidCtl = TextEditingController();
  final _goods = <_Opt>[], _regions = <_Opt>[], _lines = <_Opt>[], _pkgs = <_Opt>[];
  final _users = <int, String>{};
  final _avatars = <int, String>{};
  DateTime _from = DateTime.now().subtract(const Duration(days: 30)), _to = DateTime.now();
  String _quick = '30d', _level = 'overall';
  int? _goodsId, _regionId, _lineId, _pkgId, _uid;
  int _page = 1, _pageSize = 20, _total = 0;
  Map<String, dynamic> _overview = {};
  List<Map<String, dynamic>> _top = [], _trend = [], _detail = [];
  List<_UserRank> _userRanks = [];

  @override
  void dispose() {
    _uidCtl.dispose();
    super.dispose();
  }

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    final client = context.read<AppState>().apiClient;
    if (client != null &&
        (client.baseUrl != _c?.baseUrl || client.token != _c?.token || client.apiKey != _c?.apiKey)) {
      _c = client;
      _init();
    }
  }

  Map<String, dynamic> _q({bool detail = false}) => {
    'from_at': _from.toUtc().toIso8601String(),
    'to_at': _to.toUtc().toIso8601String(),
    'level': _level,
    if (_goodsId != null) 'goods_type_id': _goodsId,
    if (_regionId != null) 'region_id': _regionId,
    if (_lineId != null) 'line_id': _lineId,
    if (_pkgId != null) 'package_id': _pkgId,
    if (_uid != null) 'user_id': _uid,
    if (detail) 'page': _page,
    if (detail) 'page_size': _pageSize,
    if (detail) 'sort_field': 'paid_at',
    if (detail) 'sort_order': 'desc',
  };

  Future<void> _init() async {
    setState(() => _loading = true);
    try {
      await _loadGoods();
      await _reload();
    } finally {
      if (mounted) setState(() => _loading = false);
    }
  }

  Future<void> _loadGoods() async {
    final r = await _c!.getJson('/admin/api/v1/goods-types');
    _goods
      ..clear()
      ..addAll(_ls(r['items']).map((e) => _Opt(_i(e['id']), _s(e['name'], '类型'))));
  }

  Future<void> _loadRegions() async {
    if (_goodsId == null) return _regions.clear();
    final r = await _c!.getJson('/admin/api/v1/regions', query: {'goods_type_id': '$_goodsId'});
    _regions
      ..clear()
      ..addAll(_ls(r['items']).map((e) => _Opt(_i(e['id']), _s(e['name'], '地区'))));
  }

  Future<void> _loadLines() async {
    if (_goodsId == null || _regionId == null) return _lines.clear();
    final r = await _c!.getJson('/admin/api/v1/plan-groups', query: {'goods_type_id': '$_goodsId', 'region_id': '$_regionId'});
    _lines
      ..clear()
      ..addAll(_ls(r['items']).map((e) => _Opt(_i(e['id']), _s(e['name'], '线路'))));
  }

  Future<void> _loadPkgs() async {
    if (_goodsId == null || _regionId == null) return _pkgs.clear();
    final r = await _c!.getJson('/admin/api/v1/packages', query: {'goods_type_id': '$_goodsId'});
    _pkgs
      ..clear()
      ..addAll(_ls(r['items']).where((e) => _lineId == null || _i(e['line_id']) == _lineId).map((e) => _Opt(_i(e['id']), _s(e['name'], '套餐'))));
  }

  Future<void> _reload() async {
    setState(() => _loading = true);
    try {
      final x = await Future.wait([
        _c!.postJson('/admin/api/v1/dashboard/revenue-analytics/overview', body: _q()),
        _c!.postJson('/admin/api/v1/dashboard/revenue-analytics/top', body: _q()),
        _c!.postJson('/admin/api/v1/dashboard/revenue-analytics/trend', body: _q()),
        _c!.postJson('/admin/api/v1/dashboard/revenue-analytics/details', body: _q(detail: true)),
      ]);
      _overview = _m(x[0]['data'], x[0]);
      _top = _ls(x[1]['items'] ?? _m(x[1]['data'])['items']);
      _trend = _ls(x[2]['items'] ?? _m(x[2]['data'])['items']);
      _detail = _ls(x[3]['items'] ?? _m(x[3]['data'])['items']);
      _total = _i(x[3]['total'] ?? _m(x[3]['data'])['total']);
      await _loadUsernames(_detail.map((e) => _i(e['user_id'])).where((e) => e > 0).toSet().toList());
      _refreshUserRanks();
    } finally {
      if (mounted) setState(() => _loading = false);
    }
  }

  Future<void> _reloadDetail() async {
    setState(() => _loading = true);
    try {
      final x = await _c!.postJson('/admin/api/v1/dashboard/revenue-analytics/details', body: _q(detail: true));
      _detail = _ls(x['items'] ?? _m(x['data'])['items']);
      _total = _i(x['total'] ?? _m(x['data'])['total']);
      await _loadUsernames(_detail.map((e) => _i(e['user_id'])).where((e) => e > 0).toSet().toList());
    } finally {
      if (mounted) setState(() => _loading = false);
    }
  }

  Future<List<Map<String, dynamic>>> _allRows({int? uid}) async {
    final out = <Map<String, dynamic>>[];
    var p = 1;
    for (var i = 0; i < 30; i++) {
      final x = await _c!.postJson('/admin/api/v1/dashboard/revenue-analytics/details', body: {..._q(), if (uid != null) 'user_id': uid, 'page': p, 'page_size': 200, 'sort_field': 'paid_at', 'sort_order': 'desc'});
      final items = _ls(x['items'] ?? _m(x['data'])['items']);
      final total = _i(x['total'] ?? _m(x['data'])['total']);
      out.addAll(items);
      if (items.length < 200 || out.length >= total) break;
      p++;
    }
    return out;
  }

  Future<void> _loadUsernames(List<int> ids) async {
    for (final id in ids) {
      if (_users.containsKey(id)) continue;
      try {
        final x = await _c!.getJson('/admin/api/v1/users/$id');
        final d = _m(x['user'], _m(x['data'], x));
        _users[id] = _s(d['username'], '用户$id');
        _avatars[id] = resolveAvatarUrl(
          baseUrl: _c?.baseUrl ?? '',
          qq: _s(d['qq'], ''),
          avatarUrl: _s(d['avatar_url'], _s(d['avatar'], '')),
        );
      } catch (_) {
        _users[id] = '用户$id';
        _avatars[id] = '';
      }
    }
  }

  Future<void> _refreshUserRanks() async {
    if (_rankLoading) return;
    setState(() => _rankLoading = true);
    try {
      final rows = await _allRows();
      final agg = <int, _Agg>{};
      for (final r in rows) {
        final u = _i(r['user_id']);
        if (u <= 0) continue;
        final it = agg.putIfAbsent(u, () => _Agg());
        it.sum += _i(r['amount_cents']);
        final oid = _i(r['order_id']);
        if (oid > 0) it.orders.add(oid);
      }
      final ranked = agg.entries.map((e) => _UserRank(userId: e.key, revenueCents: e.value.sum, orderCount: e.value.orders.length)).toList()
        ..sort((a, b) => b.revenueCents.compareTo(a.revenueCents));
      _userRanks = ranked.take(20).toList();
      for (var i = 0; i < _userRanks.length; i++) {
        _userRanks[i] = _userRanks[i].copyWith(rank: i + 1);
      }
      await _loadUsernames(_userRanks.map((e) => e.userId).toList());
    } catch (_) {
    } finally {
      if (mounted) setState(() => _rankLoading = false);
    }
  }

  void _syncLevel() {
    _level = _goodsId == null ? 'overall' : _regionId == null ? 'goods_type' : _lineId == null ? 'region' : _pkgId == null ? 'line' : 'package';
  }

  Future<void> _drill(Map<String, dynamic> it) async {
    final id = _i(it['dimension_id']);
    if (id <= 0) return;
    if (_level == 'overall') {
      _goodsId = id;
      _regionId = _lineId = _pkgId = null;
      await _loadRegions();
      _lines.clear();
      _pkgs.clear();
    } else if (_level == 'goods_type') {
      _regionId = id;
      _lineId = _pkgId = null;
      await _loadLines();
      await _loadPkgs();
    } else if (_level == 'region') {
      _lineId = id;
      _pkgId = null;
      await _loadPkgs();
    } else if (_level == 'line') {
      _pkgId = id;
    }
    _syncLevel();
    _page = 1;
    if (mounted) setState(() {});
    await _reload();
  }

  Future<void> _goLevelUp() async {
    if (_level == 'overall') return;
    if (_level == 'package') {
      _pkgId = null;
    } else if (_level == 'line') {
      _lineId = null;
      _pkgId = null;
      await _loadPkgs();
    } else if (_level == 'region') {
      _regionId = null;
      _lineId = null;
      _pkgId = null;
      _lines.clear();
      _pkgs.clear();
    } else if (_level == 'goods_type') {
      _goodsId = null;
      _regionId = null;
      _lineId = null;
      _pkgId = null;
      _regions.clear();
      _lines.clear();
      _pkgs.clear();
    }
    _syncLevel();
    _page = 1;
    if (mounted) setState(() {});
    await _reload();
  }

  void _quickApply(String k) {
    final n = DateTime.now();
    var f = DateTime(n.year, n.month, n.day);
    final t = DateTime(n.year, n.month, n.day, 23, 59, 59);
    if (k == '7d') f = f.subtract(const Duration(days: 6));
    if (k == '30d') f = f.subtract(const Duration(days: 30));
    if (k == 'month') f = DateTime(n.year, n.month, 1);
    setState(() {
      _quick = k;
      _from = f;
      _to = t;
    });
  }

  Future<void> _pickRange() async {
    final s = await showDatePicker(context: context, initialDate: _from, firstDate: DateTime(2020), lastDate: DateTime.now().add(const Duration(days: 365)));
    if (s == null) return;
    final e = await showDatePicker(context: context, initialDate: _to, firstDate: s, lastDate: DateTime.now().add(const Duration(days: 365)));
    if (e == null) return;
    setState(() {
      _from = DateTime(s.year, s.month, s.day);
      _to = DateTime(e.year, e.month, e.day, 23, 59, 59);
    });
  }

  int get _pages => (_total / _pageSize).ceil().clamp(1, 1 << 20);

  @override
  Widget build(BuildContext context) {
    final s = _m(_overview['summary']);
    final insight = _detailInsight(_detail);
    final momRatio = _d(s['mom_ratio']);
    final momComparable = _b(s['mom_comparable']);
    final yoyRatio = _d(s['yoy_ratio']);
    final yoyComparable = _b(s['yoy_comparable']);
    final trendPts = _trend.map((e) => _Pt(_s(e['bucket'], '-'), _i(e['revenue_cents']) / 100.0)).toList();
    return Scaffold(
      appBar: AppBar(title: const Text('收入统计'), actions: [IconButton(onPressed: _loading ? null : _reload, icon: const Icon(Icons.refresh_rounded))]),
      body: RefreshIndicator(
        onRefresh: _reload,
        child: ListView(
          physics: const AlwaysScrollableScrollPhysics(),
          padding: const EdgeInsets.fromLTRB(12, 12, 12, 20),
          children: [
            _FilterCard(goods: _goods, regions: _regions, lines: _lines, pkgs: _pkgs, goodsId: _goodsId, regionId: _regionId, lineId: _lineId, pkgId: _pkgId, uidCtl: _uidCtl, from: _from, to: _to, quick: _quick, level: _level, loading: _loading, expanded: _filterExpanded, onToggleExpand: () => setState(() => _filterExpanded = !_filterExpanded), onPickRange: _pickRange, onQuick: _quickApply, onGoods: (v) async { setState(() { _goodsId = v; _regionId = _lineId = _pkgId = null; _syncLevel();}); await _loadRegions(); if (mounted) setState(() {});}, onRegion: (v) async { setState(() { _regionId = v; _lineId = _pkgId = null; _syncLevel();}); await _loadLines(); await _loadPkgs(); if (mounted) setState(() {});}, onLine: (v) async { setState(() { _lineId = v; _pkgId = null; _syncLevel();}); await _loadPkgs(); if (mounted) setState(() {});}, onPkg: (v) => setState(() { _pkgId = v; _syncLevel();}), onReset: () async { setState(() { _goodsId = _regionId = _lineId = _pkgId = _uid = null; _uidCtl.clear(); _from = DateTime.now().subtract(const Duration(days: 30)); _to = DateTime.now(); _quick = '30d'; _page = 1; _syncLevel();}); await _reload();}, onSearch: () { setState(() { _uid = int.tryParse(_uidCtl.text.trim()); _page = 1; _syncLevel();}); _reload();}),
            const SizedBox(height: 12),
            SizedBox(height: 138, child: ListView(scrollDirection: Axis.horizontal, children: [
              _Kpi(
                '总收入',
                '¥${(_i(s['total_revenue_cents']) / 100).toStringAsFixed(2)}',
                '订单数: ${_i(s['order_count'])}',
                const [Color(0xFFF0F7FF), Color(0xFFFFFFFF)],
                animatedValue: _i(s['total_revenue_cents']) / 100.0,
                valueBuilder: (v) => '¥${v.toStringAsFixed(2)}',
              ),
              _Kpi(
                '环比',
                _ratio(momRatio, momComparable),
                _ratioDesc(momRatio, momComparable, '环比'),
                const [Color(0xFFF4FBF5), Color(0xFFFFFFFF)],
                animatedValue: (momComparable && momRatio != null) ? momRatio * 100 : null,
                valueBuilder: (v) => '${v.toStringAsFixed(2)}%',
              ),
              _Kpi(
                '同比',
                _ratio(yoyRatio, yoyComparable),
                _ratioDesc(yoyRatio, yoyComparable, '同比'),
                const [Color(0xFFFFF6EE), Color(0xFFFFFFFF)],
                animatedValue: (yoyComparable && yoyRatio != null) ? yoyRatio * 100 : null,
                valueBuilder: (v) => '${v.toStringAsFixed(2)}%',
              ),
              _Kpi(
                '当前页净额',
                '¥${(insight.$1 / 100).toStringAsFixed(2)}',
                '退款:${insight.$2} 活跃用户:${insight.$3}',
                const [Color(0xFFF5F3FF), Color(0xFFFFFFFF)],
                animatedValue: insight.$1 / 100.0,
                valueBuilder: (v) => '¥${v.toStringAsFixed(2)}',
              ),
            ])),
            const SizedBox(height: 12),
            _Panel(
              title: '收入占比（可点下钻）',
              icon: Icons.pie_chart_outline_rounded,
              action: TextButton.icon(
                onPressed: _level == 'overall' ? null : _goLevelUp,
                icon: const Icon(Icons.reply_rounded, size: 16),
                label: const Text('返回上一级'),
                style: TextButton.styleFrom(
                  padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
                  minimumSize: const Size(0, 30),
                ),
              ),
              child: _SharePanelBody(items: _top, onDrill: _drill),
            ),
            const SizedBox(height: 12),
            _Panel(title: '收入趋势', icon: Icons.show_chart_rounded, child: SizedBox(height: 216, child: trendPts.length < 2 ? const Center(child: Text('暂无趋势数据')) : _Trend(points: trendPts))),
            const SizedBox(height: 12),
            _Panel(title: 'Top 榜单', icon: Icons.leaderboard_rounded, child: DefaultTabController(length: 2, child: Column(children: [const TabBar(tabs: [Tab(text: '维度Top'), Tab(text: '用户消费榜')]), SizedBox(height: 220, child: TabBarView(children: [ListView(children: _top.take(8).map((e) => ListTile(dense: true, onTap: () => _drill(e), title: Row(children: [_Tag(_i(e['rank'])), const SizedBox(width: 8), Expanded(child: Text(_s(e['dimension_name'], '-'), maxLines: 1, overflow: TextOverflow.ellipsis))]), trailing: Text('¥${(_i(e['revenue_cents']) / 100).toStringAsFixed(2)}'))).toList()), _rankLoading ? const Center(child: CircularProgressIndicator()) : ListView(children: _userRanks.map((e) => ListTile(dense: true, onTap: () => _showUser(e.userId), title: Row(children: [_Tag(e.rank), const SizedBox(width: 8), _UserAvatar(url: _avatars[e.userId] ?? ''), const SizedBox(width: 8), Expanded(child: Text(_users[e.userId] ?? '用户${e.userId}'))]), trailing: Text('¥${(e.revenueCents / 100).toStringAsFixed(2)}'))).toList())]))]))),
            const SizedBox(height: 12),
            _Panel(title: '明细表', icon: Icons.table_rows_rounded, child: Column(children: [
              AnimatedSize(
                duration: const Duration(milliseconds: 280),
                curve: Curves.easeOutCubic,
                child: Column(
                  key: ValueKey('detail-${_detail.length}-${_loading ? 1 : 0}'),
                  children: [
                    if (_loading && _detail.isEmpty) const Padding(padding: EdgeInsets.all(24), child: CircularProgressIndicator()),
                    ..._detail.map((e) => _DetailItem(e: e, user: _users[_i(e['user_id'])] ?? '用户${_i(e['user_id'])}', avatarUrl: _avatars[_i(e['user_id'])] ?? '', onTapUser: _showUser)),
                  ],
                ),
              ),
              Row(children: [Expanded(child: Text('第 $_page / $_pages 页 · 共 $_total 条', style: Theme.of(context).textTheme.bodySmall)), OutlinedButton(onPressed: _page > 1 && !_loading ? () { setState(() => _page -= 1); _reloadDetail(); } : null, child: const Text('上一页')), const SizedBox(width: 8), OutlinedButton(onPressed: _page < _pages && !_loading ? () { setState(() => _page += 1); _reloadDetail(); } : null, child: const Text('下一页'))]),
            ])),
          ],
        ),
      ),
    );
  }

  Future<void> _showUser(int id) async {
    if (id <= 0) return;
    final rows = await _allRows(uid: id);
    final summary = _userSummary(rows);
    if (!mounted) return;
    await showModalBottomSheet<void>(
      context: context,
      isScrollControlled: true,
      showDragHandle: true,
      builder: (context) => DraggableScrollableSheet(
        expand: false,
        initialChildSize: 0.82,
        builder: (context, ctl) => ListView(
          controller: ctl,
          padding: const EdgeInsets.fromLTRB(16, 4, 16, 24),
          children: [
            Text('用户财务详情 #$id', style: Theme.of(context).textTheme.titleLarge?.copyWith(fontWeight: FontWeight.w700)),
            const SizedBox(height: 10),
            _UserSummaryCard(user: _users[id] ?? '用户$id', summary: summary),
            const SizedBox(height: 10),
            FilledButton.icon(
              onPressed: () {
                setState(() {
                  _uid = id;
                  _uidCtl.text = '$id';
                  _page = 1;
                });
                Navigator.pop(context);
                _reload();
              },
              icon: const Icon(Icons.person_search_rounded),
              label: const Text('按此用户筛选主视图'),
            ),
            const SizedBox(height: 10),
            ...rows.take(100).map((e) => _DetailItem(e: e, user: _users[id] ?? '用户$id', avatarUrl: _avatars[id] ?? '', onTapUser: null)),
          ],
        ),
      ),
    );
  }

  _UserSummary _userSummary(List<Map<String, dynamic>> rows) {
    final orders = <int, int>{};
    DateTime? last;
    for (final r in rows) {
      final oid = _i(r['order_id']);
      if (oid > 0) orders[oid] = (orders[oid] ?? 0) + _i(r['amount_cents']);
      final dt = DateTime.tryParse(_s(r['paid_at'], ''));
      if (dt != null && (last == null || dt.isAfter(last))) last = dt;
    }
    final list = orders.values.toList();
    final total = list.fold<int>(0, (p, e) => p + e);
    final avg = list.isEmpty ? 0 : (total / list.length).truncate();
    return _UserSummary(
      total: total,
      count: list.length,
      positive: list.where((e) => e > 0).length,
      negative: list.where((e) => e < 0).length,
      avg: avg,
      last: _fmt(last?.toLocal()),
    );
  }
}

class _FilterCard extends StatelessWidget {
  final List<_Opt> goods, regions, lines, pkgs;
  final int? goodsId, regionId, lineId, pkgId;
  final TextEditingController uidCtl;
  final DateTime from, to;
  final String quick, level;
  final bool loading;
  final bool expanded;
  final VoidCallback onToggleExpand;
  final Future<void> Function() onPickRange;
  final void Function(String) onQuick;
  final Future<void> Function(int?) onGoods, onRegion, onLine;
  final void Function(int?) onPkg;
  final Future<void> Function() onReset;
  final VoidCallback onSearch;
  const _FilterCard({required this.goods, required this.regions, required this.lines, required this.pkgs, required this.goodsId, required this.regionId, required this.lineId, required this.pkgId, required this.uidCtl, required this.from, required this.to, required this.quick, required this.level, required this.loading, required this.expanded, required this.onToggleExpand, required this.onPickRange, required this.onQuick, required this.onGoods, required this.onRegion, required this.onLine, required this.onPkg, required this.onReset, required this.onSearch});

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.all(12),
      decoration: BoxDecoration(color: Colors.white, borderRadius: BorderRadius.circular(14), border: Border.all(color: const Color(0xFFE5EAF2))),
      child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
        Row(children: [
          const Icon(Icons.tune_rounded, size: 18, color: Color(0xFF1E88E5)),
          const SizedBox(width: 6),
          const Text('筛选条件', style: TextStyle(fontWeight: FontWeight.w700)),
          const Spacer(),
          Text('层级: ${_levelText(level)}', style: Theme.of(context).textTheme.bodySmall),
          IconButton(
            onPressed: onToggleExpand,
            icon: Icon(expanded ? Icons.expand_less_rounded : Icons.expand_more_rounded),
            tooltip: expanded ? '收起筛选' : '展开筛选',
          ),
        ]),
        AnimatedCrossFade(
          firstChild: const SizedBox.shrink(),
          secondChild: Column(
            children: [
              const SizedBox(height: 8),
              Wrap(spacing: 8, runSpacing: 8, children: [
                _Drop(label: '类型', value: goodsId, opts: goods, onChanged: onGoods),
                _Drop(label: '地区', value: regionId, opts: regions, enabled: goodsId != null, onChanged: onRegion),
                _Drop(label: '线路', value: lineId, opts: lines, enabled: regionId != null, onChanged: onLine),
                _Drop(label: '套餐', value: pkgId, opts: pkgs, enabled: lineId != null, onChanged: (v) async => onPkg(v)),
              ]),
              const SizedBox(height: 10),
              TextField(controller: uidCtl, keyboardType: TextInputType.number, decoration: const InputDecoration(labelText: '用户ID', prefixIcon: Icon(Icons.person_outline))),
              const SizedBox(height: 10),
              OutlinedButton.icon(onPressed: onPickRange, icon: const Icon(Icons.date_range_rounded), label: Text('${from.year}-${_p(from.month)}-${_p(from.day)} ~ ${to.year}-${_p(to.month)}-${_p(to.day)}')),
              const SizedBox(height: 10),
              SizedBox(height: 38, child: ListView(scrollDirection: Axis.horizontal, children: [
                _QuickChip('今天', quick == 'today', () => onQuick('today')),
                const SizedBox(width: 8),
                _QuickChip('近7天', quick == '7d', () => onQuick('7d')),
                const SizedBox(width: 8),
                _QuickChip('近30天', quick == '30d', () => onQuick('30d')),
                const SizedBox(width: 8),
                _QuickChip('本月', quick == 'month', () => onQuick('month')),
              ])),
            ],
          ),
          crossFadeState: expanded ? CrossFadeState.showSecond : CrossFadeState.showFirst,
          duration: const Duration(milliseconds: 180),
        ),
        const SizedBox(height: 10),
        Row(children: [
          Expanded(child: OutlinedButton(onPressed: loading ? null : onReset, child: const Text('重置'))),
          const SizedBox(width: 8),
          Expanded(child: FilledButton(onPressed: loading ? null : onSearch, child: const Text('查询'))),
        ]),
      ]),
    );
  }
}

class _Drop extends StatelessWidget {
  final String label;
  final int? value;
  final List<_Opt> opts;
  final bool enabled;
  final Future<void> Function(int?) onChanged;
  const _Drop({required this.label, required this.value, required this.opts, this.enabled = true, required this.onChanged});
  @override
  Widget build(BuildContext context) {
    return SizedBox(
      width: 155,
      child: DropdownButtonFormField<int>(
        value: value,
        isExpanded: true,
        decoration: InputDecoration(labelText: label),
        items: [const DropdownMenuItem<int>(value: null, child: Text('全部')), ...opts.map((e) => DropdownMenuItem<int>(value: e.id, child: Text(e.name)))],
        onChanged: enabled ? (v) => onChanged(v) : null,
      ),
    );
  }
}

class _QuickChip extends StatelessWidget {
  final String text;
  final bool selected;
  final VoidCallback onTap;
  const _QuickChip(this.text, this.selected, this.onTap);
  @override
  Widget build(BuildContext context) {
    return InkWell(
      onTap: onTap,
      borderRadius: BorderRadius.circular(12),
      child: Container(
        padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
        decoration: BoxDecoration(color: selected ? const Color(0xFFE8F2FF) : Colors.white, borderRadius: BorderRadius.circular(12), border: Border.all(color: selected ? const Color(0xFF1E88E5) : const Color(0xFFD7DFEB))),
        child: Text(text, style: TextStyle(color: selected ? const Color(0xFF1E88E5) : null, fontWeight: FontWeight.w600, fontSize: 12.5)),
      ),
    );
  }
}

class _Panel extends StatelessWidget {
  final String title;
  final IconData icon;
  final Widget child;
  final Widget? action;
  const _Panel({required this.title, required this.icon, required this.child, this.action});
  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.all(12),
      decoration: BoxDecoration(color: Colors.white, borderRadius: BorderRadius.circular(14), border: Border.all(color: const Color(0xFFE5EAF2))),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(children: [
            Icon(icon, color: const Color(0xFF1E88E5)),
            const SizedBox(width: 6),
            Text(title, style: const TextStyle(fontWeight: FontWeight.w700)),
            const Spacer(),
            if (action != null) action!,
          ]),
          const SizedBox(height: 8),
          child,
        ],
      ),
    );
  }
}

class _SharePanelBody extends StatelessWidget {
  final List<Map<String, dynamic>> items;
  final Future<void> Function(Map<String, dynamic>) onDrill;
  const _SharePanelBody({required this.items, required this.onDrill});
  @override
  Widget build(BuildContext context) {
    if (items.isEmpty) return const Text('暂无数据');
    final top = items.take(6).toList();
    final maxVal = top.map((e) => _i(e['revenue_cents'])).fold<int>(1, math.max);
    final donutData = <_DonutSlice>[];
    for (var i = 0; i < top.length; i++) {
      final e = top[i];
      donutData.add(
        _DonutSlice(
          label: _s(e['dimension_name'], '-'),
          value: _i(e['revenue_cents']).clamp(0, 1 << 30),
          color: _palette[i % _palette.length],
        ),
      );
    }
    return AnimatedSize(
      duration: const Duration(milliseconds: 280),
      curve: Curves.easeOutCubic,
      child: Column(
        key: ValueKey('share-${top.length}-${top.fold<int>(0, (p, e) => p + _i(e['revenue_cents']))}'),
      children: [
        SizedBox(
          height: 140,
          child: Row(
            children: [
              SizedBox(
                width: 140,
                child: _DonutChart(
                  slices: donutData,
                  onSliceTap: (index) {
                    if (index >= 0 && index < top.length) {
                      onDrill(top[index]);
                    }
                  },
                ),
              ),
              const SizedBox(width: 8),
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  mainAxisAlignment: MainAxisAlignment.center,
                  children: donutData.take(4).map((d) {
                    final total = donutData.fold<int>(0, (p, e) => p + e.value).clamp(1, 1 << 30);
                    final ratio = d.value * 100 / total;
                    return Padding(
                      padding: const EdgeInsets.symmetric(vertical: 3),
                      child: Row(
                        children: [
                          Container(width: 8, height: 8, decoration: BoxDecoration(color: d.color, borderRadius: BorderRadius.circular(8))),
                          const SizedBox(width: 6),
                          Expanded(child: Text(d.label, maxLines: 1, overflow: TextOverflow.ellipsis, style: const TextStyle(fontSize: 12))),
                          Text('${ratio.toStringAsFixed(1)}%', style: const TextStyle(fontSize: 12, fontWeight: FontWeight.w600)),
                        ],
                      ),
                    );
                  }).toList(),
                ),
              ),
            ],
          ),
        ),
        ...top.map((e) {
          final v = _i(e['revenue_cents']);
          final progress = maxVal <= 0 ? 0.0 : (v / maxVal).clamp(0.0, 1.0);
          return InkWell(
            onTap: () => onDrill(e),
            child: Padding(
              padding: const EdgeInsets.only(bottom: 8),
              child: Column(
                children: [
                  Row(children: [Expanded(child: Text(_s(e['dimension_name'], '-'))), Text('¥${(v / 100).toStringAsFixed(2)}'), const SizedBox(width: 8), Text('${((_d(e['ratio']) ?? 0) * 100).toStringAsFixed(2)}%')]),
                  const SizedBox(height: 5),
                  TweenAnimationBuilder<double>(
                    key: ValueKey('bar-${_s(e['dimension_name'], '-')}-$v-${top.length}'),
                    tween: Tween<double>(begin: 0, end: progress),
                    duration: const Duration(milliseconds: 420),
                    curve: Curves.easeOutCubic,
                    builder: (context, t, _) => LinearProgressIndicator(value: t, minHeight: 6, borderRadius: BorderRadius.circular(99)),
                  ),
                ],
              ),
            ),
          );
        }),
      ],
    ));
  }
}

const _palette = <Color>[
  Color(0xFF2563EB),
  Color(0xFF06B6D4),
  Color(0xFF22C55E),
  Color(0xFFF59E0B),
  Color(0xFFEF4444),
  Color(0xFF8B5CF6),
];

class _DonutSlice {
  final String label;
  final int value;
  final Color color;
  const _DonutSlice({required this.label, required this.value, required this.color});
}

class _DonutChart extends StatelessWidget {
  final List<_DonutSlice> slices;
  final ValueChanged<int>? onSliceTap;
  const _DonutChart({required this.slices, this.onSliceTap});
  @override
  Widget build(BuildContext context) {
    if (slices.isEmpty || slices.every((e) => e.value <= 0)) {
      return const Center(child: Text('暂无数据'));
    }
    final total = slices.fold<int>(0, (p, e) => p + e.value).clamp(1, 1 << 30);
    return LayoutBuilder(
      builder: (context, constraints) {
        final size = Size(constraints.maxWidth, constraints.maxHeight);
        final sig = slices.map((e) => '${e.label}:${e.value}').join('|');
        return TweenAnimationBuilder<double>(
          key: ValueKey('donut-$sig'),
          tween: Tween<double>(begin: 0, end: 1),
          duration: const Duration(milliseconds: 420),
          curve: Curves.easeOutCubic,
          builder: (context, progress, _) => GestureDetector(
            behavior: HitTestBehavior.opaque,
            onTapUp: (details) {
              if (progress < 0.95) return;
              final idx = _hitTestSlice(details.localPosition, size, slices, total);
              if (idx != null) {
                onSliceTap?.call(idx);
              }
            },
            child: CustomPaint(
              painter: _DonutPainter(slices: slices, total: total, progress: progress),
              child: Center(
                child: Container(
                  width: 62,
                  height: 62,
                  decoration: BoxDecoration(
                    color: Colors.white,
                    borderRadius: BorderRadius.circular(99),
                    border: Border.all(color: const Color(0xFFE5EAF2)),
                  ),
                  alignment: Alignment.center,
                  child: const Text('占比', style: TextStyle(fontSize: 12, fontWeight: FontWeight.w700)),
                ),
              ),
            ),
          ),
        );
      },
    );
  }
}

int? _hitTestSlice(
  Offset pos,
  Size size,
  List<_DonutSlice> slices,
  int total,
) {
  if (slices.isEmpty || total <= 0) return null;
  final center = Offset(size.width / 2, size.height / 2);
  final dx = pos.dx - center.dx;
  final dy = pos.dy - center.dy;
  final dist = math.sqrt(dx * dx + dy * dy);
  final outerR = math.min(size.width, size.height) / 2 - 6;
  final innerR = outerR - 20; // match strokeWidth
  if (dist < innerR || dist > outerR) return null;

  var angle = math.atan2(dy, dx) + math.pi / 2;
  if (angle < 0) angle += math.pi * 2;
  var acc = 0.0;
  for (var i = 0; i < slices.length; i++) {
    final v = slices[i].value;
    if (v <= 0) continue;
    final sweep = (v / total) * math.pi * 2;
    final next = acc + sweep;
    if (angle >= acc && angle < next) return i;
    acc = next;
  }
  return slices.isEmpty ? null : slices.length - 1;
}

class _DonutPainter extends CustomPainter {
  final List<_DonutSlice> slices;
  final int total;
  final double progress;
  const _DonutPainter({required this.slices, required this.total, required this.progress});
  @override
  void paint(Canvas canvas, Size size) {
    final center = Offset(size.width / 2, size.height / 2);
    final r = math.min(size.width, size.height) / 2 - 6;
    final rect = Rect.fromCircle(center: center, radius: r);
    final paint = Paint()
      ..style = PaintingStyle.stroke
      ..strokeWidth = 20
      ..strokeCap = StrokeCap.butt;
    var start = -math.pi / 2;
    for (final s in slices) {
      if (s.value <= 0) continue;
      final sweep = (s.value / total) * math.pi * 2 * progress;
      paint.color = s.color;
      canvas.drawArc(rect, start, sweep, false, paint);
      start += sweep;
    }
  }
  @override
  bool shouldRepaint(covariant _DonutPainter oldDelegate) => oldDelegate.slices != slices || oldDelegate.total != total || oldDelegate.progress != progress;
}

class _Kpi extends StatelessWidget {
  final String title, value, subtitle;
  final List<Color> gradient;
  final double? animatedValue;
  final String Function(double)? valueBuilder;
  const _Kpi(this.title, this.value, this.subtitle, this.gradient, {this.animatedValue, this.valueBuilder});
  @override
  Widget build(BuildContext context) {
    final valueWidget = (animatedValue != null && valueBuilder != null)
        ? TweenAnimationBuilder<double>(
            key: ValueKey('kpi-$title-${animatedValue!.toStringAsFixed(2)}'),
            tween: Tween<double>(begin: 0, end: animatedValue!),
            duration: const Duration(milliseconds: 420),
            curve: Curves.easeOutCubic,
            builder: (context, v, _) => Text(valueBuilder!(v), style: const TextStyle(fontSize: 24, fontWeight: FontWeight.w800)),
          )
        : Text(value, style: const TextStyle(fontSize: 24, fontWeight: FontWeight.w800));
    return Container(
      width: 190,
      margin: const EdgeInsets.only(right: 10),
      padding: const EdgeInsets.all(12),
      decoration: BoxDecoration(gradient: LinearGradient(colors: gradient, begin: Alignment.topLeft, end: Alignment.bottomRight), borderRadius: BorderRadius.circular(14), border: Border.all(color: const Color(0xFFE5EAF2))),
      child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [Text(title, style: const TextStyle(fontWeight: FontWeight.w700)), const SizedBox(height: 10), valueWidget, const SizedBox(height: 8), Text(subtitle, style: Theme.of(context).textTheme.bodySmall)]),
    );
  }
}

class _Tag extends StatelessWidget {
  final int rank;
  const _Tag(this.rank);
  @override
  Widget build(BuildContext context) {
    final c = rank > 0 && rank <= 3 ? const Color(0xFFD97706) : const Color(0xFF64748B);
    return Container(padding: const EdgeInsets.symmetric(horizontal: 7, vertical: 3), decoration: BoxDecoration(color: c.withOpacity(0.12), borderRadius: BorderRadius.circular(99)), child: Text('#${rank <= 0 ? '-' : rank}', style: TextStyle(color: c, fontWeight: FontWeight.w700, fontSize: 11)));
  }
}

class _DetailItem extends StatelessWidget {
  final Map<String, dynamic> e;
  final String user;
  final String avatarUrl;
  final Future<void> Function(int)? onTapUser;
  const _DetailItem({required this.e, required this.user, required this.avatarUrl, required this.onTapUser});
  @override
  Widget build(BuildContext context) {
    final uid = _i(e['user_id']);
    final amount = _i(e['amount_cents']);
    final color = amount > 0 ? const Color(0xFF15803D) : amount < 0 ? const Color(0xFFB91C1C) : const Color(0xFF475569);
    final sign = amount > 0 ? '+' : amount < 0 ? '-' : '';
    return Container(
      margin: const EdgeInsets.only(bottom: 8),
      padding: const EdgeInsets.all(10),
      decoration: BoxDecoration(borderRadius: BorderRadius.circular(12), border: Border.all(color: const Color(0xFFE5EAF2))),
      child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
        Row(children: [Expanded(child: Text(_s(e['order_no'], '-'), maxLines: 1, overflow: TextOverflow.ellipsis, style: const TextStyle(fontWeight: FontWeight.w700, fontFamily: 'monospace'))), Text('$sign¥${(amount.abs() / 100).toStringAsFixed(2)}', style: TextStyle(color: color, fontWeight: FontWeight.w800))]),
        const SizedBox(height: 6),
        Row(children: [
          _UserAvatar(url: avatarUrl),
          const SizedBox(width: 8),
          Expanded(child: InkWell(onTap: onTapUser == null || uid <= 0 ? null : () => onTapUser!(uid), child: Text('用户: $user (#$uid)', style: TextStyle(color: onTapUser == null ? null : const Color(0xFF2563EB), fontWeight: FontWeight.w600)))),
          _StatusTag(_s(e['status'], '-'))
        ]),
        const SizedBox(height: 4),
        Text('支付时间: ${_fmt(DateTime.tryParse(_s(e['paid_at'], ''))?.toLocal())}', style: Theme.of(context).textTheme.bodySmall),
      ]),
    );
  }
}

class _UserAvatar extends StatelessWidget {
  final String url;
  const _UserAvatar({required this.url});
  @override
  Widget build(BuildContext context) {
    if (url.isEmpty) {
      return const CircleAvatar(
        radius: 11,
        backgroundColor: Color(0xFFE2E8F0),
        child: Icon(Icons.person, size: 12, color: Color(0xFF64748B)),
      );
    }
    return CircleAvatar(
      radius: 11,
      backgroundColor: const Color(0xFFE2E8F0),
      backgroundImage: NetworkImage(url),
      onBackgroundImageError: (_, __) {},
    );
  }
}

class _StatusTag extends StatelessWidget {
  final String status;
  const _StatusTag(this.status);
  @override
  Widget build(BuildContext context) {
    final m = _statusMeta(status);
    return Container(padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 3), decoration: BoxDecoration(color: m.$2.withOpacity(0.1), borderRadius: BorderRadius.circular(999)), child: Text(m.$1, style: TextStyle(color: m.$2, fontWeight: FontWeight.w700, fontSize: 11)));
  }
}

class _Trend extends StatefulWidget {
  final List<_Pt> points;
  const _Trend({required this.points});
  @override
  State<_Trend> createState() => _TrendState();
}

class _TrendState extends State<_Trend> {
  int? _selected;
  static const double _left = 10;
  static const double _top = 10;
  static const double _bottomPad = 20;

  void _updateSelection(Offset localPos, Size size) {
    final w = size.width - _left * 2;
    if (widget.points.length < 2 || w <= 0) return;
    final raw = ((localPos.dx - _left) / w) * (widget.points.length - 1);
    final idx = raw.round().clamp(0, widget.points.length - 1);
    setState(() => _selected = idx);
  }

  @override
  Widget build(BuildContext context) {
    final pts = widget.points;
    final first = pts.isEmpty ? '-' : pts.first.label;
    final mid = pts.isEmpty ? '-' : pts[(pts.length / 2).floor()].label;
    final last = pts.isEmpty ? '-' : pts.last.label;
    return LayoutBuilder(
      builder: (context, c) {
        final size = Size(c.maxWidth, c.maxHeight - _bottomPad);
        final sig = pts.map((e) => '${e.label}:${e.v.toStringAsFixed(2)}').join('|');
        return TweenAnimationBuilder<double>(
          key: ValueKey('trend-$sig'),
          tween: Tween<double>(begin: 0, end: 1),
          duration: const Duration(milliseconds: 460),
          curve: Curves.easeOutCubic,
          builder: (context, progress, _) {
            final sel = (_selected != null && _selected! >= 0 && _selected! < pts.length) ? pts[_selected!] : null;
            final marker = (sel == null) ? null : _pointAt(pts, _selected!, size, progress);
            return GestureDetector(
              behavior: HitTestBehavior.opaque,
              onTapDown: (d) => _updateSelection(d.localPosition, size),
              onHorizontalDragStart: (d) => _updateSelection(d.localPosition, size),
              onHorizontalDragUpdate: (d) => _updateSelection(d.localPosition, size),
              onLongPressStart: (d) => _updateSelection(d.localPosition, size),
              child: Stack(
                children: [
                  Positioned.fill(
                    child: Padding(
                      padding: const EdgeInsets.only(bottom: _bottomPad),
                      child: CustomPaint(
                        painter: _TrendPainter(pts, selectedIndex: _selected, progress: progress),
                      ),
                    ),
                  ),
                  if (sel != null && marker != null)
                    Positioned(
                      left: (marker.dx - 56).clamp(2, c.maxWidth - 112),
                      top: (marker.dy - 46).clamp(2, size.height - 44),
                      child: Container(
                        width: 112,
                        padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 6),
                        decoration: BoxDecoration(
                          color: const Color(0xEE0F172A),
                          borderRadius: BorderRadius.circular(8),
                        ),
                        child: Column(
                          crossAxisAlignment: CrossAxisAlignment.start,
                          children: [
                            Text(sel.label, style: const TextStyle(color: Colors.white, fontSize: 11)),
                            Text('¥${(sel.v * progress).toStringAsFixed(2)}', style: const TextStyle(color: Colors.white, fontWeight: FontWeight.w700)),
                          ],
                        ),
                      ),
                    ),
                  Positioned(
                    left: 0,
                    right: 0,
                    bottom: 0,
                    child: Row(
                      children: [
                        Expanded(child: Text(first, style: const TextStyle(fontSize: 11, color: Color(0xFF64748B)), overflow: TextOverflow.ellipsis)),
                        Expanded(child: Text(mid, textAlign: TextAlign.center, style: const TextStyle(fontSize: 11, color: Color(0xFF64748B)), overflow: TextOverflow.ellipsis)),
                        Expanded(child: Text(last, textAlign: TextAlign.right, style: const TextStyle(fontSize: 11, color: Color(0xFF64748B)), overflow: TextOverflow.ellipsis)),
                      ],
                    ),
                  ),
                ],
              ),
            );
          },
        );
      },
    );
  }

  Offset? _pointAt(List<_Pt> pts, int? idx, Size s, double progress) {
    if (idx == null || pts.length < 2) return null;
    final w = s.width - _left * 2;
    final h = s.height - _top * 2;
    final anim = pts.map((e) => e.v * progress).toList();
    final maxV = anim.reduce(math.max);
    final minV = anim.reduce(math.min);
    final span = math.max(0.001, maxV - minV);
    final x = _left + w * idx / (pts.length - 1);
    final y = _top + h - ((pts[idx].v * progress) - minV) / span * h;
    return Offset(x, y);
  }
}

class _TrendPainter extends CustomPainter {
  final List<_Pt> p;
  final int? selectedIndex;
  final double progress;
  _TrendPainter(this.p, {this.selectedIndex, required this.progress});
  @override
  void paint(Canvas c, Size s) {
    if (p.length < 2) return;
    const l = 10.0, t = 10.0;
    final w = s.width - l * 2, h = s.height - t * 2;
    final values = p.map((e) => e.v * progress).toList();
    final maxV = values.reduce(math.max), minV = values.reduce(math.min), span = math.max(0.001, maxV - minV);
    final grid = Paint()..color = const Color(0xFFE6EDF7);
    for (var i = 0; i < 4; i++) {
      final y = t + h * i / 3;
      c.drawLine(Offset(l, y), Offset(l + w, y), grid);
    }
    final path = Path();
    for (var i = 0; i < p.length; i++) {
      final x = l + w * i / (p.length - 1), y = t + h - ((p[i].v * progress) - minV) / span * h;
      if (i == 0) {
        path.moveTo(x, y);
      } else {
        path.lineTo(x, y);
      }
    }
    final fill = Path.from(path)..lineTo(l + w, t + h)..lineTo(l, t + h)..close();
    c.drawPath(fill, Paint()..shader = const LinearGradient(colors: [Color(0x553B82F6), Color(0x003B82F6)], begin: Alignment.topCenter, end: Alignment.bottomCenter).createShader(Rect.fromLTWH(l, t, w, h)));
    c.drawPath(path, Paint()..color = const Color(0xFF2563EB)..strokeWidth = 2.2..style = PaintingStyle.stroke);
    if (selectedIndex != null && selectedIndex! >= 0 && selectedIndex! < p.length) {
      final i = selectedIndex!;
      final x = l + w * i / (p.length - 1), y = t + h - ((p[i].v * progress) - minV) / span * h;
      c.drawCircle(Offset(x, y), 5, Paint()..color = Colors.white);
      c.drawCircle(Offset(x, y), 3.2, Paint()..color = const Color(0xFF2563EB));
    }
  }
  @override
  bool shouldRepaint(covariant _TrendPainter oldDelegate) => oldDelegate.p != p || oldDelegate.selectedIndex != selectedIndex || oldDelegate.progress != progress;
}

class _UserSummaryCard extends StatelessWidget {
  final String user;
  final _UserSummary summary;
  const _UserSummaryCard({required this.user, required this.summary});
  @override
  Widget build(BuildContext context) {
    Widget kv(String k, String v) => Padding(padding: const EdgeInsets.symmetric(vertical: 3), child: Row(children: [SizedBox(width: 96, child: Text(k, style: const TextStyle(color: Color(0xFF475569)))), Expanded(child: Text(v, style: const TextStyle(fontWeight: FontWeight.w600)))]));
    return Container(
      padding: const EdgeInsets.all(12),
      decoration: BoxDecoration(color: const Color(0xFFF8FAFD), borderRadius: BorderRadius.circular(12), border: Border.all(color: const Color(0xFFE5EAF2))),
      child: Column(children: [kv('用户', user), kv('总收入贡献', '¥${(summary.total / 100).toStringAsFixed(2)}'), kv('订单数', '${summary.count}'), kv('净收入订单', '${summary.positive}'), kv('退款/负数订单', '${summary.negative}'), kv('客单价', '¥${(summary.avg / 100).toStringAsFixed(2)}'), kv('最近支付时间', summary.last)]),
    );
  }
}

class _Opt { final int id; final String name; const _Opt(this.id, this.name); }
class _Agg { int sum = 0; final Set<int> orders = {}; }
class _UserRank {
  final int rank, userId, revenueCents, orderCount;
  const _UserRank({this.rank = 0, required this.userId, required this.revenueCents, required this.orderCount});
  _UserRank copyWith({int? rank, int? userId, int? revenueCents, int? orderCount}) => _UserRank(rank: rank ?? this.rank, userId: userId ?? this.userId, revenueCents: revenueCents ?? this.revenueCents, orderCount: orderCount ?? this.orderCount);
}
class _Pt { final String label; final double v; const _Pt(this.label, this.v); }
class _UserSummary { final int total, count, positive, negative, avg; final String last; const _UserSummary({required this.total, required this.count, required this.positive, required this.negative, required this.avg, required this.last}); }

List<Map<String, dynamic>> _ls(dynamic v) => v is List ? v.map((e) => _m(e)).where((e) => e.isNotEmpty).toList() : const [];
Map<String, dynamic> _m(dynamic v, [Map<String, dynamic>? fb]) => v is Map<String, dynamic> ? v : v is Map ? v.map((k, val) => MapEntry(k.toString(), val)) : (fb ?? <String, dynamic>{});
int _i(dynamic v) => v is int ? v : v is num ? v.toInt() : v is String ? int.tryParse(v) ?? 0 : 0;
double? _d(dynamic v) => v == null ? null : v is double ? v : v is int ? v.toDouble() : v is num ? v.toDouble() : v is String ? double.tryParse(v) : null;
bool _b(dynamic v) => v is bool ? v : v is num ? v != 0 : v is String ? ['1', 'true', 'yes'].contains(v.toLowerCase()) : false;
String _s(dynamic v, String fb) { if (v == null) return fb; final t = v.toString().trim(); return t.isEmpty ? fb : t; }
String _ratio(double? r, bool ok) => (!ok || r == null) ? '不可比' : '${(r * 100).toStringAsFixed(2)}%';
String _ratioDesc(double? r, bool ok, String p) => (!ok || r == null) ? '$p: 不可比' : r > 0 ? '$p: 上升 ${(r * 100).toStringAsFixed(2)}%' : r < 0 ? '$p: 下降 ${(r.abs() * 100).toStringAsFixed(2)}%' : '$p: 持平';
String _levelText(String l) => l == 'goods_type' ? '类型' : l == 'region' ? '地区' : l == 'line' ? '线路' : l == 'package' ? '套餐' : '整体';
(String, Color) _statusMeta(String s) {
  switch (s.toLowerCase()) {
    case 'approved':
    case 'active': return ('已完成', const Color(0xFF15803D));
    case 'pending_payment': return ('待支付', const Color(0xFF2563EB));
    case 'pending_review': return ('待审核', const Color(0xFFD97706));
    case 'canceled': return ('已取消', const Color(0xFF64748B));
    case 'failed':
    case 'rejected': return ('失败', const Color(0xFFB91C1C));
    default: return (s.isEmpty ? '-' : s, const Color(0xFF64748B));
  }
}
String _fmt(DateTime? dt) => dt == null ? '-' : '${dt.year}-${_p(dt.month)}-${_p(dt.day)} ${_p(dt.hour)}:${_p(dt.minute)}:${_p(dt.second)}';
String _p(int n) => n.toString().padLeft(2, '0');
(int, int, int) _detailInsight(List<Map<String, dynamic>> rows) {
  final orders = <int, int>{}, users = <int>{};
  for (final r in rows) {
    final oid = _i(r['order_id']);
    if (oid > 0) orders[oid] = (orders[oid] ?? 0) + _i(r['amount_cents']);
    final uid = _i(r['user_id']);
    if (uid > 0) users.add(uid);
  }
  return (orders.values.fold(0, (p, e) => p + e), orders.values.where((e) => e < 0).length, users.length);
}
