// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint
// ignore_for_file: unused_element, deprecated_member_use, deprecated_member_use_from_same_package, use_function_type_syntax_for_parameters, unnecessary_const, avoid_init_to_null, invalid_override_different_default_values_named, prefer_expression_function_bodies, annotate_overrides, invalid_annotation_target, unnecessary_question_mark

part of 'user.dart';

// **************************************************************************
// FreezedGenerator
// **************************************************************************

T _$identity<T>(T value) => value;

final _privateConstructorUsedError = UnsupportedError(
    'It seems like you constructed your class using `MyClass._()`. This constructor is only meant to be used by freezed and you are not supposed to need it nor use it.\nPlease check the documentation here for more information: https://github.com/rrousselGit/freezed#adding-getters-and-methods-to-our-models');

User _$UserFromJson(Map<String, dynamic> json) {
  return _User.fromJson(json);
}

/// @nodoc
mixin _$User {
  int? get id => throw _privateConstructorUsedError;
  @JsonKey(name: 'username')
  String? get username => throw _privateConstructorUsedError;
  String? get email => throw _privateConstructorUsedError;
  String? get qq => throw _privateConstructorUsedError;
  String? get phone => throw _privateConstructorUsedError;
  String? get avatar => throw _privateConstructorUsedError;
  @JsonKey(name: 'avatar_url')
  String? get avatarUrl => throw _privateConstructorUsedError;
  String? get bio => throw _privateConstructorUsedError;
  String? get role => throw _privateConstructorUsedError;
  String? get status => throw _privateConstructorUsedError;
  double? get balance => throw _privateConstructorUsedError;
  String? get currency => throw _privateConstructorUsedError;
  @JsonKey(name: 'created_at')
  String? get createdAt => throw _privateConstructorUsedError;
  @JsonKey(name: 'updated_at')
  String? get updatedAt => throw _privateConstructorUsedError;

  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;
  @JsonKey(ignore: true)
  $UserCopyWith<User> get copyWith => throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $UserCopyWith<$Res> {
  factory $UserCopyWith(User value, $Res Function(User) then) =
      _$UserCopyWithImpl<$Res, User>;
  @useResult
  $Res call(
      {int? id,
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
      @JsonKey(name: 'updated_at') String? updatedAt});
}

/// @nodoc
class _$UserCopyWithImpl<$Res, $Val extends User>
    implements $UserCopyWith<$Res> {
  _$UserCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? id = freezed,
    Object? username = freezed,
    Object? email = freezed,
    Object? qq = freezed,
    Object? phone = freezed,
    Object? avatar = freezed,
    Object? avatarUrl = freezed,
    Object? bio = freezed,
    Object? role = freezed,
    Object? status = freezed,
    Object? balance = freezed,
    Object? currency = freezed,
    Object? createdAt = freezed,
    Object? updatedAt = freezed,
  }) {
    return _then(_value.copyWith(
      id: freezed == id
          ? _value.id
          : id // ignore: cast_nullable_to_non_nullable
              as int?,
      username: freezed == username
          ? _value.username
          : username // ignore: cast_nullable_to_non_nullable
              as String?,
      email: freezed == email
          ? _value.email
          : email // ignore: cast_nullable_to_non_nullable
              as String?,
      qq: freezed == qq
          ? _value.qq
          : qq // ignore: cast_nullable_to_non_nullable
              as String?,
      phone: freezed == phone
          ? _value.phone
          : phone // ignore: cast_nullable_to_non_nullable
              as String?,
      avatar: freezed == avatar
          ? _value.avatar
          : avatar // ignore: cast_nullable_to_non_nullable
              as String?,
      avatarUrl: freezed == avatarUrl
          ? _value.avatarUrl
          : avatarUrl // ignore: cast_nullable_to_non_nullable
              as String?,
      bio: freezed == bio
          ? _value.bio
          : bio // ignore: cast_nullable_to_non_nullable
              as String?,
      role: freezed == role
          ? _value.role
          : role // ignore: cast_nullable_to_non_nullable
              as String?,
      status: freezed == status
          ? _value.status
          : status // ignore: cast_nullable_to_non_nullable
              as String?,
      balance: freezed == balance
          ? _value.balance
          : balance // ignore: cast_nullable_to_non_nullable
              as double?,
      currency: freezed == currency
          ? _value.currency
          : currency // ignore: cast_nullable_to_non_nullable
              as String?,
      createdAt: freezed == createdAt
          ? _value.createdAt
          : createdAt // ignore: cast_nullable_to_non_nullable
              as String?,
      updatedAt: freezed == updatedAt
          ? _value.updatedAt
          : updatedAt // ignore: cast_nullable_to_non_nullable
              as String?,
    ) as $Val);
  }
}

