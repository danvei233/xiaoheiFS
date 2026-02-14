import 'package:dio/dio.dart';
import 'package:url_launcher/url_launcher.dart';

import '../config/update_config.dart';

class AppUpdateInfo {
  AppUpdateInfo({
    required this.hasUpdate,
    required this.latestVersion,
    required this.latestVersionCode,
    required this.forceUpdate,
    required this.apkUrl,
    required this.changelog,
  });

  final bool hasUpdate;
  final String latestVersion;
  final int latestVersionCode;
  final bool forceUpdate;
  final String? apkUrl;
  final String changelog;
}

class UpdateService {
  UpdateService({Dio? dio}) : _dio = dio ?? Dio();

  final Dio _dio;

  Future<AppUpdateInfo?> checkForUpdate({
    required String packageName,
    required int versionCode,
  }) async {
    try {
      final response = await _dio.get(
        '${UpdateConfig.serverBaseUrl}${UpdateConfig.checkPath}',
        queryParameters: {'pkg': packageName, 'versionCode': versionCode},
        options: Options(
          sendTimeout: const Duration(seconds: 6),
          receiveTimeout: const Duration(seconds: 6),
        ),
      );

      final data = response.data;
      if (data is! Map<String, dynamic>) {
        return null;
      }

      final hasUpdate = _asBool(data['hasUpdate']);
      final latestVersion = (data['latestVersion'] ?? '').toString();
      final latestVersionCode = _asInt(data['latestVersionCode']);
      final forceUpdate = _asBool(data['forceUpdate']);
      final apkUrl = _asNullableString(data['apkUrl']);
      final changelog = (data['changelog'] ?? '').toString();

      if (!hasUpdate) return null;

      return AppUpdateInfo(
        hasUpdate: hasUpdate,
        latestVersion: latestVersion,
        latestVersionCode: latestVersionCode,
        forceUpdate: forceUpdate,
        apkUrl: apkUrl,
        changelog: changelog,
      );
    } catch (_) {
      return null;
    }
  }

  Future<bool> openUpdateLink(String? url) async {
    if (url == null || url.isEmpty) return false;
    final uri = Uri.tryParse(url);
    if (uri == null) return false;
    return launchUrl(uri, mode: LaunchMode.externalApplication);
  }

  bool _asBool(dynamic v) {
    if (v is bool) return v;
    if (v is num) return v != 0;
    if (v is String) {
      final s = v.toLowerCase().trim();
      return s == 'true' || s == '1';
    }
    return false;
  }

  int _asInt(dynamic v) {
    if (v is int) return v;
    if (v is num) return v.toInt();
    if (v is String) return int.tryParse(v) ?? 0;
    return 0;
  }

  String? _asNullableString(dynamic v) {
    if (v == null) return null;
    final s = v.toString().trim();
    return s.isEmpty ? null : s;
  }
}
