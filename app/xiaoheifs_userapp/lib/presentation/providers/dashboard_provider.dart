import 'dart:math';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../../core/utils/map_utils.dart';
import '../../data/repositories/dashboard_repository.dart';

class DashboardMetrics {
  final int vpsTotal;
  final int expiring;
  final int ordersTotal;
  final int pendingOrders;
  final double spend30d;
  final double balance;
  final String currency;
  final String realnameStatus;
  final int cartItems;
  final int pendingPayment;

  const DashboardMetrics({
    this.vpsTotal = 0,
    this.expiring = 0,
    this.ordersTotal = 0,
    this.pendingOrders = 0,
    this.spend30d = 0,
    this.balance = 0,
    this.currency = 'CNY',
    this.realnameStatus = '',
    this.cartItems = 0,
    this.pendingPayment = 0,
  });
}

class DashboardCharts {
  final List<String> spendLabels;
  final List<double> spendValues;
  final List<Map<String, dynamic>> orderStatus;
  final List<Map<String, dynamic>> expiringList;

  const DashboardCharts({
    this.spendLabels = const [],
    this.spendValues = const [],
    this.orderStatus = const [],
    this.expiringList = const [],
  });
}

class DashboardState {
  final bool loading;
  final DashboardMetrics metrics;
  final DashboardCharts charts;
  final Map<String, dynamic>? wallet;
  final Map<String, dynamic>? realname;
  final String? error;

  const DashboardState({
    this.loading = false,
    this.metrics = const DashboardMetrics(),
    this.charts = const DashboardCharts(),
    this.wallet,
    this.realname,
    this.error,
  });

  DashboardState copyWith({
    bool? loading,
    DashboardMetrics? metrics,
    DashboardCharts? charts,
    Map<String, dynamic>? wallet,
    Map<String, dynamic>? realname,
    String? error,
  }) {
    return DashboardState(
      loading: loading ?? this.loading,
      metrics: metrics ?? this.metrics,
      charts: charts ?? this.charts,
      wallet: wallet ?? this.wallet,
      realname: realname ?? this.realname,
      error: error,
    );
  }
}

final dashboardRepositoryProvider = Provider<DashboardRepository>((ref) {
  return DashboardRepository();
});

final dashboardProvider = StateNotifierProvider<DashboardNotifier, DashboardState>((ref) {
  return DashboardNotifier(ref.read(dashboardRepositoryProvider));
});

class DashboardNotifier extends StateNotifier<DashboardState> {
  DashboardNotifier(this._repository) : super(const DashboardState()) {
    loadDashboard();
  }

  final DashboardRepository _repository;
  DateTime? _lastFetchedAt;
  final Duration _cacheTtl = const Duration(seconds: 30);

  Future<void> loadDashboard({bool force = false}) async {
    if (!force &&
        state.metrics.vpsTotal != 0 &&
        _lastFetchedAt != null &&
        DateTime.now().difference(_lastFetchedAt!) < _cacheTtl) {
      return;
    }
    state = state.copyWith(loading: true, error: null);
    try {
      final results = await Future.wait([
        _repository.getDashboard(),
        _repository.listOrders(limit: 200, offset: 0),
        _repository.listVps(),
        _repository.getWallet(),
        _repository.getRealnameStatus(),
      ]);

      final dash = ensureMap(results[0]);
      final ordersPayload = ensureMap(results[1]);
      final vpsPayload = ensureMap(results[2]);
      final walletPayload = ensureMap(results[3]);
      final realnamePayload = ensureMap(results[4]);

      final orders = (ordersPayload['items'] is List)
          ? (ordersPayload['items'] as List).map((e) => ensureMap(e)).toList()
          : <Map<String, dynamic>>[];
      final vpsList = (vpsPayload['items'] is List)
          ? (vpsPayload['items'] as List).map((e) => ensureMap(e)).toList()
          : (vpsPayload is List)
              ? (vpsPayload as List).map((e) => ensureMap(e)).toList()
              : <Map<String, dynamic>>[];

      final spend30 = _calcSpend30(orders);
      final realnameStatus = _resolveRealnameStatus(realnamePayload);

      final metrics = DashboardMetrics(
        vpsTotal: (dash['vps'] ?? vpsList.length ?? 0) as int,
        expiring: (dash['expiring'] ?? _countExpiring(vpsList, 7)) as int,
        ordersTotal: (dash['orders'] ?? orders.length ?? 0) as int,
        pendingOrders: (dash['pending_review'] ?? _countStatus(orders, 'pending_review')) as int,
        spend30d: (dash['spend_30d'] ?? spend30).toDouble(),
        balance: _normalizeWalletBalance(walletPayload),
        currency: _normalizeWalletCurrency(walletPayload),
        realnameStatus: realnameStatus,
        cartItems: (dash['cart_items'] ?? 0) as int,
        pendingPayment: (dash['pending_payment'] ?? 0) as int,
      );

      final charts = DashboardCharts(
        spendLabels: _spendTrendLabels(orders),
        spendValues: _spendTrendValues(orders),
        orderStatus: _orderStatusList(orders),
        expiringList: _expiringList(vpsList),
      );

      state = state.copyWith(
        loading: false,
        metrics: metrics,
        charts: charts,
        wallet: walletPayload,
        realname: realnamePayload,
      );
      _lastFetchedAt = DateTime.now();
    } catch (e) {
      state = state.copyWith(loading: false, error: e.toString());
    }
  }

