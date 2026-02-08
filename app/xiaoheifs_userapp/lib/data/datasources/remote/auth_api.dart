import 'package:dio/dio.dart';
import '../../../core/network/api_client.dart';
import '../../../core/constants/api_endpoints.dart';
import '../../models/user.dart';

/// 认证API服务
class AuthApi {
  final Dio _dio = ApiClient.instance.dio;

  /// 获取认证设置
  Future<AuthSettings> getAuthSettings() async {
    final response = await _dio.get(ApiEndpoints.authSettings);
    return AuthSettings.fromJson(response.data);
  }

  /// 用户登录
  Future<LoginResponse> login(LoginRequest request) async {
    final response = await _dio.post(
      ApiEndpoints.authLogin,
      data: request.toJson(),
    );
    return LoginResponse.fromJson(response.data);
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
    return LoginResponse.fromJson(response.data);
  }

  /// 获取当前用户信息
  Future<User> getMe() async {
    final response = await _dio.get(ApiEndpoints.me);
    return User.fromJson(response.data);
  }

  /// 更新用户信息
  Future<User> updateMe(Map<String, dynamic> data) async {
    final response = await _dio.patch(
      ApiEndpoints.me,
      data: data,
    );
    return User.fromJson(response.data);
  }
}
