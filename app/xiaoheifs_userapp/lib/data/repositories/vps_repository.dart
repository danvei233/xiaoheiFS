import 'dart:convert';

import 'package:dio/dio.dart';
import 'package:flutter/foundation.dart' show kReleaseMode;
import '../../core/constants/api_endpoints.dart';
import '../../core/network/api_client.dart';
import '../../core/utils/map_utils.dart';

class VpsRepository {
  final Dio _dio = ApiClient.instance.dio;

  Future<List<Map<String, dynamic>>> listVps() async {
    final response = await _dio.get(ApiEndpoints.vps);
    final data = ensureMap(response.data);
    final items = data['items'];
    if (items is List) {
      return items.map((e) => ensureMap(e)).toList();
    }
    if (response.data is List) {
      return (response.data as List).map((e) => ensureMap(e)).toList();
    }
    return [];
  }

  Future<Map<String, dynamic>> getDetail(int id) async {
    final response = await _dio.get(ApiEndpoints.vpsDetail(id));
    return ensureMap(response.data);
  }

  Future<void> refresh(int id) async {
    await _dio.post(ApiEndpoints.vpsRefresh(id));
  }

  Future<void> start(int id) async => _dio.post(ApiEndpoints.vpsStart(id));
  Future<void> shutdown(int id) async =>
      _dio.post(ApiEndpoints.vpsShutdown(id));
  Future<void> reboot(int id) async => _dio.post(ApiEndpoints.vpsReboot(id));

  Future<Map<String, dynamic>> getMonitor(int id) async {
    final response = await _dio.get(ApiEndpoints.vpsMonitor(id));
    return ensureMap(response.data);
  }

  Future<void> resetOs(int id, Map<String, dynamic> payload) async {
    await _dio.post(ApiEndpoints.vpsResetOs(id), data: payload);
  }

  Future<void> resetOsPassword(int id, Map<String, dynamic> payload) async {
    await _dio.post(ApiEndpoints.vpsResetOsPassword(id), data: payload);
  }

  Future<List<Map<String, dynamic>>> listSnapshots(int id) async {
    final response = await _dio.get(ApiEndpoints.vpsSnapshots(id));
    final data = ensureMap(response.data);
    final items = data['items'] ?? data['data'];
    if (items is List) return items.map((e) => ensureMap(e)).toList();
    if (response.data is List) {
      return (response.data as List).map((e) => ensureMap(e)).toList();
    }
    return [];
  }

  Future<void> createSnapshot(int id) async =>
      _dio.post(ApiEndpoints.vpsSnapshots(id));
  Future<void> deleteSnapshot(int id, int snapshotId) async =>
      _dio.delete(ApiEndpoints.vpsSnapshotDetail(id, snapshotId));
  Future<void> restoreSnapshot(int id, int snapshotId) async =>
      _dio.post(ApiEndpoints.vpsSnapshotRestore(id, snapshotId));

  Future<List<Map<String, dynamic>>> listBackups(int id) async {
    final response = await _dio.get(ApiEndpoints.vpsBackups(id));
    final data = ensureMap(response.data);
    final items = data['items'] ?? data['data'];
    if (items is List) return items.map((e) => ensureMap(e)).toList();
    if (response.data is List) {
      return (response.data as List).map((e) => ensureMap(e)).toList();
    }
    return [];
  }

  Future<void> createBackup(int id) async =>
      _dio.post(ApiEndpoints.vpsBackups(id));
  Future<void> deleteBackup(int id, int backupId) async =>
      _dio.delete(ApiEndpoints.vpsBackupDetail(id, backupId));
  Future<void> restoreBackup(int id, int backupId) async =>
      _dio.post(ApiEndpoints.vpsBackupRestore(id, backupId));

  Future<List<Map<String, dynamic>>> listFirewallRules(int id) async {
    final response = await _dio.get(ApiEndpoints.vpsFirewall(id));
    final data = ensureMap(response.data);
    final items = data['items'] ?? data['data'];
    if (items is List) return items.map((e) => ensureMap(e)).toList();
    if (response.data is List) {
      return (response.data as List).map((e) => ensureMap(e)).toList();
    }
    return [];
  }

  Future<void> addFirewallRule(int id, Map<String, dynamic> payload) async {
    await _dio.post(ApiEndpoints.vpsFirewall(id), data: payload);
  }

  Future<void> deleteFirewallRule(int id, int ruleId) async {
    await _dio.delete(ApiEndpoints.vpsFirewallRule(id, ruleId));
  }

  Future<List<Map<String, dynamic>>> listPortMappings(int id) async {
    final response = await _dio.get(ApiEndpoints.vpsPorts(id));
    final data = ensureMap(response.data);
    final items = data['items'] ?? data['data'];
    if (items is List) return items.map((e) => ensureMap(e)).toList();
    if (response.data is List) {
      return (response.data as List).map((e) => ensureMap(e)).toList();
    }
    return [];
  }

  Future<List<dynamic>> listPortCandidates(int id, {String? keywords}) async {
    final response = await _dio.get(
      ApiEndpoints.vpsPortCandidates(id),
      queryParameters: {
        if (keywords != null && keywords.isNotEmpty) 'keywords': keywords,
      },
    );
    final data = ensureMap(response.data);
    final items = data['items'] ?? data['data'];
    if (items is List) return items;
    if (response.data is List) return response.data as List;
    return const [];
  }