  Future<void> refresh() async {
    await loadDashboard(force: true);
  }

  double _calcSpend30(List<Map<String, dynamic>> orders) {
    final thirtyDaysAgo = DateTime.now().subtract(const Duration(days: 30));
    double sum = 0;
    for (final order in orders) {
      final created = _parseDate(order['created_at'] ?? order['CreatedAt']);
      if (created != null && created.isAfter(thirtyDaysAgo)) {
        final amount = double.tryParse('${order['total_amount'] ?? order['TotalAmount'] ?? 0}') ?? 0;
        sum += amount;
      }
    }
    return sum;
  }

  int _countStatus(List<Map<String, dynamic>> orders, String status) {
    return orders.where((o) => (o['status'] ?? o['Status']) == status).length;
  }

  int _countExpiring(List<Map<String, dynamic>> vpsList, int withinDays) {
    final now = DateTime.now();
    return vpsList.where((v) {
      final expire = _parseDate(v['expire_at'] ?? v['ExpireAt']);
      if (expire == null) return false;
      final diff = expire.difference(now).inDays;
      return diff >= 0 && diff <= withinDays;
    }).length;
  }

  List<String> _spendTrendLabels(List<Map<String, dynamic>> orders) {
    final map = _spendTrendMap(orders);
    final keys = map.keys.toList()..sort();
    return keys;
  }

  List<double> _spendTrendValues(List<Map<String, dynamic>> orders) {
    final map = _spendTrendMap(orders);
    final keys = map.keys.toList()..sort();
    return keys.map((k) => map[k] ?? 0).toList();
  }

  Map<String, double> _spendTrendMap(List<Map<String, dynamic>> orders) {
    final map = <String, double>{};
    final thirtyDaysAgo = DateTime.now().subtract(const Duration(days: 30));
    for (final order in orders) {
      final created = _parseDate(order['created_at'] ?? order['CreatedAt']);
      if (created == null || created.isBefore(thirtyDaysAgo)) continue;
      final key = '${created.year.toString().padLeft(4, '0')}-${created.month.toString().padLeft(2, '0')}-${created.day.toString().padLeft(2, '0')}';
      final amount = double.tryParse('${order['total_amount'] ?? order['TotalAmount'] ?? 0}') ?? 0;
      map[key] = (map[key] ?? 0) + amount;
    }
    return map;
  }

  List<Map<String, dynamic>> _orderStatusList(List<Map<String, dynamic>> orders) {
    final map = <String, int>{};
    for (final order in orders) {
      final status = (order['status'] ?? order['Status'] ?? 'unknown').toString();
      map[status] = (map[status] ?? 0) + 1;
    }
    return map.entries.map((e) => {'name': e.key, 'value': e.value}).toList();
  }

  List<Map<String, dynamic>> _expiringList(List<Map<String, dynamic>> vpsList) {
    return vpsList
        .where((v) => v['expire_at'] != null || v['ExpireAt'] != null)
        .take(5)
        .toList();
  }

  DateTime? _parseDate(dynamic value) {
    if (value == null) return null;
    if (value is DateTime) return value;
    if (value is num) {
      final epoch = value.toInt();
      final ms = epoch > 1000000000000 ? epoch : epoch * 1000;
      return DateTime.fromMillisecondsSinceEpoch(ms);
    }
    final text = value.toString();
    final numeric = int.tryParse(text);
    if (numeric != null) {
      final ms = numeric > 1000000000000 ? numeric : numeric * 1000;
      return DateTime.fromMillisecondsSinceEpoch(ms);
    }
    return DateTime.tryParse(text);
  }

  double _normalizeWalletBalance(Map<String, dynamic> walletPayload) {
    final wallet = walletPayload['wallet'] is Map ? ensureMap(walletPayload['wallet']) : walletPayload;
    final raw = wallet['balance'] ?? 0;
    return double.tryParse('$raw') ?? 0;
  }

  String _normalizeWalletCurrency(Map<String, dynamic> walletPayload) {
    final wallet = walletPayload['wallet'] is Map ? ensureMap(walletPayload['wallet']) : walletPayload;
    return wallet['currency']?.toString() ?? 'CNY';
  }

  String _resolveRealnameStatus(Map<String, dynamic> payload) {
    if (payload['verified'] == true) return 'verified';
    final verification = payload['verification'] is Map ? ensureMap(payload['verification']) : null;
    final status = verification?['status']?.toString();
    if (status != null && status.isNotEmpty) return status;
    final enabled = payload['enabled'];
    if (enabled == false) return 'disabled';
    return 'unverified';
  }
}
