import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../../core/network/api_client.dart';
import '../../core/storage/storage_service.dart';
import '../../data/repositories/auth_repository.dart';
import '../../data/models/user.dart';
import 'api_key_provider.dart';
import 'cart_provider.dart';
import 'dashboard_provider.dart';
import 'notification_provider.dart';
import 'order_provider.dart';
import 'realname_provider.dart';
import 'ticket_provider.dart';
import 'vps_provider.dart';
import 'wallet_provider.dart';

enum AuthStatus { initial, loading, authenticated, unauthenticated, error }

class AuthState {
  final AuthStatus status;
  final User? user;
  final String? error;

  const AuthState({this.status = AuthStatus.initial, this.user, this.error});

  AuthState copyWith({AuthStatus? status, User? user, String? error}) {
    return AuthState(
      status: status ?? this.status,
      user: user ?? this.user,
      error: error ?? this.error,
    );
  }

  bool get isAuthenticated => status == AuthStatus.authenticated;
  bool get isLoading => status == AuthStatus.loading;
}

final authRepositoryProvider = Provider<AuthRepository>((ref) {
  return AuthRepository();
});

final authProvider = StateNotifierProvider<AuthNotifier, AuthState>((ref) {
  return AuthNotifier(ref.read(authRepositoryProvider), ref);
});

class AuthNotifier extends StateNotifier<AuthState> {
  final AuthRepository _authRepository;
  final Ref _ref;

  AuthNotifier(this._authRepository, this._ref) : super(const AuthState()) {
    _init();
  }

  Future<void> _init() async {
    if (await _authRepository.isLoggedIn()) {
      state = state.copyWith(status: AuthStatus.loading);
      try {
        final user = await _authRepository.getMe();
        state = AuthState(status: AuthStatus.authenticated, user: user);
      } catch (_) {
        state = const AuthState(status: AuthStatus.unauthenticated);
      }
    } else {
      state = const AuthState(status: AuthStatus.unauthenticated);
    }
  }

  Future<void> login({
    required String username,
    required String password,
    String? apiUrl,
    String? captchaId,
    String? captchaCode,
    String? lotNumber,
    String? captchaOutput,
    String? passToken,
    String? genTime,
  }) async {
    state = state.copyWith(status: AuthStatus.loading, error: null);

    if (apiUrl != null && apiUrl.isNotEmpty) {
      final normalized = _normalizeApiUrl(apiUrl);
      await StorageService.instance.setApiBaseUrl(normalized);
      ApiClient.instance.updateBaseUrl(normalized);
    }

    try {
      final response = await _authRepository.login(
        username: username,
        password: password,
        captchaId: captchaId,
        captchaCode: captchaCode,
        lotNumber: lotNumber,
        captchaOutput: captchaOutput,
        passToken: passToken,
        genTime: genTime,
      );

      User? user = response.user;
      try {
        user = await _authRepository.getMe();
      } catch (_) {
        // Keep login response user as fallback when profile endpoint is temporarily unavailable.
      }

      state = AuthState(status: AuthStatus.authenticated, user: user);
    } catch (e) {
      state = AuthState(status: AuthStatus.error, error: e.toString());
      rethrow;
    }
  }

  String _normalizeApiUrl(String value) {
    var url = value.trim();
    if (url.endsWith('/')) {
      url = url.substring(0, url.length - 1);
    }
    if (!url.endsWith('/api')) {
      url = '$url/api';
    }
    return url;
  }

  Future<void> logout() async {
    VpsMonitorNotifier.stopAllActivePolling();
    await _authRepository.logout();
    state = const AuthState(status: AuthStatus.unauthenticated);
    _resetBusinessState();
  }

  /// 会话失效（401 强制登出）时清空本地认证状态与业务 provider。
  ///
  /// 与 [logout] 不同，这里不调用登出 API（token 已失效），只做本地清理，
  /// 防止下一账号在 30s TTL 缓存窗口内看到上一账号的数据。
  Future<void> handleSessionExpired() async {
    VpsMonitorNotifier.stopAllActivePolling();
    state = const AuthState(status: AuthStatus.unauthenticated);
    _resetBusinessState();
  }

  /// 清除所有已创建的业务 provider 状态与缓存，防止跨账号数据串台。
  ///
  /// 只在 provider 已存活时调用 reset()，避免在此处首次创建 provider
  /// 从而触发其构造函数的自动网络请求。non-autoDispose 的 family provider
  /// （firewall/ports/snapshots/backups/monitor）不实例化 notifier，用
  /// `invalidate` 使已创建元素失效（未创建的 family 元素是 no-op，不会
  /// 触发网络请求），下次读取自动重新拉取新账号的数据。
  void _resetBusinessState() {
    if (_ref.exists(orderListProvider)) {
      _ref.read(orderListProvider.notifier).reset();
    }
    if (_ref.exists(orderDetailProvider)) {
      _ref.read(orderDetailProvider.notifier).reset();
    }
    if (_ref.exists(vpsListProvider)) {
      _ref.read(vpsListProvider.notifier).reset();
    }
    if (_ref.exists(vpsDetailProvider)) {
      _ref.read(vpsDetailProvider.notifier).reset();
    }
    if (_ref.exists(walletProvider)) {
      _ref.read(walletProvider.notifier).reset();
    }
    if (_ref.exists(notificationProvider)) {
      _ref.read(notificationProvider.notifier).reset();
    }
    if (_ref.exists(apiKeyProvider)) {
      _ref.read(apiKeyProvider.notifier).reset();
    }
    if (_ref.exists(ticketListProvider)) {
      _ref.read(ticketListProvider.notifier).reset();
    }
    if (_ref.exists(ticketDetailProvider)) {
      _ref.read(ticketDetailProvider.notifier).reset();
    }
    if (_ref.exists(cartProvider)) {
      _ref.read(cartProvider.notifier).reset();
    }
    if (_ref.exists(dashboardProvider)) {
      _ref.read(dashboardProvider.notifier).reset();
    }
    if (_ref.exists(realnameProvider)) {
      _ref.read(realnameProvider.notifier).reset();
    }
    // non-autoDispose family providers：失效全部已创建元素，跨账号不得复用旧缓存。
    _ref.invalidate(vpsFirewallProvider);
    _ref.invalidate(vpsPortsProvider);
    _ref.invalidate(vpsSnapshotsProvider);
    _ref.invalidate(vpsBackupsProvider);
    _ref.invalidate(vpsMonitorStateProvider);
  }

  Future<void> refreshUser() async {
    try {
      final user = await _authRepository.getMe();
      state = state.copyWith(user: user);
    } catch (_) {}
  }

  Future<void> updateUserInfo(Map<String, dynamic> data) async {
    try {
      final user = await _authRepository.updateMe(data);
      state = state.copyWith(user: user);
    } catch (e) {
      state = state.copyWith(error: e.toString());
      rethrow;
    }
  }
}
