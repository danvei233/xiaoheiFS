import 'package:flutter_test/flutter_test.dart';

import 'package:opensource_userapp/presentation/routes/redirect_guard.dart';

/// F11 回归测试：登录后 redirect 参数只允许站内 /console 路径，
/// 拒绝外域 URL 与可造成循环导航的 /login 链。
void main() {
  group('sanitizeRedirectParam (F11)', () {
    test('允许 /console 站内路径', () {
      expect(sanitizeRedirectParam('/console'), '/console');
      expect(sanitizeRedirectParam('/console/orders/1'), '/console/orders/1');
      expect(
        sanitizeRedirectParam('%2Fconsole%2Forders%2F1'),
        '/console/orders/1',
      );
    });

    test('拒绝外域 URL', () {
      expect(sanitizeRedirectParam('https://evil.com'), isNull);
      expect(sanitizeRedirectParam('%2F%2Fevil.com'), isNull);
    });

    test('拒绝非 /console 路径与循环链', () {
      expect(sanitizeRedirectParam('/login'), isNull);
      expect(sanitizeRedirectParam('/register'), isNull);
      expect(sanitizeRedirectParam('/login?redirect=%2Fconsole'), isNull);
      expect(
        sanitizeRedirectParam('%2Flogin%3Fredirect%3D%252Flogin'),
        isNull,
      );
    });

    test('null 与空字符串返回 null', () {
      expect(sanitizeRedirectParam(null), isNull);
      expect(sanitizeRedirectParam(''), isNull);
    });
  });
}
