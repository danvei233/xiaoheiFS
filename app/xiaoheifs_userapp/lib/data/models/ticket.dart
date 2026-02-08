import 'package:freezed_annotation/freezed_annotation.dart';

part 'ticket.freezed.dart';
part 'ticket.g.dart';

/// 工单模型
@freezed
class Ticket with _$Ticket {
  const factory Ticket({
    int? id,
    String? subject,
    String? status,
    @JsonKey(name: 'created_at') String? createdAt,
    @JsonKey(name: 'updated_at') String? updatedAt,
    @JsonKey(name: 'last_message_at') String? lastMessageAt,
    @JsonKey(name: 'message_count') int? messageCount,
    List<TicketMessage>? messages,
    List<TicketResource>? resources,
  }) = _Ticket;

  factory Ticket.fromJson(Map<String, dynamic> json) => _$TicketFromJson(json);
}

/// 工单消息模型
@freezed
class TicketMessage with _$TicketMessage {
  const factory TicketMessage({
    int? id,
    @JsonKey(name: 'is_admin') bool? isAdmin,
    @JsonKey(name: 'user_name') String? userName,
    String? content,
    @JsonKey(name: 'created_at') String? createdAt,
  }) = _TicketMessage;

  factory TicketMessage.fromJson(Map<String, dynamic> json) =>
      _$TicketMessageFromJson(json);
}

/// 工单关联资源模型
@freezed
class TicketResource with _$TicketResource {
  const factory TicketResource({
    @JsonKey(name: 'resource_type') String? resourceType,
    @JsonKey(name: 'resource_id') int? resourceId,
    @JsonKey(name: 'resource_name') String? resourceName,
  }) = _TicketResource;

  factory TicketResource.fromJson(Map<String, dynamic> json) =>
      _$TicketResourceFromJson(json);
}
