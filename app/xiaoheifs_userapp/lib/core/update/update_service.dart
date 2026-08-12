import 'dart:convert';
import 'dart:io';

import 'package:crypto/crypto.dart';
import 'package:dio/dio.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter/services.dart';
import 'package:url_launcher/url_launcher.dart';

import '../config/update_config.dart';
import 'manifest_verifier.dart';
import 'secure_manifest_state_store.dart';
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

  /// 更新包 SHA-256（hex 小写）。来自经 Ed25519 验签的更新清单，把「下载
  /// 字节」与「发布端签名」绑定。P2-13：受控下载路径
  /// （[UpdateService.downloadAndVerifyApk]）在安装前校验实际字节与它一致，
  /// 不一致则 fail-closed（不启动安装、不写同意记录）。
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
    Dio? dio,
    Map<int, String>? publicKeys,
    DateTime Function()? clock,
    ManifestStateStore? stateStore,
    Future<bool> Function(Uri url)? launcher,
  }) : _dio = dio ?? Dio(),
       _publicKeys = publicKeys ?? updateSignPublicKeys,
       _clock = clock ?? DateTime.now,
       _launcher = launcher ?? _defaultLauncher,
       _replayGuard = ManifestReplayGuard(
         stateStore ?? SecureManifestStateStore(),
       );

  final Dio _dio;

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
  /// FileProvider 解析为 content:// URI（见 MainActivity 的
  /// `xiaohei/update` method channel）再交给系统安装器；其它平台直接用
  /// 默认处理器打开该文件。
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
      final hasSignature =
          data['signature'] is String &&
          (data['signature'] as String).isNotEmpty;

      // 无签名且 hasUpdate=false：正常的「无更新」响应。
      if (!hasSignature) {
        return null;
      }

      // Fail closed：清单必须携带可验证的 Ed25519 签名，且必须先验签再信任
      // hasUpdate——`hasUpdate` 参与签名，攻击者无法在不破坏签名的情况下改写
      // 更新决策。缺签名/算法不支持/公钥不符/超期/降级一律拒绝。
      final signed = await ManifestVerifier.verifyManifest(
        manifest: data,
        packageName: packageName,
        currentVersionCode: versionCode,
        publicKeys: _publicKeys,
        now: _clock(),
      );
      if (!signed) return null;

      // 验签通过后 hasUpdate 才可信。
      if (!hasUpdate) return null;

      final apkUrl = _asNullableString(data['apkUrl']);
      // Fail closed：更新包必须经 HTTPS 下载，且与更新服务器同源。
      if (!isSafeDownloadUrl(apkUrl)) return null;

      // 跨会话防重放：比已接受的最高记录更新才接受，旧签名清单重放/降级拒绝。
      // 注意：这里只「检查」是否可接受，**不记录**——记录发生在用户真正点
      // 「立即更新」时（[recordAcceptedConsent]），因此被用户「稍后」关闭的
      // 非强制更新在下次启动仍会再次提示，不会因一次展示就被永久吞掉。
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

  Future<bool> openUpdateLink(String? url) async {
    if (!isSafeDownloadUrl(url)) return false;
    try {
      return await _launcher(Uri.parse(url!));
    } catch (_) {
      return false;
    }
  }

  /// 记录「用户已同意安装」该更新（在用户点击「立即更新」后、打开下载/安装器
  /// 成功后调用一次）。此后同一签名清单不再重复提示（跨会话防重放）。
  ///
  /// 与旧实现（在 `checkForUpdate` 内无条件记录）的区别：**展示/被「稍后」关闭
  /// 不会写入记录**，因此非强制更新被用户推迟后，下次启动会再次提示；只有
  /// 用户真正选择安装才记录。`update` 必须是本实例 `checkForUpdate` 返回的
  /// `AppUpdateInfo`（携带 keyEpoch/issuedAt）。
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

  /// 启动更新，**仅当外部交接成功后才记录「已同意安装」**。
  ///
  /// 受控下载（P2-13）：先经 [downloadAndVerifyApk] 把 APK 下载到本地并校验
  /// SHA-256（清单声明的 apkSha256 约束实际安装字节），校验通过才把**已验证
  /// 的本地文件**交给安装器；只有安装器成功交接后才写同意记录。
  ///
  /// fail-closed：清单缺 sha256 / 下载失败 / hash mismatch / 启动失败，一律
  /// 返回 false 且**不写记录**——同版本强更在下一次启动仍会再次提示，不会因
  /// 一次失败的交接被永久抑制。返回是否成功交接。
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
      return false; // 下载失败 / hash mismatch：fail-closed，不写同意。
    }
    final launched = await _launcher(await _installUriForFile(verifiedPath));
    if (launched) {
      await recordAcceptedConsent(update, packageName: packageName);
    }
    return launched;
  }

  /// 仅允许 https 且与更新服务器**同源**的下载地址（与 admin 端
  /// `_isTrustedApkUrl` 一致的纵深防御），拒绝 `intent://`、`file://`、
  /// `http://` 以及跨主机 CDN 地址。清单虽经 Ed25519 验签，但同源约束
  /// 保证即使更新服务器被攻破也只能分发其本域下的 URL。
  static bool isSafeDownloadUrl(String? url) {
    if (url == null || url.isEmpty) return false;
    final uri = Uri.tryParse(url);
    if (uri == null) return false;
    if (uri.scheme != 'https') return false;
    final server = Uri.tryParse(UpdateConfig.serverBaseUrl);
    if (server == null || !server.hasAuthority) return false;
    return uri.host == server.host;
  }

  /// 校验 APK 字节流与清单声明的 SHA-256 是否一致。
  ///
  /// 纯函数，可在单元测试中直接验证。`expectedSha256` 需为 64 位 hex。
  static bool verifyApkSha256(List<int> apkBytes, String? expectedSha256) {
    final normalized = _normalizeSha256(expectedSha256);
    if (normalized == null || normalized.length != 64) return false;
    final actual = sha256.convert(apkBytes).toString();
    return _constantTimeEquals(actual, normalized);
  }

  /// 下载更新包到临时目录并校验 SHA-256。
  ///
  /// 校验通过返回本地文件路径；否则删除文件并抛出 [UpdateIntegrityException]。
  /// 调用方必须用返回的文件路径触发系统安装器（Android 需 FileProvider）。
  ///
  /// 注意：当前发布流程（`login_page.dart`/`login_screen.dart`）走
  /// [openUpdateLink] 外部浏览器分发，不调用本方法；本方法是自托管/桌面
  /// 自安装路径的完整性校验入口（与 `docs/update-signing-contract.md` 一致）。
  Future<String> downloadAndVerifyApk({
    required String url,
    required String? expectedSha256,
  }) async {
    if (!isSafeDownloadUrl(url)) {
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
      await _dio.download(url, file.path);
      final bytes = await file.readAsBytes();
      if (!verifyApkSha256(bytes, normalizedHash)) {
        throw UpdateIntegrityException(
          'APK sha256 mismatch, refusing to install',
        );
      }
      return file.path;
    } catch (e) {
      try {
        if (await file.exists()) await file.delete();
      } catch (_) {}
      if (e is UpdateIntegrityException) rethrow;
      throw UpdateIntegrityException('download failed: $e');
    }
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

  /// 常量时间比较，避免哈希比较的时序侧信道。
  static bool _constantTimeEquals(String a, String b) {
    if (a.length != b.length) return false;
    var diff = 0;
    for (var i = 0; i < a.length; i++) {
      diff |= a.codeUnitAt(i) ^ b.codeUnitAt(i);
    }
    return diff == 0;
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
