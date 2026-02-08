import 'dart:html' as html;
import 'platform_utils.dart';

class _PlatformUtilsWeb implements PlatformUtils {
  @override
  bool get isWeb => true;
  @override
  bool get isWindows => _uaContains('windows');
  @override
  bool get isAndroid => _uaContains('android');
  @override
  bool get isIOS => _uaContains('iphone') || _uaContains('ipad') || _uaContains('ipod');
  @override
  bool get isMacOS => _uaContains('macintosh');
  @override
  bool get isLinux => _uaContains('linux') && !_uaContains('android');
  @override
  bool get isMobile => isAndroid || isIOS;

  bool _uaContains(String token) {
    final ua = html.window.navigator.userAgent.toLowerCase();
    return ua.contains(token);
  }
}

final PlatformUtils platformUtils = _PlatformUtilsWeb();

void downloadTextFileImpl(String filename, String content, String mimeType) {
  final blob = html.Blob([content], mimeType);
  final url = html.Url.createObjectUrlFromBlob(blob);
  final anchor = html.AnchorElement(href: url)
    ..setAttribute('download', filename)
    ..click();
  html.Url.revokeObjectUrl(url);
}
