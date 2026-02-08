import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../../core/utils/map_utils.dart';
import '../../data/repositories/notification_repository.dart';

class NotificationState {
  final bool loading;
  final List<Map<String, dynamic>> items;
  final int total;
  final int unreadCount;
  final String? error;

  const NotificationState({
    this.loading = false,
    this.items = const [],
    this.total = 0,
    this.unreadCount = 0,
    this.error,
  });

  NotificationState copyWith({
    bool? loading,
    List<Map<String, dynamic>>? items,
    int? total,
    int? unreadCount,
    String? error,
  }) {
    return NotificationState(
      loading: loading ?? this.loading,
      items: items ?? this.items,
      total: total ?? this.total,
      unreadCount: unreadCount ?? this.unreadCount,
      error: error,
    );
  }
}

final notificationRepositoryProvider = Provider<NotificationRepository>((ref) {
  return NotificationRepository();
});

final notificationProvider = StateNotifierProvider<NotificationNotifier, NotificationState>((ref) {
  return NotificationNotifier(ref.read(notificationRepositoryProvider));
});

class NotificationNotifier extends StateNotifier<NotificationState> {
  NotificationNotifier(this._repo) : super(const NotificationState());

  final NotificationRepository _repo;
  final Map<String, _NotificationCacheEntry> _cache = {};
  final Duration _cacheTtl = const Duration(seconds: 30);

  Future<void> fetchUnreadCount() async {
    try {
      final payload = await _repo.getUnreadCount();
      final unread = payload['unread'] ?? payload['count'] ?? 0;
      state = state.copyWith(unreadCount: int.tryParse('$unread') ?? 0);
    } catch (_) {
      // ignore
    }
  }

  Future<void> fetchNotifications({
    String? status,
    int limit = 20,
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
      final payload = await _repo.listNotifications(status: status, limit: limit, offset: offset);
      final items = payload['items'];
      final list = items is List ? items.map((e) => ensureMap(e)).toList() : <Map<String, dynamic>>[];
      final total = payload['total'] ?? list.length;
      final normalizedTotal = total is int ? total : int.tryParse('$total') ?? list.length;
      _cache[cacheKey] = _NotificationCacheEntry(items: list, total: normalizedTotal);
      state = state.copyWith(loading: false, items: list, total: normalizedTotal);
    } catch (e) {
      state = state.copyWith(loading: false, error: e.toString());
    }
  }

  Future<void> markRead(int id) async {
    await _repo.markRead(id);
    final updated = state.items.map((item) {
      if ((item['id'] ?? item['ID']) == id) {
        final copy = Map<String, dynamic>.from(item);
        copy['read_at'] = DateTime.now().toIso8601String();
        return copy;
      }
      return item;
    }).toList();
    state = state.copyWith(
      items: updated,
      unreadCount: state.unreadCount > 0 ? state.unreadCount - 1 : 0,
    );
  }

  Future<void> markAllRead() async {
    await _repo.markAllRead();
    final updated = state.items.map((item) {
      if (item['read_at'] == null) {
        final copy = Map<String, dynamic>.from(item);
        copy['read_at'] = DateTime.now().toIso8601String();
        return copy;
      }
      return item;
    }).toList();
    state = state.copyWith(items: updated, unreadCount: 0);
  }

  String _cacheKey(String? status, int limit, int offset) =>
      '${status ?? 'all'}|$limit|$offset';
}

class _NotificationCacheEntry {
  final List<Map<String, dynamic>> items;
  final int total;
  final DateTime fetchedAt;

  _NotificationCacheEntry({required this.items, required this.total}) : fetchedAt = DateTime.now();
}
