import 'package:dio/dio.dart';
import '../datasources/remote/auth_api.dart';
import '../models/user.dart';
import '../../core/storage/storage_service.dart';

/// 认证仓储
/// 负责认证相关的数据操作
class AuthRepository {
  final AuthApi _authApi = AuthApi();
  final StorageService _storage = StorageService.instance;

  /// 获取认证设置
  Future<AuthSettings> getAuthSettings() async {
    try {
      return await _authApi.getAuthSettings();
    } catch (e) {
      throw _handleError(e);
    }
  }

  /// 登录
  Future<LoginResponse> login({
    required String username,
    required String password,
    String? captchaId,
    String? captchaCode,
  }) async {
    try {
      final response = await _authApi.login(
        LoginRequest(
          username: username,
          password: password,
          captchaId: captchaId,
          captchaCode: captchaCode,
        ),
      );

      // 保存Token和用户信息
      if (response.accessToken != null) {
        await _storage.setAccessToken(response.accessToken!);
      }
      if (response.refreshToken != null) {
        await _storage.setRefreshToken(response.refreshToken!);
      }
      if (response.user?.id != null) {
        await _storage.setUserId(response.user!.id!);
      }

      return response;
    } catch (e) {
      throw _handleError(e);
    }
  }

  /// 登出
  Future<void> logout() async {
    try {
      await _authApi.logout();
    } catch (e) {
      // 即使API调用失败，也清除本地数据
    } finally {
      await _storage.clearAuthData();
    }
  }

  /// 获取当前用户信息
  Future<User> getMe() async {
    try {
      return await _authApi.getMe();
    } catch (e) {
      throw _handleError(e);
    }
  }

  /// 更新用户信息
  Future<User> updateMe(Map<String, dynamic> data) async {
    try {
      return await _authApi.updateMe(data);
    } catch (e) {
      throw _handleError(e);
    }
  }

  /// 检查是否已登录
  bool isLoggedIn() {
    return _storage.getAccessToken() != null;
  }

  /// 获取存储的Token
  String? getAccessToken() {
    return _storage.getAccessToken();
  }

  /// 错误处理
  Exception _handleError(dynamic error) {
    if (error is DioException) {
      final message = error.response?.data?['message'] ?? error.message;
      return Exception(message ?? '登录失败');
    }
    if (error is Exception) {
      return error;
    }
    return Exception(error.toString());
  }
}
