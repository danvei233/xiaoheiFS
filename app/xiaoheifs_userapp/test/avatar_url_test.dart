import 'package:flutter_test/flutter_test.dart';
import 'package:opensource_userapp/core/utils/avatar_url.dart';

void main() {
  group('resolveUserAvatarUrl', () {
    test('QQ avatar endpoint uses https', () {
      final url = resolveUserAvatarUrl(
        baseUrl: 'https://api.example.com/api',
        qq: '10001',
      );
      expect(url, startsWith('https://q1.qlogo.cn/'));
      expect(url, isNot(contains('http://q1.qlogo.cn')));
    });

    test('direct avatar URL keeps its own scheme', () {
      final url = resolveUserAvatarUrl(
        baseUrl: 'https://api.example.com',
        avatarUrl: 'http://cdn.example.com/a.png',
      );
      expect(url, 'http://cdn.example.com/a.png');
    });

    test('protocol-relative avatar URL is upgraded to https', () {
      final url = resolveUserAvatarUrl(
        baseUrl: 'https://api.example.com',
        avatarUrl: '//cdn.example.com/a.png',
      );
      expect(url, 'https://cdn.example.com/a.png');
    });
  });
}
