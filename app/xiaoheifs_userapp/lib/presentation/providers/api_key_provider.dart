import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../core/utils/map_utils.dart';
import '../../data/repositories/api_key_repository.dart';

class ApiKeyState {
  final bool loading;
  final List<Map<String, dynamic>> items;
  final int total;
  final String? error;

  const ApiKeyState({
    this.loading = false,
    this.items = const [],
    this.total = 0,
    this.error,
  });

  ApiKeyState copyWith({
    bool? loading,
    List<Map<String, dynamic>>? items,
    int? total,
    String? error,
  }) {
    return ApiKeyState(
      loading: loading ?? this.loading,
      items: items ?? this.items,
      total: total ?? this.total,
      error: error,
    );
  }
}

final apiKeyRepositoryProvider = Provider<ApiKeyRepository>((ref) {
  return ApiKeyRepository();
});

final apiKeyProvider = StateNotifierProvider<ApiKeyNotifier, ApiKeyState>((
  ref,
) {
  return ApiKeyNotifier(ref.read(apiKeyRepositoryProvider));
});

class ApiKeyNotifier extends StateNotifier<ApiKeyState> {
  ApiKeyNotifier(this._repo) : super(const ApiKeyState());

  final ApiKeyRepository _repo;
  final Map<String, _ApiKeyCacheEntry> _cache = {};
  final Duration _cacheTtl = const Duration(seconds: 30);

  Future<void> fetchApiKeys({
    int limit = 100,
    int offset = 0,
    bool force = false,
  }) async {
    final cacheKey = '$limit|$offset';
    if (!force) {
      final cached = _cache[cacheKey];
      if (cached != null &&
          DateTime.now().difference(cached.fetchedAt) < _cacheTtl) {
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
      final payload = await _repo.listApiKeys(limit: limit, offset: offset);
      final rawItems = payload['items'];
      final list = rawItems is List
          ? rawItems.map((e) => ensureMap(e)).toList()
          : <Map<String, dynamic>>[];
      final total = payload['total'] ?? list.length;
      final normalizedTotal = total is int
          ? total
          : int.tryParse('$total') ?? list.length;
      _cache[cacheKey] = _ApiKeyCacheEntry(items: list, total: normalizedTotal);
      state = state.copyWith(
        loading: false,
        items: list,
        total: normalizedTotal,
      );
    } catch (e) {
      state = state.copyWith(loading: false, error: e.toString());
    }
  }

  Future<Map<String, dynamic>> createApiKey(String name) async {
    final payload = await _repo.createApiKey(name: name, scopes: const []);
    _cache.clear();
    return payload;
  }

  Future<void> toggleStatus(int id, String currentStatus) async {
    final nextStatus = currentStatus.toLowerCase() == 'active'
        ? 'disabled'
        : 'active';
    await _repo.updateApiKeyStatus(id, nextStatus);
    _cache.clear();
  }

  Future<void> deleteApiKey(int id) async {
    await _repo.deleteApiKey(id);
    _cache.clear();
  }
}

class _ApiKeyCacheEntry {
  final List<Map<String, dynamic>> items;
  final int total;
  final DateTime fetchedAt;

  _ApiKeyCacheEntry({required this.items, required this.total})
    : fetchedAt = DateTime.now();
}
