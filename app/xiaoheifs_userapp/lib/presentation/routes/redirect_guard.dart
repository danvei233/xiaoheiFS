/// 登录后 `redirect` 查询参数的白名单校验。
///
/// 只允许跳转到站内 `/console` 前缀路径；外域 URL、以及构造的
/// `/login?redirect=%2Flogin...` 循环链一律拒绝，返回 null 表示
/// 调用方应回退到默认控制台首页。
String? sanitizeRedirectParam(String? raw) {
  if (raw == null || raw.isEmpty) return null;
  final decoded = Uri.decodeComponent(raw);
  if (decoded.startsWith('/console')) return decoded;
  return null;
}
