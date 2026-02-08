// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint
// ignore_for_file: unused_element, deprecated_member_use, deprecated_member_use_from_same_package, use_function_type_syntax_for_parameters, unnecessary_const, avoid_init_to_null, invalid_override_different_default_values_named, prefer_expression_function_bodies, annotate_overrides, invalid_annotation_target, unnecessary_question_mark

part of 'realname.dart';

// **************************************************************************
// FreezedGenerator
// **************************************************************************

T _$identity<T>(T value) => value;

final _privateConstructorUsedError = UnsupportedError(
    'It seems like you constructed your class using `MyClass._()`. This constructor is only meant to be used by freezed and you are not supposed to need it nor use it.\nPlease check the documentation here for more information: https://github.com/rrousselGit/freezed#adding-getters-and-methods-to-our-models');

RealnameStatus _$RealnameStatusFromJson(Map<String, dynamic> json) {
  return _RealnameStatus.fromJson(json);
}

/// @nodoc
mixin _$RealnameStatus {
  bool? get enabled => throw _privateConstructorUsedError;
  String? get provider => throw _privateConstructorUsedError;
  @JsonKey(name: 'block_actions')
  List<String>? get blockActions => throw _privateConstructorUsedError;
  bool? get verified => throw _privateConstructorUsedError;
  RealnameVerification? get verification => throw _privateConstructorUsedError;

  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;
  @JsonKey(ignore: true)
  $RealnameStatusCopyWith<RealnameStatus> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $RealnameStatusCopyWith<$Res> {
  factory $RealnameStatusCopyWith(
          RealnameStatus value, $Res Function(RealnameStatus) then) =
      _$RealnameStatusCopyWithImpl<$Res, RealnameStatus>;
  @useResult
  $Res call(
      {bool? enabled,
      String? provider,
      @JsonKey(name: 'block_actions') List<String>? blockActions,
      bool? verified,
      RealnameVerification? verification});

  $RealnameVerificationCopyWith<$Res>? get verification;
}

/// @nodoc
class _$RealnameStatusCopyWithImpl<$Res, $Val extends RealnameStatus>
    implements $RealnameStatusCopyWith<$Res> {
  _$RealnameStatusCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? enabled = freezed,
    Object? provider = freezed,
    Object? blockActions = freezed,
    Object? verified = freezed,
    Object? verification = freezed,
  }) {
    return _then(_value.copyWith(
      enabled: freezed == enabled
          ? _value.enabled
          : enabled // ignore: cast_nullable_to_non_nullable
              as bool?,
      provider: freezed == provider
          ? _value.provider
          : provider // ignore: cast_nullable_to_non_nullable
              as String?,
      blockActions: freezed == blockActions
          ? _value.blockActions
          : blockActions // ignore: cast_nullable_to_non_nullable
              as List<String>?,
      verified: freezed == verified
          ? _value.verified
          : verified // ignore: cast_nullable_to_non_nullable
              as bool?,
      verification: freezed == verification
          ? _value.verification
          : verification // ignore: cast_nullable_to_non_nullable
              as RealnameVerification?,
    ) as $Val);
  }

  @override
  @pragma('vm:prefer-inline')
  $RealnameVerificationCopyWith<$Res>? get verification {
    if (_value.verification == null) {
      return null;
    }

    return $RealnameVerificationCopyWith<$Res>(_value.verification!, (value) {
      return _then(_value.copyWith(verification: value) as $Val);
    });
  }
}

/// @nodoc
abstract class _$$RealnameStatusImplCopyWith<$Res>
    implements $RealnameStatusCopyWith<$Res> {
  factory _$$RealnameStatusImplCopyWith(_$RealnameStatusImpl value,
          $Res Function(_$RealnameStatusImpl) then) =
      __$$RealnameStatusImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call(
      {bool? enabled,
      String? provider,
      @JsonKey(name: 'block_actions') List<String>? blockActions,
      bool? verified,
      RealnameVerification? verification});

  @override
  $RealnameVerificationCopyWith<$Res>? get verification;
}

