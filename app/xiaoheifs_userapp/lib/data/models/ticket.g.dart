// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'ticket.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

_$TicketImpl _$$TicketImplFromJson(Map<String, dynamic> json) => _$TicketImpl(
      id: (json['id'] as num?)?.toInt(),
      subject: json['subject'] as String?,
      status: json['status'] as String?,
      createdAt: json['created_at'] as String?,
      updatedAt: json['updated_at'] as String?,
      lastMessageAt: json['last_message_at'] as String?,
      messageCount: (json['message_count'] as num?)?.toInt(),
      messages: (json['messages'] as List<dynamic>?)
          ?.map((e) => TicketMessage.fromJson(e as Map<String, dynamic>))
          .toList(),
      resources: (json['resources'] as List<dynamic>?)
          ?.map((e) => TicketResource.fromJson(e as Map<String, dynamic>))
          .toList(),
    );

Map<String, dynamic> _$$TicketImplToJson(_$TicketImpl instance) =>
    <String, dynamic>{
      'id': instance.id,
      'subject': instance.subject,
      'status': instance.status,
      'created_at': instance.createdAt,
      'updated_at': instance.updatedAt,
      'last_message_at': instance.lastMessageAt,
      'message_count': instance.messageCount,
      'messages': instance.messages,
      'resources': instance.resources,
    };

_$TicketMessageImpl _$$TicketMessageImplFromJson(Map<String, dynamic> json) =>
    _$TicketMessageImpl(
      id: (json['id'] as num?)?.toInt(),
      isAdmin: json['is_admin'] as bool?,
      userName: json['user_name'] as String?,
      content: json['content'] as String?,
      createdAt: json['created_at'] as String?,
    );

Map<String, dynamic> _$$TicketMessageImplToJson(_$TicketMessageImpl instance) =>
    <String, dynamic>{
      'id': instance.id,
      'is_admin': instance.isAdmin,
      'user_name': instance.userName,
      'content': instance.content,
      'created_at': instance.createdAt,
    };

_$TicketResourceImpl _$$TicketResourceImplFromJson(Map<String, dynamic> json) =>
    _$TicketResourceImpl(
      resourceType: json['resource_type'] as String?,
      resourceId: (json['resource_id'] as num?)?.toInt(),
      resourceName: json['resource_name'] as String?,
    );

Map<String, dynamic> _$$TicketResourceImplToJson(
        _$TicketResourceImpl instance) =>
    <String, dynamic>{
      'resource_type': instance.resourceType,
      'resource_id': instance.resourceId,
      'resource_name': instance.resourceName,
    };
