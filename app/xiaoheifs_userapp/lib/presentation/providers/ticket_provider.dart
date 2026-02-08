import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../../data/repositories/ticket_repository.dart';
import '../../core/utils/map_utils.dart';

class TicketListState {
  final bool loading;
  final List<Map<String, dynamic>> items;
  final int total;
  final String? error;

  const TicketListState({
    this.loading = false,
    this.items = const [],
    this.total = 0,
    this.error,
  });

  TicketListState copyWith({
    bool? loading,
    List<Map<String, dynamic>>? items,
    int? total,
    String? error,
  }) {
    return TicketListState(
      loading: loading ?? this.loading,
      items: items ?? this.items,
      total: total ?? this.total,
      error: error,
    );
  }
}

class TicketDetailState {
  final bool loading;
  final Map<String, dynamic>? ticket;
  final List<Map<String, dynamic>> messages;
  final List<Map<String, dynamic>> resources;
  final String? error;

  const TicketDetailState({
    this.loading = false,
    this.ticket,
    this.messages = const [],
    this.resources = const [],
    this.error,
  });

  TicketDetailState copyWith({
    bool? loading,
    Map<String, dynamic>? ticket,
    List<Map<String, dynamic>>? messages,
    List<Map<String, dynamic>>? resources,
    String? error,
  }) {
    return TicketDetailState(
      loading: loading ?? this.loading,
      ticket: ticket ?? this.ticket,
      messages: messages ?? this.messages,
      resources: resources ?? this.resources,
      error: error,
    );
  }
}

final ticketRepositoryProvider = Provider<TicketRepository>((ref) {
  return TicketRepository();
});

final ticketListProvider = StateNotifierProvider<TicketListNotifier, TicketListState>((ref) {
  return TicketListNotifier(ref.read(ticketRepositoryProvider));
});

final ticketDetailProvider = StateNotifierProvider<TicketDetailNotifier, TicketDetailState>((ref) {
  return TicketDetailNotifier(ref.read(ticketRepositoryProvider));
});

class TicketListNotifier extends StateNotifier<TicketListState> {
  TicketListNotifier(this._repo) : super(const TicketListState());
  final TicketRepository _repo;
  final Map<String, _TicketCacheEntry> _cache = {};
  final Duration _cacheTtl = const Duration(seconds: 30);

  Future<void> fetchTickets({
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
      final payload = await _repo.listTickets(status: status, limit: limit, offset: offset);
      final items = payload['items'];
      final list = items is List ? items.map((e) => ensureMap(e)).toList() : <Map<String, dynamic>>[];
      final total = payload['total'] ?? list.length;
      final normalizedTotal = total is int ? total : int.tryParse('$total') ?? list.length;
      _cache[cacheKey] = _TicketCacheEntry(items: list, total: normalizedTotal);
      state = state.copyWith(
        loading: false,
        items: list,
        total: normalizedTotal,
      );
    } catch (e) {
      state = state.copyWith(loading: false, error: e.toString());
    }
  }

  Future<void> refresh() async => fetchTickets(force: true);

  Future<void> createTicket({
    required String subject,
    required String content,
    List<int>? resourceIds,
    List<Map<String, dynamic>>? resources,
  }) async {
    final payload = {
      'subject': subject,
      'content': content,
      if (resources != null && resources.isNotEmpty) 'resources': resources,
    };
    if (resources == null && resourceIds != null && resourceIds.isNotEmpty) {
      payload['resources'] = resourceIds
          .map((id) => {
                'resource_type': 'vps',
                'resource_id': id,
              })
          .toList();
    }
    await _repo.createTicket(payload);
    await fetchTickets(force: true);
  }

  String _cacheKey(String? status, int limit, int offset) =>
      '${status ?? 'all'}|$limit|$offset';
}

class _TicketCacheEntry {
  final List<Map<String, dynamic>> items;
  final int total;
  final DateTime fetchedAt;

  _TicketCacheEntry({required this.items, required this.total}) : fetchedAt = DateTime.now();
}

class TicketDetailNotifier extends StateNotifier<TicketDetailState> {
  TicketDetailNotifier(this._repo) : super(const TicketDetailState());
  final TicketRepository _repo;

  Future<void> fetchDetail(int id, {bool showLoading = true}) async {
    if (showLoading) {
      state = state.copyWith(loading: true, error: null);
    }
    try {
      final payload = await _repo.getDetail(id);
      final ticket = ensureMap(payload['ticket'] ?? payload);
      final messages = payload['messages'] is List
          ? (payload['messages'] as List).map((e) => ensureMap(e)).toList()
          : <Map<String, dynamic>>[];
      final resources = payload['resources'] is List
          ? (payload['resources'] as List).map((e) => ensureMap(e)).toList()
          : <Map<String, dynamic>>[];
      state = state.copyWith(
        loading: false,
        ticket: ticket,
        messages: messages,
        resources: resources,
      );
    } catch (e) {
      state = state.copyWith(loading: false, error: e.toString());
    }
  }

  Future<void> addMessage(int id, String content) async {
    await _repo.addMessage(id, {'content': content});
    await fetchDetail(id, showLoading: false);
  }

  Future<void> closeTicket(int id) async {
    await _repo.closeTicket(id);
    await fetchDetail(id, showLoading: false);
  }
}
