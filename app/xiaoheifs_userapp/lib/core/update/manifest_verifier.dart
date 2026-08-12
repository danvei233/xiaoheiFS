import 'dart:convert';
import 'dart:typed_data';

import 'package:cryptography/cryptography.dart';

/// 更新清单真实性校验器（Ed25519）。
///
/// 更新服务器发布的清单必须携带 Ed25519 签名：`algorithm` 固定为
/// `ed25519`，`signature` 为对「规范化载荷」的 base64 签名。规范化载荷由
/// 固定顺序、无歧义的字段（含 `hasUpdate`、`keyEpoch`、制品 SHA-256 与下载
/// URL）拼接而成，见 [buildCanonicalPayload]。同一逻辑的签名端契约见
/// `docs/update-signing-contract.md`。
///
/// 校验规则（全部 fail-closed）：
/// - 算法不是 `ed25519` 一律拒绝（不支持即拒绝，不做降级）；
/// - 签名缺失、非 base64、长度非 64 字节一律拒绝；
/// - `keyEpoch` 缺失/非正数/在客户端内嵌公钥表中不存在一律拒绝；
/// - 公钥缺失、非 hex、长度非 32 字节，或为占位（全零）公钥一律拒绝；
/// - 被签名字段缺失或类型与发布契约不符一律拒绝（含 `hasUpdate`，防止在
///   不破坏签名的情况下改写更新决策）；
/// - 下载 URL 必须是 https；
/// - 验签通过后再做防降级（签名 versionCode 必须高于当前安装版本）与
///   防重放（`issuedAt` 必须在 ±7 天时间窗内）检查。
///
/// 跨会话防重放（客户端在安全存储中维护「已接受的最高 versionCode/
/// issuedAt」，按 package + keyEpoch 记录）由调用方
/// [UpdateService.checkForUpdate] 配合安全存储实现，见
/// `docs/update-signing-contract.md` §5。
///
/// 本文件只依赖纯 Dart 包（`package:cryptography`），不依赖 Flutter，
/// 因此可在无 Flutter 的 CI 上用 `dart test` 直接执行单元测试。
class ManifestVerifier {
  ManifestVerifier._();

  static const String algorithmEd25519 = 'ed25519';
  static const int signatureLength = 64;
  static const int publicKeyLength = 32;
  static const Duration maxManifestAge = Duration(days: 7);
  static const Duration maxClockSkew = Duration(minutes: 5);

  static final RegExp _sha256Pattern = RegExp(r'^[0-9a-f]{64}$');

  /// 占位公钥（全零 hex）。未配置生产公钥的构建用它 fail-closed，
  /// 见 `update_keys.dart` 与发布契约 §6。
  static const String placeholderPublicKeyHex =
      '0000000000000000000000000000000000000000000000000000000000000000';

  /// 验证更新清单。
  ///
  /// [manifest] 是服务器返回的完整 JSON 对象；[packageName] 与
  /// [currentVersionCode] 是当前应用的包名与版本号，用于把签名绑定到
  /// 「当前应用 + 当前版本」；[publicKeys] 是「keyEpoch -> 编译期内嵌
  /// Ed25519 公钥 hex」映射（key rotation 时可在同一构建里携带新旧两把，
  /// 实现显式双 key 过渡）；[now] 仅用于测试注入时间。
  static Future<bool> verifyManifest({
    required Map<String, dynamic> manifest,
    required String packageName,
    required int currentVersionCode,
    required Map<int, String> publicKeys,
    DateTime? now,
  }) async {
    // 1. 算法必须为 ed25519。
    final algo = manifest['algorithm'];
    if (algo is! String || algo.toLowerCase() != algorithmEd25519) {
      return false;
    }

    // 2. 签名必须存在、base64 可解码、长度 64 字节。
    final signatureB64 = manifest['signature'];
    if (signatureB64 is! String || signatureB64.isEmpty) return false;
    final List<int> signature;
    try {
      signature = base64Decode(signatureB64);
    } on FormatException {
      return false;
    }
    if (signature.length != signatureLength) return false;

    // 3. keyEpoch 必须是正数，且客户端内嵌该 epoch 的公钥。
    final keyEpoch = manifest['keyEpoch'];
    if (keyEpoch is! int || keyEpoch <= 0) return false;
    final publicKeyHex = publicKeys[keyEpoch];
    if (publicKeyHex == null) return false;

    // 4. 公钥必须存在、hex 可解码、长度 32 字节，且不是占位公钥。
    final publicKey = _hexToBytes(publicKeyHex);
    if (publicKey.length != publicKeyLength) return false;
    if (publicKey.every((b) => b == 0)) return false;

    // 5. 被签名字段必须齐全且类型严格匹配发布契约。hasUpdate 参与签名，
    //    因此攻击者无法在不破坏签名的情况下改写更新决策。
    final hasUpdate = manifest['hasUpdate'];
    final latestVersion = manifest['latestVersion'];
    final latestVersionCode = manifest['latestVersionCode'];
    final forceUpdate = manifest['forceUpdate'];
    final apkUrl = manifest['apkUrl'];
    final apkSha256 = manifest['sha256'] ?? manifest['apkSha256'];
    final changelog = manifest['changelog'];
    final issuedAt = manifest['issuedAt'];
    if (hasUpdate is! bool ||
        latestVersion is! String ||
        latestVersionCode is! int ||
        forceUpdate is! bool ||
        apkUrl is! String ||
        apkSha256 is! String ||
        changelog is! String ||
        issuedAt is! int) {
      return false;
    }
    if (!_isHttpsUrl(apkUrl)) return false;
    if (!_sha256Pattern.hasMatch(apkSha256)) return false;
    if (issuedAt <= 0) return false;

    // 6. 构造规范化载荷并验签。
    final canonical = buildCanonicalPayload(
      packageName: packageName,
      hasUpdate: hasUpdate,
      keyEpoch: keyEpoch,
      latestVersion: latestVersion,
      latestVersionCode: latestVersionCode,
      forceUpdate: forceUpdate,
      apkUrl: apkUrl,
      apkSha256: apkSha256,
      changelog: changelog,
      issuedAt: issuedAt,
    );
    final ed25519 = Ed25519();
    final verified = await ed25519.verify(
      utf8.encode(canonical),
      signature: Signature(
        signature,
        publicKey: SimplePublicKey(publicKey, type: KeyPairType.ed25519),
      ),
    );
    if (!verified) return false;

    // 7. 防降级：签名 versionCode 必须高于当前安装版本。
    if (latestVersionCode <= currentVersionCode) return false;

    // 8. 防重放：issuedAt 必须在合理时间窗内。
    //    先做范围检查再乘 1000：巨大的正 issuedAt 在 64 位 VM 上乘 1000 会
    //    整数回绕成一个「落在窗口内」的正数，从而绕过时间窗（健壮性缺陷）。
    final nowMillis = (now ?? DateTime.now()).millisecondsSinceEpoch;
    final maxIssuedAt = 0x7FFFFFFFFFFFFFFF ~/ 1000;
    if (issuedAt > maxIssuedAt) return false;
    final issuedMillis = issuedAt * 1000;
    if (issuedMillis < nowMillis - maxManifestAge.inMilliseconds) return false;
    if (issuedMillis > nowMillis + maxClockSkew.inMilliseconds) return false;

    return true;
  }

