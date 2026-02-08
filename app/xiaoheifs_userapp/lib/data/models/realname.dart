import 'package:freezed_annotation/freezed_annotation.dart';

part 'realname.freezed.dart';
part 'realname.g.dart';

/// 实名认证状态模型
@freezed
class RealnameStatus with _$RealnameStatus {
  const factory RealnameStatus({
    bool? enabled,
    String? provider,
    @JsonKey(name: 'block_actions') List<String>? blockActions,
    bool? verified,
    RealnameVerification? verification,
  }) = _RealnameStatus;

  factory RealnameStatus.fromJson(Map<String, dynamic> json) =>
      _$RealnameStatusFromJson(json);
}

/// 实名认证记录模型
@freezed
class RealnameVerification with _$RealnameVerification {
  const factory RealnameVerification({
    int? id,
    @JsonKey(name: 'real_name') String? realName,
    @JsonKey(name: 'id_number') String? idNumber,
    String? status,
    @JsonKey(name: 'submitted_at') String? submittedAt,
    @JsonKey(name: 'reviewed_at') String? reviewedAt,
    String? remark,
  }) = _RealnameVerification;

  factory RealnameVerification.fromJson(Map<String, dynamic> json) =>
      _$RealnameVerificationFromJson(json);
}

/// 实名认证提交请求
@freezed
class RealnameRequest with _$RealnameRequest {
  const factory RealnameRequest({
    @JsonKey(name: 'real_name') required String realName,
    @JsonKey(name: 'id_number') required String idNumber,
  }) = _RealnameRequest;

  factory RealnameRequest.fromJson(Map<String, dynamic> json) =>
      _$RealnameRequestFromJson(json);
}
