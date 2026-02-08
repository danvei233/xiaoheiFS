import 'desktop_launcher_stub.dart'
    if (dart.library.io) 'desktop_launcher_io.dart'
    if (dart.library.html) 'desktop_launcher_stub.dart';

abstract class DesktopLauncher {
  Future<void> launchWindowsRdp({
    required String host,
    required String port,
    required String username,
    required String password,
  });

  Future<void> launchWindowsSsh({
    required String host,
    required String port,
    required String username,
  });
}

DesktopLauncher getDesktopLauncher() => desktopLauncher;
