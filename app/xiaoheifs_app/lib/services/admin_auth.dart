import 'dart:convert';

import 'package:http/http.dart' as http;

import '../models/auth_tokens.dart';

class AdminAuthService {
  Future<AdminLoginResult> login({
    required String apiUrl,
    required String username,
    required String password,
  }) async {
    final base = _normalizeBase(apiUrl);
    final uri = Uri.parse('$base/admin/api/v1/auth/login');
    final resp = await http.post(
      uri,
      headers: const {
        'Content-Type': 'application/json',
        'Accept': 'application/json',
      },
      body: jsonEncode({'username': username, 'password': password}),
    );
    return _decodeLogin(resp, '登录返回异常');
  }

  Future<AuthTokens> refresh({
    required String apiUrl,
    required String refreshToken,
  }) async {
    final base = _normalizeBase(apiUrl);
    final uri = Uri.parse('$base/admin/api/v1/auth/refresh');
    final resp = await http.post(
      uri,
      headers: const {
        'Content-Type': 'application/json',
        'Accept': 'application/json',
      },
      body: jsonEncode({'refresh_token': refreshToken}),
    );
    return _decodeTokens(resp, '刷新登录失败');
  }

  Future<Admin2FASetupResult> setup2FA({
    required String apiUrl,
    required String accessToken,
    String? password,
    String? currentCode,
  }) async {
    final base = _normalizeBase(apiUrl);
    final uri = Uri.parse('$base/admin/api/v1/auth/2fa/setup');
    final body = <String, dynamic>{};
    if (password != null && password.trim().isNotEmpty) {
      body['password'] = password.trim();
    }
    if (currentCode != null && currentCode.trim().isNotEmpty) {
      body['current_code'] = currentCode.trim();
    }
    final resp = await http.post(
      uri,
      headers: {
        'Content-Type': 'application/json',
        'Accept': 'application/json',
        'Authorization': 'Bearer $accessToken',
      },
      body: jsonEncode(body),
    );
    final decoded = _decodeJson(resp, '2FA 绑定初始化失败');
    final secret = decoded['secret'] as String?;
    final otpauthUrl = decoded['otpauth_url'] as String?;
    if (secret == null || secret.trim().isEmpty) {
      throw AuthException('2FA 绑定初始化失败');
    }
    return Admin2FASetupResult(
      secret: secret.trim(),
      otpauthUrl: (otpauthUrl ?? '').trim(),
    );
  }

  Future<void> confirm2FA({
    required String apiUrl,
    required String accessToken,
    required String code,
  }) async {
    final base = _normalizeBase(apiUrl);
    final uri = Uri.parse('$base/admin/api/v1/auth/2fa/confirm');
    final resp = await http.post(
      uri,
      headers: {
        'Content-Type': 'application/json',
        'Accept': 'application/json',
        'Authorization': 'Bearer $accessToken',
      },
      body: jsonEncode({'code': code.trim()}),
    );
    _decodeJson(resp, '2FA 绑定确认失败');
  }

  Future<AuthTokens> unlock2FA({
    required String apiUrl,
    required String accessToken,
    required String totpCode,
  }) async {
    final base = _normalizeBase(apiUrl);
    final uri = Uri.parse('$base/admin/api/v1/auth/2fa/unlock');
    final resp = await http.post(
      uri,
      headers: {
        'Content-Type': 'application/json',
        'Accept': 'application/json',
        'Authorization': 'Bearer $accessToken',
      },
      body: jsonEncode({'totp_code': totpCode.trim()}),
    );
    return _decodeTokens(resp, '2FA 验证失败');
  }

  String _normalizeBase(String apiUrl) {
    return apiUrl.endsWith('/')
        ? apiUrl.substring(0, apiUrl.length - 1)
        : apiUrl;
  }

  AdminLoginResult _decodeLogin(http.Response resp, String fallback) {
    final decoded = _decodeJson(resp, fallback);
    if (decoded['access_token'] is! String ||
        decoded['refresh_token'] is! String) {
      throw AuthException(fallback);
    }
    final expires = (decoded['expires_in'] is num)
        ? (decoded['expires_in'] as num)
        : 0;
    return AdminLoginResult(
      tokens: AuthTokens(
        accessToken: decoded['access_token'] as String,
        refreshToken: decoded['refresh_token'] as String,
        expiresIn: expires.toInt(),
      ),
      mfaRequired: decoded['mfa_required'] == true,
      mfaBindRequired: decoded['mfa_bind_required'] == true,
      mfaUnlocked: decoded['mfa_unlocked'] == true,
      totpEnabled: decoded['totp_enabled'] == true,
    );
  }

  AuthTokens _decodeTokens(http.Response resp, String fallback) {
    final decoded = _decodeJson(resp, fallback);
    if (decoded['access_token'] is String &&
        decoded['refresh_token'] is String) {
      final expires = (decoded['expires_in'] is num)
          ? (decoded['expires_in'] as num)
          : 0;
      return AuthTokens(
        accessToken: decoded['access_token'] as String,
        refreshToken: decoded['refresh_token'] as String,
        expiresIn: expires.toInt(),
      );
    }
    throw AuthException(fallback);
  }

  Map<String, dynamic> _decodeJson(http.Response resp, String fallback) {
    final body = resp.body.isEmpty ? '{}' : resp.body;
    final decoded = jsonDecode(body);
    if (resp.statusCode >= 200 && resp.statusCode < 300) {
      if (decoded is Map<String, dynamic>) {
        return decoded;
      }
      throw AuthException(fallback);
    }
    if (decoded is Map<String, dynamic> && decoded['error'] is String) {
      throw AuthException(decoded['error'] as String);
    }
    throw AuthException(fallback);
  }
}

class AdminLoginResult {
  final AuthTokens tokens;
  final bool mfaRequired;
  final bool mfaBindRequired;
  final bool mfaUnlocked;
  final bool totpEnabled;

  const AdminLoginResult({
    required this.tokens,
    required this.mfaRequired,
    required this.mfaBindRequired,
    required this.mfaUnlocked,
    required this.totpEnabled,
  });
}

class Admin2FASetupResult {
  final String secret;
  final String otpauthUrl;

  const Admin2FASetupResult({required this.secret, required this.otpauthUrl});
}

class AuthException implements Exception {
  final String message;
  AuthException(this.message);

  @override
  String toString() => message;
}
