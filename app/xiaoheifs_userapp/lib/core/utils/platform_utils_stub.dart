import 'platform_utils.dart';

class _PlatformUtilsStub implements PlatformUtils {
  @override
  bool get isWeb => false;
  @override
  bool get isWindows => false;
  @override
  bool get isAndroid => false;
  @override
  bool get isIOS => false;
  @override
  bool get isMacOS => false;
  @override
  bool get isLinux => false;
  @override
  bool get isMobile => false;
}

final PlatformUtils platformUtils = _PlatformUtilsStub();

void downloadTextFileImpl(String filename, String content, String mimeType) {}
