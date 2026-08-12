import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;
import 'package:http/testing.dart';
import 'package:opensource_adminapp/models/auth_tokens.dart';
import 'package:opensource_adminapp/services/api_client.dart';

void main() {
  group('ApiClient auth expiry', () {
    test('calls onAuthExpired when a 401 refresh fails', () async {
      var refreshCalls = 0;
      var expired = false;
      final client = ApiClient(
        baseUrl: 'https://api.example.com',
        token: 'expired-token',
        refreshAuth: () async {
          refreshCalls++;
          return null; // refresh token revoked / refresh endpoint failing
        },
        onAuthExpired: () => expired = true,
        client: MockClient((req) async {
          return http.Response('{"error":"unauthorized"}', 401);
        }),
      );
      await expectLater(
        client.getJson('/admin/api/v1/profile'),
        throwsA(isA<ApiException>()),
      );
      expect(refreshCalls, 1);
      expect(expired, isTrue);
    });

    test('does not call onAuthExpired when refresh succeeds', () async {
      var requests = 0;
      var expired = false;
      final client = ApiClient(
        baseUrl: 'https://api.example.com',
        token: 'expired-token',
        refreshAuth: () async {
          return const AuthTokens(
            accessToken: 'fresh-token',
            refreshToken: 'fresh-refresh',
            expiresIn: 3600,
          );
        },
        onTokens: (t) {},
        onAuthExpired: () => expired = true,
        client: MockClient((req) async {
          requests++;
          if (requests == 1) {
            return http.Response('{"error":"unauthorized"}', 401);
          }
          return http.Response('{"ok":true}', 200);
        }),
      );
      final result = await client.getJson('/admin/api/v1/profile');
      expect(result['ok'], isTrue);
      expect(requests, 2, reason: 'one 401 then one retried request');
      expect(expired, isFalse);
    });
  });
  group('ApiClient error decoding', () {
    test('non-JSON error body throws ApiException, not FormatException', () async {
      final client = ApiClient(
        baseUrl: 'https://api.example.com',
        token: 'tok',
        client: MockClient(
          (_) async => http.Response('<html><body>Bad Gateway</body></html>', 502),
        ),
      );
      await expectLater(
        client.getJson('/admin/api/v1/profile'),
        throwsA(isA<ApiException>()),
      );
    });

    test('non-JSON 401 body in the refresh path is still handled', () async {
      var refreshed = 0;
      final client = ApiClient(
        baseUrl: 'https://api.example.com',
        token: 'tok',
        refreshAuth: () async {
          refreshed++;
          return null;
        },
        client: MockClient((_) async => http.Response('not json', 401)),
      );
      await expectLater(
        client.getJson('/admin/api/v1/profile'),
        throwsA(isA<ApiException>()),
      );
      expect(refreshed, 1);
    });
  });
}
