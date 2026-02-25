import 'dart:convert';

import 'package:http/http.dart' as http;

import '../models/auth_tokens.dart';

class ApiClient {
  final String baseUrl;
  String? token;
  final String? apiKey;
  final Future<AuthTokens?> Function()? refreshAuth;
  final void Function(AuthTokens tokens)? onTokens;

  ApiClient({
    required this.baseUrl,
    this.token,
    this.apiKey,
    this.refreshAuth,
    this.onTokens,
  });

  Uri _buildUri(String path, [Map<String, String>? query]) {
    final normalizedBase = baseUrl.endsWith('/')
        ? baseUrl.substring(0, baseUrl.length - 1)
        : baseUrl;
    final normalizedPath = path.startsWith('/') ? path : '/$path';
    final uri = Uri.parse('$normalizedBase$normalizedPath');
    if (query == null || query.isEmpty) {
      return uri;
    }
    return uri.replace(queryParameters: query);
  }

  Map<String, String> _headers() {
    final auth = token?.isNotEmpty == true
        ? token
        : (apiKey?.isNotEmpty == true ? apiKey : null);
    return {
      if (auth != null) 'Authorization': 'Bearer $auth',
      'Content-Type': 'application/json',
      'Accept': 'application/json',
    };
  }

  Future<Map<String, dynamic>> getJson(
    String path, {
    Map<String, String>? query,
  }) async {
    return _request('GET', path, query: query);
  }

  Future<Map<String, dynamic>> postJson(
    String path, {
    Object? body,
    Map<String, String>? query,
  }) async {
    return _request('POST', path, body: body, query: query);
  }

  Future<Map<String, dynamic>> patchJson(
    String path, {
    Object? body,
    Map<String, String>? query,
  }) async {
    return _request('PATCH', path, body: body, query: query);
  }

  Future<Map<String, dynamic>> putJson(
    String path, {
    Object? body,
    Map<String, String>? query,
  }) async {
    return _request('PUT', path, body: body, query: query);
  }

  Future<Map<String, dynamic>> deleteJson(
    String path, {
    Object? body,
    Map<String, String>? query,
  }) async {
    return _request('DELETE', path, body: body, query: query);
  }

  Future<Map<String, dynamic>> _request(
    String method,
    String path, {
    Object? body,
    Map<String, String>? query,
  }) async {
    final uri = _buildUri(path, query);
    final payload = body == null ? null : jsonEncode(body);
    http.Response resp = await _send(method, uri, payload);
    if (resp.statusCode == 401 && refreshAuth != null && apiKey == null) {
      final tokens = await refreshAuth!.call();
      if (tokens != null) {
        token = tokens.accessToken;
        onTokens?.call(tokens);
        resp = await _send(method, uri, payload);
      }
    }
    return _decode(resp);
  }

  Future<http.Response> _send(
    String method,
    Uri uri,
    String? payload,
  ) {
    final headers = _headers();
    switch (method) {
      case 'POST':
        return http.post(uri, headers: headers, body: payload);
      case 'PATCH':
        return http.patch(uri, headers: headers, body: payload);
      case 'PUT':
        return http.put(uri, headers: headers, body: payload);
      case 'DELETE':
        return http.delete(uri, headers: headers, body: payload);
      default:
        return http.get(uri, headers: headers);
    }
  }

  Map<String, dynamic> _decode(http.Response resp) {
    final status = resp.statusCode;
    final body = resp.body.isEmpty ? '{}' : resp.body;
    final decoded = jsonDecode(body);
    if (status >= 200 && status < 300) {
      if (decoded is Map<String, dynamic>) {
        return decoded;
      }
      return {'data': decoded};
    }
    String message = '请求失败';
    if (decoded is Map<String, dynamic> && decoded['error'] is String) {
      message = decoded['error'] as String;
    }
    throw ApiException(status: status, message: message);
  }
}

class ApiException implements Exception {
  final int status;
  final String message;

  ApiException({required this.status, required this.message});

  @override
  String toString() => 'ApiException($status): $message';
}
