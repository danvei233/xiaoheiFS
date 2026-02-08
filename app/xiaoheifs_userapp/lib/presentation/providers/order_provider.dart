import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../../data/repositories/order_repository.dart';
import '../../core/utils/map_utils.dart';

class OrderListState {
  final bool loading;
  final List<Map<String, dynamic>> items;
  final int total;
  final String? error;

  const OrderListState({
    this.loading = false,
    this.items = const [],
    this.total = 0,
    this.error,
  });

  OrderListState copyWith({
    bool? loading,
    List<Map<String, dynamic>>? items,
    int? total,
    String? error,
  }) {
    return OrderListState(
      loading: loading ?? this.loading,
      items: items ?? this.items,
      total: total ?? this.total,
      error: error,
    );
  }
}

class OrderDetailState {
  final bool loading;
  final Map<String, dynamic>? order;
  final List<Map<String, dynamic>> items;
  final List<Map<String, dynamic>> payments;
  final String? error;

  const OrderDetailState({
    this.loading = false,
    this.order,
    this.items = const [],
    this.payments = const [],
    this.error,
  });

  OrderDetailState copyWith({
    bool? loading,
    Map<String, dynamic>? order,
    List<Map<String, dynamic>>? items,
    List<Map<String, dynamic>>? payments,
    String? error,
  }) {
    return OrderDetailState(
      loading: loading ?? this.loading,
      order: order ?? this.order,
      items: items ?? this.items,
      payments: payments ?? this.payments,
      error: error,
    );
  }
}

final orderRepositoryProvider = Provider<OrderRepository>((ref) {
  return OrderRepository();
});

final orderListProvider = StateNotifierProvider<OrderListNotifier, OrderListState>((ref) {
  return OrderListNotifier(ref.read(orderRepositoryProvider));
});

final orderDetailProvider = StateNotifierProvider<OrderDetailNotifier, OrderDetailState>((ref) {
  return OrderDetailNotifier(ref.read(orderRepositoryProvider));
});

class OrderListNotifier extends StateNotifier<OrderListState> {
  OrderListNotifier(this._repo) : super(const OrderListState()) {
    fetchOrders();
  }

  final OrderRepository _repo;
  final Map<String, _OrderCacheEntry> _cache = {};
  final Duration _cacheTtl = const Duration(seconds: 30);

  Future<void> fetchOrders({
    String? status,
    int limit = 10,
    int offset = 0,
    bool force = false,
  }) async {
    final cacheKey = _cacheKey(status, limit, offset);
    if (!force) {
      final cached = _cache[cacheKey];
      if (cached != null && DateTime.now().difference(cached.fetchedAt) < _cacheTtl) {
        state = state.copyWith(
          loading: false,
          items: cached.items,
          total: cached.total,
          error: null,
        );
        return;
      }
    }
    state = state.copyWith(loading: true, error: null);
    try {
      final payload = await _repo.listOrders(status: status, limit: limit, offset: offset);
      final items = payload['items'];
      final list = items is List ? items.map((e) => ensureMap(e)).toList() : <Map<String, dynamic>>[];
      final total = payload['total'] ?? list.length;
      final normalizedTotal = total is int ? total : int.tryParse('$total') ?? list.length;
      _cache[cacheKey] = _OrderCacheEntry(items: list, total: normalizedTotal);
      state = state.copyWith(loading: false, items: list, total: normalizedTotal);
    } catch (e) {
      state = state.copyWith(loading: false, error: e.toString());
    }
  }

  Future<void> refresh({String? status, int limit = 10, int offset = 0}) async {
    await fetchOrders(status: status, limit: limit, offset: offset, force: true);
  }

  String _cacheKey(String? status, int limit, int offset) =>
      '${status ?? 'all'}|$limit|$offset';
}

class _OrderCacheEntry {
  final List<Map<String, dynamic>> items;
  final int total;
  final DateTime fetchedAt;

  _OrderCacheEntry({required this.items, required this.total}) : fetchedAt = DateTime.now();
}

class OrderDetailNotifier extends StateNotifier<OrderDetailState> {
  OrderDetailNotifier(this._repo) : super(const OrderDetailState());
  final OrderRepository _repo;

  Future<void> fetchDetail(int id) async {
    state = state.copyWith(loading: true, error: null);
    try {
      final payload = await _repo.getOrderDetail(id);
      final order = ensureMap(payload['order'] ?? payload['Order']);
      final items = payload['items'] is List ? (payload['items'] as List).map((e) => ensureMap(e)).toList() : <Map<String, dynamic>>[];
      final payments = payload['payments'] is List ? (payload['payments'] as List).map((e) => ensureMap(e)).toList() : <Map<String, dynamic>>[];
      state = state.copyWith(loading: false, order: order, items: items, payments: payments);
    } catch (e) {
      state = state.copyWith(loading: false, error: e.toString());
    }
  }

  Future<void> refresh(int id) async {
    await _repo.refreshOrder(id);
    await fetchDetail(id);
  }
}
