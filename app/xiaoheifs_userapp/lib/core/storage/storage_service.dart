import 'package:shared_preferences/shared_preferences.dart';
import '../config/storage_keys.dart';

/// 存储服务
/// 封装SharedPreferences，提供类型安全的存储操作
class StorageService {
  static StorageService? _instance;
  static SharedPreferences? _prefs;

  StorageService._();

  /// 获取单例实例
  static StorageService get instance {
    _instance ??= StorageService._();
    return _instance!;
  }

  /// 初始化
  static Future<void> init() async {
    _prefs ??= await SharedPreferences.getInstance();
  }

  // ==================== 通用方法 ====================

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

  /// 清空所有数据
  Future<bool> clear() async {
    return await _prefs!.clear();
  }

  /// 检查是否包含指定key
  bool containsKey(String key) {
    return _prefs!.containsKey(key);
  }

  // ==================== Token相关 ====================

  /// 保存访问令牌
  Future<bool> setAccessToken(String token) async {
    return await setString(StorageKeys.accessToken, token);
  }

  /// 获取访问令牌
  String? getAccessToken() {
    return getString(StorageKeys.accessToken);
  }

  /// 删除访问令牌
  Future<bool> clearAccessToken() async {
    return await remove(StorageKeys.accessToken);
  }

  /// 保存刷新令牌
  Future<bool> setRefreshToken(String token) async {
    return await setString(StorageKeys.refreshToken, token);
  }

  /// 获取刷新令牌
  String? getRefreshToken() {
    return getString(StorageKeys.refreshToken);
  }

  /// 清除所有认证信息
  Future<void> clearAuthData() async {
    await remove(StorageKeys.accessToken);
    await remove(StorageKeys.refreshToken);
    await remove(StorageKeys.userId);
    await remove(StorageKeys.userInfo);
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
  Future<bool> setUserId(int userId) async {
    return await setInt(StorageKeys.userId, userId);
  }

  /// 获取用户ID
  int? getUserId() {
    return getInt(StorageKeys.userId);
  }

  /// 保存用户信息(JSON字符串)
  Future<bool> setUserInfo(String userInfo) async {
    return await setString(StorageKeys.userInfo, userInfo);
  }

  /// 获取用户信息(JSON字符串)
  String? getUserInfo() {
    return getString(StorageKeys.userInfo);
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