/// @nodoc
abstract class _$$UserImplCopyWith<$Res> implements $UserCopyWith<$Res> {
  factory _$$UserImplCopyWith(
          _$UserImpl value, $Res Function(_$UserImpl) then) =
      __$$UserImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call(
      {int? id,
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
      @JsonKey(name: 'updated_at') String? updatedAt});
}

/// @nodoc
class __$$UserImplCopyWithImpl<$Res>
    extends _$UserCopyWithImpl<$Res, _$UserImpl>
    implements _$$UserImplCopyWith<$Res> {
  __$$UserImplCopyWithImpl(_$UserImpl _value, $Res Function(_$UserImpl) _then)
      : super(_value, _then);

  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? id = freezed,
    Object? username = freezed,
    Object? email = freezed,
    Object? qq = freezed,
    Object? phone = freezed,
    Object? avatar = freezed,
    Object? avatarUrl = freezed,
    Object? bio = freezed,
    Object? role = freezed,
    Object? status = freezed,
    Object? balance = freezed,
    Object? currency = freezed,
    Object? createdAt = freezed,
    Object? updatedAt = freezed,
  }) {
    return _then(_$UserImpl(
      id: freezed == id
          ? _value.id
          : id // ignore: cast_nullable_to_non_nullable
              as int?,
      username: freezed == username
          ? _value.username
          : username // ignore: cast_nullable_to_non_nullable
              as String?,
      email: freezed == email
          ? _value.email
          : email // ignore: cast_nullable_to_non_nullable
              as String?,
      qq: freezed == qq
          ? _value.qq
          : qq // ignore: cast_nullable_to_non_nullable
              as String?,
      phone: freezed == phone
          ? _value.phone
          : phone // ignore: cast_nullable_to_non_nullable
              as String?,
      avatar: freezed == avatar
          ? _value.avatar
          : avatar // ignore: cast_nullable_to_non_nullable
              as String?,
      avatarUrl: freezed == avatarUrl
          ? _value.avatarUrl
          : avatarUrl // ignore: cast_nullable_to_non_nullable
              as String?,
      bio: freezed == bio
          ? _value.bio
          : bio // ignore: cast_nullable_to_non_nullable
              as String?,
      role: freezed == role
          ? _value.role
          : role // ignore: cast_nullable_to_non_nullable
              as String?,
      status: freezed == status
          ? _value.status
          : status // ignore: cast_nullable_to_non_nullable
              as String?,
      balance: freezed == balance
          ? _value.balance
          : balance // ignore: cast_nullable_to_non_nullable
              as double?,
      currency: freezed == currency
          ? _value.currency
          : currency // ignore: cast_nullable_to_non_nullable
              as String?,
      createdAt: freezed == createdAt
          ? _value.createdAt
          : createdAt // ignore: cast_nullable_to_non_nullable
              as String?,
      updatedAt: freezed == updatedAt
          ? _value.updatedAt
          : updatedAt // ignore: cast_nullable_to_non_nullable
              as String?,
    ));
  }
}

/// @nodoc
@JsonSerializable()
class _$UserImpl implements _User {
  const _$UserImpl(
      {this.id,
      @JsonKey(name: 'username') this.username,
      this.email,
      this.qq,
      this.phone,
      this.avatar,
      @JsonKey(name: 'avatar_url') this.avatarUrl,
      this.bio,
      this.role,
      this.status,
      this.balance,
      this.currency,
      @JsonKey(name: 'created_at') this.createdAt,
      @JsonKey(name: 'updated_at') this.updatedAt});

