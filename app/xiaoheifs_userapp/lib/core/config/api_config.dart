/// API配置类
/// 管理API基础URL和相关配置
class ApiConfig {
  /// 默认API URL
  static const String defaultUrl = 'http://localhost:8080/api';

  /// 当前API URL（可由用户在登录页配置）
  static String _baseUrl = defaultUrl;

  /// 获取当前API URL
  static String get baseUrl => _baseUrl;

  /// 设置API URL（用于用户自定义服务器地址）
  static void setBaseUrl(String url) {
    _baseUrl = url;
  }

  /// 重置为默认URL
  static void reset() {
    _baseUrl = defaultUrl;
  }

  /// 获取完整的API端点URL
  static String getEndpoint(String path) {
    return '$baseUrl$path';
  }
}
