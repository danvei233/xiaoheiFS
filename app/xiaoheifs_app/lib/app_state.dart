import 'package:flutter/material.dart';
import 'package:http/http.dart' as http;

import 'models/session.dart';
import 'models/auth_tokens.dart';
import 'services/app_storage.dart';
import 'services/api_client.dart';
import 'services/admin_auth.dart';

class AppState extends ChangeNotifier {
  final AppStorage _storage;
  final AdminAuthService _auth;

  /// 测试注入点：为 apiClient 提供统一的 http.Client（生产为 null）。
  final http.Client? _clientOverride;

  AppState({
    AppStorage? storage,
    AdminAuthService? authService,
    http.Client? clientOverride,
  }) : _storage = storage ?? AppStorage(),
       _auth = authService ?? AdminAuthService(),
       _clientOverride = clientOverride;

  Session? _session;
  bool _isReady = false;

  bool get isReady => _isReady;
  bool get isLoggedIn => _session != null;
  Session? get session => _session;

  ApiClient? get apiClient {
    final s = _session;
    if (s == null) return null;
    return ApiClient(
      baseUrl: s.apiUrl,
      token: s.token,
      apiKey: s.apiKey,
      refreshAuth: s.authType == 'password' ? _refreshTokens : null,
      onTokens: _applyTokens,
      onAuthExpired: () => _handleAuthExpiredFor(s),
      client: _clientOverride,
    );
  }

  Future<void> load() async {
    final session = await _storage.loadSession();
    if (session != null &&
        session.tokenExpiresAt != null &&
        !session.tokenExpiresAt!.isAfter(DateTime.now())) {
      // The stored access token has already expired: clear the stale session
      // (same as logout) instead of loading a dead "logged in" state, so the
      // app returns to the login screen at startup rather than serving 401s.
      _session = null;
      await _storage.clearSession();
    } else {
      _session = session;
    }
    _isReady = true;
    notifyListeners();
  }

  Future<void> loginWithApiKey({
    required String apiUrl,
    required String apiKey,
    required String username,
  }) async {
    _session = Session(
      apiUrl: apiUrl,
      apiKey: apiKey,
      username: username,
      authType: 'api_key',
    );
    await _storage.saveSession(_session!);
    notifyListeners();
  }

  Future<void> loginWithPassword({
    required String apiUrl,
    required AuthTokens tokens,
    required String username,
    String? email,
  }) async {
    final expiresAt = tokens.expiresIn > 0
        ? DateTime.now().add(Duration(seconds: tokens.expiresIn))
        : null;
    _session = Session(
      apiUrl: apiUrl,
      token: tokens.accessToken,
      refreshToken: tokens.refreshToken,
      tokenExpiresAt: expiresAt,
      username: username,
      email: email,
      authType: 'password',
    );
    await _storage.saveSession(_session!);
    notifyListeners();
  }

  Future<void> logout() async {
    _session = null;
    await _storage.clearSession();
    notifyListeners();
  }

  Future<void> updateProfile({
    String? apiUrl,
    String? username,
    String? token,
    String? refreshToken,
    DateTime? tokenExpiresAt,
    String? apiKey,
    String? email,
  }) async {
    if (_session == null) return;
    _session = _session!.copyWith(
      apiUrl: apiUrl,
      username: username,
      token: token,
      refreshToken: refreshToken,
      tokenExpiresAt: tokenExpiresAt,
      apiKey: apiKey,
      email: email,
    );
    await _storage.saveSession(_session!);
    notifyListeners();
  }

  /// 串行化并发 401 刷新：多个页面同时收到 401 时只发起一次 refresh，
  /// 其余调用共享同一个 in-flight future。若后端未来改为「用后作废」式
  /// refresh token 轮换，这能避免除首个外全部 refresh 失败被误判为过期。
  ///
  /// `_refreshFutureSession` 绑定该 in-flight refresh 所属的会话身份
  /// （apiUrl + refreshToken）。logout/login 换号后身份变化，新会话不会复用
  /// 旧会话的 in-flight future，旧会话的 waiter 也不能清掉新会话的新一代
  /// future（identical 守卫）。
  Future<AuthTokens?>? _refreshFuture;
  String? _refreshFutureSession;

  String? _sessionIdentity() {
    final s = _session;
    if (s == null || s.refreshToken == null || s.refreshToken!.isEmpty) {
      return null;
    }
    return '${s.apiUrl}|${s.refreshToken}';
  }

  Future<AuthTokens?> _refreshTokens() {
    final identity = _sessionIdentity();
    final existing = _refreshFuture;
    if (existing != null && _refreshFutureSession == identity) {
      return existing;
    }
    final future = _doRefreshTokens();
    _refreshFuture = future;
    _refreshFutureSession = identity;
    future.whenComplete(() {
      if (identical(_refreshFuture, future)) {
        _refreshFuture = null;
        _refreshFutureSession = null;
      }
    });
    return future;
  }

  Future<AuthTokens?> _doRefreshTokens() async {
    final s = _session;
    if (s == null || s.refreshToken == null || s.refreshToken!.isEmpty) {
      return null;
    }
    final identity = '${s.apiUrl}|${s.refreshToken}';
    try {
      final tokens = await _auth.refresh(
        apiUrl: s.apiUrl,
        refreshToken: s.refreshToken!,
      );
      // 会话代次守卫：refresh 结果只能写回「发起它的那个会话」。若期间
      // logout/login 换号（账号 A -> 账号 B），当前会话身份已变化——丢弃
      // 结果，账号 A 的 refresh 永不覆盖账号 B 的 storage/state。
      final current = _session;
      if (current == null ||
          current.refreshToken == null ||
          '${current.apiUrl}|${current.refreshToken}' != identity) {
        return null;
      }
      _applyTokens(tokens);
      return tokens;
    } catch (_) {
      return null;
    }
  }

  /// Called when the access token can no longer be refreshed (401 + failed
  /// refresh). Clears the stale session so the UI returns to the login screen
  /// instead of leaving a dead "logged in" state behind.
  void _handleAuthExpired() {
    if (_session == null) return;
    logout();
  }

  /// 会话绑定版过期处理：只有 [expected]（创建该 ApiClient 的会话）仍是当前
  /// 会话时才登出。账号 A 的旧 ApiClient 在 logout/login 为 B 后收到 401，
  /// 绝不能把账号 B 踢下线。
  void _handleAuthExpiredFor(Session expected) {
    final current = _session;
    if (current == null) return;
    if (current.apiUrl != expected.apiUrl ||
        current.refreshToken != expected.refreshToken ||
        current.apiKey != expected.apiKey ||
        current.token != expected.token) {
      return; // 会话已切换；旧会话的过期回调不得影响新账号。
    }
    logout();
  }

  void _applyTokens(AuthTokens tokens) {
    // 会话已登出（null）时不再写回任何令牌；配合 _doRefreshTokens 的代次
    // 守卫，旧账号的 refresh 结果绝不可能覆盖新账号的 storage/state。
    if (_session == null) return;
    final expiresAt = tokens.expiresIn > 0
        ? DateTime.now().add(Duration(seconds: tokens.expiresIn))
        : null;
    updateProfile(
      token: tokens.accessToken,
      refreshToken: tokens.refreshToken,
      tokenExpiresAt: expiresAt,
    );
  }
}
