import 'dart:convert';
import 'dart:io';

import 'package:flutter/foundation.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:path_provider/path_provider.dart';
import 'package:shared_preferences/shared_preferences.dart';

import '../models/session.dart';

/// 持久化存储。
///
/// 敏感凭据（api_key / token / refresh_token）使用 `flutter_secure_storage`
/// （Android Keystore / iOS Keychain）加密存储；非敏感会话元数据
/// （api_url / username / auth_type / token_expires_at / email）留在
/// SharedPreferences。启动时会把旧版本写入 SharedPreferences 的明文凭据
/// 迁移进安全存储并删除明文。
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

  static const FlutterSecureStorage _secure = FlutterSecureStorage();

  Future<Session?> loadSession() async {
    Session? session;
    try {
      final prefs = await SharedPreferences.getInstance();
      await _migrateLegacyCredentials(prefs);
      session = await _fromPrefs(prefs);
    } catch (_) {}
    if (session != null) return session;
    return _migrateFromFile();
  }

  /// 旧版本把凭据明文存在 SharedPreferences，这里一次性迁移到安全存储。
  Future<void> _migrateLegacyCredentials(SharedPreferences prefs) async {
    Future<void> move(String key) async {
      final value = prefs.getString(key);
      if (value != null && value.isNotEmpty) {
        await _secure.write(key: key, value: value);
        await prefs.remove(key);
      }
    }

    await move(_keyApiKey);
    await move(_keyToken);
    await move(_keyRefreshToken);
  }

  Future<Session?> _fromPrefs(SharedPreferences prefs) async {
    final apiUrl = prefs.getString(_keyApiUrl);
    final storedAuthType = prefs.getString(_keyAuthType);
    final authType = storedAuthType ?? 'api_key';
    final apiKey = await _secure.read(key: _keyApiKey);
    final token = await _secure.read(key: _keyToken);
    final refreshToken = await _secure.read(key: _keyRefreshToken);
    final expiresRaw = prefs.getString(_keyTokenExpiresAt);
    final tokenExpiresAt = expiresRaw == null
        ? null
        : DateTime.tryParse(expiresRaw);
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
        await _secure.write(key: _keyApiKey, value: session.apiKey!);
      } else {
        await _secure.delete(key: _keyApiKey);
      }
      if (session.token != null) {
        await _secure.write(key: _keyToken, value: session.token!);
      } else {
        await _secure.delete(key: _keyToken);
      }
      if (session.refreshToken != null) {
        await _secure.write(
          key: _keyRefreshToken,
          value: session.refreshToken!,
        );
      } else {
        await _secure.delete(key: _keyRefreshToken);
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
    // 凭据不再镜像到明文 session.json（见 _migrateFromFile 一次性迁移读取）。
  }

  Future<void> clearSession() async {
    try {
      final prefs = await SharedPreferences.getInstance();
      await _secure.delete(key: _keyApiKey);
      await _secure.delete(key: _keyToken);
      await _secure.delete(key: _keyRefreshToken);
      await prefs.remove(_keyApiUrl);
      await prefs.remove(_keyUsername);
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

/// 一次性迁移：旧版本把会话明文镜像到 `session.json`。首次读到该文件时把
/// 凭据迁入安全存储、元数据写入 SharedPreferences，随后**删除明文文件**，
/// 后续启动不再读取明文。返回迁移后的会话（与文件内容一致）。
///
/// 失败路径（JSON 损坏 / 非对象 / 缺 api_url / 凭据不完整）也必须**删除**该
/// 文件：损坏或不完整的 session.json 若残留，其中可能的敏感明文会永久驻留
/// 应用支持目录，且每次启动都会重复尝试解析失败。
Future<Session?> _migrateFromFile() async {
  File? file;
  try {
    if (kIsWeb) return null;
    file = await _sessionFile();
    if (!await file.exists()) return null;
    final raw = await file.readAsString();
    final decoded = jsonDecode(raw);
    if (decoded is! Map<String, dynamic>) {
      await _deleteQuietly(file);
      return null;
    }
    final apiUrl = (decoded['api_url'] as String?) ?? '';
    final authType = (decoded['auth_type'] as String?) ?? 'api_key';
    if (apiUrl.isEmpty) {
      await _deleteQuietly(file);
      return null;
    }
    final token = decoded['token'] as String?;
    final refreshToken = decoded['refresh_token'] as String?;
    final expiresRaw = decoded['token_expires_at'] as String?;
    final tokenExpiresAt = expiresRaw == null
        ? null
        : DateTime.tryParse(expiresRaw);
    final apiKey = decoded['api_key'] as String?;
    if (authType == 'password' && (token == null || token.isEmpty)) {
      await _deleteQuietly(file);
      return null;
    }
    if (authType == 'api_key' && (apiKey == null || apiKey.isEmpty)) {
      await _deleteQuietly(file);
      return null;
    }

    // 迁移：凭据进安全存储，元数据进 prefs，随后删除明文文件。
    final prefs = await SharedPreferences.getInstance();
    await prefs.setString(_keyApiUrl, apiUrl);
    await prefs.setString(_keyAuthType, authType);
    await prefs.setString(
      _keyUsername,
      (decoded['username'] as String?) ?? '管理员',
    );
    if (apiKey != null) {
      await _secure.write(key: _keyApiKey, value: apiKey);
    }
    if (token != null) {
      await _secure.write(key: _keyToken, value: token);
    }
    if (refreshToken != null) {
      await _secure.write(key: _keyRefreshToken, value: refreshToken);
    }
    if (expiresRaw != null && expiresRaw.isNotEmpty) {
      await prefs.setString(_keyTokenExpiresAt, expiresRaw);
    }
    final email = decoded['email'] as String?;
    if (email != null) {
      await prefs.setString(_keyEmail, email);
    }
    // 删除明文文件：凭据不再驻留应用支持目录。
    await file.delete();

    return Session(
      apiUrl: apiUrl,
      apiKey: apiKey,
      token: token,
      refreshToken: refreshToken,
      tokenExpiresAt: tokenExpiresAt,
      username: (decoded['username'] as String?) ?? '管理员',
      email: email,
      authType: authType,
    );
  } catch (_) {
    // 损坏/无法解析的 session.json 必须删除，不能把可能的敏感明文永久留在
    // 磁盘上；迁移失败返回 null（等同无会话）。
    if (file != null) {
      await _deleteQuietly(file);
    }
    return null;
  }
}

/// 尽力删除文件，失败不掩盖调用方结果。
Future<void> _deleteQuietly(File file) async {
  try {
    if (await file.exists()) {
      await file.delete();
    }
  } catch (_) {}
}