/// @nodoc
class __$$RealnameStatusImplCopyWithImpl<$Res>
    extends _$RealnameStatusCopyWithImpl<$Res, _$RealnameStatusImpl>
    implements _$$RealnameStatusImplCopyWith<$Res> {
  __$$RealnameStatusImplCopyWithImpl(
      _$RealnameStatusImpl _value, $Res Function(_$RealnameStatusImpl) _then)
      : super(_value, _then);

  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? enabled = freezed,
    Object? provider = freezed,
    Object? blockActions = freezed,
    Object? verified = freezed,
    Object? verification = freezed,
  }) {
    return _then(_$RealnameStatusImpl(
      enabled: freezed == enabled
          ? _value.enabled
          : enabled // ignore: cast_nullable_to_non_nullable
              as bool?,
      provider: freezed == provider
          ? _value.provider
          : provider // ignore: cast_nullable_to_non_nullable
              as String?,
      blockActions: freezed == blockActions
          ? _value._blockActions
          : blockActions // ignore: cast_nullable_to_non_nullable
              as List<String>?,
      verified: freezed == verified
          ? _value.verified
          : verified // ignore: cast_nullable_to_non_nullable
              as bool?,
      verification: freezed == verification
          ? _value.verification
          : verification // ignore: cast_nullable_to_non_nullable
              as RealnameVerification?,
    ));
  }
}

/// @nodoc
@JsonSerializable()
class _$RealnameStatusImpl implements _RealnameStatus {
  const _$RealnameStatusImpl(
      {this.enabled,
      this.provider,
      @JsonKey(name: 'block_actions') final List<String>? blockActions,
      this.verified,
      this.verification})
      : _blockActions = blockActions;

  factory _$RealnameStatusImpl.fromJson(Map<String, dynamic> json) =>
      _$$RealnameStatusImplFromJson(json);

  @override
  final bool? enabled;
  @override
  final String? provider;
  final List<String>? _blockActions;
  @override
  @JsonKey(name: 'block_actions')
  List<String>? get blockActions {
    final value = _blockActions;
    if (value == null) return null;
    if (_blockActions is EqualUnmodifiableListView) return _blockActions;
    // ignore: implicit_dynamic_type
    return EqualUnmodifiableListView(value);
  }

  @override
  final bool? verified;
  @override
  final RealnameVerification? verification;

  @override
  String toString() {
    return 'RealnameStatus(enabled: $enabled, provider: $provider, blockActions: $blockActions, verified: $verified, verification: $verification)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$RealnameStatusImpl &&
            (identical(other.enabled, enabled) || other.enabled == enabled) &&
            (identical(other.provider, provider) ||
                other.provider == provider) &&
            const DeepCollectionEquality()
                .equals(other._blockActions, _blockActions) &&
            (identical(other.verified, verified) ||
                other.verified == verified) &&
            (identical(other.verification, verification) ||
                other.verification == verification));
  }

  @JsonKey(ignore: true)
  @override
  int get hashCode => Object.hash(
      runtimeType,
      enabled,
      provider,
      const DeepCollectionEquality().hash(_blockActions),
      verified,
      verification);

  @JsonKey(ignore: true)
  @override
  @pragma('vm:prefer-inline')
  _$$RealnameStatusImplCopyWith<_$RealnameStatusImpl> get copyWith =>
      __$$RealnameStatusImplCopyWithImpl<_$RealnameStatusImpl>(
          this, _$identity);

  @override
  Map<String, dynamic> toJson() {
    return _$$RealnameStatusImplToJson(
      this,
    );
  }
}

abstract class _RealnameStatus implements RealnameStatus {
  const factory _RealnameStatus(
      {final bool? enabled,
      final String? provider,
      @JsonKey(name: 'block_actions') final List<String>? blockActions,
      final bool? verified,
      final RealnameVerification? verification}) = _$RealnameStatusImpl;

  factory _RealnameStatus.fromJson(Map<String, dynamic> json) =
      _$RealnameStatusImpl.fromJson;

  @override
  bool? get enabled;
  @override
  String? get provider;
  @override
  @JsonKey(name: 'block_actions')
  List<String>? get blockActions;
  @override
  bool? get verified;
  @override
  RealnameVerification? get verification;
  @override
  @JsonKey(ignore: true)
  _$$RealnameStatusImplCopyWith<_$RealnameStatusImpl> get copyWith =>
      throw _privateConstructorUsedError;
}

RealnameVerification _$RealnameVerificationFromJson(Map<String, dynamic> json) {
  return _RealnameVerification.fromJson(json);
}

