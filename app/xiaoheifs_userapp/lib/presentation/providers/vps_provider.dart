import 'dart:async';

import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:intl/intl.dart';
import '../../data/repositories/vps_repository.dart';

class VpsListState {
  final bool loading;
  final List<Map<String, dynamic>> items;
  final String? error;

  const VpsListState({
    this.loading = false,
    this.items = const [],
    this.error,
  });

  VpsListState copyWith({
    bool? loading,
    List<Map<String, dynamic>>? items,
    String? error,
  }) {
    return VpsListState(
      loading: loading ?? this.loading,
      items: items ?? this.items,
      error: error,
    );
  }
}

class VpsDetailState {
  final bool loading;
  final Map<String, dynamic>? detail;
  final String? error;

  const VpsDetailState({
    this.loading = false,
    this.detail,
    this.error,
  });

  VpsDetailState copyWith({
    bool? loading,
    Map<String, dynamic>? detail,
    String? error,
  }) {
    return VpsDetailState(
      loading: loading ?? this.loading,
      detail: detail ?? this.detail,
      error: error,
    );
  }
}

final vpsRepositoryProvider = Provider<VpsRepository>((ref) {
  return VpsRepository();
});

final vpsListProvider = StateNotifierProvider<VpsListNotifier, VpsListState>((ref) {
  return VpsListNotifier(ref.read(vpsRepositoryProvider));
});

final vpsDetailProvider = StateNotifierProvider<VpsDetailNotifier, VpsDetailState>((ref) {
  return VpsDetailNotifier(ref.read(vpsRepositoryProvider));
});

final vpsMonitorStateProvider =
    StateNotifierProvider.family<VpsMonitorNotifier, VpsMonitorState, int>((ref, id) {
  return VpsMonitorNotifier(ref, ref.read(vpsRepositoryProvider), id);
});

final vpsFirewallProvider = FutureProvider.family<List<Map<String, dynamic>>, int>((ref, id) {
  return ref.read(vpsRepositoryProvider).listFirewallRules(id);
});

final vpsPortsProvider = FutureProvider.family<List<Map<String, dynamic>>, int>((ref, id) {
  return ref.read(vpsRepositoryProvider).listPortMappings(id);
});

final vpsSnapshotsProvider = FutureProvider.family<List<Map<String, dynamic>>, int>((ref, id) {
  return ref.read(vpsRepositoryProvider).listSnapshots(id);
});

final vpsBackupsProvider = FutureProvider.family<List<Map<String, dynamic>>, int>((ref, id) {
  return ref.read(vpsRepositoryProvider).listBackups(id);
});

class VpsListNotifier extends StateNotifier<VpsListState> {
  VpsListNotifier(this._repo) : super(const VpsListState());

  final VpsRepository _repo;
  DateTime? _lastFetchedAt;
  final Duration _cacheTtl = const Duration(seconds: 30);

  Future<void> fetch({bool force = false}) async {
    if (!force &&
        state.items.isNotEmpty &&
        _lastFetchedAt != null &&
        DateTime.now().difference(_lastFetchedAt!) < _cacheTtl) {
      return;
    }
    state = state.copyWith(loading: true, error: null);
    try {
      final items = await _repo.listVps();
      _lastFetchedAt = DateTime.now();
      state = state.copyWith(loading: false, items: items);
    } catch (e) {
      state = state.copyWith(loading: false, error: e.toString());
    }
  }

  Future<void> refresh() async => fetch(force: true);
}

class VpsDetailNotifier extends StateNotifier<VpsDetailState> {
  VpsDetailNotifier(this._repo) : super(const VpsDetailState());

  final VpsRepository _repo;

  Future<void> fetch(int id) async {
    state = state.copyWith(loading: true, error: null);
    try {
      final detail = await _repo.getDetail(id);
      state = state.copyWith(loading: false, detail: detail);
    } catch (e) {
      state = state.copyWith(loading: false, error: e.toString());
    }
  }

  Future<void> refresh(int id) async {
    await _repo.refresh(id);
    await fetch(id);
  }