  factory _$UserImpl.fromJson(Map<String, dynamic> json) =>
      _$$UserImplFromJson(json);

  @override
  final int? id;
  @override
  @JsonKey(name: 'username')
  final String? username;
  @override
  final String? email;
  @override
  final String? qq;
  @override
  final String? phone;
  @override
  final String? avatar;
  @override
  @JsonKey(name: 'avatar_url')
  final String? avatarUrl;
  @override
  final String? bio;
  @override
  final String? role;
  @override
  final String? status;
  @override
  final double? balance;
  @override
  final String? currency;
  @override
  @JsonKey(name: 'created_at')
  final String? createdAt;
  @override
  @JsonKey(name: 'updated_at')
  final String? updatedAt;

  @override
  String toString() {
    return 'User(id: $id, username: $username, email: $email, qq: $qq, phone: $phone, avatar: $avatar, avatarUrl: $avatarUrl, bio: $bio, role: $role, status: $status, balance: $balance, currency: $currency, createdAt: $createdAt, updatedAt: $updatedAt)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$UserImpl &&
            (identical(other.id, id) || other.id == id) &&
            (identical(other.username, username) ||
                other.username == username) &&
            (identical(other.email, email) || other.email == email) &&
            (identical(other.qq, qq) || other.qq == qq) &&
            (identical(other.phone, phone) || other.phone == phone) &&
            (identical(other.avatar, avatar) || other.avatar == avatar) &&
            (identical(other.avatarUrl, avatarUrl) ||
                other.avatarUrl == avatarUrl) &&
            (identical(other.bio, bio) || other.bio == bio) &&
            (identical(other.role, role) || other.role == role) &&
            (identical(other.status, status) || other.status == status) &&
            (identical(other.balance, balance) || other.balance == balance) &&
            (identical(other.currency, currency) ||
                other.currency == currency) &&
            (identical(other.createdAt, createdAt) ||
                other.createdAt == createdAt) &&
            (identical(other.updatedAt, updatedAt) ||
                other.updatedAt == updatedAt));
  }

  @JsonKey(ignore: true)
  @override
  int get hashCode => Object.hash(
      runtimeType,
      id,
      username,
      email,
      qq,
      phone,
      avatar,
      avatarUrl,
      bio,
      role,
      status,
      balance,
      currency,
      createdAt,
      updatedAt);

  @JsonKey(ignore: true)
  @override
  @pragma('vm:prefer-inline')
  _$$UserImplCopyWith<_$UserImpl> get copyWith =>
      __$$UserImplCopyWithImpl<_$UserImpl>(this, _$identity);

  @override
  Map<String, dynamic> toJson() {
    return _$$UserImplToJson(
      this,
    );
  }
}

abstract class _User implements User {
  const factory _User(
      {final int? id,
      @JsonKey(name: 'username') final String? username,
      final String? email,
      final String? qq,
      final String? phone,
      final String? avatar,
      @JsonKey(name: 'avatar_url') final String? avatarUrl,
      final String? bio,
      final String? role,
      final String? status,
      final double? balance,
      final String? currency,
      @JsonKey(name: 'created_at') final String? createdAt,
      @JsonKey(name: 'updated_at') final String? updatedAt}) = _$UserImpl;

  factory _User.fromJson(Map<String, dynamic> json) = _$UserImpl.fromJson;

  @override
  int? get id;
  @override
  @JsonKey(name: 'username')
  String? get username;
  @override
  String? get email;
  @override
  String? get qq;
  @override
  String? get phone;
  @override
  String? get avatar;
  @override
  @JsonKey(name: 'avatar_url')
  String? get avatarUrl;
  @override
  String? get bio;
  @override
  String? get role;
  @override
  String? get status;
  @override
  double? get balance;
  @override
  String? get currency;
  @override
  @JsonKey(name: 'created_at')
  String? get createdAt;
  @override
  @JsonKey(name: 'updated_at')
  String? get updatedAt;
  @override
  @JsonKey(ignore: true)
  _$$UserImplCopyWith<_$UserImpl> get copyWith =>
      throw _privateConstructorUsedError;
}

