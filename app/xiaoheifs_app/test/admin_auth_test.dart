import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;
import 'package:http/testing.dart';
import 'package:opensource_adminapp/services/admin_auth.dart';

void main() {
  group('AdminAuthService error decoding', () {
    test('non-JSON error body throws AuthException, not FormatException',
        () async {
      final auth = AdminAuthService(
        client: MockClient(
          (_) async => http.Response('<html>Bad Gateway</html>', 502),
        ),
      );
      await expectLater(
        auth.login(
          apiUrl: 'https://api.example.com',
          username: 'admin',
          password: 'pw',
        ),
        throwsA(isA<AuthException>()),
      );
    });
  });
}
