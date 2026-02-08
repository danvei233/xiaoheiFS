import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../../data/repositories/notifications_repository.dart';
import '../../core/utils/map_utils.dart';

class NotificationsState {
  final bool loading;
  final List<Map<String, dynamic>> items;
  final int unreadCount;
  final String? error;

  const NotificationsState({
    this.loading = false,
    this.items = const [],
    this.unreadCount = 0,
    this.error,
  });

  NotificationsState copyWith({
    bool? loading,
    List<Map<String, dynamic>>? items,
    int? unreadCount,
    String? error,
  }) {
    return NotificationsState(
      loading: loading ?? this.loading,
      items: items ?? this.items,
      unreadCount: unreadCount ?? this.unreadCount,
      error: error,
    );
  }
}

final notificationsRepositoryProvider = Provider<NotificationsRepository>((ref) {
  return NotificationsRepository();
});

final notificationsProvider =
    StateNotifierProvider<NotificationsNotifier, NotificationsState>((ref) {
  return NotificationsNotifier(ref.read(notificationsRepositoryProvider));
});

class NotificationsNotifier extends StateNotifier<NotificationsState> {
  NotificationsNotifier(this._repo) : super(const NotificationsState());
  final NotificationsRepository _repo;

  Future<void> fetchNotifications({String? status}) async {
    state = state.copyWith(loading: true, error: null);
    try {
      final payload = await _repo.listNotifications(status: status, limit: 20, offset: 0);
      final items = payload['items'];
      final list = items is List ? items.map((e) => ensureMap(e)).toList() : <Map<String, dynamic>>[];
      state = state.copyWith(loading: false, items: list);
    } catch (e) {
      state = state.copyWith(loading: false, error: e.toString());
    }
  }

  Future<void> fetchUnreadCount() async {
    try {
      final count = await _repo.getUnreadCount();
      state = state.copyWith(unreadCount: count);
    } catch (_) {}
  }

  Future<void> markAsRead(int id) async {
    await _repo.markRead(id);
    state = state.copyWith(
      items: state.items.map((item) {
        if (item['id'] == id || item['ID'] == id) {
          return {...item, 'read_at': DateTime.now().toIso8601String()};
        }
        return item;
      }).toList(),
      unreadCount: state.unreadCount > 0 ? state.unreadCount - 1 : 0,
    );
  }

  Future<void> markAllRead() async {
    await _repo.markAllRead();
    state = state.copyWith(
      items: state.items.map((item) => {...item, 'read_at': DateTime.now().toIso8601String()}).toList(),
      unreadCount: 0,
    );
  }
}
