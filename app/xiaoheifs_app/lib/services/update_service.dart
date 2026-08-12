import 'dart:convert';
import 'dart:io';

import 'package:crypto/crypto.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter/services.dart';
import 'package:http/http.dart' as http;
import 'package:url_launcher/url_launcher.dart';

import 'manifest_verifier.dart';
import 'secure_manifest_state_store.dart';
import 'update_config.dart';
import 'update_keys.dart';
import 'update_replay_guard.dart';

class AppUpdateInfo {
  AppUpdateInfo({
    required this.hasUpdate,
    required this.latestVersion,
    required this.latestVersionCode,
    required this.forceUpdate,
    required this.apkUrl,
    required this.changelog,
    this.apkSha256,
    this.keyEpoch,
    this.issuedAt,
  });

  final bool hasUpdate;
  final String latestVersion;
  final int latestVersionCode;
  final bool forceUpdate;
  final String? apkUrl;

  /// 更新包 SHA-256（hex 小写）。来自经 Ed25519 验签的更新清单，
  /// 下载完成后用于校验 APK 未被篡改。
  final String? apkSha256;
  final String changelog;

  /// 清单声明的密钥代次与签发时间。用于「用户同意安装时才记录已接受」：
  /// 见 [UpdateService.recordAcceptedConsent] 与
  /// `docs/update-signing-contract.md` §5.12。
  final int? keyEpoch;
  final int? issuedAt;
}

/// 更新包完整性校验失败时抛出，调用方应中止安装并提示用户。
class UpdateIntegrityException implements Exception {
  UpdateIntegrityException(this.message);
  final String message;

  @override
  String toString() => 'UpdateIntegrityException: $message';
}

class UpdateService {
  UpdateService({
    http.Client? client,
    Map<int, String>? publicKeys,
    DateTime Function()? clock,
    ManifestStateStore? stateStore,
    Future<bool> Function(Uri url)? launcher,
  }) : _client = client ?? http.Client(),
       _publicKeys = publicKeys ?? updateSignPublicKeys,
       _clock = clock ?? DateTime.now,
       _launcher = launcher ?? _defaultLauncher,
       _replayGuard = ManifestReplayGuard(
         stateStore ?? SecureManifestStateStore(),
       );

  final http.Client _client;

  /// keyEpoch -> 验签公钥（hex）。构建期内嵌，测试可注入测试公钥表。
  final Map<int, String> _publicKeys;

  /// 时间源，测试可注入固定时间验证防重放/防降级逻辑。
  final DateTime Function() _clock;

  /// 外部应用启动器。测试可注入替身模拟启动成功/失败，验证
  /// [consentAndLaunchUpdate] 只在成功交接后才记录同意。
  final Future<bool> Function(Uri url) _launcher;

  /// 默认启动器：把更新 URL/已验证的本地 APK 交给系统浏览器/安装器。
  ///
  /// Android 上打开本地 file:// 会抛 FileUriExposedException，因此先经
  /// FileProvider 解析为 content:// URI（见 MainActivity 的 `xiaohei/update`
  /// method channel）再交给系统安装器；其它平台直接用默认处理器打开该文件。
  static Future<bool> _defaultLauncher(Uri url) async {
    if (url.scheme == 'file') {
      return launchUrl(
        await _installUriForFile(url.toFilePath()),
        mode: LaunchMode.externalApplication,
      );
    }
    return launchUrl(url, mode: LaunchMode.externalApplication);
  }

  /// 解析本地已验证 APK 的安装 URI：Android 经 FileProvider 得到 content://，
  /// 其它平台（含测试宿主）直接返回 file://。
  static Future<Uri> _installUriForFile(String path) async {
    if (!kIsWeb && Platform.isAndroid) {
      try {
        final contentUri = await const MethodChannel(
          'xiaohei/update',
        ).invokeMethod<String>('getContentUriForPath', {'path': path});
        if (contentUri != null && contentUri.isNotEmpty) {
          return Uri.parse(contentUri);
        }
      } catch (_) {}
    }
    return Uri.file(path);
  }

  /// 跨会话防重放守卫（安全存储持久化最高已接受 versionCode/issuedAt）。
  final ManifestReplayGuard _replayGuard;

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

