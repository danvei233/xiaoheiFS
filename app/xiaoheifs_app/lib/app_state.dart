import 'package:flutter/material.dart';

import 'models/session.dart';
import 'services/app_storage.dart';
import 'services/api_client.dart';

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
    return ApiClient(baseUrl: s.apiUrl, token: s.token, apiKey: s.apiKey);
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
    required String token,
    required String username,
    String? email,
  }) async {
    _session = Session(
      apiUrl: apiUrl,
      token: token,
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
    String? apiKey,
    String? email,
  }) async {
    if (_session == null) return;
    _session = _session!.copyWith(
      apiUrl: apiUrl,
      username: username,
      token: token,
      apiKey: apiKey,
      email: email,
    );
    await _storage.saveSession(_session!);
    notifyListeners();
  }
}
