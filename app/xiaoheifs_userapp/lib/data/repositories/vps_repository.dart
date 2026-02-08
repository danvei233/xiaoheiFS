import 'package:dio/dio.dart';
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
  Future<void> shutdown(int id) async => _dio.post(ApiEndpoints.vpsShutdown(id));
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

  Future<void> createSnapshot(int id) async => _dio.post(ApiEndpoints.vpsSnapshots(id));
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

  Future<void> createBackup(int id) async => _dio.post(ApiEndpoints.vpsBackups(id));
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

  Future<Map<String, dynamic>> createRenewOrder(int id, Map<String, dynamic> payload) async {
    final response = await _dio.post(ApiEndpoints.vpsRenew(id), data: payload);
    return ensureMap(response.data);
  }

  Future<Map<String, dynamic>> quoteResize(int id, Map<String, dynamic> payload) async {
    final response = await _dio.post(ApiEndpoints.vpsResizeQuote(id), data: payload);
    return ensureMap(response.data);
  }

  Future<Map<String, dynamic>> createResizeOrder(int id, Map<String, dynamic> payload) async {
    final response = await _dio.post(ApiEndpoints.vpsResize(id), data: payload);
    return ensureMap(response.data);
  }

  Future<void> emergencyRenew(int id) async {
    await _dio.post(ApiEndpoints.vpsEmergencyRenew(id));
  }

  Future<Map<String, dynamic>> requestRefund(int id, Map<String, dynamic> payload) async {
    final response = await _dio.post(ApiEndpoints.vpsRefund(id), data: payload);
    return ensureMap(response.data);
  }
}
