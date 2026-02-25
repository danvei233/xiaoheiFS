import 'dart:async';
import 'dart:convert';
import 'dart:io';

import 'package:flutter/material.dart';
import 'package:http/http.dart' as http;
import 'package:path_provider/path_provider.dart';
import 'package:provider/provider.dart';

import '../app_state.dart';
import '../services/api_client.dart';
import '../services/avatar.dart';
import 'users_screen.dart';
import 'wallet_orders_screen.dart';
import 'tickets_screen.dart';
import 'audit_logs_screen.dart';
import 'scheduled_tasks_screen.dart';
import 'payment_providers_screen.dart';
import 'settings_kv_screen.dart';
import 'catalog/catalog_hub_screen.dart';
import 'permissions_screen.dart';
import 'root_tab_switch_notification.dart';
import 'settings_screen.dart';
import 'probes_screen.dart';
import 'realname_records_screen.dart';
import 'revenue_analytics_screen.dart';

class HomeScreen extends StatefulWidget {
  const HomeScreen({super.key});

  @override
  State<HomeScreen> createState() => _HomeScreenState();
}

class _HomeScreenState extends State<HomeScreen> {
  Future<DashboardData>? _future;
  ApiClient? _client;
  DateTime? _lastBackPressedAt;

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

  Future<DashboardData> _load(ApiClient client) async {
    final overview = await client.postJson('/admin/api/v1/dashboard/overview');
    final usersTotal = await _safeTotal(client, '/admin/api/v1/users');
    final walletTotal = await _safeTotal(client, '/admin/api/v1/wallet/orders');
    return DashboardData(
      totalOrders: _asInt(overview['total_orders']),
      pendingReview: _asInt(overview['pending_review']),
      revenueCents: _asInt(overview['revenue']),
      vpsCount: _asInt(overview['vps_count']),
      expiringSoon: _asInt(overview['expiring_soon']),
      usersTotal: usersTotal,
      walletOrdersTotal: walletTotal,
    );
  }

  Future<int?> _safeTotal(ApiClient client, String path) async {
    try {
      final resp = await client.getJson(path, query: {'limit': '1'});
      return _asInt(resp['total']);
    } catch (_) {
      return null;
    }
  }

  int _asInt(dynamic value) {
    if (value is int) return value;
    if (value is double) return value.toInt();
    if (value is String) return int.tryParse(value) ?? 0;
    return 0;
  }

  String _formatAmount(int cents) {
    final amount = cents / 100.0;
    return '¥${amount.toStringAsFixed(2)}';
  }

  Future<bool> _onWillPop() async {
    final now = DateTime.now();
    if (_lastBackPressedAt == null ||
        now.difference(_lastBackPressedAt!) > const Duration(seconds: 2)) {
      _lastBackPressedAt = now;
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(content: Text('请再次按返回退出')),
      );
      return false;
    }
    return true;
  }

  @override
  Widget build(BuildContext context) {
    return WillPopScope(
      onWillPop: _onWillPop,
      child: Scaffold(
        floatingActionButton: FloatingActionButton(
          onPressed: () => Navigator.push(
            context,
            MaterialPageRoute(builder: (_) => const SettingsScreen()),
          ),
          child: const Icon(Icons.settings),
        ),
        body: FutureBuilder<DashboardData>(
          future: _future,
          builder: (context, snapshot) {
            if (snapshot.connectionState == ConnectionState.waiting) {
              return const Center(child: CircularProgressIndicator());
            }
            if (snapshot.hasError) {
              return _ErrorState(
                message: '加载概览失败，请检查 API Key 权限或 API 地址。',
                onRetry: () {
                  final client = context.read<AppState>().apiClient;
                  if (client != null) {
                    setState(() {
                      _future = _load(client);
                    });
                  }
                },
              );
            }
            final data = snapshot.data;
            if (data == null) {
              return const _EmptyState();
            }

            return RefreshIndicator(
              onRefresh: () async {
                final client = context.read<AppState>().apiClient;
                if (client != null) {
                  setState(() {
                    _future = _load(client);
                  });
                }
                await _future;
              },
              child: CustomScrollView(
                slivers: [
                  _HeaderSliver(data: data),
                  SliverToBoxAdapter(
                    child: _StatsSection(data: data, formatAmount: _formatAmount),
                  ),
                  const SliverToBoxAdapter(child: _QuickEntrySection()),
                  const SliverToBoxAdapter(child: _ManagementSection()),
                  const SliverToBoxAdapter(child: SizedBox(height: 100)),
                ],
              ),
            );
          },
        ),
      ),
    );
  }
}

