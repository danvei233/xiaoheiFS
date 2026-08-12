import 'package:flutter_test/flutter_test.dart';
import 'package:opensource_userapp/core/constants/input_limits.dart';
import 'package:opensource_userapp/core/utils/validators.dart';

void main() {
  group('Validators.validateUsername', () {
    test('accepts usernames up to the backend limit of 64 chars', () {
      final longName = 'a' * InputLimits.username;
      expect(Validators.validateUsername(longName), isNull);

      final tooLong = 'a' * (InputLimits.username + 1);
      expect(Validators.validateUsername(tooLong), isNotNull);
    });

    test('rejects shell-unsafe characters', () {
      expect(Validators.validateUsername('a;drop'), isNotNull);
      expect(Validators.validateUsername('root|admin'), isNotNull);
    });
  });

  group('Validators.validatePassword', () {
    test('enforces a minimum length', () {
      expect(Validators.validatePassword('12345'), isNotNull);
      expect(Validators.validatePassword('123456'), isNull);
    });

    test('enforces the backend maximum length', () {
      final maxLen = 'p' * InputLimits.password;
      expect(Validators.validatePassword(maxLen), isNull);

      final tooLong = 'p' * (InputLimits.password + 1);
      expect(Validators.validatePassword(tooLong), isNotNull);
    });
  });
}
