import 'package:flutter_test/flutter_test.dart';
import 'package:opensource_adminapp/services/avatar.dart';

void main() {
  group('avatarHeaders', () {
    test('attaches Bearer header for same-origin avatar URL', () {
      final headers = avatarHeaders(
        token: 'tok123',
        baseUrl: 'https://api.example.com',
        url: 'https://api.example.com/admin/api/v1/avatar/qq/123456',
      );
      expect(headers['Authorization'], 'Bearer tok123');
    });

    test('omits auth header for cross-origin absolute avatar URL', () {
      // A user-controlled avatar_url pointing at an attacker host must NOT
      // receive the admin token (native Image.network ignores CORS).
      final headers = avatarHeaders(
        token: 'tok123',
        baseUrl: 'https://api.example.com',
        url: 'https://evil.example.net/avatar.png',
      );
      expect(headers.containsKey('Authorization'), isFalse);
    });

    test('omits auth header when scheme differs even if host matches', () {
      final headers = avatarHeaders(
        token: 'tok123',
        baseUrl: 'https://api.example.com',
        url: 'http://api.example.com/avatar.png',
      );
      expect(headers.containsKey('Authorization'), isFalse);
    });

    test('omits auth header when host differs but scheme matches', () {
      final headers = avatarHeaders(
        token: 'tok123',
        baseUrl: 'https://api.example.com',
        url: 'https://cdn.example.com/avatar.png',
      );
      expect(headers.containsKey('Authorization'), isFalse);
    });

    test('omits auth header for protocol-relative URL resolved to another host',
        () {
      final headers = avatarHeaders(
        token: 'tok123',
        baseUrl: 'https://api.example.com',
        url: 'https://evil.example.net/avatar.png',
      );
      expect(headers.containsKey('Authorization'), isFalse);
    });

    test('omits auth header when url is empty', () {
      final headers = avatarHeaders(
        token: 'tok123',
        baseUrl: 'https://api.example.com',
        url: '',
      );
      expect(headers, isEmpty);
    });

    test('omits auth header when baseUrl is missing', () {
      final headers = avatarHeaders(
        token: 'tok123',
        url: 'https://api.example.com/avatar.png',
      );
      expect(headers, isEmpty);
    });

    test('falls back to apiKey when no token is present', () {
      final headers = avatarHeaders(
        apiKey: 'ak_live_secret',
        baseUrl: 'https://api.example.com',
        url: 'https://api.example.com/avatar.png',
      );
      expect(headers['Authorization'], 'Bearer ak_live_secret');
    });
  });

  group('resolveAvatarUrl', () {
    test('passes through absolute http(s) avatar URL', () {
      expect(
        resolveAvatarUrl(
          baseUrl: 'https://api.example.com',
          avatarUrl: 'https://cdn.example.com/a.png',
        ),
        'https://cdn.example.com/a.png',
      );
    });

    test('builds same-origin QQ proxy URL when qq is present', () {
      expect(
        resolveAvatarUrl(
          baseUrl: 'https://api.example.com/',
          qq: '123456',
        ),
        'https://api.example.com/admin/api/v1/avatar/qq/123456',
      );
    });
  });
}
