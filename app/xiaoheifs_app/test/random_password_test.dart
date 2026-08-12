import 'package:flutter_test/flutter_test.dart';
import 'package:opensource_adminapp/screens/users_screen.dart';

void main() {
  group('randomPassword', () {
    test('produces a 10-character password from the safe alphabet', () {
      final password = randomPassword();
      expect(password.length, 10);
      final allowed = RegExp(r'^[ABCDEFGHJKMNPQRSTUVWXYZabcdefghjkmnpqrstuvwxyz23456789]{10}$');
      expect(allowed.hasMatch(password), isTrue,
          reason: 'password must only contain the unambiguous charset');
    });

    test('does not derive the same password from a single point in time', () {
      // The previous implementation derived every character from
      // (now.millisecondsSinceEpoch + i*37) % 51, so calls made within the
      // same millisecond were byte-for-byte identical. With a CSPRNG the
      // outputs must be distinct.
      final seen = <String>{};
      for (var i = 0; i < 200; i++) {
        seen.add(randomPassword());
      }
      expect(seen.length, greaterThan(150));
    });
  });
}
