import 'desktop_launcher.dart';

class _DesktopLauncherStub implements DesktopLauncher {
  @override
  Future<void> launchWindowsRdp({
    required String host,
    required String port,
    required String username,
    required String password,
  }) async {
    throw UnsupportedError('Desktop launch is not supported on this platform.');
  }

  @override
  Future<void> launchWindowsSsh({
    required String host,
    required String port,
    required String username,
  }) async {
    throw UnsupportedError('Desktop launch is not supported on this platform.');
  }
}

final DesktopLauncher desktopLauncher = _DesktopLauncherStub();
