import 'dart:convert';
import 'dart:io';

import 'package:flutter/foundation.dart';
import 'package:path_provider/path_provider.dart';
import 'package:shared_preferences/shared_preferences.dart';

import '../models/session.dart';

class AppStorage {
  static const _keyApiUrl = 'api_url';
  static const _keyApiKey = 'api_key';
  static const _keyUsername = 'username';
  static const _keyToken = 'token';
  static const _keyRefreshToken = 'refresh_token';
  static const _keyTokenExpiresAt = 'token_expires_at';
  static const _keyEmail = 'email';
  static const _keyAuthType = 'auth_type';
  static const _fileName = 'session.json';

  Future<Session?> loadSession() async {
    Session? session;
    try {
      final prefs = await SharedPreferences.getInstance();
      session = _fromPrefs(prefs);
    } catch (_) {}
    if (session != null) return session;
    return _loadFromFile();
  }

  Session? _fromPrefs(SharedPreferences prefs) {
    final apiUrl = prefs.getString(_keyApiUrl);
    final storedAuthType = prefs.getString(_keyAuthType);
    final authType = storedAuthType ?? 'api_key';
    final apiKey = prefs.getString(_keyApiKey);
    final token = prefs.getString(_keyToken);
    final refreshToken = prefs.getString(_keyRefreshToken);
    final expiresRaw = prefs.getString(_keyTokenExpiresAt);
    final tokenExpiresAt =
        expiresRaw == null ? null : DateTime.tryParse(expiresRaw);
    if (apiUrl == null || apiUrl.isEmpty) {
      return null;
    }
    if (storedAuthType == null && token != null && token.isNotEmpty) {
      return Session(
        apiUrl: apiUrl,
        apiKey: apiKey,
        token: token,
        refreshToken: refreshToken,
        tokenExpiresAt: tokenExpiresAt,
        username: prefs.getString(_keyUsername) ?? '管理员',
        email: prefs.getString(_keyEmail),
        authType: 'password',
      );
    }
    if (authType == 'password' && (token == null || token.isEmpty)) {
      return null;
    }
    if (authType == 'api_key' && (apiKey == null || apiKey.isEmpty)) {
      return null;
    }
    final username = prefs.getString(_keyUsername) ?? '管理员';
    final email = prefs.getString(_keyEmail);
    return Session(
      apiUrl: apiUrl,
      apiKey: apiKey,
      token: token,
      refreshToken: refreshToken,
      tokenExpiresAt: tokenExpiresAt,
      username: username,
      email: email,
      authType: authType,
    );
  }

  Future<void> saveSession(Session session) async {
    try {
      final prefs = await SharedPreferences.getInstance();
      await prefs.setString(_keyApiUrl, session.apiUrl);
      await prefs.setString(_keyUsername, session.username);
      await prefs.setString(_keyAuthType, session.authType);
      if (session.apiKey != null) {
        await prefs.setString(_keyApiKey, session.apiKey!);
      } else {
        await prefs.remove(_keyApiKey);
      }
      if (session.token != null) {
        await prefs.setString(_keyToken, session.token!);
      } else {
        await prefs.remove(_keyToken);
      }
      if (session.refreshToken != null) {
        await prefs.setString(_keyRefreshToken, session.refreshToken!);
      } else {
        await prefs.remove(_keyRefreshToken);
      }
      if (session.tokenExpiresAt != null) {
        await prefs.setString(
          _keyTokenExpiresAt,
          session.tokenExpiresAt!.toIso8601String(),
        );
      } else {
        await prefs.remove(_keyTokenExpiresAt);
      }
      if (session.email != null) {
        await prefs.setString(_keyEmail, session.email!);
      } else {
        await prefs.remove(_keyEmail);
      }
    } catch (_) {}
    await _saveToFile(session);
  }

  Future<void> clearSession() async {
    try {
      final prefs = await SharedPreferences.getInstance();
      await prefs.remove(_keyApiKey);
      await prefs.remove(_keyUsername);
      await prefs.remove(_keyToken);
      await prefs.remove(_keyRefreshToken);
      await prefs.remove(_keyTokenExpiresAt);
      await prefs.remove(_keyEmail);
      await prefs.remove(_keyAuthType);
    } catch (_) {}
    try {
      final file = await _sessionFile();
      if (await file.exists()) {
        await file.delete();
      }
    } catch (_) {}
  }

  Future<String?> loadApiUrl() async {
    try {
      final prefs = await SharedPreferences.getInstance();
      final apiUrl = prefs.getString(_keyApiUrl);
      if (apiUrl == null || apiUrl.trim().isEmpty) return null;
      return apiUrl.trim();
    } catch (_) {
      return null;
    }
  }

  Future<void> saveApiUrl(String apiUrl) async {
    final value = apiUrl.trim();
    if (value.isEmpty) return;
    try {
      final prefs = await SharedPreferences.getInstance();
      await prefs.setString(_keyApiUrl, value);
    } catch (_) {}
  }
}

Future<File> _sessionFile() async {
  if (kIsWeb) {
    throw UnsupportedError('No file storage on web');
  }
  final dir = await getApplicationSupportDirectory();
  return File('${dir.path}${Platform.pathSeparator}${AppStorage._fileName}');
}

Future<void> _saveToFile(Session session) async {
  if (kIsWeb) return;
  final file = await _sessionFile();
  final payload = {
    'api_url': session.apiUrl,
    'api_key': session.apiKey,
    'token': session.token,
    'refresh_token': session.refreshToken,
    'token_expires_at': session.tokenExpiresAt?.toIso8601String(),
    'username': session.username,
    'email': session.email,
    'auth_type': session.authType,
  };
  await file.writeAsString(jsonEncode(payload));
}

Future<Session?> _loadFromFile() async {
  try {
    if (kIsWeb) return null;
    final file = await _sessionFile();
    if (!await file.exists()) return null;
    final raw = await file.readAsString();
    final decoded = jsonDecode(raw);
    if (decoded is! Map<String, dynamic>) return null;
    final apiUrl = (decoded['api_url'] as String?) ?? '';
    final authType = (decoded['auth_type'] as String?) ?? 'api_key';
    if (apiUrl.isEmpty) return null;
    final token = decoded['token'] as String?;
    final refreshToken = decoded['refresh_token'] as String?;
    final expiresRaw = decoded['token_expires_at'] as String?;
    final tokenExpiresAt =
        expiresRaw == null ? null : DateTime.tryParse(expiresRaw);
    final apiKey = decoded['api_key'] as String?;
    if (authType == 'password' && (token == null || token.isEmpty)) return null;
    if (authType == 'api_key' && (apiKey == null || apiKey.isEmpty)) return null;
    return Session(
      apiUrl: apiUrl,
      apiKey: apiKey,
      token: token,
      refreshToken: refreshToken,
      tokenExpiresAt: tokenExpiresAt,
      username: (decoded['username'] as String?) ?? '管理员',
      email: decoded['email'] as String?,
      authType: authType,
    );
  } catch (_) {
    return null;
  }
}