LoginRequest _$LoginRequestFromJson(Map<String, dynamic> json) {
  return _LoginRequest.fromJson(json);
}

/// @nodoc
mixin _$LoginRequest {
  String get username => throw _privateConstructorUsedError;
  String get password => throw _privateConstructorUsedError;
  @JsonKey(name: 'captcha_id')
  String? get captchaId => throw _privateConstructorUsedError;
  @JsonKey(name: 'captcha_code')
  String? get captchaCode => throw _privateConstructorUsedError;

  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;
  @JsonKey(ignore: true)
  $LoginRequestCopyWith<LoginRequest> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $LoginRequestCopyWith<$Res> {
  factory $LoginRequestCopyWith(
          LoginRequest value, $Res Function(LoginRequest) then) =
      _$LoginRequestCopyWithImpl<$Res, LoginRequest>;
  @useResult
  $Res call(
      {String username,
      String password,
      @JsonKey(name: 'captcha_id') String? captchaId,
      @JsonKey(name: 'captcha_code') String? captchaCode});
}

/// @nodoc
class _$LoginRequestCopyWithImpl<$Res, $Val extends LoginRequest>
    implements $LoginRequestCopyWith<$Res> {
  _$LoginRequestCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? username = null,
    Object? password = null,
    Object? captchaId = freezed,
    Object? captchaCode = freezed,
  }) {
    return _then(_value.copyWith(
      username: null == username
          ? _value.username
          : username // ignore: cast_nullable_to_non_nullable
              as String,
      password: null == password
          ? _value.password
          : password // ignore: cast_nullable_to_non_nullable
              as String,
      captchaId: freezed == captchaId
          ? _value.captchaId
          : captchaId // ignore: cast_nullable_to_non_nullable
              as String?,
      captchaCode: freezed == captchaCode
          ? _value.captchaCode
          : captchaCode // ignore: cast_nullable_to_non_nullable
              as String?,
    ) as $Val);
  }
}

/// @nodoc
abstract class _$$LoginRequestImplCopyWith<$Res>
    implements $LoginRequestCopyWith<$Res> {
  factory _$$LoginRequestImplCopyWith(
          _$LoginRequestImpl value, $Res Function(_$LoginRequestImpl) then) =
      __$$LoginRequestImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call(
      {String username,
      String password,
      @JsonKey(name: 'captcha_id') String? captchaId,
      @JsonKey(name: 'captcha_code') String? captchaCode});
}

/// @nodoc
class __$$LoginRequestImplCopyWithImpl<$Res>
    extends _$LoginRequestCopyWithImpl<$Res, _$LoginRequestImpl>
    implements _$$LoginRequestImplCopyWith<$Res> {
  __$$LoginRequestImplCopyWithImpl(
      _$LoginRequestImpl _value, $Res Function(_$LoginRequestImpl) _then)
      : super(_value, _then);

  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? username = null,
    Object? password = null,
    Object? captchaId = freezed,
    Object? captchaCode = freezed,
  }) {
    return _then(_$LoginRequestImpl(
      username: null == username
          ? _value.username
          : username // ignore: cast_nullable_to_non_nullable
              as String,
      password: null == password
          ? _value.password
          : password // ignore: cast_nullable_to_non_nullable
              as String,
      captchaId: freezed == captchaId
          ? _value.captchaId
          : captchaId // ignore: cast_nullable_to_non_nullable
              as String?,
      captchaCode: freezed == captchaCode
          ? _value.captchaCode
          : captchaCode // ignore: cast_nullable_to_non_nullable
              as String?,
    ));
  }
}

/// @nodoc
@JsonSerializable()
class _$LoginRequestImpl implements _LoginRequest {
  const _$LoginRequestImpl(
      {required this.username,
      required this.password,
      @JsonKey(name: 'captcha_id') this.captchaId,
      @JsonKey(name: 'captcha_code') this.captchaCode});

