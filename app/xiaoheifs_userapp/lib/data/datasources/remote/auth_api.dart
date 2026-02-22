import 'package:dio/dio.dart';
import '../../../core/network/api_client.dart';
import '../../../core/constants/api_endpoints.dart';
import '../../models/user.dart';

/// 认证API服务
class AuthApi {
  final Dio _dio = ApiClient.instance.dio;

  Map<String, dynamic> _asMap(dynamic data) {
    if (data is Map<String, dynamic>) return data;
    if (data is Map) {
      return data.map((key, value) => MapEntry(key.toString(), value));
    }
    return <String, dynamic>{};
  }

  /// 获取认证设置
  Future<Map<String, dynamic>> getAuthSettingsRaw() async {
    final response = await _dio.get(ApiEndpoints.authSettings);
    return _asMap(response.data);
  }

  /// 获取验证码（图形/极验）
  Future<Map<String, dynamic>> getCaptcha() async {
    final response = await _dio.get(ApiEndpoints.captcha);
    return _asMap(response.data);
  }

  /// 用户登录
  Future<LoginResponse> login(Map<String, dynamic> payload) async {
    final response = await _dio.post(ApiEndpoints.authLogin, data: payload);
    return LoginResponse.fromJson(_asMap(response.data));
  }

  /// 请求注册验证码
  Future<void> requestRegisterCode(Map<String, dynamic> payload) async {
    await _dio.post(ApiEndpoints.authRegisterCode, data: payload);
  }

  /// 用户注册
  Future<void> register(Map<String, dynamic> payload) async {
    await _dio.post(ApiEndpoints.authRegister, data: payload);
  }

  /// 找回密码：查询可用渠道
  Future<Map<String, dynamic>> getPasswordResetOptions(String account) async {
    final response = await _dio.post(
      ApiEndpoints.authPasswordResetOptions,
      data: {'account': account},
    );
    return _asMap(response.data);
  }

  /// 找回密码：发送验证码
  Future<void> sendPasswordResetCode(Map<String, dynamic> payload) async {
    await _dio.post(ApiEndpoints.authPasswordResetSendCode, data: payload);
  }

  /// 找回密码：校验验证码
  Future<Map<String, dynamic>> verifyPasswordResetCode(
    Map<String, dynamic> payload,
  ) async {
    final response = await _dio.post(
      ApiEndpoints.authPasswordResetVerifyCode,
      data: payload,
    );
    return _asMap(response.data);
  }

  /// 找回密码：重置密码
  Future<void> confirmPasswordReset(Map<String, dynamic> payload) async {
    await _dio.post(ApiEndpoints.authPasswordResetConfirm, data: payload);
  }

  /// 用户登出
  Future<void> logout() async {
    await _dio.post(ApiEndpoints.authLogout);
  }

  /// 刷新Token
  Future<LoginResponse> refreshToken(String refreshToken) async {
    final response = await _dio.post(
      ApiEndpoints.authRefresh,
      data: {'refresh_token': refreshToken},
    );
    return LoginResponse.fromJson(_asMap(response.data));
  }

  /// 获取当前用户信息
  Future<User> getMe() async {
    final response = await _dio.get(ApiEndpoints.me);
    return User.fromJson(_asMap(response.data));
  }

  /// 更新用户信息
  Future<User> updateMe(Map<String, dynamic> data) async {
    final response = await _dio.patch(ApiEndpoints.me, data: data);
    return User.fromJson(_asMap(response.data));
  }

  /// 获取当前用户等级/分组信息
  Future<Map<String, dynamic>> getMyUserTier() async {
    final response = await _dio.get(ApiEndpoints.meUserTier);
    return _asMap(response.data);
  }

  /// 修改密码
  Future<void> changeMyPassword(Map<String, dynamic> payload) async {
    await _dio.post(ApiEndpoints.mePasswordChange, data: payload);
  }

  /// 获取安全联系人绑定状态
  Future<Map<String, dynamic>> getMySecurityContacts() async {
    final response = await _dio.get(ApiEndpoints.meSecurityContacts);
    return _asMap(response.data);
  }

  Future<Map<String, dynamic>> verifyMyEmailBind2FA(
    Map<String, dynamic> payload,
  ) async {
    final response = await _dio.post(
      ApiEndpoints.meSecurityEmailVerify2fa,
      data: payload,
    );
    return _asMap(response.data);
  }

  Future<void> sendMyEmailBindCode(Map<String, dynamic> payload) async {
    await _dio.post(ApiEndpoints.meSecurityEmailSendCode, data: payload);
  }

  Future<void> confirmMyEmailBind(Map<String, dynamic> payload) async {
    await _dio.post(ApiEndpoints.meSecurityEmailConfirm, data: payload);
  }

  Future<Map<String, dynamic>> verifyMyPhoneBind2FA(
    Map<String, dynamic> payload,
  ) async {
    final response = await _dio.post(
      ApiEndpoints.meSecurityPhoneVerify2fa,
      data: payload,
    );
    return _asMap(response.data);
  }

  Future<void> sendMyPhoneBindCode(Map<String, dynamic> payload) async {
    await _dio.post(ApiEndpoints.meSecurityPhoneSendCode, data: payload);
  }

  Future<void> confirmMyPhoneBind(Map<String, dynamic> payload) async {
    await _dio.post(ApiEndpoints.meSecurityPhoneConfirm, data: payload);
  }

  Future<Map<String, dynamic>> getTwoFAStatus() async {
    final response = await _dio.get(ApiEndpoints.meSecurity2faStatus);
    return _asMap(response.data);
  }

  Future<Map<String, dynamic>> setupTwoFA(Map<String, dynamic> payload) async {
    final response = await _dio.post(
      ApiEndpoints.meSecurity2faSetup,
      data: payload,
    );
    return _asMap(response.data);
  }

  Future<void> confirmTwoFA(Map<String, dynamic> payload) async {
    await _dio.post(ApiEndpoints.meSecurity2faConfirm, data: payload);
  }
}