/// @nodoc
mixin _$RealnameVerification {
  int? get id => throw _privateConstructorUsedError;
  @JsonKey(name: 'real_name')
  String? get realName => throw _privateConstructorUsedError;
  @JsonKey(name: 'id_number')
  String? get idNumber => throw _privateConstructorUsedError;
  String? get status => throw _privateConstructorUsedError;
  @JsonKey(name: 'submitted_at')
  String? get submittedAt => throw _privateConstructorUsedError;
  @JsonKey(name: 'reviewed_at')
  String? get reviewedAt => throw _privateConstructorUsedError;
  String? get remark => throw _privateConstructorUsedError;

  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;
  @JsonKey(ignore: true)
  $RealnameVerificationCopyWith<RealnameVerification> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $RealnameVerificationCopyWith<$Res> {
  factory $RealnameVerificationCopyWith(RealnameVerification value,
          $Res Function(RealnameVerification) then) =
      _$RealnameVerificationCopyWithImpl<$Res, RealnameVerification>;
  @useResult
  $Res call(
      {int? id,
      @JsonKey(name: 'real_name') String? realName,
      @JsonKey(name: 'id_number') String? idNumber,
      String? status,
      @JsonKey(name: 'submitted_at') String? submittedAt,
      @JsonKey(name: 'reviewed_at') String? reviewedAt,
      String? remark});
}

/// @nodoc
class _$RealnameVerificationCopyWithImpl<$Res,
        $Val extends RealnameVerification>
    implements $RealnameVerificationCopyWith<$Res> {
  _$RealnameVerificationCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? id = freezed,
    Object? realName = freezed,
    Object? idNumber = freezed,
    Object? status = freezed,
    Object? submittedAt = freezed,
    Object? reviewedAt = freezed,
    Object? remark = freezed,
  }) {
    return _then(_value.copyWith(
      id: freezed == id
          ? _value.id
          : id // ignore: cast_nullable_to_non_nullable
              as int?,
      realName: freezed == realName
          ? _value.realName
          : realName // ignore: cast_nullable_to_non_nullable
              as String?,
      idNumber: freezed == idNumber
          ? _value.idNumber
          : idNumber // ignore: cast_nullable_to_non_nullable
              as String?,
      status: freezed == status
          ? _value.status
          : status // ignore: cast_nullable_to_non_nullable
              as String?,
      submittedAt: freezed == submittedAt
          ? _value.submittedAt
          : submittedAt // ignore: cast_nullable_to_non_nullable
              as String?,
      reviewedAt: freezed == reviewedAt
          ? _value.reviewedAt
          : reviewedAt // ignore: cast_nullable_to_non_nullable
              as String?,
      remark: freezed == remark
          ? _value.remark
          : remark // ignore: cast_nullable_to_non_nullable
              as String?,
    ) as $Val);
  }
}

/// @nodoc
abstract class _$$RealnameVerificationImplCopyWith<$Res>
    implements $RealnameVerificationCopyWith<$Res> {
  factory _$$RealnameVerificationImplCopyWith(_$RealnameVerificationImpl value,
          $Res Function(_$RealnameVerificationImpl) then) =
      __$$RealnameVerificationImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call(
      {int? id,
      @JsonKey(name: 'real_name') String? realName,
      @JsonKey(name: 'id_number') String? idNumber,
      String? status,
      @JsonKey(name: 'submitted_at') String? submittedAt,
      @JsonKey(name: 'reviewed_at') String? reviewedAt,
      String? remark});
}

/// @nodoc
class __$$RealnameVerificationImplCopyWithImpl<$Res>
    extends _$RealnameVerificationCopyWithImpl<$Res, _$RealnameVerificationImpl>
    implements _$$RealnameVerificationImplCopyWith<$Res> {
  __$$RealnameVerificationImplCopyWithImpl(_$RealnameVerificationImpl _value,
      $Res Function(_$RealnameVerificationImpl) _then)
      : super(_value, _then);

  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? id = freezed,
    Object? realName = freezed,
    Object? idNumber = freezed,
    Object? status = freezed,
    Object? submittedAt = freezed,
    Object? reviewedAt = freezed,
    Object? remark = freezed,
  }) {
    return _then(_$RealnameVerificationImpl(
      id: freezed == id
          ? _value.id
          : id // ignore: cast_nullable_to_non_nullable
              as int?,
      realName: freezed == realName
          ? _value.realName
          : realName // ignore: cast_nullable_to_non_nullable
              as String?,
      idNumber: freezed == idNumber
          ? _value.idNumber
          : idNumber // ignore: cast_nullable_to_non_nullable
              as String?,
      status: freezed == status
          ? _value.status
          : status // ignore: cast_nullable_to_non_nullable
              as String?,
      submittedAt: freezed == submittedAt
          ? _value.submittedAt
          : submittedAt // ignore: cast_nullable_to_non_nullable
              as String?,
      reviewedAt: freezed == reviewedAt
          ? _value.reviewedAt
          : reviewedAt // ignore: cast_nullable_to_non_nullable
              as String?,
      remark: freezed == remark
          ? _value.remark
          : remark // ignore: cast_nullable_to_non_nullable
              as String?,
    ));
  }
}