class _HeaderSliver extends StatefulWidget {
  final DashboardData data;

  const _HeaderSliver({required this.data});

  @override
  State<_HeaderSliver> createState() => _HeaderSliverState();
}

class _HeaderSliverState extends State<_HeaderSliver> {
  Future<Map<String, dynamic>>? _future;
  Future<String>? _siteNameFuture;
  ApiClient? _client;
  bool _bingLoaded = false;
  List<dynamic> _bingImages = const [];
  int _bannerIndex = 0;
  Timer? _bannerTimer;
  Directory? _bannerCacheDir;
  final Map<String, String> _cachedPaths = {};
  final Set<String> _cacheInFlight = <String>{};
  static const int _maxCacheFiles = 10;
  static const int _maxCacheBytes = 22 * 1024 * 1024;

  @override
  void dispose() {
    _bannerTimer?.cancel();
    super.dispose();
  }

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    final client = context.read<AppState>().apiClient;
    if (client?.baseUrl != _client?.baseUrl ||
        client?.apiKey != _client?.apiKey ||
        client?.token != _client?.token) {
      _client = client;
      if (client != null) {
        _future = client.getJson('/admin/api/v1/profile');
        _siteNameFuture = _loadSiteName(client);
        if (!_bingLoaded) {
          _bingLoaded = true;
          _loadBingImages();
        }
      }
    }
  }

  Future<void> _loadBingImages() async {
    try {
      final uri = Uri.parse(
        'https://cn.bing.com/HPImageArchive.aspx?format=js&idx=0&n=8&mkt=zh-CN',
      );
      final resp = await http.get(uri).timeout(const Duration(seconds: 8));
      if (resp.statusCode < 200 || resp.statusCode >= 300) return;
      final json = jsonDecode(resp.body) as Map<String, dynamic>;
      final images = (json['images'] as List<dynamic>? ?? [])
          .whereType<Map<String, dynamic>>()
          .map(_mapBingBanner)
          .where((e) => e.imageUrl.isNotEmpty)
          .toSet()
          .toList();
      if (!mounted || images.isEmpty) return;
      setState(() {
        _bingImages = images;
        _bannerIndex = 0;
      });
      _prefetchBannerImages();
      _startBannerAutoPlay();
    } catch (_) {}
  }

  _BingBanner _mapBingBanner(Map<String, dynamic> row) {
    final urlBase = (row['urlbase'] ?? '').toString();
    final rawUrl = (row['url'] ?? '').toString();
    final title = (row['title'] ?? row['copyright'] ?? '').toString().trim();
    String imageUrl = '';
    if (urlBase.isNotEmpty) {
      // Use lower resolution first to reduce bandwidth and improve switch smoothness.
      imageUrl = 'https://cn.bing.com${urlBase}_1366x768.jpg';
    } else if (rawUrl.isNotEmpty) {
      imageUrl = rawUrl.startsWith('http')
          ? rawUrl
          : 'https://cn.bing.com$rawUrl';
    }
    return _BingBanner(imageUrl: imageUrl, title: title);
  }

  void _startBannerAutoPlay() {
    _bannerTimer?.cancel();
    if (_bingImages.length <= 1) return;
    _bannerTimer = Timer.periodic(const Duration(seconds: 9), (_) {
      if (!mounted) return;
      _shiftBanner(1);
    });
  }

  void _shiftBanner(int delta) {
    if (_bingImages.length <= 1) return;
    setState(() {
      _bannerIndex = (_bannerIndex + delta) % _bingImages.length;
      if (_bannerIndex < 0) {
        _bannerIndex += _bingImages.length;
      }
    });
    _prefetchBannerImages();
  }

  _BingBanner _bannerAt(int index) {
    const fallback = _BingBanner(
      imageUrl:
          'https://cn.bing.com/th?id=OHR.MalaysiaTea_ZH-CN5756169294_1920x1080.jpg&rf=LaDigue_1920x1080.jpg&pid=hp',
      title: '',
    );
    if (_bingImages.isEmpty) return fallback;
    final raw = _bingImages[index % _bingImages.length];
    if (raw is _BingBanner) return raw;
    if (raw is String && raw.isNotEmpty) {
      return _BingBanner(imageUrl: raw, title: '');
    }
    return fallback;
  }

  Future<void> _initBannerCache() async {
    if (_bannerCacheDir != null) return;
    try {
      final root = await getTemporaryDirectory();
      final dir = Directory(
        '${root.path}${Platform.pathSeparator}bing_banner_cache',
      );
      if (!await dir.exists()) {
        await dir.create(recursive: true);
      }
      _bannerCacheDir = dir;
      await _pruneBannerCache();
    } catch (_) {}
  }

  String _cacheFileName(String url) {
    final safe = base64UrlEncode(utf8.encode(url)).replaceAll('=', '');
    return '$safe.img';
  }

  File? _cacheFileFor(String url) {
    final dir = _bannerCacheDir;
    if (dir == null) return null;
    return File('${dir.path}${Platform.pathSeparator}${_cacheFileName(url)}');
  }

  Future<void> _prefetchBannerImages() async {
    await _initBannerCache();
    if (_bingImages.isEmpty) return;
    final targets = <_BingBanner>[];
    for (var i = 0; i < 4 && i < _bingImages.length; i++) {
      final idx = (_bannerIndex + i) % _bingImages.length;
      targets.add(_bannerAt(idx));
    }
    for (final t in targets) {
      await _ensureBannerCached(t.imageUrl);
    }
  }

  Future<void> _ensureBannerCached(String url) async {
    if (url.isEmpty || _cachedPaths.containsKey(url)) return;
    await _initBannerCache();
    final file = _cacheFileFor(url);
    if (file == null) return;
    try {
      if (await file.exists()) {
        _cachedPaths[url] = file.path;
        await file.setLastModified(DateTime.now());
        if (mounted) setState(() {});
        return;
      }
      if (_cacheInFlight.contains(url)) return;
      _cacheInFlight.add(url);
      final resp = await http
          .get(Uri.parse(url))
          .timeout(const Duration(seconds: 12));
      if (resp.statusCode >= 200 &&
          resp.statusCode < 300 &&
          resp.bodyBytes.isNotEmpty &&
          resp.bodyBytes.length <= 4 * 1024 * 1024) {
        await file.writeAsBytes(resp.bodyBytes, flush: true);
        _cachedPaths[url] = file.path;
        await _pruneBannerCache();
        if (mounted) setState(() {});
      }
    } catch (_) {
    } finally {
      _cacheInFlight.remove(url);
    }
  }

  Future<void> _pruneBannerCache() async {
    final dir = _bannerCacheDir;
    if (dir == null || !await dir.exists()) return;
    final files = await dir
        .list()
        .where((e) => e is File)
        .cast<File>()
        .toList();
    final stats = <({File file, FileStat stat})>[];
    var totalBytes = 0;
    for (final f in files) {
      try {
        final s = await f.stat();
        stats.add((file: f, stat: s));
        totalBytes += s.size;
      } catch (_) {}
    }
    stats.sort((a, b) => a.stat.modified.compareTo(b.stat.modified));
    while (stats.length > _maxCacheFiles || totalBytes > _maxCacheBytes) {
      final victim = stats.removeAt(0);
      try {
        await victim.file.delete();
      } catch (_) {}
      totalBytes -= victim.stat.size;
      _cachedPaths.removeWhere((_, path) => path == victim.file.path);
    }
  }

  Widget _buildBannerImage(String url, int cacheWidth) {
    _ensureBannerCached(url);
    final localPath = _cachedPaths[url];
    if (localPath != null) {
      final file = File(localPath);
      return Image.file(
        file,
        fit: BoxFit.cover,
        alignment: Alignment.center,
        gaplessPlayback: true,
        filterQuality: FilterQuality.medium,
        errorBuilder: (context, error, stack) => _bannerFallback(),
      );
    }
    return Image.network(
      url,
      fit: BoxFit.cover,
      alignment: Alignment.center,
      gaplessPlayback: true,
      filterQuality: FilterQuality.medium,
      cacheWidth: cacheWidth > 0 ? cacheWidth : null,
      loadingBuilder: (context, child, progress) {
        if (progress == null) return child;
        return _bannerFallback();
      },
      errorBuilder: (context, error, stack) => _bannerFallback(),
    );
  }

  Widget _bannerFallback() {
    return Container(
      decoration: const BoxDecoration(
        gradient: LinearGradient(
          colors: [Color(0xFF00796B), Color(0xFF005B4F)],
          begin: Alignment.topLeft,
          end: Alignment.bottomRight,
        ),
      ),
    );
  }

  Future<String> _loadSiteName(ApiClient client) async {
    try {
      final resp = await client.getJson('/admin/api/v1/settings');
      final items = (resp['items'] as List<dynamic>? ?? [])
          .whereType<Map<String, dynamic>>()
          .toList();
      String? pickByKey(String key) {
        for (final item in items) {
          if ((item['key']?.toString() ?? '').toLowerCase() == key) {
            return item['value']?.toString();
          }
        }
        return null;
      }

      final candidates = [
        pickByKey('site_name'),
        pickByKey('site_title'),
        pickByKey('web_name'),
        pickByKey('web_title'),
        pickByKey('website_name'),
        pickByKey('title'),
        pickByKey('name'),
      ];
      for (final c in candidates) {
        if (c != null && c.trim().isNotEmpty) return c.trim();
      }
    } catch (_) {}
    return '';
  }

  @override
  Widget build(BuildContext context) {
    final session = context.read<AppState>().session;
    final username = session?.username ?? '管理员';
    final topPadding = MediaQuery.of(context).padding.top;
    final media = MediaQuery.of(context);
    final currentBanner = _bannerAt(_bannerIndex);
    final cacheWidth = (media.size.width * media.devicePixelRatio * 2).round();

    return SliverAppBar(
      expandedHeight: 260,
      pinned: true,
      floating: false,
      toolbarHeight: 56,
      backgroundColor: Colors.transparent,
      surfaceTintColor: Colors.transparent,
      elevation: 0,
      flexibleSpace: FlexibleSpaceBar(
        collapseMode: CollapseMode.parallax,
        background: GestureDetector(
          behavior: HitTestBehavior.opaque,
          onHorizontalDragEnd: (details) {
            final vx = details.primaryVelocity ?? 0;
            if (vx < -120) {
              _shiftBanner(1);
              _startBannerAutoPlay();
            } else if (vx > 120) {
              _shiftBanner(-1);
              _startBannerAutoPlay();
            }
          },
          child: Stack(
            fit: StackFit.expand,
            children: [
              AnimatedSwitcher(
                duration: const Duration(milliseconds: 700),
                switchInCurve: Curves.easeOutCubic,
                switchOutCurve: Curves.easeInCubic,
                layoutBuilder: (currentChild, previousChildren) {
                  return Stack(
                    fit: StackFit.expand,
                    children: [
                      ...previousChildren,
                      if (currentChild != null) currentChild,
                    ],
                  );
                },
                child: ClipRect(
                  key: ValueKey(currentBanner.imageUrl),
                  child: Transform.scale(
                    scale: 1.08,
                    child: _buildBannerImage(currentBanner.imageUrl, cacheWidth),
                  ),
                ),
              ),
              Container(
                decoration: const BoxDecoration(
                  gradient: LinearGradient(
                    colors: [
                      Color(0xD9001B18),
                      Color(0x9941342A),
                      Color(0x33009E84),
                    ],
                    begin: Alignment.topLeft,
                    end: Alignment.bottomRight,
                  ),
                ),
              ),
              SafeArea(
                top: false,
                bottom: false,
                child: Padding(
                  padding: EdgeInsets.fromLTRB(16, topPadding + 8, 16, 12),
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Row(
                        children: [
                          const Spacer(),
                          FutureBuilder<Map<String, dynamic>>(
                            future: _future,
                            builder: (context, snapshot) {
                              final profile = snapshot.data ?? {};
                              final baseUrl = _client?.baseUrl ?? '';
                              final avatarUrl = resolveAvatarUrl(
                                baseUrl: baseUrl,
                                qq: profile['qq']?.toString(),
                                avatarUrl: profile['avatar_url']?.toString(),
                              );
                              final headers = avatarHeaders(
                                token: session?.token,
                                apiKey: session?.apiKey,
                              );
                              return CircleAvatar(
                                radius: 18,
                                backgroundColor: Colors.white24,
                                child: avatarUrl.isNotEmpty
                                    ? ClipOval(
                                        child: Image.network(
                                          avatarUrl,
                                          width: 36,
                                          height: 36,
                                          fit: BoxFit.cover,
                                          headers: headers.isEmpty
                                              ? null
                                              : headers,
                                          errorBuilder: (context, error, stack) {
                                            return const Icon(
                                              Icons.person,
                                              color: Colors.white,
                                            );
                                          },
                                        ),
                                      )
                                    : Text(
                                        username.isNotEmpty
                                            ? username.characters.first
                                            : '?',
                                        style: const TextStyle(
                                          color: Colors.white,
                                          fontWeight: FontWeight.bold,
                                          fontSize: 16,
                                        ),
                                      ),
                              );
                            },
                          ),
                        ],
                      ),
                      const Spacer(),
                      FutureBuilder<String>(
                        future: _siteNameFuture,
                        builder: (context, snapshot) {
                          final name = snapshot.data;
                          return Text(
                            (name == null || name.isEmpty) ? '小黑云' : name,
                            style: const TextStyle(
                              fontSize: 22,
                              fontWeight: FontWeight.w700,
                              color: Colors.white,
                            ),
                          );
                        },
                      ),
                      const SizedBox(height: 6),
                      Text(
                        '欢迎回来，$username~',
                        style: const TextStyle(
                          fontSize: 16,
                          color: Colors.white,
                          fontWeight: FontWeight.w500,
                        ),
                      ),
                      const SizedBox(height: 8),
                      Row(
                        children: [
                          Container(
                            padding: const EdgeInsets.symmetric(
                              horizontal: 8,
                              vertical: 4,
                            ),
                            decoration: BoxDecoration(
                              color: Colors.white.withOpacity(0.16),
                              borderRadius: BorderRadius.circular(999),
                              border: Border.all(color: Colors.white30),
                            ),
                            child: const Text(
                              'Bing 每日图',
                              style: TextStyle(
                                color: Colors.white,
                                fontSize: 11,
                                fontWeight: FontWeight.w600,
                              ),
                            ),
                          ),
                          const SizedBox(width: 8),
                          if (_bingImages.length > 1)
                            Builder(
                              builder: (context) {
                                final dotCount = _bingImages.length > 6
                                    ? 6
                                    : _bingImages.length;
                                final selected = _bannerIndex % dotCount;
                                return Row(
                                  children: List.generate(
                                    dotCount,
                                    (i) => AnimatedContainer(
                                      duration: const Duration(milliseconds: 300),
                                      width: i == selected ? 12 : 6,
                                      height: 6,
                                      margin: const EdgeInsets.only(right: 4),
                                      decoration: BoxDecoration(
                                        color: i == selected
                                            ? Colors.white
                                            : Colors.white54,
                                        borderRadius: BorderRadius.circular(99),
                                      ),
                                    ),
                                  ),
                                );
                              },
                            ),
                        ],
                      ),
                      if (currentBanner.title.isNotEmpty) ...[
                        const SizedBox(height: 6),
                        Text(
                          currentBanner.title,
                          maxLines: 1,
                          overflow: TextOverflow.ellipsis,
                          style: const TextStyle(
                            color: Colors.white70,
                            fontSize: 11,
                          ),
                        ),
                      ],
                    ],
                  ),
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}

class _StatsSection extends StatelessWidget {
  final DashboardData data;
  final String Function(int) formatAmount;

  const _StatsSection({required this.data, required this.formatAmount});

  @override
  Widget build(BuildContext context) {
    final stats = [
      _StatItem(
        '待审核',
        '${data.pendingReview}',
        Icons.pending,
        const Color(0xFFFF6B6B),
        preferredTabIndex: 1,
      ),
      _StatItem(
        '用户数',
        '${data.usersTotal ?? '--'}',
        Icons.people,
        const Color(0xFF4ECDC4),
        preferredTabIndex: 2,
      ),
      _StatItem(
        '服务器',
        '${data.vpsCount}',
        Icons.dns,
        const Color(0xFF45B7D1),
        preferredTabIndex: 3,
      ),
      _StatItem(
        '钱包订单',
        '${data.walletOrdersTotal ?? '--'}',
        Icons.account_balance_wallet,
        const Color(0xFF96CEB4),
        fallbackScreen: const WalletOrdersScreen(),
      ),
      _StatItem(
        '累计收入',
        formatAmount(data.revenueCents),
        Icons.trending_up,
        const Color(0xFFFFA94D),
        fallbackScreen: const RevenueAnalyticsScreen(),
      ),
    ];

    return Container(
      margin: const EdgeInsets.fromLTRB(12, 16, 12, 8),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          const Padding(
            padding: EdgeInsets.symmetric(horizontal: 4),
            child: Row(
              children: [
                Icon(
                  Icons.insights_outlined,
                  size: 18,
                  color: Color(0xFF1E88E5),
                ),
                SizedBox(width: 6),
                Text(
                  '数据概览',
                  style: TextStyle(fontSize: 17, fontWeight: FontWeight.bold),
                ),
              ],
            ),
          ),
          const SizedBox(height: 12),
          SizedBox(
            height: 120,
            child: ListView.separated(
              scrollDirection: Axis.horizontal,
              itemCount: stats.length,
              padding: const EdgeInsets.symmetric(horizontal: 4),
              separatorBuilder: (_, i) => const SizedBox(width: 10),
              itemBuilder: (context, index) {
                final item = stats[index];
                return Material(
                  color: Colors.transparent,
                  child: InkWell(
                    borderRadius: BorderRadius.circular(16),
                    onTap: () => _openStatTarget(context, item),
                    child: Container(
                      width: 126,
                      padding: const EdgeInsets.all(14),
                      decoration: BoxDecoration(
                        gradient: LinearGradient(
                          begin: Alignment.topLeft,
                          end: Alignment.bottomRight,
                          colors: [item.color, item.color.withValues(alpha: 0.85)],
                        ),
                        borderRadius: BorderRadius.circular(16),
                        border: Border.all(
                          color: Colors.white.withOpacity(0.28),
                          width: 0.8,
                        ),
                        boxShadow: [
                          BoxShadow(
                            color: item.color.withValues(alpha: 0.3),
                            blurRadius: 14,
                            offset: const Offset(0, 4),
                          ),
                        ],
                      ),
                      child: Column(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        mainAxisAlignment: MainAxisAlignment.spaceBetween,
                        children: [
                          Container(
                            padding: const EdgeInsets.all(6),
                            decoration: BoxDecoration(
                              color: Colors.white.withValues(alpha: 0.2),
                              borderRadius: BorderRadius.circular(10),
                            ),
                            child: Icon(item.icon, color: Colors.white, size: 18),
                          ),
                          Column(
                            crossAxisAlignment: CrossAxisAlignment.start,
                            children: [
                              Text(
                                item.value,
                                style: const TextStyle(
                                  fontSize: 24,
                                  fontWeight: FontWeight.bold,
                                  color: Colors.white,
                                ),
                              ),
                              Text(
                                item.label,
                                style: const TextStyle(
                                  fontSize: 12,
                                  color: Colors.white,
                                  fontWeight: FontWeight.w500,
                                ),
                              ),
                            ],
                          ),
                        ],
                      ),
                    ),
                  ),
                );
              },
            ),
          ),
        ],
      ),
    );
  }

  void _openStatTarget(BuildContext context, _StatItem item) {
    if (item.preferredTabIndex != null) {
      RootTabSwitchNotification(item.preferredTabIndex!).dispatch(context);
      return;
    }
    if (item.fallbackScreen != null) {
      _pushStatFallback(context, item.fallbackScreen!);
    }
  }

  void _pushStatFallback(BuildContext context, Widget screen) {
    Navigator.of(context).push(
      PageRouteBuilder<void>(
        transitionDuration: const Duration(milliseconds: 320),
        reverseTransitionDuration: const Duration(milliseconds: 260),
        pageBuilder: (context, animation, secondaryAnimation) => screen,
        transitionsBuilder: (context, animation, secondaryAnimation, child) {
          final fade = CurvedAnimation(
            parent: animation,
            curve: Curves.easeOutCubic,
          );
          final slide = Tween<Offset>(
            begin: const Offset(0.08, 0),
            end: Offset.zero,
          ).animate(
            CurvedAnimation(parent: animation, curve: Curves.easeOutCubic),
          );
          return FadeTransition(
            opacity: fade,
            child: SlideTransition(position: slide, child: child),
          );
        },
      ),
    );
  }
}

class _QuickEntrySection extends StatelessWidget {
  const _QuickEntrySection();

  @override
  Widget build(BuildContext context) {
    final entries = [
      _EntryItem(
        '钱包订单',
        Icons.account_balance_wallet,
        const WalletOrdersScreen(),
      ),
      _EntryItem('财务统计', Icons.analytics_outlined, const RevenueAnalyticsScreen()),
      _EntryItem('用户管理', Icons.people, const UsersScreen()),
      _EntryItem('工单管理', Icons.support_agent, const TicketsScreen()),
      _EntryItem('实名认证', Icons.verified_user, const RealnameRecordsScreen()),
      _EntryItem('操作日志', Icons.history, const AuditLogsScreen()),
      _EntryItem('定时任务', Icons.schedule, const ScheduledTasksScreen()),
      _EntryItem('探针管理', Icons.radar, const ProbesScreen()),
    ];

    return LayoutBuilder(
      builder: (context, constraints) {
        final width = constraints.maxWidth;
        final crossAxisCount = width < 360 ? 3 : (width < 720 ? 4 : 6);
        final itemExtent =
            ((width - 24 - (crossAxisCount - 1) * 10) / crossAxisCount).clamp(
              64.0,
              120.0,
            );
        return Container(
          margin: const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
          padding: const EdgeInsets.all(16),
          decoration: BoxDecoration(
            gradient: const LinearGradient(
              colors: [Colors.white, Color(0xFFF8FBFF)],
              begin: Alignment.topLeft,
              end: Alignment.bottomRight,
            ),
            borderRadius: BorderRadius.circular(16),
            border: Border.all(color: const Color(0xFFDDE6F3)),
            boxShadow: const [
              BoxShadow(
                color: Color(0x140F172A),
                blurRadius: 16,
                offset: Offset(0, 6),
              ),
            ],
          ),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              const Row(
                children: [
                  Icon(Icons.flash_on, size: 18, color: Color(0xFF1E88E5)),
                  SizedBox(width: 6),
                  Text(
                    '快捷入口',
                    style: TextStyle(fontSize: 16, fontWeight: FontWeight.bold),
                  ),
                ],
              ),
              const SizedBox(height: 12),
              GridView.builder(
                shrinkWrap: true,
                physics: const NeverScrollableScrollPhysics(),
                gridDelegate: SliverGridDelegateWithFixedCrossAxisCount(
                  crossAxisCount: crossAxisCount,
                  mainAxisSpacing: 10,
                  crossAxisSpacing: 10,
                  mainAxisExtent: itemExtent,
                ),
                itemCount: entries.length,
                itemBuilder: (context, index) {
                  final e = entries[index];
                  return Material(
                    color: Colors.white,
                    borderRadius: BorderRadius.circular(14),
                    child: InkWell(
                      borderRadius: BorderRadius.circular(14),
                      onTap: () => Navigator.push(
                        context,
                        MaterialPageRoute(builder: (_) => e.screen),
                      ),
                      child: Padding(
                        padding: const EdgeInsets.symmetric(
                          horizontal: 8,
                          vertical: 10,
                        ),
                        child: Column(
                          mainAxisAlignment: MainAxisAlignment.center,
                          children: [
                            Container(
                              width: 44,
                              height: 44,
                              decoration: BoxDecoration(
                                gradient: const LinearGradient(
                                  colors: [
                                    Color(0xFF1E88E5),
                                    Color(0xFF42A5F5),
                                  ],
                                  begin: Alignment.topLeft,
                                  end: Alignment.bottomRight,
                                ),
                                borderRadius: BorderRadius.circular(12),
                              ),
                              child: Icon(
                                e.icon,
                                color: Colors.white,
                                size: 22,
                              ),
                            ),
                            const SizedBox(height: 8),
                            Text(
                              e.label,
                              textAlign: TextAlign.center,
                              style: const TextStyle(
                                fontSize: 12.5,
                                fontWeight: FontWeight.w600,
                              ),
                              maxLines: 2,
                              overflow: TextOverflow.ellipsis,
                            ),
                          ],
                        ),
                      ),
                    ),
                  );
                },
              ),
            ],
          ),
        );
      },
    );
  }
}

class _ManagementSection extends StatelessWidget {
  const _ManagementSection();

  @override
  Widget build(BuildContext context) {
    final items = [
      _ManagementItem(
        '支付渠道',
        '启用/停用支付方式',
        Icons.payments,
        const PaymentProvidersScreen(),
      ),
      _ManagementItem(
        '系统设置',
        '配置键值管理',
        Icons.settings,
        const SettingsKvScreen(),
      ),
      _ManagementItem(
        '商品与计费',
        '区域/线路/套餐管理',
        Icons.category,
        const CatalogHubScreen(),
      ),
      _ManagementItem(
        '权限列表',
        '系统权限定义',
        Icons.security,
        const PermissionsScreen(),
      ),
      _ManagementItem(
        '实名认证记录',
        '移动端快捷审核实名',
        Icons.verified_user,
        const RealnameRecordsScreen(),
      ),
    ];

    return Container(
      margin: const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
      padding: const EdgeInsets.all(16),
      decoration: BoxDecoration(
        gradient: const LinearGradient(
          colors: [Colors.white, Color(0xFFF8FBFF)],
          begin: Alignment.topLeft,
          end: Alignment.bottomRight,
        ),
        borderRadius: BorderRadius.circular(16),
        border: Border.all(color: const Color(0xFFDDE6F3)),
        boxShadow: const [
          BoxShadow(
            color: Color(0x140F172A),
            blurRadius: 16,
            offset: Offset(0, 6),
          ),
        ],
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          const Row(
            children: [
              Icon(
                Icons.admin_panel_settings_outlined,
                size: 18,
                color: Color(0xFF1E88E5),
              ),
              SizedBox(width: 6),
              Text(
                '系统管理',
                style: TextStyle(fontSize: 16, fontWeight: FontWeight.bold),
              ),
            ],
          ),
          const SizedBox(height: 10),
          ...items.map((item) {
            return Padding(
              padding: const EdgeInsets.only(bottom: 8),
              child: Material(
                color: Colors.white,
                borderRadius: BorderRadius.circular(14),
                child: InkWell(
                  borderRadius: BorderRadius.circular(14),
                  onTap: () => Navigator.push(
                    context,
                    MaterialPageRoute(builder: (_) => item.screen),
                  ),
                  child: Padding(
                    padding: const EdgeInsets.symmetric(
                      horizontal: 10,
                      vertical: 10,
                    ),
                    child: Row(
                      children: [
                        Container(
                          width: 42,
                          height: 42,
                          decoration: BoxDecoration(
                            gradient: const LinearGradient(
                              colors: [Color(0xFF1E88E5), Color(0xFF42A5F5)],
                              begin: Alignment.topLeft,
                              end: Alignment.bottomRight,
                            ),
                            borderRadius: BorderRadius.circular(10),
                          ),
                          child: Icon(item.icon, color: Colors.white, size: 20),
                        ),
                        const SizedBox(width: 12),
                        Expanded(
                          child: Column(
                            crossAxisAlignment: CrossAxisAlignment.start,
                            children: [
                              Text(
                                item.title,
                                style: const TextStyle(
                                  fontSize: 15,
                                  fontWeight: FontWeight.w600,
                                ),
                              ),
                              const SizedBox(height: 2),
                              Text(
                                item.subtitle,
                                style: TextStyle(
                                  fontSize: 12.5,
                                  color: Colors.grey[600],
                                ),
                              ),
                            ],
                          ),
                        ),
                        Icon(
                          Icons.chevron_right_rounded,
                          color: Colors.grey[400],
                          size: 22,
                        ),
                      ],
                    ),
                  ),
                ),
              ),
            );
          }),
        ],
      ),
    );
  }
}