  /// 规范化载荷：固定字段顺序 + 转义，签名端与验签端必须逐字节一致。
  ///
  /// 字段值中的反斜杠与换行符等控制字符会被转义（`\\` `\n` `\r` `\t`），
  /// 因此「key=value\n」编码无歧义。覆盖影响展示（hasUpdate/version/notes）、
  /// 下载（url/sha256）、强制更新（force）、版本/防降级（versionCode）、
  /// 时效/防重放（issuedAt）、包名绑定（pkg）与密钥轮换（epoch）的全部字段。
  static String buildCanonicalPayload({
    required String packageName,
    required bool hasUpdate,
    required int keyEpoch,
    required String latestVersion,
    required int latestVersionCode,
    required bool forceUpdate,
    required String apkUrl,
    required String apkSha256,
    required String changelog,
    required int issuedAt,
  }) {
    final lines = <String>[
      'pkg=${_escape(packageName)}',
      'hasUpdate=${hasUpdate ? '1' : '0'}',
      'epoch=$keyEpoch',
      'version=${_escape(latestVersion)}',
      'versionCode=$latestVersionCode',
      'force=${forceUpdate ? '1' : '0'}',
      'url=${_escape(apkUrl)}',
      'sha256=${_escape(apkSha256)}',
      'notes=${_escape(changelog)}',
      'issuedAt=$issuedAt',
    ];
    return lines.join('\n');
  }

  static String _escape(String value) {
    final buf = StringBuffer();
    for (final code in value.codeUnits) {
      if (code == 0x5C) {
        buf.write(r'\\');
      } else if (code == 0x0A) {
        buf.write(r'\n');
      } else if (code == 0x0D) {
        buf.write(r'\r');
      } else if (code == 0x09) {
        buf.write(r'\t');
      } else {
        buf.writeCharCode(code);
      }
    }
    return buf.toString();
  }

  static bool _isHttpsUrl(String url) {
    final uri = Uri.tryParse(url);
    return uri != null && uri.scheme == 'https' && uri.host.isNotEmpty;
  }

  static Uint8List _hexToBytes(String hex) {
    final clean = hex.trim();
    if (clean.length % 2 != 0) return Uint8List(0);
    final out = Uint8List(clean.length ~/ 2);
    for (var i = 0; i < out.length; i++) {
      final hi = _hexNibble(clean.codeUnitAt(i * 2));
      final lo = _hexNibble(clean.codeUnitAt(i * 2 + 1));
      if (hi < 0 || lo < 0) return Uint8List(0);
      out[i] = (hi << 4) | lo;
    }
    return out;
  }

  static int _hexNibble(int code) {
    if (code >= 0x30 && code <= 0x39) return code - 0x30; // 0-9
    if (code >= 0x61 && code <= 0x66) return code - 0x61 + 10; // a-f
    if (code >= 0x41 && code <= 0x46) return code - 0x41 + 10; // A-F
    return -1;
  }
}
