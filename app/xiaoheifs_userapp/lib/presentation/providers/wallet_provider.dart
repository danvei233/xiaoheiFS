import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../../data/repositories/wallet_repository.dart';
import '../../core/utils/map_utils.dart';

class WalletState {
  final bool loading;
  final Map<String, dynamic>? wallet;
  final List<Map<String, dynamic>> orders;
  final List<Map<String, dynamic>> transactions;
  final String? error;

  const WalletState({
    this.loading = false,
    this.wallet,
    this.orders = const [],
    this.transactions = const [],
    this.error,
  });

  WalletState copyWith({
    bool? loading,
    Map<String, dynamic>? wallet,
    List<Map<String, dynamic>>? orders,
    List<Map<String, dynamic>>? transactions,
    String? error,
  }) {
    return WalletState(
      loading: loading ?? this.loading,
      wallet: wallet ?? this.wallet,
      orders: orders ?? this.orders,
      transactions: transactions ?? this.transactions,
      error: error,
    );
  }
}

final walletRepositoryProvider = Provider<WalletRepository>((ref) {
  return WalletRepository();
});

final walletProvider = StateNotifierProvider<WalletNotifier, WalletState>((ref) {
  return WalletNotifier(ref.read(walletRepositoryProvider));
});

class WalletNotifier extends StateNotifier<WalletState> {
  WalletNotifier(this._repo) : super(const WalletState()) {
    refresh();
  }

  final WalletRepository _repo;

  Future<void> loadWallet() async {
    state = state.copyWith(loading: true, error: null);
    try {
      final wallet = await _repo.getWallet();
      state = state.copyWith(loading: false, wallet: wallet);
    } catch (e) {
      state = state.copyWith(loading: false, error: e.toString());
    }
  }

  Future<void> refresh() async {
    await loadWallet();
    await loadOrders();
    await loadTransactions();
  }

  Future<void> loadOrders({int limit = 100, int offset = 0}) async {
    final res = await _repo.listWalletOrders(limit: limit, offset: offset);
    final items = res['items'];
    final list = items is List ? items.map((e) => ensureMap(e)).toList() : <Map<String, dynamic>>[];
    state = state.copyWith(orders: list);
  }

  Future<void> loadTransactions({int limit = 100, int offset = 0}) async {
    final res = await _repo.listWalletTransactions(limit: limit, offset: offset);
    final items = res['items'];
    final list = items is List ? items.map((e) => ensureMap(e)).toList() : <Map<String, dynamic>>[];
    state = state.copyWith(transactions: list);
  }

  Future<void> recharge(Map<String, dynamic> payload) async {
    await _repo.createRecharge(payload);
  }

  Future<void> withdraw(Map<String, dynamic> payload) async {
    await _repo.createWithdraw(payload);
  }
}
