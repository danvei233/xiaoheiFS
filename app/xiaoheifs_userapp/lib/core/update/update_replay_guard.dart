import 'dart:convert';

/// 跨会话防重放守卫：记录「当前应用 + keyEpoch」下已接受的最高
/// versionCode / issuedAt，并拒绝任何不比它更新的签名清单（旧签名清单
/// 降级/重放）。
///
/// 纯 Dart（不依赖 Flutter）：测试可用 [InMemoryManifestStateStore]；
/// 应用内用 `flutter_secure_storage` 实现 [ManifestStateStore] 持久化。
/// 契约见 `docs/update-signing-contract.md` §5。
class AcceptedManifestRecord {
  AcceptedManifestRecord({required this.versionCode, required this.issuedAt});

  final int versionCode;
  final int issuedAt;

  Map<String, dynamic> toJson() => {
    'versionCode': versionCode,
    'issuedAt': issuedAt,
  };

  static AcceptedManifestRecord? fromJson(dynamic raw) {
    if (raw is! Map) return null;
    final vc = raw['versionCode'];
    final ia = raw['issuedAt'];
    if (vc is! int || ia is! int || vc <= 0 || ia <= 0) return null;
    return AcceptedManifestRecord(versionCode: vc, issuedAt: ia);
  }
}

/// 已接受状态的后端存储。应用内由安全存储（flutter_secure_storage）实现。
abstract class ManifestStateStore {
  Future<String?> read(String key);
  Future<void> write(String key, String value);
}

/// 内存存储，仅供单元测试与纯 Dart 验证。
class InMemoryManifestStateStore implements ManifestStateStore {
  final Map<String, String> _data = <String, String>{};

  @override
  Future<String?> read(String key) async => _data[key];

  @override
  Future<void> write(String key, String value) async {
    _data[key] = value;
  }
}

/// 防重放守卫。key = `packageName/keyEpoch`，value 为该组合下已接受的最高
/// versionCode/issuedAt。严格单调：新的 (versionCode, issuedAt) 必须比已记录
/// 的更大才允许接受，否则判定为重放/降级。
class ManifestReplayGuard {
  ManifestReplayGuard(this._store);

  final ManifestStateStore _store;

  String _key(String packageName, int keyEpoch) => '$packageName/$keyEpoch';

  Future<AcceptedManifestRecord?> _load(String key) async {
    final raw = await _store.read(key);
    if (raw == null || raw.isEmpty) return null;
    try {
      return AcceptedManifestRecord.fromJson(jsonDecode(raw));
    } catch (_) {
      return null;
    }
  }

  /// 返回 true 表示该清单比已接受的最新记录更新（可接受）；false 表示
  /// 重放/降级。
  Future<bool> isAcceptable({
    required String packageName,
    required int keyEpoch,
    required int versionCode,
    required int issuedAt,
  }) async {
    final prev = await _load(_key(packageName, keyEpoch));
    if (prev == null) return true;
    if (versionCode < prev.versionCode) return false;
    if (versionCode == prev.versionCode && issuedAt <= prev.issuedAt) {
      return false;
    }
    return true;
  }

  /// 接受后记录新的最高值。
  Future<void> recordAccepted({
    required String packageName,
    required int keyEpoch,
    required int versionCode,
    required int issuedAt,
  }) async {
    final key = _key(packageName, keyEpoch);
    final prev = await _load(key);
    final next = AcceptedManifestRecord(
      versionCode: prev == null || versionCode > prev.versionCode
          ? versionCode
          : prev.versionCode,
      issuedAt: prev == null || issuedAt > prev.issuedAt
          ? issuedAt
          : prev.issuedAt,
    );
    await _store.write(key, jsonEncode(next.toJson()));
  }
}
