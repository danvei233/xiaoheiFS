import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'package:opensource_userapp/core/config/storage_keys.dart';
import 'package:opensource_userapp/core/storage/storage_service.dart';

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  setUp(() async {
    SharedPreferences.setMockInitialValues({});
    FlutterSecureStorage.setMockInitialValues({});
    await StorageService.init();
  });

  group('StorageService token storage', () {
    test('access and refresh tokens are stored under distinct keys', () async {
      final storage = StorageService.instance;
      await storage.setAccessToken('access-token-value');
      await storage.setRefreshToken('refresh-token-value');

      // If both tokens collapse into one slot, getAccessToken() would return
      // the refresh token written last.
      expect(await storage.getAccessToken(), 'access-token-value');
      expect(await storage.getRefreshToken(), 'refresh-token-value');

      // Clearing the access token must not wipe the refresh token.
      await storage.clearAccessToken();
      expect(await storage.getAccessToken(), isNull);
      expect(await storage.getRefreshToken(), 'refresh-token-value');
    });

    test('token keys are distinct and not scrubbed placeholders', () {
      expect(StorageKeys.accessToken, isNot(contains('<REDACTED>')));
      expect(StorageKeys.refreshToken, isNot(contains('<REDACTED>')));
      expect(StorageKeys.accessToken, isNot(StorageKeys.refreshToken));
      expect(StorageKeys.accessToken, isNotEmpty);
      expect(StorageKeys.refreshToken, isNotEmpty);
    });

    test('legacy plaintext tokens are migrated to secure storage and removed',
        () async {
      // 模拟旧版本：token 明文存在于 SharedPreferences。
      SharedPreferences.setMockInitialValues({
        StorageKeys.accessToken: 'legacy-access',
        StorageKeys.refreshToken: 'legacy-refresh',
      });
      await StorageService.init();

      expect(await StorageService.instance.getAccessToken(), 'legacy-access');
      expect(await StorageService.instance.getRefreshToken(), 'legacy-refresh');
      // 明文必须已从 SharedPreferences 移除。
      final prefs = await SharedPreferences.getInstance();
      expect(prefs.getString(StorageKeys.accessToken), isNull);
      expect(prefs.getString(StorageKeys.refreshToken), isNull);
    });
  });
}
