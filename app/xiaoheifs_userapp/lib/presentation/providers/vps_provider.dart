import 'dart:async';

import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:intl/intl.dart';
import '../../core/storage/storage_service.dart';
import '../../data/repositories/vps_repository.dart';

class VpsListState {
  final bool loading;
  final List<Map<String, dynamic>> items;
  final String? error;

  const VpsListState({this.loading = false, this.items = const [], this.error});

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
  /// 哨兵值：用于区分「未提供 detail 参数」与「显式清空 detail」。
  static const Object _unset = Object();

  final bool loading;
  final Map<String, dynamic>? detail;
  final String? error;

  const VpsDetailState({this.loading = false, this.detail, this.error});

  VpsDetailState copyWith({
    bool? loading,
    Object? detail = _unset,
    String? error,
  }) {
    return VpsDetailState(
      loading: loading ?? this.loading,
      detail: identical(detail, _unset)
          ? this.detail
          : detail as Map<String, dynamic>?,
      error: error,
    );
  }
}

final vpsRepositoryProvider = Provider<VpsRepository>((ref) {
  return VpsRepository();
});

final vpsListProvider = StateNotifierProvider<VpsListNotifier, VpsListState>((
  ref,
) {
  return VpsListNotifier(ref.read(vpsRepositoryProvider));
});

final vpsDetailProvider =
    StateNotifierProvider<VpsDetailNotifier, VpsDetailState>((ref) {
      return VpsDetailNotifier(ref.read(vpsRepositoryProvider));
    });

final vpsMonitorStateProvider =
    StateNotifierProvider.family<VpsMonitorNotifier, VpsMonitorState, int>((
      ref,
      id,
    ) {
      return VpsMonitorNotifier(ref, ref.read(vpsRepositoryProvider), id);
    });

final vpsFirewallProvider =
    FutureProvider.family<List<Map<String, dynamic>>, int>((ref, id) {
      return ref.read(vpsRepositoryProvider).listFirewallRules(id);
    });

final vpsPortsProvider = FutureProvider.family<List<Map<String, dynamic>>, int>(
  (ref, id) {
    return ref.read(vpsRepositoryProvider).listPortMappings(id);
  },
);

final vpsSnapshotsProvider =
    FutureProvider.family<List<Map<String, dynamic>>, int>((ref, id) {
      return ref.read(vpsRepositoryProvider).listSnapshots(id);
    });