  void applyMonitorUpdate(Map<String, dynamic> data) {
    if (state.detail == null) return;
    final next = Map<String, dynamic>.from(state.detail!);
    if (data['status'] != null) next['status'] = data['status'];
    if (data['automation_state'] != null) next['automation_state'] = data['automation_state'];
    if (data['access_info'] != null) next['access_info'] = data['access_info'];
    if (data['spec'] != null) next['spec'] = data['spec'];
    state = state.copyWith(detail: next);
  }
}

class VpsMonitorSeries {
  final List<String> labels;
  final List<double> values;

  const VpsMonitorSeries({this.labels = const [], this.values = const []});

  VpsMonitorSeries push(double value, {int maxPoints = 20}) {
    final nextLabels = List<String>.from(labels);
    final nextValues = List<double>.from(values);
    nextLabels.add(DateFormat.Hms().format(DateTime.now()));
    nextValues.add(value);
    if (nextLabels.length > maxPoints) {
      nextLabels.removeAt(0);
    }
    if (nextValues.length > maxPoints) {
      nextValues.removeAt(0);
    }
    return VpsMonitorSeries(labels: nextLabels, values: nextValues);
  }
}

class VpsMonitorState {
  final bool loading;
  final String? error;
  final VpsMonitorSeries cpu;
  final VpsMonitorSeries memory;
  final VpsMonitorSeries trafficIn;
  final VpsMonitorSeries trafficOut;

  const VpsMonitorState({
    this.loading = false,
    this.error,
    this.cpu = const VpsMonitorSeries(),
    this.memory = const VpsMonitorSeries(),
    this.trafficIn = const VpsMonitorSeries(),
    this.trafficOut = const VpsMonitorSeries(),
  });

  VpsMonitorState copyWith({
    bool? loading,
    String? error,
    VpsMonitorSeries? cpu,
    VpsMonitorSeries? memory,
    VpsMonitorSeries? trafficIn,
    VpsMonitorSeries? trafficOut,
  }) {
    return VpsMonitorState(
      loading: loading ?? this.loading,
      error: error,
      cpu: cpu ?? this.cpu,
      memory: memory ?? this.memory,
      trafficIn: trafficIn ?? this.trafficIn,
      trafficOut: trafficOut ?? this.trafficOut,
    );
  }
}

class VpsMonitorNotifier extends StateNotifier<VpsMonitorState> {
  VpsMonitorNotifier(this._ref, this._repo, this._id) : super(const VpsMonitorState()) {
    _startPolling();
  }

  final Ref _ref;
  final VpsRepository _repo;
  final int _id;
  Timer? _timer;

  void _startPolling() {
    _timer?.cancel();
    fetchOnce();
    _timer = Timer.periodic(const Duration(seconds: 10), (_) => fetchOnce(silent: true));
  }

  void stopPolling() {
    _timer?.cancel();
    _timer = null;
  }

  Future<void> fetchOnce({bool silent = false}) async {
    final currentDetail = _ref.read(vpsDetailProvider).detail;
    final currentId = currentDetail?['id'] ?? currentDetail?['ID'];
    if (currentId != null && currentId.toString() != _id.toString()) {
      stopPolling();
      return;
    }
    if (!silent) {
      state = state.copyWith(loading: true, error: null);
    }
    try {
      final data = await _repo.getMonitor(_id);
      _ref.read(vpsDetailProvider.notifier).applyMonitorUpdate(data);
      final cpu = _toDouble(data['cpu']);
      final memory = _toDouble(data['memory']);
      final trafficIn =
          _toDouble(data['bytes_in'] ?? data['in_bytes'] ?? data['rx_bytes'] ?? data['in']) /
              1024;
      final trafficOut =
          _toDouble(data['bytes_out'] ?? data['out_bytes'] ?? data['tx_bytes'] ?? data['out']) /
              1024;
      state = state.copyWith(
        loading: false,
        error: null,
        cpu: state.cpu.push(cpu),
        memory: state.memory.push(memory),
        trafficIn: state.trafficIn.push(trafficIn),
        trafficOut: state.trafficOut.push(trafficOut),
      );
    } catch (e) {
      if (!silent) {
        state = state.copyWith(loading: false, error: e.toString());
      }
    }
  }

  double _toDouble(dynamic value) {
    if (value == null) return 0;
    if (value is num) return value.toDouble();
    return double.tryParse(value.toString()) ?? 0;
  }

  @override
  void dispose() {
    stopPolling();
    super.dispose();
  }
}
