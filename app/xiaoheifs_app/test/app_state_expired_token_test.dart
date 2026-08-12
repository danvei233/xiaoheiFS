import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:opensource_adminapp/app_state.dart';
import 'package:shared_preferences/shared_preferences.dart';

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  setUp(() {
    SharedPreferences.setMockInitialValues({});
    FlutterSecureStorage.setMockInitialValues({});
  });

  test(
    'AppState.load clears a session whose token has already expired',
    () async {
      final storage = AppStorage();
      await storage.saveSession(
        Session(
          apiUrl: 'https://api.example.com',
          username: 'admin',
          authType: 'password',
          token: 'expired-token',
          refreshToken: 'expired-refresh',
          tokenExpiresAt: DateTime.now().subtract(const Duration(seconds: 1)),
        ),
      );

      final state = AppState(storage: storage);
      await state.load();

      expect(state.isReady, isTrue, reason: 'the app must still become ready');
      expect(state.isLoggedIn, isFalse,
          reason: 'an expired token must not load as a valid session');
      expect(state.session, isNull);
    },
  );

  test('AppState.load keeps a session whose token has not expired', () async {
    final storage = AppStorage();
    await storage.saveSession(
      Session(
        apiUrl: 'https://api.example.com',
        username: 'admin',
        authType: 'password',
        token: 'valid-token',
        refreshToken: 'valid-refresh',
        tokenExpiresAt: DateTime.now().add(const Duration(hours: 1)),
      ),
    );

    final state = AppState(storage: storage);
    await state.load();

    expect(state.isReady, isTrue);
    expect(state.isLoggedIn, isTrue);
    expect(state.session?.token, 'valid-token');
  });

  test('AppState.load keeps an API-key session (no token expiry)', () async {
    final storage = AppStorage();
    await storage.saveSession(
      Session(
        apiUrl: 'https://api.example.com',
        username: 'svc',
        authType: 'api_key',
        apiKey: 'k-123',
      ),
    );

    final state = AppState(storage: storage);
    await state.load();

    expect(state.isReady, isTrue);
    expect(state.isLoggedIn, isTrue);
    expect(state.session?.apiKey, 'k-123');
  });
}
