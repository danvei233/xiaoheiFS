// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'realname.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

_$RealnameStatusImpl _$$RealnameStatusImplFromJson(Map<String, dynamic> json) =>
    _$RealnameStatusImpl(
      enabled: json['enabled'] as bool?,
      provider: json['provider'] as String?,
      blockActions: (json['block_actions'] as List<dynamic>?)
          ?.map((e) => e as String)
          .toList(),
      verified: json['verified'] as bool?,
      verification: json['verification'] == null
          ? null
          : RealnameVerification.fromJson(
              json['verification'] as Map<String, dynamic>),
    );

Map<String, dynamic> _$$RealnameStatusImplToJson(
        _$RealnameStatusImpl instance) =>
    <String, dynamic>{
      'enabled': instance.enabled,
      'provider': instance.provider,
      'block_actions': instance.blockActions,
      'verified': instance.verified,
      'verification': instance.verification,
    };

_$RealnameVerificationImpl _$$RealnameVerificationImplFromJson(
        Map<String, dynamic> json) =>
    _$RealnameVerificationImpl(
      id: (json['id'] as num?)?.toInt(),
      realName: json['real_name'] as String?,
      idNumber: json['id_number'] as String?,
      status: json['status'] as String?,
      submittedAt: json['submitted_at'] as String?,
      reviewedAt: json['reviewed_at'] as String?,
      remark: json['remark'] as String?,
    );

Map<String, dynamic> _$$RealnameVerificationImplToJson(
        _$RealnameVerificationImpl instance) =>
    <String, dynamic>{
      'id': instance.id,
      'real_name': instance.realName,
      'id_number': instance.idNumber,
      'status': instance.status,
      'submitted_at': instance.submittedAt,
      'reviewed_at': instance.reviewedAt,
      'remark': instance.remark,
    };

_$RealnameRequestImpl _$$RealnameRequestImplFromJson(
        Map<String, dynamic> json) =>
    _$RealnameRequestImpl(
      realName: json['real_name'] as String,
      idNumber: json['id_number'] as String,
    );

Map<String, dynamic> _$$RealnameRequestImplToJson(
        _$RealnameRequestImpl instance) =>
    <String, dynamic>{
      'real_name': instance.realName,
      'id_number': instance.idNumber,
    };
