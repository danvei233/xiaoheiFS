import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../../core/network/api_client.dart';
import '../../core/storage/storage_service.dart';
import '../../data/repositories/auth_repository.dart';
import '../../data/models/user.dart';
import 'vps_provider.dart';

enum AuthStatus {
  initial,
  loading,
  authenticated,
  unauthenticated,
  error,
}

class AuthState {
  final AuthStatus status;
  final User? user;
  final String? error;

  const AuthState({
    this.status = AuthStatus.initial,
    this.user,
    this.error,
  });

  AuthState copyWith({
    AuthStatus? status,
    User? user,
    String? error,
  }) {
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
  return AuthNotifier(ref.read(authRepositoryProvider));
});

class AuthNotifier extends StateNotifier<AuthState> {
  final AuthRepository _authRepository;

  AuthNotifier(this._authRepository) : super(const AuthState()) {
    _init();
  }

  Future<void> _init() async {
    if (_authRepository.isLoggedIn()) {
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
      );

      User? user = response.user;
      try {
        user = await _authRepository.getMe();
      } catch (_) {
        // Keep login response user as fallback when profile endpoint is temporarily unavailable.
      }

      state = AuthState(
        status: AuthStatus.authenticated,
        user: user,
      );
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


