import 'dart:convert';

import 'package:http/http.dart' as http;

import '../models/auth_tokens.dart';

class AdminAuthService {
  Future<AuthTokens> login({
    required String apiUrl,
    required String username,
    required String password,
  }) async {
    final base = apiUrl.endsWith('/')
        ? apiUrl.substring(0, apiUrl.length - 1)
        : apiUrl;
    final uri = Uri.parse('$base/admin/api/v1/auth/login');
    final resp = await http.post(
      uri,
      headers: const {
        'Content-Type': 'application/json',
        'Accept': 'application/json',
      },
      body: jsonEncode({'username': username, 'password': password}),
    );
    return _decodeTokens(resp, '鐧诲綍杩斿洖寮傚父');
  }

  Future<AuthTokens> refresh({
    required String apiUrl,
    required String refreshToken,
  }) async {
    final base = apiUrl.endsWith('/')
        ? apiUrl.substring(0, apiUrl.length - 1)
        : apiUrl;
    final uri = Uri.parse('$base/admin/api/v1/auth/refresh');
    final resp = await http.post(
      uri,
      headers: const {
        'Content-Type': 'application/json',
        'Accept': 'application/json',
      },
      body: jsonEncode({'refresh_token': refreshToken}),
    );
    return _decodeTokens(resp, '鍒锋柊鐧诲綍澶辫触');
  }

  AuthTokens _decodeTokens(http.Response resp, String fallback) {
    final body = resp.body.isEmpty ? '{}' : resp.body;
    final decoded = jsonDecode(body);
    if (resp.statusCode >= 200 && resp.statusCode < 300) {
      if (decoded is Map<String, dynamic> &&
          decoded['access_token'] is String &&
          decoded['refresh_token'] is String) {
        final expires =
            (decoded['expires_in'] is num) ? (decoded['expires_in'] as num) : 0;
        return AuthTokens(
          accessToken: decoded['access_token'] as String,
          refreshToken: decoded['refresh_token'] as String,
          expiresIn: expires.toInt(),
        );
      }
      throw AuthException(fallback);
    }
    if (decoded is Map<String, dynamic> && decoded['error'] is String) {
      throw AuthException(decoded['error'] as String);
    }
    throw AuthException(fallback);
  }
}

class AuthException implements Exception {
  final String message;
  AuthException(this.message);

  @override
  String toString() => message;
}
