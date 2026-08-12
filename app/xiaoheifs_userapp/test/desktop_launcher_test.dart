import 'package:flutter_test/flutter_test.dart';
import 'package:opensource_userapp/core/utils/desktop_launcher_io.dart';

void main() {
  group('safe remote-target validation', () {
    test('accepts hostnames and IPv4 addresses', () {
      expect(isSafeRemoteHost('vps.example.com'), isTrue);
      expect(isSafeRemoteHost('127.0.0.1'), isTrue);
      expect(isSafeRemoteHost('my-vps_1'), isTrue);
    });

    test('rejects shell metacharacters in host', () {
      expect(isSafeRemoteHost('127.0.0.1; calc.exe'), isFalse);
      expect(isSafeRemoteHost('x & y'), isFalse);
      expect(isSafeRemoteHost('a|b'), isFalse);
      expect(isSafeRemoteHost('`id`'), isFalse);
      expect(isSafeRemoteHost(r'$(id)'), isFalse);
    });

    test('accepts plain usernames', () {
      expect(isSafeRemoteUsername('root'), isTrue);
      expect(isSafeRemoteUsername('administrator'), isTrue);
      expect(isSafeRemoteUsername('ops_user.1'), isTrue);
    });

    test('rejects shell metacharacters in username', () {
      expect(isSafeRemoteUsername('root;whoami'), isFalse);
      expect(isSafeRemoteUsername('a&b'), isFalse);
    });

    test('accepts numeric ports', () {
      expect(isSafeRemotePort('22'), isTrue);
      expect(isSafeRemotePort('65535'), isTrue);
    });

    test('rejects non-numeric or out-of-range ports', () {
      expect(isSafeRemotePort('22;calc'), isFalse);
      expect(isSafeRemotePort('abc'), isFalse);
      expect(isSafeRemotePort('70000'), isFalse);
    });
  });

  group('launchWindowsRdp validation', () {
    test('rejects invalid host before touching the process layer', () {
      expect(
        () => desktopLauncher.launchWindowsRdp(
          host: '127.0.0.1; calc.exe',
          port: '3389',
          username: 'admin',
          password: 'p@ss',
        ),
        throwsArgumentError,
      );
    });

    test('rejects invalid port for the RDP branch', () {
      expect(
        () => desktopLauncher.launchWindowsRdp(
          host: '127.0.0.1',
          port: '3389;calc',
          username: 'admin',
          password: 'p@ss',
        ),
        throwsArgumentError,
      );
    });

    test('rejects invalid username for the RDP branch', () {
      expect(
        () => desktopLauncher.launchWindowsRdp(
          host: '127.0.0.1',
          port: '3389',
          username: 'admin&whoami',
          password: 'p@ss',
        ),
        throwsArgumentError,
      );
    });
  });
}
