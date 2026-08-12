import 'package:flutter_secure_storage/flutter_secure_storage.dart';

import 'update_replay_guard.dart';

/// flutter_secure_storage-backed accepted-update state store.
///
/// Persists the highest accepted versionCode/issuedAt per "packageName/keyEpoch"
/// so a signed manifest from the past is never replayed across sessions. The
/// values live in the system Keychain/Keystore like other credential data.
class SecureManifestStateStore implements ManifestStateStore {
  static const FlutterSecureStorage _secure = FlutterSecureStorage();

  static const String _prefix = 'xiaohei_update_state_';

  @override
  Future<String?> read(String key) async =>
      await _secure.read(key: '$_prefix$key');

  @override
  Future<void> write(String key, String value) async {
    await _secure.write(key: '$_prefix$key', value: value);
  }
}