/// @nodoc
@JsonSerializable()
class _$RealnameVerificationImpl implements _RealnameVerification {
  const _$RealnameVerificationImpl(
      {this.id,
      @JsonKey(name: 'real_name') this.realName,
      @JsonKey(name: 'id_number') this.idNumber,
      this.status,
      @JsonKey(name: 'submitted_at') this.submittedAt,
      @JsonKey(name: 'reviewed_at') this.reviewedAt,
      this.remark});

  factory _$RealnameVerificationImpl.fromJson(Map<String, dynamic> json) =>
      _$$RealnameVerificationImplFromJson(json);

  @override
  final int? id;
  @override
  @JsonKey(name: 'real_name')
  final String? realName;
  @override
  @JsonKey(name: 'id_number')
  final String? idNumber;
  @override
  final String? status;
  @override
  @JsonKey(name: 'submitted_at')
  final String? submittedAt;
  @override
  @JsonKey(name: 'reviewed_at')
  final String? reviewedAt;
  @override
  final String? remark;

  @override
  String toString() {
    return 'RealnameVerification(id: $id, realName: $realName, idNumber: $idNumber, status: $status, submittedAt: $submittedAt, reviewedAt: $reviewedAt, remark: $remark)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$RealnameVerificationImpl &&
            (identical(other.id, id) || other.id == id) &&
            (identical(other.realName, realName) ||
                other.realName == realName) &&
            (identical(other.idNumber, idNumber) ||
                other.idNumber == idNumber) &&
            (identical(other.status, status) || other.status == status) &&
            (identical(other.submittedAt, submittedAt) ||
                other.submittedAt == submittedAt) &&
            (identical(other.reviewedAt, reviewedAt) ||
                other.reviewedAt == reviewedAt) &&
            (identical(other.remark, remark) || other.remark == remark));
  }

  @JsonKey(ignore: true)
  @override
  int get hashCode => Object.hash(runtimeType, id, realName, idNumber, status,
      submittedAt, reviewedAt, remark);

  @JsonKey(ignore: true)
  @override
  @pragma('vm:prefer-inline')
  _$$RealnameVerificationImplCopyWith<_$RealnameVerificationImpl>
      get copyWith =>
          __$$RealnameVerificationImplCopyWithImpl<_$RealnameVerificationImpl>(
              this, _$identity);

  @override
  Map<String, dynamic> toJson() {
    return _$$RealnameVerificationImplToJson(
      this,
    );
  }
}

abstract class _RealnameVerification implements RealnameVerification {
  const factory _RealnameVerification(
      {final int? id,
      @JsonKey(name: 'real_name') final String? realName,
      @JsonKey(name: 'id_number') final String? idNumber,
      final String? status,
      @JsonKey(name: 'submitted_at') final String? submittedAt,
      @JsonKey(name: 'reviewed_at') final String? reviewedAt,
      final String? remark}) = _$RealnameVerificationImpl;

  factory _RealnameVerification.fromJson(Map<String, dynamic> json) =
      _$RealnameVerificationImpl.fromJson;

  @override
  int? get id;
  @override
  @JsonKey(name: 'real_name')
  String? get realName;
  @override
  @JsonKey(name: 'id_number')
  String? get idNumber;
  @override
  String? get status;
  @override
  @JsonKey(name: 'submitted_at')
  String? get submittedAt;
  @override
  @JsonKey(name: 'reviewed_at')
  String? get reviewedAt;
  @override
  String? get remark;
  @override
  @JsonKey(ignore: true)
  _$$RealnameVerificationImplCopyWith<_$RealnameVerificationImpl>
      get copyWith => throw _privateConstructorUsedError;
}

RealnameRequest _$RealnameRequestFromJson(Map<String, dynamic> json) {
  return _RealnameRequest.fromJson(json);
}