  factory _$LoginRequestImpl.fromJson(Map<String, dynamic> json) =>
      _$$LoginRequestImplFromJson(json);

  @override
  final String username;
  @override
  final String password;
  @override
  @JsonKey(name: 'captcha_id')
  final String? captchaId;
  @override
  @JsonKey(name: 'captcha_code')
  final String? captchaCode;

  @override
  String toString() {
    return 'LoginRequest(username: $username, password: $password, captchaId: $captchaId, captchaCode: $captchaCode)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$LoginRequestImpl &&
            (identical(other.username, username) ||
                other.username == username) &&
            (identical(other.password, password) ||
                other.password == password) &&
            (identical(other.captchaId, captchaId) ||
                other.captchaId == captchaId) &&
            (identical(other.captchaCode, captchaCode) ||
                other.captchaCode == captchaCode));
  }

  @JsonKey(ignore: true)
  @override
  int get hashCode =>
      Object.hash(runtimeType, username, password, captchaId, captchaCode);

  @JsonKey(ignore: true)
  @override
  @pragma('vm:prefer-inline')
  _$$LoginRequestImplCopyWith<_$LoginRequestImpl> get copyWith =>
      __$$LoginRequestImplCopyWithImpl<_$LoginRequestImpl>(this, _$identity);

  @override
  Map<String, dynamic> toJson() {
    return _$$LoginRequestImplToJson(
      this,
    );
  }
}

abstract class _LoginRequest implements LoginRequest {
  const factory _LoginRequest(
          {required final String username,
          required final String password,
          @JsonKey(name: 'captcha_id') final String? captchaId,
          @JsonKey(name: 'captcha_code') final String? captchaCode}) =
      _$LoginRequestImpl;

  factory _LoginRequest.fromJson(Map<String, dynamic> json) =
      _$LoginRequestImpl.fromJson;

  @override
  String get username;
  @override
  String get password;
  @override
  @JsonKey(name: 'captcha_id')
  String? get captchaId;
  @override
  @JsonKey(name: 'captcha_code')
  String? get captchaCode;
  @override
  @JsonKey(ignore: true)
  _$$LoginRequestImplCopyWith<_$LoginRequestImpl> get copyWith =>
      throw _privateConstructorUsedError;
}

LoginResponse _$LoginResponseFromJson(Map<String, dynamic> json) {
  return _LoginResponse.fromJson(json);
}

/// @nodoc
mixin _$LoginResponse {
  @JsonKey(name: 'access_token')
  String? get accessToken => throw _privateConstructorUsedError;
  @JsonKey(name: 'expires_in')
  int? get expiresIn => throw _privateConstructorUsedError;
  @JsonKey(name: 'refresh_token')
  String? get refreshToken => throw _privateConstructorUsedError;
  User? get user => throw _privateConstructorUsedError;

  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;
  @JsonKey(ignore: true)
  $LoginResponseCopyWith<LoginResponse> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $LoginResponseCopyWith<$Res> {
  factory $LoginResponseCopyWith(
          LoginResponse value, $Res Function(LoginResponse) then) =
      _$LoginResponseCopyWithImpl<$Res, LoginResponse>;
  @useResult
  $Res call(
      {@JsonKey(name: 'access_token') String? accessToken,
      @JsonKey(name: 'expires_in') int? expiresIn,
      @JsonKey(name: 'refresh_token') String? refreshToken,
      User? user});

  $UserCopyWith<$Res>? get user;
}

/// @nodoc
class _$LoginResponseCopyWithImpl<$Res, $Val extends LoginResponse>
    implements $LoginResponseCopyWith<$Res> {
  _$LoginResponseCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? accessToken = freezed,
    Object? expiresIn = freezed,
    Object? refreshToken = freezed,
    Object? user = freezed,
  }) {
    return _then(_value.copyWith(
      accessToken: freezed == accessToken
          ? _value.accessToken
          : accessToken // ignore: cast_nullable_to_non_nullable
              as String?,
      expiresIn: freezed == expiresIn
          ? _value.expiresIn
          : expiresIn // ignore: cast_nullable_to_non_nullable
              as int?,
      refreshToken: freezed == refreshToken
          ? _value.refreshToken
          : refreshToken // ignore: cast_nullable_to_non_nullable
              as String?,
      user: freezed == user
          ? _value.user
          : user // ignore: cast_nullable_to_non_nullable
              as User?,
    ) as $Val);
  }

  @override
  @pragma('vm:prefer-inline')
  $UserCopyWith<$Res>? get user {
    if (_value.user == null) {
      return null;
    }

    return $UserCopyWith<$Res>(_value.user!, (value) {
      return _then(_value.copyWith(user: value) as $Val);
    });
  }
}

