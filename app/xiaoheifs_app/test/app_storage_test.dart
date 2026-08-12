import 'dart:convert';
import 'dart:io';

import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:opensource_adminapp/models/session.dart';
import 'package:opensource_adminapp/services/app_storage.dart';
import 'package:path_provider_platform_interface/path_provider_platform_interface.dart';
import 'package:plugin_platform_interface/plugin_platform_interface.dart';
import 'package:shared_preferences/shared_preferences.dart';

class _FakePathProvider extends PathProviderPlatform
    with MockPlatformInterfaceMixin {
  final String path;
  _FakePathProvider(this.path);

  @override
  Future<String?> getApplicationSupportPath() async => path;
}

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  test(
    'saveSession does not persist a plaintext session.json fallback',
    () async {
      final dir = await Directory.systemTemp.createTemp('app_storage_test');
      addTearDown(() => dir.delete(recursive: true));
      PathProviderPlatform.instance = _FakePathProvider(dir.path);
      SharedPreferences.setMockInitialValues({});
      FlutterSecureStorage.setMockInitialValues({});

      final storage = AppStorage();
      await storage.saveSession(
        const Session(
          apiUrl: 'https://api.example.com',
          username: 'admin',
          authType: 'password',
          token: 'secret-token',
          refreshToken: 'secret-refresh',
          apiKey: 'ak_secret',
        ),
      );

      final file = File('${dir.path}${Platform.pathSeparator}session.json');
      expect(
        await file.exists(),
        isFalse,
        reason: 'credentials must not be mirrored to a plaintext file',
      );
    },
  );

  test(
    'loadSession migrates a legacy plaintext session.json and deletes it',
    () async {
      final dir = await Directory.systemTemp.createTemp('app_storage_migrate');
      addTearDown(() => dir.delete(recursive: true));
      PathProviderPlatform.instance = _FakePathProvider(dir.path);
      SharedPreferences.setMockInitialValues({});
      FlutterSecureStorage.setMockInitialValues({});

      // 预置一个旧版本写入的明文 session.json。
      final file = File('${dir.path}${Platform.pathSeparator}session.json');
      await file.writeAsString(
        jsonEncode({
          'api_url': 'https://api.example.com',
          'auth_type': 'password',
          'username': 'admin',
          'token': 'legacy-token',
          'refresh_token': 'legacy-refresh',
          'token_expires_at': '2026-12-31T00:00:00.000',
          'email': 'admin@example.com',
          'api_key': null,
        }),
      );

      final storage = AppStorage();
      final session = await storage.loadSession();

      // (a) 返回的 Session 与文件一致。
      expect(session, isNotNull);
      expect(session!.apiUrl, 'https://api.example.com');
      expect(session.authType, 'password');
      expect(session.token, 'legacy-token');
      expect(session.refreshToken, 'legacy-refresh');

      // (b) 明文文件被删除，不再驻留。
      expect(
        await file.exists(),
        isFalse,
        reason: 'legacy plaintext session.json must be deleted after migration',
      );

      // (c) 凭据已迁入安全存储。
      const secure = FlutterSecureStorage();
      expect(await secure.read(key: 'token'), 'legacy-token');
      expect(await secure.read(key: 'refresh_token'), 'legacy-refresh');

      // 再次 loadSession：prefs 已有数据，不会再走文件迁移。
      final second = await storage.loadSession();
      expect(second!.token, 'legacy-token');
    },
  );

  test(
    'loadSession deletes a corrupted session.json (no lingering plaintext)',
    () async {
      final dir = await Directory.systemTemp.createTemp('app_storage_corrupt');
      addTearDown(() => dir.delete(recursive: true));
      PathProviderPlatform.instance = _FakePathProvider(dir.path);
      SharedPreferences.setMockInitialValues({});
      FlutterSecureStorage.setMockInitialValues({});

      // 损坏的 JSON：可能包含截断的敏感明文，迁移失败也必须删除。
      final file = File('${dir.path}${Platform.pathSeparator}session.json');
      await file.writeAsString(
        '{"api_url":"https://api.example.com","token":"partial-',
      );

      final storage = AppStorage();
      final session = await storage.loadSession();

      expect(
        session,
        isNull,
        reason: 'corrupted session.json yields no session',
      );
      expect(
        await file.exists(),
        isFalse,
        reason:
            'corrupted plaintext session.json must be deleted, not left on disk',
      );
    },
  );

  test(
    'loadSession deletes an incomplete session.json (missing api_url)',
    () async {
      final dir = await Directory.systemTemp.createTemp(
        'app_storage_incomplete',
      );
      addTearDown(() => dir.delete(recursive: true));
      PathProviderPlatform.instance = _FakePathProvider(dir.path);
      SharedPreferences.setMockInitialValues({});
      FlutterSecureStorage.setMockInitialValues({});

      final file = File('${dir.path}${Platform.pathSeparator}session.json');
      await file.writeAsString(
        jsonEncode({
          'auth_type': 'password',
          'username': 'admin',
          'token': 'should-not-linger',
        }),
      );

      final storage = AppStorage();
      final session = await storage.loadSession();

      expect(
        session,
        isNull,
        reason: 'incomplete session.json yields no session',
      );
      expect(
        await file.exists(),
        isFalse,
        reason:
            'incomplete plaintext session.json must be deleted (no permanent residue)',
      );
    },
  );

  test(
    'loadSession deletes a password session.json whose token is missing',
    () async {
      final dir = await Directory.systemTemp.createTemp('app_storage_notoken');
      addTearDown(() => dir.delete(recursive: true));
      PathProviderPlatform.instance = _FakePathProvider(dir.path);
      SharedPreferences.setMockInitialValues({});
      FlutterSecureStorage.setMockInitialValues({});

      final file = File('${dir.path}${Platform.pathSeparator}session.json');
      await file.writeAsString(
        jsonEncode({
          'api_url': 'https://api.example.com',
          'auth_type': 'password',
          'username': 'admin',
          // 无 token：password 会话不完整。
        }),
      );

      final storage = AppStorage();
      final session = await storage.loadSession();

      expect(
        session,
        isNull,
        reason: 'password session without token yields no session',
      );
      expect(
        await file.exists(),
        isFalse,
        reason:
            'incomplete plaintext session.json must be deleted even without credentials',
      );
    },
  );
}
