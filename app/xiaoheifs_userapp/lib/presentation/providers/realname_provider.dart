import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../../data/repositories/realname_repository.dart';
import '../../core/utils/map_utils.dart';

class RealnameState {
  final bool loading;
  final Map<String, dynamic>? data;
  final String? error;

  const RealnameState({this.loading = false, this.data, this.error});

  RealnameState copyWith({bool? loading, Map<String, dynamic>? data, String? error}) {
    return RealnameState(
      loading: loading ?? this.loading,
      data: data ?? this.data,
      error: error,
    );
  }
}

final realnameRepositoryProvider = Provider<RealnameRepository>((ref) {
  return RealnameRepository();
});

final realnameProvider = StateNotifierProvider<RealnameNotifier, RealnameState>((ref) {
  return RealnameNotifier(ref.read(realnameRepositoryProvider));
});

class RealnameNotifier extends StateNotifier<RealnameState> {
  RealnameNotifier(this._repo) : super(const RealnameState()) {
    fetch();
  }
  final RealnameRepository _repo;

  Future<void> fetch() async {
    state = state.copyWith(loading: true, error: null);
    try {
      final res = await _repo.getStatus();
      state = state.copyWith(loading: false, data: ensureMap(res));
    } catch (e) {
      state = state.copyWith(loading: false, error: e.toString());
    }
  }

  Future<void> submit(Map<String, dynamic> payload) async {
    await _repo.submit(payload);
    await fetch();
  }
}
