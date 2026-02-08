import 'dart:io';
import 'desktop_launcher.dart';

class _DesktopLauncherIo implements DesktopLauncher {
  @override
  Future<void> launchWindowsRdp({
    required String host,
    required String port,
    required String username,
    required String password,
  }) async {
    final target = 'TERMSRV/$host:$port';
    await Process.run(
      'cmdkey',
      [
        '/generic:$target',
        '/user:$username',
        '/pass:$password',
      ],
      runInShell: true,
    );
    await Process.start(
      'mstsc',
      ['/v:$host:$port'],
      runInShell: true,
    );
  }

  @override
  Future<void> launchWindowsSsh({
    required String host,
    required String port,
    required String username,
  }) async {
    final command = 'ssh $username@$host -p $port';
    await Process.start(
      'powershell',
      ['-NoExit', '-Command', command],
      runInShell: true,
    );
  }
}

final DesktopLauncher desktopLauncher = _DesktopLauncherIo();
