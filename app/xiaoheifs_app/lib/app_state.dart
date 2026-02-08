import 'package:flutter/material.dart';

import 'models/session.dart';
import 'models/auth_tokens.dart';
import 'services/app_storage.dart';
import 'services/api_client.dart';
import 'services/admin_auth.dart';

class AppState extends ChangeNotifier {
  final AppStorage _storage = AppStorage();

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
    );
  }

  Future<void> load() async {
    _session = await _storage.loadSession();
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

  Future<AuthTokens?> _refreshTokens() async {
    final s = _session;
    if (s == null || s.refreshToken == null || s.refreshToken!.isEmpty) {
      return null;
    }
    final auth = AdminAuthService();
    try {
      final tokens = await auth.refresh(
        apiUrl: s.apiUrl,
        refreshToken: s.refreshToken!,
      );
      _applyTokens(tokens);
      return tokens;
    } catch (_) {
      return null;
    }
  }

  void _applyTokens(AuthTokens tokens) {
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
