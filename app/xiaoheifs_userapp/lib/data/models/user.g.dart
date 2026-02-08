// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'user.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

_$UserImpl _$$UserImplFromJson(Map<String, dynamic> json) => _$UserImpl(
      id: (json['id'] as num?)?.toInt(),
      username: json['username'] as String?,
      email: json['email'] as String?,
      qq: json['qq'] as String?,
      phone: json['phone'] as String?,
      avatar: json['avatar'] as String?,
      avatarUrl: json['avatar_url'] as String?,
      bio: json['bio'] as String?,
      role: json['role'] as String?,
      status: json['status'] as String?,
      balance: (json['balance'] as num?)?.toDouble(),
      currency: json['currency'] as String?,
      createdAt: json['created_at'] as String?,
      updatedAt: json['updated_at'] as String?,
    );

Map<String, dynamic> _$$UserImplToJson(_$UserImpl instance) =>
    <String, dynamic>{
      'id': instance.id,
      'username': instance.username,
      'email': instance.email,
      'qq': instance.qq,
      'phone': instance.phone,
      'avatar': instance.avatar,
      'avatar_url': instance.avatarUrl,
      'bio': instance.bio,
      'role': instance.role,
      'status': instance.status,
      'balance': instance.balance,
      'currency': instance.currency,
      'created_at': instance.createdAt,
      'updated_at': instance.updatedAt,
    };

_$LoginRequestImpl _$$LoginRequestImplFromJson(Map<String, dynamic> json) =>
    _$LoginRequestImpl(
      username: json['username'] as String,
      password: json['password'] as String,
      captchaId: json['captcha_id'] as String?,
      captchaCode: json['captcha_code'] as String?,
    );

Map<String, dynamic> _$$LoginRequestImplToJson(_$LoginRequestImpl instance) =>
    <String, dynamic>{
      'username': instance.username,
      'password': instance.password,
      'captcha_id': instance.captchaId,
      'captcha_code': instance.captchaCode,
    };

_$LoginResponseImpl _$$LoginResponseImplFromJson(Map<String, dynamic> json) =>
    _$LoginResponseImpl(
      accessToken: json['access_token'] as String?,
      expiresIn: (json['expires_in'] as num?)?.toInt(),
      refreshToken: json['refresh_token'] as String?,
      user: json['user'] == null
          ? null
          : User.fromJson(json['user'] as Map<String, dynamic>),
    );

Map<String, dynamic> _$$LoginResponseImplToJson(_$LoginResponseImpl instance) =>
    <String, dynamic>{
      'access_token': instance.accessToken,
      'expires_in': instance.expiresIn,
      'refresh_token': instance.refreshToken,
      'user': instance.user,
    };

_$AuthSettingsImpl _$$AuthSettingsImplFromJson(Map<String, dynamic> json) =>
    _$AuthSettingsImpl(
      registerEnabled: json['register_enabled'] as bool?,
      loginCaptchaEnabled: json['login_captcha_enabled'] as bool?,
      passwordMinLen: (json['password_min_len'] as num?)?.toInt(),
    );

Map<String, dynamic> _$$AuthSettingsImplToJson(_$AuthSettingsImpl instance) =>
    <String, dynamic>{
      'register_enabled': instance.registerEnabled,
      'login_captcha_enabled': instance.loginCaptchaEnabled,
      'password_min_len': instance.passwordMinLen,
    };
