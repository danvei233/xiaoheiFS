/// 存储键常量
/// 用于SharedPreferences存储的键名定义
class StorageKeys {
  StorageKeys._();

  // 认证相关
  static const String accessToken = 'access_token';
  static const String refreshToken = 'refresh_token';
  static const String userId = 'user_id';
  static const String userInfo = 'user_info';

  // API配置
  static const String apiBaseUrl = 'api_base_url';

  // 主题设置
  static const String themeMode = 'theme_mode';

  // 语言设置
  static const String locale = 'locale';
}
