import 'package:flutter_secure_storage/flutter_secure_storage.dart';

import 'update_replay_guard.dart';

/// 基于 `flutter_secure_storage` 的已接受更新状态存储。
///
/// 记录每个 `packageName/keyEpoch` 下已接受的最高 versionCode/issuedAt，
/// 用于跨会话拒绝旧签名清单的降级/重放。凭据级数据与 token 一样走系统
/// Keychain / Keystore 加密存储，不落明文。
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
