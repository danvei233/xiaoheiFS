import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../../data/repositories/cart_repository.dart';
import '../../core/utils/map_utils.dart';

class CartState {
  final bool loading;
  final List<Map<String, dynamic>> items;
  final String? error;

  const CartState({
    this.loading = false,
    this.items = const [],
    this.error,
  });

  CartState copyWith({
    bool? loading,
    List<Map<String, dynamic>>? items,
    String? error,
  }) {
    return CartState(
      loading: loading ?? this.loading,
      items: items ?? this.items,
      error: error,
    );
  }
}

final cartRepositoryProvider = Provider<CartRepository>((ref) {
  return CartRepository();
});

final cartProvider = StateNotifierProvider<CartNotifier, CartState>((ref) {
  return CartNotifier(ref.read(cartRepositoryProvider));
});

class CartNotifier extends StateNotifier<CartState> {
  CartNotifier(this._repo) : super(const CartState());
  final CartRepository _repo;
  DateTime? _lastFetchedAt;
  final Duration _cacheTtl = const Duration(seconds: 30);

  Future<void> fetchCart({bool force = false}) async {
    if (!force &&
        state.items.isNotEmpty &&
        _lastFetchedAt != null &&
        DateTime.now().difference(_lastFetchedAt!) < _cacheTtl) {
      return;
    }
    state = state.copyWith(loading: true, error: null);
    try {
      final payload = await _repo.listCart();
      final items = _normalizeItems(payload['items'] ?? payload);
      _lastFetchedAt = DateTime.now();
      state = state.copyWith(loading: false, items: items);
    } catch (e) {
      state = state.copyWith(loading: false, error: e.toString());
    }
  }

  Future<void> addItem(Map<String, dynamic> payload) async {
    await _repo.addItem(payload);
    await fetchCart(force: true);
  }

  Future<void> updateItem(int id, Map<String, dynamic> payload) async {
    await _repo.updateItem(id, payload);
    await fetchCart(force: true);
  }

  Future<void> removeItem(int id) async {
    await _repo.deleteItem(id);
    await fetchCart(force: true);
  }

  Future<void> clear() async {
    await _repo.clear();
    state = state.copyWith(items: []);
  }

  List<Map<String, dynamic>> _normalizeItems(dynamic raw) {
    if (raw is List) {
      return raw.map((e) => ensureMap(e)).toList();
    }
    if (raw is Map<String, dynamic>) {
      return [raw];
    }
    return [];
  }
}
