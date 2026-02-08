import 'dart:io';
import 'platform_utils.dart';

class _PlatformUtilsIo implements PlatformUtils {
  @override
  bool get isWeb => false;
  @override
  bool get isWindows => Platform.isWindows;
  @override
  bool get isAndroid => Platform.isAndroid;
  @override
  bool get isIOS => Platform.isIOS;
  @override
  bool get isMacOS => Platform.isMacOS;
  @override
  bool get isLinux => Platform.isLinux;
  @override
  bool get isMobile => Platform.isAndroid || Platform.isIOS;
}

final PlatformUtils platformUtils = _PlatformUtilsIo();

void downloadTextFileImpl(String filename, String content, String mimeType) {}
