import 'dart:convert';

/// Cross-session replay guard: records the highest accepted versionCode /
/// issuedAt per "packageName/keyEpoch" and rejects any signed manifest that is
/// not strictly newer (downgrade / replay of an already-accepted manifest).
///
/// Pure Dart (no Flutter dependency): tests use [InMemoryManifestStateStore];
/// the app persists via a secure-storage-backed [ManifestStateStore].
/// Contract: `docs/update-signing-contract.md` §5.
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

/// Backing store for the accepted state. The app implements this with
/// flutter_secure_storage.
abstract class ManifestStateStore {
  Future<String?> read(String key);
  Future<void> write(String key, String value);
}

/// In-memory store for unit tests / pure-Dart verification.
class InMemoryManifestStateStore implements ManifestStateStore {
  final Map<String, String> _data = <String, String>{};

  @override
  Future<String?> read(String key) async => _data[key];

  @override
  Future<void> write(String key, String value) async {
    _data[key] = value;
  }
}

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

  /// True when the incoming manifest is strictly newer than the highest
  /// accepted record for this package/epoch; false on replay/downgrade.
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

  /// Record the newly accepted highest value.
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
