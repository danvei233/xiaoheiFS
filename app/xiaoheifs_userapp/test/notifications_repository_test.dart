import 'package:flutter_test/flutter_test.dart';
import 'package:opensource_userapp/data/repositories/notifications_repository.dart';

void main() {
  group('NotificationsRepository.parseUnreadCount', () {
    test('accepts int values', () {
      expect(NotificationsRepository.parseUnreadCount(3), 3);
      expect(NotificationsRepository.parseUnreadCount(0), 0);
    });

    test('accepts num values', () {
      expect(NotificationsRepository.parseUnreadCount(3.0), 3);
    });

    test('accepts numeric strings returned by misbehaving servers', () {
      expect(NotificationsRepository.parseUnreadCount('3'), 3);
      expect(NotificationsRepository.parseUnreadCount('0'), 0);
    });

    test('falls back to zero for unparseable or missing values', () {
      expect(NotificationsRepository.parseUnreadCount('abc'), 0);
      expect(NotificationsRepository.parseUnreadCount(null), 0);
      expect(NotificationsRepository.parseUnreadCount(<Object>[]), 0);
    });
  });
}