/// @nodoc
abstract class _$$LoginResponseImplCopyWith<$Res>
    implements $LoginResponseCopyWith<$Res> {
  factory _$$LoginResponseImplCopyWith(
          _$LoginResponseImpl value, $Res Function(_$LoginResponseImpl) then) =
      __$$LoginResponseImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call(
      {@JsonKey(name: 'access_token') String? accessToken,
      @JsonKey(name: 'expires_in') int? expiresIn,
      @JsonKey(name: 'refresh_token') String? refreshToken,
      User? user});

  @override
  $UserCopyWith<$Res>? get user;
}

/// @nodoc
class __$$LoginResponseImplCopyWithImpl<$Res>
    extends _$LoginResponseCopyWithImpl<$Res, _$LoginResponseImpl>
    implements _$$LoginResponseImplCopyWith<$Res> {
  __$$LoginResponseImplCopyWithImpl(
      _$LoginResponseImpl _value, $Res Function(_$LoginResponseImpl) _then)
      : super(_value, _then);

  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? accessToken = freezed,
    Object? expiresIn = freezed,
    Object? refreshToken = freezed,
    Object? user = freezed,
  }) {
    return _then(_$LoginResponseImpl(
      accessToken: freezed == accessToken
          ? _value.accessToken
          : accessToken // ignore: cast_nullable_to_non_nullable
              as String?,
      expiresIn: freezed == expiresIn
          ? _value.expiresIn
          : expiresIn // ignore: cast_nullable_to_non_nullable
              as int?,
      refreshToken: freezed == refreshToken
          ? _value.refreshToken
          : refreshToken // ignore: cast_nullable_to_non_nullable
              as String?,
      user: freezed == user
          ? _value.user
          : user // ignore: cast_nullable_to_non_nullable
              as User?,
    ));
  }
}

/// @nodoc
@JsonSerializable()
class _$LoginResponseImpl implements _LoginResponse {
  const _$LoginResponseImpl(
      {@JsonKey(name: 'access_token') this.accessToken,
      @JsonKey(name: 'expires_in') this.expiresIn,
      @JsonKey(name: 'refresh_token') this.refreshToken,
      this.user});

  factory _$LoginResponseImpl.fromJson(Map<String, dynamic> json) =>
      _$$LoginResponseImplFromJson(json);

  @override
  @JsonKey(name: 'access_token')
  final String? accessToken;
  @override
  @JsonKey(name: 'expires_in')
  final int? expiresIn;
  @override
  @JsonKey(name: 'refresh_token')
  final String? refreshToken;
  @override
  final User? user;

  @override
  String toString() {
    return 'LoginResponse(accessToken: $accessToken, expiresIn: $expiresIn, refreshToken: $refreshToken, user: $user)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$LoginResponseImpl &&
            (identical(other.accessToken, accessToken) ||
                other.accessToken == accessToken) &&
            (identical(other.expiresIn, expiresIn) ||
                other.expiresIn == expiresIn) &&
            (identical(other.refreshToken, refreshToken) ||
                other.refreshToken == refreshToken) &&
            (identical(other.user, user) || other.user == user));
  }

  @JsonKey(ignore: true)
  @override
  int get hashCode =>
      Object.hash(runtimeType, accessToken, expiresIn, refreshToken, user);

  @JsonKey(ignore: true)
  @override
  @pragma('vm:prefer-inline')
  _$$LoginResponseImplCopyWith<_$LoginResponseImpl> get copyWith =>
      __$$LoginResponseImplCopyWithImpl<_$LoginResponseImpl>(this, _$identity);

  @override
  Map<String, dynamic> toJson() {
    return _$$LoginResponseImplToJson(
      this,
    );
  }
}

