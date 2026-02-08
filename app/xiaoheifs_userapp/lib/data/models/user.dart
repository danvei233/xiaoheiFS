import 'package:freezed_annotation/freezed_annotation.dart';

part 'user.freezed.dart';
part 'user.g.dart';

/// 用户数据模型
@freezed
class User with _$User {
  const factory User({
    int? id,
    @JsonKey(name: 'username') String? username,
    String? email,
    String? qq,
    String? phone,
    String? avatar,
    @JsonKey(name: 'avatar_url') String? avatarUrl,
    String? bio,
    String? role,
    String? status,
    double? balance,
    String? currency,
    @JsonKey(name: 'created_at') String? createdAt,
    @JsonKey(name: 'updated_at') String? updatedAt,
  }) = _User;

  factory User.fromJson(Map<String, dynamic> json) => _$UserFromJson(json);
}

/// 登录请求模型
@freezed
class LoginRequest with _$LoginRequest {
  const factory LoginRequest({
    required String username,
    required String password,
    @JsonKey(name: 'captcha_id') String? captchaId,
    @JsonKey(name: 'captcha_code') String? captchaCode,
  }) = _LoginRequest;

  factory LoginRequest.fromJson(Map<String, dynamic> json) =>
      _$LoginRequestFromJson(json);
}

/// 登录响应模型
@freezed
class LoginResponse with _$LoginResponse {
  const factory LoginResponse({
    @JsonKey(name: 'access_token') String? accessToken,
    @JsonKey(name: 'expires_in') int? expiresIn,
    @JsonKey(name: 'refresh_token') String? refreshToken,
    User? user,
  }) = _LoginResponse;

  factory LoginResponse.fromJson(Map<String, dynamic> json) =>
      _$LoginResponseFromJson(json);
}

/// 认证设置模型
@freezed
class AuthSettings with _$AuthSettings {
  const factory AuthSettings({
    @JsonKey(name: 'register_enabled') bool? registerEnabled,
    @JsonKey(name: 'login_captcha_enabled') bool? loginCaptchaEnabled,
    @JsonKey(name: 'password_min_len') int? passwordMinLen,
  }) = _AuthSettings;

  factory AuthSettings.fromJson(Map<String, dynamic> json) =>
      _$AuthSettingsFromJson(json);
}