class _StatItem {
  final String label;
  final String value;
  final IconData icon;
  final Color color;
  final int? preferredTabIndex;
  final Widget? fallbackScreen;

  const _StatItem(
    this.label,
    this.value,
    this.icon,
    this.color, {
    this.preferredTabIndex,
    this.fallbackScreen,
  });
}

class _EntryItem {
  final String label;
  final IconData icon;
  final Widget screen;

  const _EntryItem(this.label, this.icon, this.screen);
}

class _ManagementItem {
  final String title;
  final String subtitle;
  final IconData icon;
  final Widget screen;

  const _ManagementItem(this.title, this.subtitle, this.icon, this.screen);
}

class _BingBanner {
  final String imageUrl;
  final String title;

  const _BingBanner({required this.imageUrl, required this.title});

  @override
  bool operator ==(Object other) {
    if (identical(this, other)) return true;
    return other is _BingBanner && other.imageUrl == imageUrl;
  }

  @override
  int get hashCode => imageUrl.hashCode;
}

class DashboardData {
  final int totalOrders;
  final int pendingReview;
  final int revenueCents;
  final int vpsCount;
  final int expiringSoon;
  final int? usersTotal;
  final int? walletOrdersTotal;

  DashboardData({
    required this.totalOrders,
    required this.pendingReview,
    required this.revenueCents,
    required this.vpsCount,
    required this.expiringSoon,
    required this.usersTotal,
    required this.walletOrdersTotal,
  });
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
    return const Center(child: Text('暂无数据'));
  }
}
