import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:shared_preferences/shared_preferences.dart';
import '../config/storage_keys.dart';

/// 存储服务
///
/// 分层存储：
/// - 敏感数据（access/refresh token、用户信息、用户 ID）使用
///   `flutter_secure_storage`（Android Keystore / iOS Keychain 加密存储），
///   不再明文落 SharedPreferences。
/// - 非敏感偏好（主题、语言、API Base URL）继续使用 SharedPreferences。
///
/// 启动时会把历史明文 SharedPreferences 中的凭据迁移进安全存储并删除明文。
class StorageService {
  static StorageService? _instance;
  static SharedPreferences? _prefs;

  static const FlutterSecureStorage _secure = FlutterSecureStorage();

  StorageService._();

  /// 获取单例实例
  static StorageService get instance {
    _instance ??= StorageService._();
    return _instance!;
  }

  /// 初始化
  static Future<void> init() async {
    _prefs ??= await SharedPreferences.getInstance();
    await _migrateLegacySensitiveData();
  }

  /// 把历史明文 SharedPreferences 中的凭据迁移到安全存储后删除明文，
  /// 避免旧版本留下的 token/userInfo 继续以明文驻留。
  static Future<void> _migrateLegacySensitiveData() async {
    final prefs = _prefs;
    if (prefs == null) return;
    Future<void> move(String key) async {
      final value = prefs.getString(key);
      if (value != null && value.isNotEmpty) {
        await _secure.write(key: key, value: value);
        await prefs.remove(key);
      }
    }

    await move(StorageKeys.accessToken);
    await move(StorageKeys.refreshToken);
    await move(StorageKeys.userId);
    await move(StorageKeys.userInfo);
  }

  // ==================== 非敏感偏好（SharedPreferences） ====================

  /// 保存字符串
  Future<bool> setString(String key, String value) async {
    return await _prefs!.setString(key, value);
  }

  /// 获取字符串
  String? getString(String key) {
    return _prefs!.getString(key);
  }

  /// 保存整数
  Future<bool> setInt(String key, int value) async {
    return await _prefs!.setInt(key, value);
  }

  /// 获取整数
  int? getInt(String key) {
    return _prefs!.getInt(key);
  }

  /// 保存布尔值
  Future<bool> setBool(String key, bool value) async {
    return await _prefs!.setBool(key, value);
  }

  /// 获取布尔值
  bool? getBool(String key) {
    return _prefs!.getBool(key);
  }

  /// 保存双精度浮点数
  Future<bool> setDouble(String key, double value) async {
    return await _prefs!.setDouble(key, value);
  }

  /// 获取双精度浮点数
  double? getDouble(String key) {
    return _prefs!.getDouble(key);
  }

  /// 保存字符串列表
  Future<bool> setStringList(String key, List<String> value) async {
    return await _prefs!.setStringList(key, value);
  }

  /// 获取字符串列表
  List<String>? getStringList(String key) {
    return _prefs!.getStringList(key);
  }

  /// 删除指定key
  Future<bool> remove(String key) async {
    return await _prefs!.remove(key);
  }

  /// 清空所有非敏感偏好数据
  Future<bool> clear() async {
    return await _prefs!.clear();
  }

  /// 检查是否包含指定key
  bool containsKey(String key) {
    return _prefs!.containsKey(key);
  }

  // ==================== Token / 用户信息（安全存储） ====================

  /// 保存访问令牌
  Future<void> setAccessToken(String token) async {
    await _secure.write(key: StorageKeys.accessToken, value: token);
  }

  /// 获取访问令牌
  Future<String?> getAccessToken() async {
    return await _secure.read(key: StorageKeys.accessToken);
  }

  /// 删除访问令牌
  Future<void> clearAccessToken() async {
    await _secure.delete(key: StorageKeys.accessToken);
  }

  /// 保存刷新令牌
  Future<void> setRefreshToken(String token) async {
    await _secure.write(key: StorageKeys.refreshToken, value: token);
  }

  /// 获取刷新令牌
  Future<String?> getRefreshToken() async {
    return await _secure.read(key: StorageKeys.refreshToken);
  }

  /// 清除所有认证信息
  Future<void> clearAuthData() async {
    await _secure.delete(key: StorageKeys.accessToken);
    await _secure.delete(key: StorageKeys.refreshToken);
    await _secure.delete(key: StorageKeys.userId);
    await _secure.delete(key: StorageKeys.userInfo);
  }

  // ==================== API配置相关 ====================

  /// 保存API基础URL
  Future<bool> setApiBaseUrl(String url) async {
    return await setString(StorageKeys.apiBaseUrl, url);
  }

  /// 获取API基础URL
  String? getApiBaseUrl() {
    return getString(StorageKeys.apiBaseUrl);
  }

  // ==================== 用户信息相关 ====================

  /// 保存用户ID
  Future<void> setUserId(int userId) async {
    await _secure.write(key: StorageKeys.userId, value: '$userId');
  }

  /// 获取用户ID
  Future<int?> getUserId() async {
    final value = await _secure.read(key: StorageKeys.userId);
    return value == null ? null : int.tryParse(value);
  }

  /// 保存用户信息(JSON字符串)
  Future<void> setUserInfo(String userInfo) async {
    await _secure.write(key: StorageKeys.userInfo, value: userInfo);
  }

  /// 获取用户信息(JSON字符串)
  Future<String?> getUserInfo() async {
    return await _secure.read(key: StorageKeys.userInfo);
  }

  // ==================== 主题和语言相关 ====================

  /// 保存主题模式
  Future<bool> setThemeMode(String mode) async {
    return await setString(StorageKeys.themeMode, mode);
  }

  /// 获取主题模式
  String? getThemeMode() {
    return getString(StorageKeys.themeMode);
  }

  /// 保存语言设置
  Future<bool> setLocale(String locale) async {
    return await setString(StorageKeys.locale, locale);
  }

  /// 获取语言设置
  String? getLocale() {
    return getString(StorageKeys.locale);
  }
}