abstract class _LoginResponse implements LoginResponse {
  const factory _LoginResponse(
      {@JsonKey(name: 'access_token') final String? accessToken,
      @JsonKey(name: 'expires_in') final int? expiresIn,
      @JsonKey(name: 'refresh_token') final String? refreshToken,
      final User? user}) = _$LoginResponseImpl;

  factory _LoginResponse.fromJson(Map<String, dynamic> json) =
      _$LoginResponseImpl.fromJson;

  @override
  @JsonKey(name: 'access_token')
  String? get accessToken;
  @override
  @JsonKey(name: 'expires_in')
  int? get expiresIn;
  @override
  @JsonKey(name: 'refresh_token')
  String? get refreshToken;
  @override
  User? get user;
  @override
  @JsonKey(ignore: true)
  _$$LoginResponseImplCopyWith<_$LoginResponseImpl> get copyWith =>
      throw _privateConstructorUsedError;
}

AuthSettings _$AuthSettingsFromJson(Map<String, dynamic> json) {
  return _AuthSettings.fromJson(json);
}

/// @nodoc
mixin _$AuthSettings {
  @JsonKey(name: 'register_enabled')
  bool? get registerEnabled => throw _privateConstructorUsedError;
  @JsonKey(name: 'login_captcha_enabled')
  bool? get loginCaptchaEnabled => throw _privateConstructorUsedError;
  @JsonKey(name: 'password_min_len')
  int? get passwordMinLen => throw _privateConstructorUsedError;

  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;
  @JsonKey(ignore: true)
  $AuthSettingsCopyWith<AuthSettings> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $AuthSettingsCopyWith<$Res> {
  factory $AuthSettingsCopyWith(
          AuthSettings value, $Res Function(AuthSettings) then) =
      _$AuthSettingsCopyWithImpl<$Res, AuthSettings>;
  @useResult
  $Res call(
      {@JsonKey(name: 'register_enabled') bool? registerEnabled,
      @JsonKey(name: 'login_captcha_enabled') bool? loginCaptchaEnabled,
      @JsonKey(name: 'password_min_len') int? passwordMinLen});
}

/// @nodoc
class _$AuthSettingsCopyWithImpl<$Res, $Val extends AuthSettings>
    implements $AuthSettingsCopyWith<$Res> {
  _$AuthSettingsCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? registerEnabled = freezed,
    Object? loginCaptchaEnabled = freezed,
    Object? passwordMinLen = freezed,
  }) {
    return _then(_value.copyWith(
      registerEnabled: freezed == registerEnabled
          ? _value.registerEnabled
          : registerEnabled // ignore: cast_nullable_to_non_nullable
              as bool?,
      loginCaptchaEnabled: freezed == loginCaptchaEnabled
          ? _value.loginCaptchaEnabled
          : loginCaptchaEnabled // ignore: cast_nullable_to_non_nullable
              as bool?,
      passwordMinLen: freezed == passwordMinLen
          ? _value.passwordMinLen
          : passwordMinLen // ignore: cast_nullable_to_non_nullable
              as int?,
    ) as $Val);
  }
}

/// @nodoc
abstract class _$$AuthSettingsImplCopyWith<$Res>
    implements $AuthSettingsCopyWith<$Res> {
  factory _$$AuthSettingsImplCopyWith(
          _$AuthSettingsImpl value, $Res Function(_$AuthSettingsImpl) then) =
      __$$AuthSettingsImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call(
      {@JsonKey(name: 'register_enabled') bool? registerEnabled,
      @JsonKey(name: 'login_captcha_enabled') bool? loginCaptchaEnabled,
      @JsonKey(name: 'password_min_len') int? passwordMinLen});
}