final vpsBackupsProvider =
    FutureProvider.family<List<Map<String, dynamic>>, int>((ref, id) {
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

  /// 重置为初始状态（登出时调用）。
  void reset() {
    _lastFetchedAt = null;
    state = const VpsListState();
  }
}

class VpsDetailNotifier extends StateNotifier<VpsDetailState> {
  VpsDetailNotifier(this._repo) : super(const VpsDetailState());

  final VpsRepository _repo;

  /// 请求代次守卫：每次 fetch/reset 递增。只有「本次请求」的响应可以写状态，
  /// 乱序返回（旧 VPS 慢响应晚于新 VPS 到达）或跨会话（登出后旧响应到达）一律
  /// 丢弃，防止把旧账号/旧 VPS 响应写入新会话。
  int _generation = 0;

  Future<void> fetch(int id) async {
    final generation = ++_generation;
    final currentId = state.detail?['id'] ?? state.detail?['ID'];
    final sameInstance =
        currentId != null && currentId.toString() == id.toString();
    // 切换实例时先清空上个实例的 detail（含 access_info 密码），
    // 避免新实例加载完成前短暂展示上一个实例的数据。
    state = state.copyWith(
      loading: true,
      error: null,
      detail: sameInstance ? state.detail : null,
    );
    try {
      final detail = await _repo.getDetail(id);
      if (generation != _generation) return; // 过期响应（乱序/已登出）丢弃
      state = state.copyWith(loading: false, detail: detail);
    } catch (e) {
      if (generation != _generation) return;
      state = state.copyWith(loading: false, error: e.toString());
    }
  }

  Future<void> refresh(int id) async {
    await _repo.refresh(id);
    await fetch(id);
  }

  /// 重置为初始状态（登出时调用）。同时递增代次，令在途请求的响应作废。
  void reset() {
    _generation++;
    state = const VpsDetailState();
  }

  void applyMonitorUpdate(Map<String, dynamic> data) {
    if (state.detail == null) return;
    final next = Map<String, dynamic>.from(state.detail!);
    if (data['status'] != null) next['status'] = data['status'];
    if (data['automation_state'] != null)
      next['automation_state'] = data['automation_state'];
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
  VpsMonitorNotifier(this._ref, this._repo, this._id)
    : super(const VpsMonitorState()) {
    _instances.add(this);
    _startPolling();
  }

  static final Set<VpsMonitorNotifier> _instances = <VpsMonitorNotifier>{};
  static void stopAllActivePolling() {
    for (final notifier in _instances.toList()) {
      notifier.stopPolling();
    }
  }

  final Ref _ref;
  final VpsRepository _repo;
  final int _id;
  Timer? _timer;

  /// 代次守卫：stopPolling/dispose 时递增，令在途 monitor 响应作废，
  /// 防止登出/切换 VPS 后旧账号/旧 VPS 的监控数据写入新会话。
  int _generation = 0;

  void _startPolling() {
    _timer?.cancel();
    fetchOnce();
    _timer = Timer.periodic(
      const Duration(seconds: 10),
      (_) => fetchOnce(silent: true),
    );
  }

  void stopPolling() {
    _generation++;
    _timer?.cancel();
    _timer = null;
  }

  Future<void> fetchOnce({bool silent = false}) async {
    final generation = _generation;
    final token = await StorageService.instance.getAccessToken();
    if (token == null || token.isEmpty) {
      stopPolling();
      return;
    }
    final currentDetail = _ref.read(vpsDetailProvider).detail;
    final currentId = currentDetail?['id'] ?? currentDetail?['ID'];
    if (currentId != null && currentId.toString() != _id.toString()) {
      // 已切换到别的 VPS：停止轮询。detail 尚为 null（新页面加载中）时继续，
      // 等 detail 就绪后再拉取。
      stopPolling();
      return;
    }
    if (currentId == null) {
      // detail 还没就绪：本次不拉取，留给下一轮定时器。
      return;
    }
    if (!silent) {
      state = state.copyWith(loading: true, error: null);
    }
    try {
      final data = await _repo.getMonitor(_id);
      // 过期响应丢弃：期间 stopPolling/登出/切换 VPS 都会递增代次。
      if (generation != _generation) return;
      // 响应到达时再次核对仍在本会话查看同一台 VPS（防跨账号/跨 VPS 串台）。
      final nowDetail = _ref.read(vpsDetailProvider).detail;
      final nowId = nowDetail?['id'] ?? nowDetail?['ID'];
      if (nowId == null || nowId.toString() != _id.toString()) return;
      _ref.read(vpsDetailProvider.notifier).applyMonitorUpdate(data);
      final cpu = _toDouble(data['cpu']);
      final memory = _toDouble(data['memory']);
      final trafficIn =
          _toDouble(
            data['bytes_in'] ??
                data['in_bytes'] ??
                data['rx_bytes'] ??
                data['in'],
          ) /
          1024;
      final trafficOut =
          _toDouble(
            data['bytes_out'] ??
                data['out_bytes'] ??
                data['tx_bytes'] ??
                data['out'],
          ) /
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
      final message = e.toString().toLowerCase();
      if (message.contains('401') ||
          message.contains('invalid token') ||
          message.contains('unauthorized')) {
        stopPolling();
      }
      if (!silent && generation == _generation) {
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
    _instances.remove(this);
    stopPolling();
    super.dispose();
  }
}