/// @nodoc
mixin _$RealnameRequest {
  @JsonKey(name: 'real_name')
  String get realName => throw _privateConstructorUsedError;
  @JsonKey(name: 'id_number')
  String get idNumber => throw _privateConstructorUsedError;

  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;
  @JsonKey(ignore: true)
  $RealnameRequestCopyWith<RealnameRequest> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $RealnameRequestCopyWith<$Res> {
  factory $RealnameRequestCopyWith(
          RealnameRequest value, $Res Function(RealnameRequest) then) =
      _$RealnameRequestCopyWithImpl<$Res, RealnameRequest>;
  @useResult
  $Res call(
      {@JsonKey(name: 'real_name') String realName,
      @JsonKey(name: 'id_number') String idNumber});
}

/// @nodoc
class _$RealnameRequestCopyWithImpl<$Res, $Val extends RealnameRequest>
    implements $RealnameRequestCopyWith<$Res> {
  _$RealnameRequestCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? realName = null,
    Object? idNumber = null,
  }) {
    return _then(_value.copyWith(
      realName: null == realName
          ? _value.realName
          : realName // ignore: cast_nullable_to_non_nullable
              as String,
      idNumber: null == idNumber
          ? _value.idNumber
          : idNumber // ignore: cast_nullable_to_non_nullable
              as String,
    ) as $Val);
  }
}

/// @nodoc
abstract class _$$RealnameRequestImplCopyWith<$Res>
    implements $RealnameRequestCopyWith<$Res> {
  factory _$$RealnameRequestImplCopyWith(_$RealnameRequestImpl value,
          $Res Function(_$RealnameRequestImpl) then) =
      __$$RealnameRequestImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call(
      {@JsonKey(name: 'real_name') String realName,
      @JsonKey(name: 'id_number') String idNumber});
}

/// @nodoc
class __$$RealnameRequestImplCopyWithImpl<$Res>
    extends _$RealnameRequestCopyWithImpl<$Res, _$RealnameRequestImpl>
    implements _$$RealnameRequestImplCopyWith<$Res> {
  __$$RealnameRequestImplCopyWithImpl(
      _$RealnameRequestImpl _value, $Res Function(_$RealnameRequestImpl) _then)
      : super(_value, _then);

  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? realName = null,
    Object? idNumber = null,
  }) {
    return _then(_$RealnameRequestImpl(
      realName: null == realName
          ? _value.realName
          : realName // ignore: cast_nullable_to_non_nullable
              as String,
      idNumber: null == idNumber
          ? _value.idNumber
          : idNumber // ignore: cast_nullable_to_non_nullable
              as String,
    ));
  }
}

/// @nodoc
@JsonSerializable()
class _$RealnameRequestImpl implements _RealnameRequest {
  const _$RealnameRequestImpl(
      {@JsonKey(name: 'real_name') required this.realName,
      @JsonKey(name: 'id_number') required this.idNumber});

  factory _$RealnameRequestImpl.fromJson(Map<String, dynamic> json) =>
      _$$RealnameRequestImplFromJson(json);

  @override
  @JsonKey(name: 'real_name')
  final String realName;
  @override
  @JsonKey(name: 'id_number')
  final String idNumber;

  @override
  String toString() {
    return 'RealnameRequest(realName: $realName, idNumber: $idNumber)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$RealnameRequestImpl &&
            (identical(other.realName, realName) ||
                other.realName == realName) &&
            (identical(other.idNumber, idNumber) ||
                other.idNumber == idNumber));
  }

  @JsonKey(ignore: true)
  @override
  int get hashCode => Object.hash(runtimeType, realName, idNumber);

  @JsonKey(ignore: true)
  @override
  @pragma('vm:prefer-inline')
  _$$RealnameRequestImplCopyWith<_$RealnameRequestImpl> get copyWith =>
      __$$RealnameRequestImplCopyWithImpl<_$RealnameRequestImpl>(
          this, _$identity);

  @override
  Map<String, dynamic> toJson() {
    return _$$RealnameRequestImplToJson(
      this,
    );
  }
}

abstract class _RealnameRequest implements RealnameRequest {
  const factory _RealnameRequest(
          {@JsonKey(name: 'real_name') required final String realName,
          @JsonKey(name: 'id_number') required final String idNumber}) =
      _$RealnameRequestImpl;

  factory _RealnameRequest.fromJson(Map<String, dynamic> json) =
      _$RealnameRequestImpl.fromJson;

  @override
  @JsonKey(name: 'real_name')
  String get realName;
  @override
  @JsonKey(name: 'id_number')
  String get idNumber;
  @override
  @JsonKey(ignore: true)
  _$$RealnameRequestImplCopyWith<_$RealnameRequestImpl> get copyWith =>
      throw _privateConstructorUsedError;
}
