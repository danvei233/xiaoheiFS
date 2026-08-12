import 'package:flutter_test/flutter_test.dart';
import 'package:opensource_userapp/data/repositories/vps_repository.dart';

void main() {
  group('VpsRepository.isSafeRedirectUrl', () {
    test('accepts https panel / vnc redirect URLs in all modes', () {
      expect(
        VpsRepository.isSafeRedirectUrl('https://panel.example.com/vnc'),
        isTrue,
      );
      expect(
        VpsRepository.isSafeRedirectUrl('https://127.0.0.1:8080/vnc'),
        isTrue,
      );
    });

    test('rejects non-localhost cleartext http (public/LAN eavesdropping)', () {
      // In debug/profile (tests), only localhost http is allowed; public/LAN
      // http is rejected because the short-lived ticket would traverse plaintext.
      expect(
        VpsRepository.isSafeRedirectUrl('http://panel.example.com/'),
        isFalse,
      );
      expect(
        VpsRepository.isSafeRedirectUrl('http://192.168.1.10:8888/vnc'),
        isFalse,
      );
      expect(
        VpsRepository.isSafeRedirectUrl('http://10.0.0.5:8888/vnc'),
        isFalse,
      );
    });

    test('allows localhost http only in non-release (dev exception)', () {
      expect(
        VpsRepository.isSafeRedirectUrl('http://localhost:8080/vnc'),
        isTrue,
      );
      expect(
        VpsRepository.isSafeRedirectUrl('http://127.0.0.1:8080/vnc'),
        isTrue,
      );
      expect(VpsRepository.isSafeRedirectUrl('http://[::1]:8080/vnc'), isTrue);
    });

    test('release mode rejects ALL cleartext http including localhost', () {
      // Simulate a release build: no cleartext at all — the ticket must never
      // travel over http, even to a local panel.
      expect(
        VpsRepository.isSafeRedirectUrl(
          'http://localhost:8080/vnc',
          allowCleartext: false,
        ),
        isFalse,
      );
      expect(
        VpsRepository.isSafeRedirectUrl(
          'http://127.0.0.1:8080/vnc',
          allowCleartext: false,
        ),
        isFalse,
      );
      expect(
        VpsRepository.isSafeRedirectUrl(
          'http://panel.example.com/vnc',
          allowCleartext: false,
        ),
        isFalse,
      );
      expect(
        VpsRepository.isSafeRedirectUrl(
          'https://panel.example.com/vnc',
          allowCleartext: false,
        ),
        isTrue,
      );
    });

    test('rejects arbitrary schemes that could launch intents', () {
      expect(
        VpsRepository.isSafeRedirectUrl('intent://evil#Intent;end'),
        isFalse,
      );
      expect(VpsRepository.isSafeRedirectUrl('file:///etc/passwd'), isFalse);
      expect(VpsRepository.isSafeRedirectUrl('javascript:alert(1)'), isFalse);
      expect(VpsRepository.isSafeRedirectUrl('tel:12345'), isFalse);
      expect(
        VpsRepository.isSafeRedirectUrl('ftp://panel.example.com/vnc'),
        isFalse,
      );
    });

    test('rejects malformed values', () {
      expect(VpsRepository.isSafeRedirectUrl(''), isFalse);
      expect(
        VpsRepository.isSafeRedirectUrl('//no-scheme.example.com'),
        isFalse,
      );
      expect(VpsRepository.isSafeRedirectUrl('example.com/vnc'), isFalse);
      expect(VpsRepository.isSafeRedirectUrl('not a url'), isFalse);
    });
  });
}