      final resp = await _client.get(uri).timeout(const Duration(seconds: 6));
      if (resp.statusCode < 200 || resp.statusCode >= 300) {
        return null;
      }
      final data = jsonDecode(resp.body);
      if (data is! Map<String, dynamic>) return null;

      final hasUpdate = _asBool(data['hasUpdate']);
      final hasSignature =
          data['signature'] is String &&
          (data['signature'] as String).isNotEmpty;

      // No signature + hasUpdate=false: normal "no update" response.
      if (!hasSignature) {
        return null;
      }

      // Fail closed: verify the Ed25519 signature BEFORE trusting hasUpdate.
      // hasUpdate is part of the signed payload, so the update decision cannot
      // be flipped without invalidating the signature.
      final signed = await ManifestVerifier.verifyManifest(
        manifest: data,
        packageName: packageName,
        currentVersionCode: versionCode,
        publicKeys: _publicKeys,
        now: _clock(),
      );
      if (!signed) return null;

      // hasUpdate is only trusted after verification.
      if (!hasUpdate) return null;

      final apkUrl = _asNullableString(data['apkUrl']);
      // Fail closed: only trust a manifest whose APK lives on the configured
      // update server over HTTPS.
      if (!_isTrustedApkUrl(apkUrl)) return null;

      // Cross-session replay guard: only a strictly newer signed manifest is
      // accepted; a replay/downgrade of an already-accepted one is rejected.
      // Recording happens only when the user consents to install
      // (recordAcceptedConsent), never when the dialog is merely shown or
      // dismissed — so a postponed non-forced update is offered again next
      // launch instead of being silently swallowed.
      final keyEpoch = data['keyEpoch'] as int;
      final latestVersionCode = data['latestVersionCode'] as int;
      final issuedAt = data['issuedAt'] as int;
      final acceptable = await _replayGuard.isAcceptable(
        packageName: packageName,
        keyEpoch: keyEpoch,
        versionCode: latestVersionCode,
        issuedAt: issuedAt,
      );
      if (!acceptable) return null;