  Future<void> addPortMapping(int id, Map<String, dynamic> payload) async {
    await _dio.post(ApiEndpoints.vpsPorts(id), data: payload);
  }

  Future<void> deletePortMapping(int id, int mappingId) async {
    await _dio.delete(ApiEndpoints.vpsPortMapping(id, mappingId));
  }

  Future<Map<String, dynamic>> createRenewOrder(
    int id,
    Map<String, dynamic> payload,
  ) async {
    final response = await _dio.post(ApiEndpoints.vpsRenew(id), data: payload);
    return ensureMap(response.data);
  }

  Future<Map<String, dynamic>> quoteResize(
    int id,
    Map<String, dynamic> payload,
  ) async {
    final response = await _dio.post(
      ApiEndpoints.vpsResizeQuote(id),
      data: payload,
    );
    return ensureMap(response.data);
  }

  Future<Map<String, dynamic>> createResizeOrder(
    int id,
    Map<String, dynamic> payload,
  ) async {
    final response = await _dio.post(ApiEndpoints.vpsResize(id), data: payload);
    return ensureMap(response.data);
  }

  Future<void> emergencyRenew(int id) async {
    await _dio.post(ApiEndpoints.vpsEmergencyRenew(id));
  }

  Future<Map<String, dynamic>> requestRefund(
    int id,
    Map<String, dynamic> payload,
  ) async {
    final response = await _dio.post(ApiEndpoints.vpsRefund(id), data: payload);
    return ensureMap(response.data);
  }

  Future<String> resolveRemoteRedirect(int id, {required String action}) async {
    final normalized = action.trim().toLowerCase();
    final path = normalized == 'panel'
        ? ApiEndpoints.vpsPanel(id)
        : ApiEndpoints.vpsVnc(id);

    final response = await _dio.get(
      path,
      options: Options(
        followRedirects: false,
        validateStatus: (_) => true,
        responseType: ResponseType.plain,
      ),
    );

    final status = response.statusCode ?? 0;
    final location = response.headers.value('location')?.trim() ?? '';
    if (_isRedirectStatus(status) && location.isNotEmpty) {
      return _requireSafeRedirectUrl(location);
    }

    if (status >= 400) {
      final msg = _extractErrorMessage(response.data);
      throw Exception(msg.isNotEmpty ? msg : '跳转失败（HTTP $status）');
    }

    final map = _toMap(response.data);
    final directUrl = _pickString(map, const [
      'url',
      'redirect_url',
      'location',
    ]);
    if (directUrl.isNotEmpty) {
      return _requireSafeRedirectUrl(directUrl);
    }

    if (location.isNotEmpty) {
      return _requireSafeRedirectUrl(location);
    }
    throw Exception('未获取到跳转地址');
  }

  /// 校验重定向/面板地址：
  ///
  /// - 生产（release）只允许 **HTTPS**：面板/VNC 跳转携带短时单次 ticket，
  ///   明文 HTTP 会把 ticket 暴露给链路窃听者（公网/LAN 均可被中间人读取）。
  /// - 开发（debug/profile）例外：`http://localhost` / `http://127.0.0.1` /
  ///   `http://[::1]` 允许（本机面板调试），公网/LAN 明文 HTTP 一律拒绝。
  /// - 任何 `intent://`、`file://`、`javascript:` 等 scheme 一律拒绝。
  ///
  /// [allowCleartext] 供测试注入：默认跟随构建模式（release 仅 HTTPS）；
  /// 传 `false` 可模拟 release 行为，传 `true` 仅放行本机 http。
  static bool isSafeRedirectUrl(String url, {bool? allowCleartext}) {
    final uri = Uri.tryParse(url);
    if (uri == null) return false;
    if (uri.scheme == 'https') return true;
    final allowHttp = allowCleartext ?? !kReleaseMode;
    if (allowHttp && uri.scheme == 'http') {
      final host = uri.host.toLowerCase();
      return host == 'localhost' || host == '127.0.0.1' || host == '::1';
    }
    return false;
  }

  String _requireSafeRedirectUrl(String url) {
    if (!isSafeRedirectUrl(url)) {
      throw Exception('跳转地址不安全：生产仅支持 HTTPS（本机调试可放行 localhost HTTP）');
    }
    return url;
  }

  bool _isRedirectStatus(int code) =>
      code == 301 || code == 302 || code == 303 || code == 307 || code == 308;

  String _extractErrorMessage(dynamic data) {
    final map = _toMap(data);
    final msg = _pickString(map, const ['error', 'message', 'msg', 'detail']);
    if (msg.isNotEmpty) {
      return msg;
    }
    if (data is String && data.trim().isNotEmpty) {
      return data.trim();
    }
    return '';
  }

  Map<String, dynamic> _toMap(dynamic data) {
    if (data is Map<String, dynamic>) {
      return data;
    }
    if (data is Map) {
      return ensureMap(data);
    }
    if (data is String) {
      final text = data.trim();
      if (text.isEmpty || (!text.startsWith('{') && !text.startsWith('['))) {
        return {};
      }
      try {
        final decoded = jsonDecode(text);
        return ensureMap(decoded);
      } catch (_) {
        return {};
      }
    }
    return {};
  }

  String _pickString(Map<String, dynamic> map, List<String> keys) {
    for (final key in keys) {
      final value = map[key];
      if (value != null && value.toString().trim().isNotEmpty) {
        return value.toString().trim();
      }
    }
    return '';
  }
}
