import 'platform_utils_stub.dart'
    if (dart.library.html) 'platform_utils_web.dart'
    if (dart.library.io) 'platform_utils_io.dart';

abstract class PlatformUtils {
  bool get isWeb;
  bool get isWindows;
  bool get isAndroid;
  bool get isIOS;
  bool get isMacOS;
  bool get isLinux;
  bool get isMobile;
}

PlatformUtils getPlatformUtils() => platformUtils;

void downloadTextFile(String filename, String content, String mimeType) =>
    downloadTextFileImpl(filename, content, mimeType);
