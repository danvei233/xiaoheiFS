import 'dart:convert';

import 'package:http/http.dart' as http;
import 'package:url_launcher/url_launcher.dart';

import 'update_config.dart';

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
  Future<AppUpdateInfo?> checkForUpdate({
    required String packageName,
    required int versionCode,
  }) async {
    try {
      final uri =
          Uri.parse(
            '${UpdateConfig.serverBaseUrl}${UpdateConfig.checkPath}',
          ).replace(
            queryParameters: {
              'pkg': packageName,
              'versionCode': '$versionCode',
            },
          );

      final resp = await http.get(uri).timeout(const Duration(seconds: 6));
      if (resp.statusCode < 200 || resp.statusCode >= 300) {
        return null;
      }
      final data = jsonDecode(resp.body);
      if (data is! Map<String, dynamic>) return null;

      final hasUpdate = _asBool(data['hasUpdate']);
      if (!hasUpdate) return null;

      return AppUpdateInfo(
        hasUpdate: hasUpdate,
        latestVersion: (data['latestVersion'] ?? '').toString(),
        latestVersionCode: _asInt(data['latestVersionCode']),
        forceUpdate: _asBool(data['forceUpdate']),
        apkUrl: _asNullableString(data['apkUrl']),
        changelog: (data['changelog'] ?? '').toString(),
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