/// @nodoc
class __$$AuthSettingsImplCopyWithImpl<$Res>
    extends _$AuthSettingsCopyWithImpl<$Res, _$AuthSettingsImpl>
    implements _$$AuthSettingsImplCopyWith<$Res> {
  __$$AuthSettingsImplCopyWithImpl(
      _$AuthSettingsImpl _value, $Res Function(_$AuthSettingsImpl) _then)
      : super(_value, _then);

  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? registerEnabled = freezed,
    Object? loginCaptchaEnabled = freezed,
    Object? passwordMinLen = freezed,
  }) {
    return _then(_$AuthSettingsImpl(
      registerEnabled: freezed == registerEnabled
          ? _value.registerEnabled
          : registerEnabled // ignore: cast_nullable_to_non_nullable
              as bool?,
      loginCaptchaEnabled: freezed == loginCaptchaEnabled
          ? _value.loginCaptchaEnabled
          : loginCaptchaEnabled // ignore: cast_nullable_to_non_nullable
              as bool?,
      passwordMinLen: freezed == passwordMinLen
          ? _value.passwordMinLen
          : passwordMinLen // ignore: cast_nullable_to_non_nullable
              as int?,
    ));
  }
}

/// @nodoc
@JsonSerializable()
class _$AuthSettingsImpl implements _AuthSettings {
  const _$AuthSettingsImpl(
      {@JsonKey(name: 'register_enabled') this.registerEnabled,
      @JsonKey(name: 'login_captcha_enabled') this.loginCaptchaEnabled,
      @JsonKey(name: 'password_min_len') this.passwordMinLen});

  factory _$AuthSettingsImpl.fromJson(Map<String, dynamic> json) =>
      _$$AuthSettingsImplFromJson(json);

  @override
  @JsonKey(name: 'register_enabled')
  final bool? registerEnabled;
  @override
  @JsonKey(name: 'login_captcha_enabled')
  final bool? loginCaptchaEnabled;
  @override
  @JsonKey(name: 'password_min_len')
  final int? passwordMinLen;

  @override
  String toString() {
    return 'AuthSettings(registerEnabled: $registerEnabled, loginCaptchaEnabled: $loginCaptchaEnabled, passwordMinLen: $passwordMinLen)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$AuthSettingsImpl &&
            (identical(other.registerEnabled, registerEnabled) ||
                other.registerEnabled == registerEnabled) &&
            (identical(other.loginCaptchaEnabled, loginCaptchaEnabled) ||
                other.loginCaptchaEnabled == loginCaptchaEnabled) &&
            (identical(other.passwordMinLen, passwordMinLen) ||
                other.passwordMinLen == passwordMinLen));
  }

  @JsonKey(ignore: true)
  @override
  int get hashCode => Object.hash(
      runtimeType, registerEnabled, loginCaptchaEnabled, passwordMinLen);

  @JsonKey(ignore: true)
  @override
  @pragma('vm:prefer-inline')
  _$$AuthSettingsImplCopyWith<_$AuthSettingsImpl> get copyWith =>
      __$$AuthSettingsImplCopyWithImpl<_$AuthSettingsImpl>(this, _$identity);

  @override
  Map<String, dynamic> toJson() {
    return _$$AuthSettingsImplToJson(
      this,
    );
  }
}

abstract class _AuthSettings implements AuthSettings {
  const factory _AuthSettings(
      {@JsonKey(name: 'register_enabled') final bool? registerEnabled,
      @JsonKey(name: 'login_captcha_enabled') final bool? loginCaptchaEnabled,
      @JsonKey(name: 'password_min_len')
      final int? passwordMinLen}) = _$AuthSettingsImpl;

  factory _AuthSettings.fromJson(Map<String, dynamic> json) =
      _$AuthSettingsImpl.fromJson;

  @override
  @JsonKey(name: 'register_enabled')
  bool? get registerEnabled;
  @override
  @JsonKey(name: 'login_captcha_enabled')
  bool? get loginCaptchaEnabled;
  @override
  @JsonKey(name: 'password_min_len')
  int? get passwordMinLen;
  @override
  @JsonKey(ignore: true)
  _$$AuthSettingsImplCopyWith<_$AuthSettingsImpl> get copyWith =>
      throw _privateConstructorUsedError;
}
