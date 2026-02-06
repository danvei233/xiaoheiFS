import 'dart:convert';

import 'package:http/http.dart' as http;

class AdminAuthService {
  Future<String> login({
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
    final body = resp.body.isEmpty ? '{}' : resp.body;
    final decoded = jsonDecode(body);
    if (resp.statusCode >= 200 && resp.statusCode < 300) {
      if (decoded is Map<String, dynamic> &&
          decoded['access_token'] is String) {
        return decoded['access_token'] as String;
      }
      throw AuthException('登录返回异常');
    }
    if (decoded is Map<String, dynamic> && decoded['error'] is String) {
      throw AuthException(decoded['error'] as String);
    }
    throw AuthException('登录失败');
  }
}

class AuthException implements Exception {
  final String message;
  AuthException(this.message);

  @override
  String toString() => message;
}