      return AppUpdateInfo(
        hasUpdate: hasUpdate,
        latestVersion: (data['latestVersion'] ?? '').toString(),
        latestVersionCode: latestVersionCode,
        forceUpdate: _asBool(data['forceUpdate']),
        apkUrl: apkUrl,
        apkSha256: _normalizeSha256(data['sha256'] ?? data['apkSha256']),
        changelog: (data['changelog'] ?? '').toString(),
        keyEpoch: keyEpoch,
        issuedAt: issuedAt,
      );
    } catch (_) {
      return null;
    }
  }

  /// 校验 APK 字节流与清单声明的 SHA-256 是否一致（常量时间比较）。
  static bool verifyApkSha256(List<int> apkBytes, String? expectedSha256) {
    final normalized = _normalizeSha256(expectedSha256);
    if (normalized == null || normalized.length != 64) return false;
    final actual = sha256.convert(apkBytes).toString();
    if (actual.length != normalized.length) return false;
    var diff = 0;
    for (var i = 0; i < actual.length; i++) {
      diff |= actual.codeUnitAt(i) ^ normalized.codeUnitAt(i);
    }
    return diff == 0;
  }

  /// 归一化 SHA-256：小写、去空白、校验 64 hex 字符。
  static String? _normalizeSha256(dynamic value) {
    if (value == null) return null;
    final s = value.toString().trim().toLowerCase();
    if (s.isEmpty || s.length != 64 || !RegExp(r'^[0-9a-f]{64}$').hasMatch(s)) {
      return null;
    }
    return s;
  }

  Future<bool> openUpdateLink(String? url) async {
    if (url == null || url.isEmpty) return false;
    final uri = Uri.tryParse(url);
    if (uri == null) return false;
    if (!_isTrustedApkUrl(url)) return false;
    try {
      return await _launcher(uri);
    } catch (_) {
      return false;
    }
  }

  /// Records the user's explicit consent to install [update] (called once after
  /// the user taps "Update now" AND the OS successfully takes over the link).
  /// After this the same signed manifest is not offered again (cross-session
  /// replay guard). Unlike the old auto-record-inside-checkForUpdate behaviour,
  /// merely showing/dismissing a non-forced update does NOT write the record,
  /// so a postponed update is offered again on the next launch.
  Future<void> recordAcceptedConsent(
    AppUpdateInfo update, {
    required String packageName,
  }) async {
    final keyEpoch = update.keyEpoch;
    final issuedAt = update.issuedAt;
    if (keyEpoch == null || issuedAt == null) return;
    await _replayGuard.recordAccepted(
      packageName: packageName,
      keyEpoch: keyEpoch,
      versionCode: update.latestVersionCode,
      issuedAt: issuedAt,
    );
  }

  /// Launches the update and ONLY records consent if the OS handoff succeeds.
  ///
  /// Controlled download (P2-13): [downloadAndVerifyApk] first downloads the APK
  /// and verifies the manifest's apkSha256 against the actual bytes; only the
  /// verified local file is handed to the installer. Ordering is a hard
  /// requirement: the verified download + successful handoff FIRST, then
  /// [recordAcceptedConsent]. A failed handoff (no browser / intent handler) or
  /// a hash mismatch must NOT write the record — a failed handoff of a forced
  /// update would otherwise permanently suppress the same-version update on
  /// every later launch. Returns whether the handoff succeeded.
  Future<bool> consentAndLaunchUpdate(
    AppUpdateInfo update, {
    required String packageName,
  }) async {
    final normalized = _normalizeSha256(update.apkSha256);
    final apkUrl = update.apkUrl;
    if (normalized == null || apkUrl == null) return false;
    String verifiedPath;
    try {
      verifiedPath = await downloadAndVerifyApk(
        url: apkUrl,
        expectedSha256: normalized,
      );
    } catch (_) {
      return false; // download failed / hash mismatch: fail closed, no consent.
    }
    final launched = await _launcher(await _installUriForFile(verifiedPath));
    if (launched) {
      await recordAcceptedConsent(update, packageName: packageName);
    }
    return launched;
  }

  /// Downloads the update package to a temp file and verifies its SHA-256
  /// against the signed manifest. Returns the verified local path; throws
  /// [UpdateIntegrityException] on unsafe URL / missing hash / download error /
  /// hash mismatch (fail closed).
  Future<String> downloadAndVerifyApk({
    required String url,
    required String? expectedSha256,
  }) async {
    if (!_isTrustedApkUrl(url)) {
      throw UpdateIntegrityException('unsafe download URL: $url');
    }
    final normalizedHash = _normalizeSha256(expectedSha256);
    if (normalizedHash == null || normalizedHash.length != 64) {
      throw UpdateIntegrityException(
        'update manifest is missing a valid apkSha256',
      );
    }
    final dir = Directory.systemTemp;
    final file = File(
      '${dir.path}${Platform.pathSeparator}xiaohei_update_${DateTime.now().millisecondsSinceEpoch}.apk',
    );
    try {
      final resp = await _client.get(Uri.parse(url));
      if (resp.statusCode < 200 || resp.statusCode >= 300) {
        throw UpdateIntegrityException(
          'download failed: HTTP ${resp.statusCode}',
        );
      }
      final bytes = resp.bodyBytes;
      if (!verifyApkSha256(bytes, normalizedHash)) {
        throw UpdateIntegrityException(
          'APK sha256 mismatch, refusing to install',
        );
      }
      await file.writeAsBytes(bytes);
      return file.path;
    } catch (e) {
      try {
        if (await file.exists()) await file.delete();
      } catch (_) {}
      if (e is UpdateIntegrityException) rethrow;
      throw UpdateIntegrityException('download failed: $e');
    }
  }

  /// The APK download must be HTTPS and hosted on the configured update
  /// server, otherwise it is untrusted.
  bool _isTrustedApkUrl(String? url) {
    if (url == null || url.isEmpty) return false;
    final uri = Uri.tryParse(url);
    if (uri == null) return false;
    if (uri.scheme != 'https') return false;
    final server = Uri.tryParse(UpdateConfig.serverBaseUrl);
    if (server == null || !server.hasAuthority) return false;
    return uri.host == server.host;
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
