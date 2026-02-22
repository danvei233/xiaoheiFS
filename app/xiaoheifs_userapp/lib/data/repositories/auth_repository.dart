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
  Future<Map<String, dynamic>> getAuthSettings() async {
    try {
      return await _authApi.getAuthSettingsRaw();
    } catch (e) {
      throw _handleError(e);
    }
  }

  /// 获取验证码
  Future<Map<String, dynamic>> getCaptcha() async {
    try {
      return await _authApi.getCaptcha();
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
    String? lotNumber,
    String? captchaOutput,
    String? passToken,
    String? genTime,
  }) async {
    try {
      final response = await _authApi.login({
        'username': username,
        'password': password,
        if ((captchaId ?? '').isNotEmpty) 'captcha_id': captchaId,
        if ((captchaCode ?? '').isNotEmpty) 'captcha_code': captchaCode,
        if ((lotNumber ?? '').isNotEmpty) 'lot_number': lotNumber,
        if ((captchaOutput ?? '').isNotEmpty) 'captcha_output': captchaOutput,
        if ((passToken ?? '').isNotEmpty) 'pass_token': passToken,
        if ((genTime ?? '').isNotEmpty) 'gen_time': genTime,
      });

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

  Future<void> requestRegisterCode(Map<String, dynamic> payload) async {
    try {
      await _authApi.requestRegisterCode(payload);
    } catch (e) {
      throw _handleError(e);
    }
  }

  Future<void> register(Map<String, dynamic> payload) async {
    try {
      await _authApi.register(payload);
    } catch (e) {
      throw _handleError(e);
    }
  }

  Future<Map<String, dynamic>> getPasswordResetOptions(String account) async {
    try {
      return await _authApi.getPasswordResetOptions(account);
    } catch (e) {
      throw _handleError(e);
    }
  }

  Future<void> sendPasswordResetCode(Map<String, dynamic> payload) async {
    try {
      await _authApi.sendPasswordResetCode(payload);
    } catch (e) {
      throw _handleError(e);
    }
  }

  Future<Map<String, dynamic>> verifyPasswordResetCode(
    Map<String, dynamic> payload,
  ) async {
    try {
      return await _authApi.verifyPasswordResetCode(payload);
    } catch (e) {
      throw _handleError(e);
    }
  }

  Future<void> confirmPasswordReset(Map<String, dynamic> payload) async {
    try {
      await _authApi.confirmPasswordReset(payload);
    } catch (e) {
      throw _handleError(e);
    }
  }

  /// 登出
  Future<void> logout() async {
    try {
      await _authApi.logout();
    } catch (_) {
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

  Future<Map<String, dynamic>> getMyUserTier() async {
    try {
      return await _authApi.getMyUserTier();
    } catch (e) {
      throw _handleError(e);
    }
  }

  Future<void> changeMyPassword(Map<String, dynamic> payload) async {
    try {
      await _authApi.changeMyPassword(payload);
    } catch (e) {
      throw _handleError(e);
    }
  }

  Future<Map<String, dynamic>> getMySecurityContacts() async {
    try {
      return await _authApi.getMySecurityContacts();
    } catch (e) {
      throw _handleError(e);
    }
  }

  Future<Map<String, dynamic>> verifyMyEmailBind2FA(
    Map<String, dynamic> payload,
  ) async {
    try {
      return await _authApi.verifyMyEmailBind2FA(payload);
    } catch (e) {
      throw _handleError(e);
    }
  }

  Future<void> sendMyEmailBindCode(Map<String, dynamic> payload) async {
    try {
      await _authApi.sendMyEmailBindCode(payload);
    } catch (e) {
      throw _handleError(e);
    }
  }

  Future<void> confirmMyEmailBind(Map<String, dynamic> payload) async {
    try {
      await _authApi.confirmMyEmailBind(payload);
    } catch (e) {
      throw _handleError(e);
    }
  }

  Future<Map<String, dynamic>> verifyMyPhoneBind2FA(
    Map<String, dynamic> payload,
  ) async {
    try {
      return await _authApi.verifyMyPhoneBind2FA(payload);
    } catch (e) {
      throw _handleError(e);
    }
  }

  Future<void> sendMyPhoneBindCode(Map<String, dynamic> payload) async {
    try {
      await _authApi.sendMyPhoneBindCode(payload);
    } catch (e) {
      throw _handleError(e);
    }
  }

  Future<void> confirmMyPhoneBind(Map<String, dynamic> payload) async {
    try {
      await _authApi.confirmMyPhoneBind(payload);
    } catch (e) {
      throw _handleError(e);
    }
  }

  Future<Map<String, dynamic>> getTwoFAStatus() async {
    try {
      return await _authApi.getTwoFAStatus();
    } catch (e) {
      throw _handleError(e);
    }
  }

  Future<Map<String, dynamic>> setupTwoFA(Map<String, dynamic> payload) async {
    try {
      return await _authApi.setupTwoFA(payload);
    } catch (e) {
      throw _handleError(e);
    }
  }

  Future<void> confirmTwoFA(Map<String, dynamic> payload) async {
    try {
      await _authApi.confirmTwoFA(payload);
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
      final data = error.response?.data;
      String? message;
      if (data is Map<String, dynamic>) {
        message = data['error']?.toString() ?? data['message']?.toString();
      } else if (data is Map) {
        message = data['error']?.toString() ?? data['message']?.toString();
      }
      return Exception(message ?? error.message ?? '请求失败');
    }
    if (error is Exception) {
      return error;
    }
    return Exception(error.toString());
  }
}
