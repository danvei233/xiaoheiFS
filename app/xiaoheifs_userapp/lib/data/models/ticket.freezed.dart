// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint
// ignore_for_file: unused_element, deprecated_member_use, deprecated_member_use_from_same_package, use_function_type_syntax_for_parameters, unnecessary_const, avoid_init_to_null, invalid_override_different_default_values_named, prefer_expression_function_bodies, annotate_overrides, invalid_annotation_target, unnecessary_question_mark

part of 'ticket.dart';

// **************************************************************************
// FreezedGenerator
// **************************************************************************

T _$identity<T>(T value) => value;

final _privateConstructorUsedError = UnsupportedError(
    'It seems like you constructed your class using `MyClass._()`. This constructor is only meant to be used by freezed and you are not supposed to need it nor use it.\nPlease check the documentation here for more information: https://github.com/rrousselGit/freezed#adding-getters-and-methods-to-our-models');

Ticket _$TicketFromJson(Map<String, dynamic> json) {
  return _Ticket.fromJson(json);
}

/// @nodoc
mixin _$Ticket {
  int? get id => throw _privateConstructorUsedError;
  String? get subject => throw _privateConstructorUsedError;
  String? get status => throw _privateConstructorUsedError;
  @JsonKey(name: 'created_at')
  String? get createdAt => throw _privateConstructorUsedError;
  @JsonKey(name: 'updated_at')
  String? get updatedAt => throw _privateConstructorUsedError;
  @JsonKey(name: 'last_message_at')
  String? get lastMessageAt => throw _privateConstructorUsedError;
  @JsonKey(name: 'message_count')
  int? get messageCount => throw _privateConstructorUsedError;
  List<TicketMessage>? get messages => throw _privateConstructorUsedError;
  List<TicketResource>? get resources => throw _privateConstructorUsedError;

  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;
  @JsonKey(ignore: true)
  $TicketCopyWith<Ticket> get copyWith => throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $TicketCopyWith<$Res> {
  factory $TicketCopyWith(Ticket value, $Res Function(Ticket) then) =
      _$TicketCopyWithImpl<$Res, Ticket>;
  @useResult
  $Res call(
      {int? id,
      String? subject,
      String? status,
      @JsonKey(name: 'created_at') String? createdAt,
      @JsonKey(name: 'updated_at') String? updatedAt,
      @JsonKey(name: 'last_message_at') String? lastMessageAt,
      @JsonKey(name: 'message_count') int? messageCount,
      List<TicketMessage>? messages,
      List<TicketResource>? resources});
}

/// @nodoc
class _$TicketCopyWithImpl<$Res, $Val extends Ticket>
    implements $TicketCopyWith<$Res> {
  _$TicketCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? id = freezed,
    Object? subject = freezed,
    Object? status = freezed,
    Object? createdAt = freezed,
    Object? updatedAt = freezed,
    Object? lastMessageAt = freezed,
    Object? messageCount = freezed,
    Object? messages = freezed,
    Object? resources = freezed,
  }) {
    return _then(_value.copyWith(
      id: freezed == id
          ? _value.id
          : id // ignore: cast_nullable_to_non_nullable
              as int?,
      subject: freezed == subject
          ? _value.subject
          : subject // ignore: cast_nullable_to_non_nullable
              as String?,
      status: freezed == status
          ? _value.status
          : status // ignore: cast_nullable_to_non_nullable
              as String?,
      createdAt: freezed == createdAt
          ? _value.createdAt
          : createdAt // ignore: cast_nullable_to_non_nullable
              as String?,
      updatedAt: freezed == updatedAt
          ? _value.updatedAt
          : updatedAt // ignore: cast_nullable_to_non_nullable
              as String?,
      lastMessageAt: freezed == lastMessageAt
          ? _value.lastMessageAt
          : lastMessageAt // ignore: cast_nullable_to_non_nullable
              as String?,
      messageCount: freezed == messageCount
          ? _value.messageCount
          : messageCount // ignore: cast_nullable_to_non_nullable
              as int?,
      messages: freezed == messages
          ? _value.messages
          : messages // ignore: cast_nullable_to_non_nullable
              as List<TicketMessage>?,
      resources: freezed == resources
          ? _value.resources
          : resources // ignore: cast_nullable_to_non_nullable
              as List<TicketResource>?,
    ) as $Val);
  }
}

/// @nodoc
abstract class _$$TicketImplCopyWith<$Res> implements $TicketCopyWith<$Res> {
  factory _$$TicketImplCopyWith(
          _$TicketImpl value, $Res Function(_$TicketImpl) then) =
      __$$TicketImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call(
      {int? id,
      String? subject,
      String? status,
      @JsonKey(name: 'created_at') String? createdAt,
      @JsonKey(name: 'updated_at') String? updatedAt,
      @JsonKey(name: 'last_message_at') String? lastMessageAt,
      @JsonKey(name: 'message_count') int? messageCount,
      List<TicketMessage>? messages,
      List<TicketResource>? resources});
}

/// @nodoc
class __$$TicketImplCopyWithImpl<$Res>
    extends _$TicketCopyWithImpl<$Res, _$TicketImpl>
    implements _$$TicketImplCopyWith<$Res> {
  __$$TicketImplCopyWithImpl(
      _$TicketImpl _value, $Res Function(_$TicketImpl) _then)
      : super(_value, _then);

  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? id = freezed,
    Object? subject = freezed,
    Object? status = freezed,
    Object? createdAt = freezed,
    Object? updatedAt = freezed,
    Object? lastMessageAt = freezed,
    Object? messageCount = freezed,
    Object? messages = freezed,
    Object? resources = freezed,
  }) {
    return _then(_$TicketImpl(
      id: freezed == id
          ? _value.id
          : id // ignore: cast_nullable_to_non_nullable
              as int?,
      subject: freezed == subject
          ? _value.subject
          : subject // ignore: cast_nullable_to_non_nullable
              as String?,
      status: freezed == status
          ? _value.status
          : status // ignore: cast_nullable_to_non_nullable
              as String?,
      createdAt: freezed == createdAt
          ? _value.createdAt
          : createdAt // ignore: cast_nullable_to_non_nullable
              as String?,
      updatedAt: freezed == updatedAt
          ? _value.updatedAt
          : updatedAt // ignore: cast_nullable_to_non_nullable
              as String?,
      lastMessageAt: freezed == lastMessageAt
          ? _value.lastMessageAt
          : lastMessageAt // ignore: cast_nullable_to_non_nullable
              as String?,
      messageCount: freezed == messageCount
          ? _value.messageCount
          : messageCount // ignore: cast_nullable_to_non_nullable
              as int?,
      messages: freezed == messages
          ? _value._messages
          : messages // ignore: cast_nullable_to_non_nullable
              as List<TicketMessage>?,
      resources: freezed == resources
          ? _value._resources
          : resources // ignore: cast_nullable_to_non_nullable
              as List<TicketResource>?,
    ));
  }
}

/// @nodoc
@JsonSerializable()
class _$TicketImpl implements _Ticket {
  const _$TicketImpl(
      {this.id,
      this.subject,
      this.status,
      @JsonKey(name: 'created_at') this.createdAt,
      @JsonKey(name: 'updated_at') this.updatedAt,
      @JsonKey(name: 'last_message_at') this.lastMessageAt,
      @JsonKey(name: 'message_count') this.messageCount,
      final List<TicketMessage>? messages,
      final List<TicketResource>? resources})
      : _messages = messages,
        _resources = resources;

  factory _$TicketImpl.fromJson(Map<String, dynamic> json) =>
      _$$TicketImplFromJson(json);

  @override
  final int? id;
  @override
  final String? subject;
  @override
  final String? status;
  @override
  @JsonKey(name: 'created_at')
  final String? createdAt;
  @override
  @JsonKey(name: 'updated_at')
  final String? updatedAt;
  @override
  @JsonKey(name: 'last_message_at')
  final String? lastMessageAt;
  @override
  @JsonKey(name: 'message_count')
  final int? messageCount;
  final List<TicketMessage>? _messages;
  @override
  List<TicketMessage>? get messages {
    final value = _messages;
    if (value == null) return null;
    if (_messages is EqualUnmodifiableListView) return _messages;
    // ignore: implicit_dynamic_type
    return EqualUnmodifiableListView(value);
  }

  final List<TicketResource>? _resources;
  @override
  List<TicketResource>? get resources {
    final value = _resources;
    if (value == null) return null;
    if (_resources is EqualUnmodifiableListView) return _resources;
    // ignore: implicit_dynamic_type
    return EqualUnmodifiableListView(value);
  }

  @override
  String toString() {
    return 'Ticket(id: $id, subject: $subject, status: $status, createdAt: $createdAt, updatedAt: $updatedAt, lastMessageAt: $lastMessageAt, messageCount: $messageCount, messages: $messages, resources: $resources)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$TicketImpl &&
            (identical(other.id, id) || other.id == id) &&
            (identical(other.subject, subject) || other.subject == subject) &&
            (identical(other.status, status) || other.status == status) &&
            (identical(other.createdAt, createdAt) ||
                other.createdAt == createdAt) &&
            (identical(other.updatedAt, updatedAt) ||
                other.updatedAt == updatedAt) &&
            (identical(other.lastMessageAt, lastMessageAt) ||
                other.lastMessageAt == lastMessageAt) &&
            (identical(other.messageCount, messageCount) ||
                other.messageCount == messageCount) &&
            const DeepCollectionEquality().equals(other._messages, _messages) &&
            const DeepCollectionEquality()
                .equals(other._resources, _resources));
  }

  @JsonKey(ignore: true)
  @override
  int get hashCode => Object.hash(
      runtimeType,
      id,
      subject,
      status,
      createdAt,
      updatedAt,
      lastMessageAt,
      messageCount,
      const DeepCollectionEquality().hash(_messages),
      const DeepCollectionEquality().hash(_resources));

  @JsonKey(ignore: true)
  @override
  @pragma('vm:prefer-inline')
  _$$TicketImplCopyWith<_$TicketImpl> get copyWith =>
      __$$TicketImplCopyWithImpl<_$TicketImpl>(this, _$identity);

  @override
  Map<String, dynamic> toJson() {
    return _$$TicketImplToJson(
      this,
    );
  }
}

abstract class _Ticket implements Ticket {
  const factory _Ticket(
      {final int? id,
      final String? subject,
      final String? status,
      @JsonKey(name: 'created_at') final String? createdAt,
      @JsonKey(name: 'updated_at') final String? updatedAt,
      @JsonKey(name: 'last_message_at') final String? lastMessageAt,
      @JsonKey(name: 'message_count') final int? messageCount,
      final List<TicketMessage>? messages,
      final List<TicketResource>? resources}) = _$TicketImpl;

  factory _Ticket.fromJson(Map<String, dynamic> json) = _$TicketImpl.fromJson;

  @override
  int? get id;
  @override
  String? get subject;
  @override
  String? get status;
  @override
  @JsonKey(name: 'created_at')
  String? get createdAt;
  @override
  @JsonKey(name: 'updated_at')
  String? get updatedAt;
  @override
  @JsonKey(name: 'last_message_at')
  String? get lastMessageAt;
  @override
  @JsonKey(name: 'message_count')
  int? get messageCount;
  @override
  List<TicketMessage>? get messages;
  @override
  List<TicketResource>? get resources;
  @override
  @JsonKey(ignore: true)
  _$$TicketImplCopyWith<_$TicketImpl> get copyWith =>
      throw _privateConstructorUsedError;
}

TicketMessage _$TicketMessageFromJson(Map<String, dynamic> json) {
  return _TicketMessage.fromJson(json);
}

/// @nodoc
mixin _$TicketMessage {
  int? get id => throw _privateConstructorUsedError;
  @JsonKey(name: 'is_admin')
  bool? get isAdmin => throw _privateConstructorUsedError;
  @JsonKey(name: 'user_name')
  String? get userName => throw _privateConstructorUsedError;
  String? get content => throw _privateConstructorUsedError;
  @JsonKey(name: 'created_at')
  String? get createdAt => throw _privateConstructorUsedError;

  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;
  @JsonKey(ignore: true)
  $TicketMessageCopyWith<TicketMessage> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $TicketMessageCopyWith<$Res> {
  factory $TicketMessageCopyWith(
          TicketMessage value, $Res Function(TicketMessage) then) =
      _$TicketMessageCopyWithImpl<$Res, TicketMessage>;
  @useResult
  $Res call(
      {int? id,
      @JsonKey(name: 'is_admin') bool? isAdmin,
      @JsonKey(name: 'user_name') String? userName,
      String? content,
      @JsonKey(name: 'created_at') String? createdAt});
}

/// @nodoc
class _$TicketMessageCopyWithImpl<$Res, $Val extends TicketMessage>
    implements $TicketMessageCopyWith<$Res> {
  _$TicketMessageCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? id = freezed,
    Object? isAdmin = freezed,
    Object? userName = freezed,
    Object? content = freezed,
    Object? createdAt = freezed,
  }) {
    return _then(_value.copyWith(
      id: freezed == id
          ? _value.id
          : id // ignore: cast_nullable_to_non_nullable
              as int?,
      isAdmin: freezed == isAdmin
          ? _value.isAdmin
          : isAdmin // ignore: cast_nullable_to_non_nullable
              as bool?,
      userName: freezed == userName
          ? _value.userName
          : userName // ignore: cast_nullable_to_non_nullable
              as String?,
      content: freezed == content
          ? _value.content
          : content // ignore: cast_nullable_to_non_nullable
              as String?,
      createdAt: freezed == createdAt
          ? _value.createdAt
          : createdAt // ignore: cast_nullable_to_non_nullable
              as String?,
    ) as $Val);
  }
}

/// @nodoc
abstract class _$$TicketMessageImplCopyWith<$Res>
    implements $TicketMessageCopyWith<$Res> {
  factory _$$TicketMessageImplCopyWith(
          _$TicketMessageImpl value, $Res Function(_$TicketMessageImpl) then) =
      __$$TicketMessageImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call(
      {int? id,
      @JsonKey(name: 'is_admin') bool? isAdmin,
      @JsonKey(name: 'user_name') String? userName,
      String? content,
      @JsonKey(name: 'created_at') String? createdAt});
}

/// @nodoc
class __$$TicketMessageImplCopyWithImpl<$Res>
    extends _$TicketMessageCopyWithImpl<$Res, _$TicketMessageImpl>
    implements _$$TicketMessageImplCopyWith<$Res> {
  __$$TicketMessageImplCopyWithImpl(
      _$TicketMessageImpl _value, $Res Function(_$TicketMessageImpl) _then)
      : super(_value, _then);

  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? id = freezed,
    Object? isAdmin = freezed,
    Object? userName = freezed,
    Object? content = freezed,
    Object? createdAt = freezed,
  }) {
    return _then(_$TicketMessageImpl(
      id: freezed == id
          ? _value.id
          : id // ignore: cast_nullable_to_non_nullable
              as int?,
      isAdmin: freezed == isAdmin
          ? _value.isAdmin
          : isAdmin // ignore: cast_nullable_to_non_nullable
              as bool?,
      userName: freezed == userName
          ? _value.userName
          : userName // ignore: cast_nullable_to_non_nullable
              as String?,
      content: freezed == content
          ? _value.content
          : content // ignore: cast_nullable_to_non_nullable
              as String?,
      createdAt: freezed == createdAt
          ? _value.createdAt
          : createdAt // ignore: cast_nullable_to_non_nullable
              as String?,
    ));
  }
}

/// @nodoc
@JsonSerializable()
class _$TicketMessageImpl implements _TicketMessage {
  const _$TicketMessageImpl(
      {this.id,
      @JsonKey(name: 'is_admin') this.isAdmin,
      @JsonKey(name: 'user_name') this.userName,
      this.content,
      @JsonKey(name: 'created_at') this.createdAt});

  factory _$TicketMessageImpl.fromJson(Map<String, dynamic> json) =>
      _$$TicketMessageImplFromJson(json);

  @override
  final int? id;
  @override
  @JsonKey(name: 'is_admin')
  final bool? isAdmin;
  @override
  @JsonKey(name: 'user_name')
  final String? userName;
  @override
  final String? content;
  @override
  @JsonKey(name: 'created_at')
  final String? createdAt;

  @override
  String toString() {
    return 'TicketMessage(id: $id, isAdmin: $isAdmin, userName: $userName, content: $content, createdAt: $createdAt)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$TicketMessageImpl &&
            (identical(other.id, id) || other.id == id) &&
            (identical(other.isAdmin, isAdmin) || other.isAdmin == isAdmin) &&
            (identical(other.userName, userName) ||
                other.userName == userName) &&
            (identical(other.content, content) || other.content == content) &&
            (identical(other.createdAt, createdAt) ||
                other.createdAt == createdAt));
  }

  @JsonKey(ignore: true)
  @override
  int get hashCode =>
      Object.hash(runtimeType, id, isAdmin, userName, content, createdAt);

  @JsonKey(ignore: true)
  @override
  @pragma('vm:prefer-inline')
  _$$TicketMessageImplCopyWith<_$TicketMessageImpl> get copyWith =>
      __$$TicketMessageImplCopyWithImpl<_$TicketMessageImpl>(this, _$identity);

  @override
  Map<String, dynamic> toJson() {
    return _$$TicketMessageImplToJson(
      this,
    );
  }
}

abstract class _TicketMessage implements TicketMessage {
  const factory _TicketMessage(
          {final int? id,
          @JsonKey(name: 'is_admin') final bool? isAdmin,
          @JsonKey(name: 'user_name') final String? userName,
          final String? content,
          @JsonKey(name: 'created_at') final String? createdAt}) =
      _$TicketMessageImpl;

  factory _TicketMessage.fromJson(Map<String, dynamic> json) =
      _$TicketMessageImpl.fromJson;

  @override
  int? get id;
  @override
  @JsonKey(name: 'is_admin')
  bool? get isAdmin;
  @override
  @JsonKey(name: 'user_name')
  String? get userName;
  @override
  String? get content;
  @override
  @JsonKey(name: 'created_at')
  String? get createdAt;
  @override
  @JsonKey(ignore: true)
  _$$TicketMessageImplCopyWith<_$TicketMessageImpl> get copyWith =>
      throw _privateConstructorUsedError;
}

TicketResource _$TicketResourceFromJson(Map<String, dynamic> json) {
  return _TicketResource.fromJson(json);
}

/// @nodoc
mixin _$TicketResource {
  @JsonKey(name: 'resource_type')
  String? get resourceType => throw _privateConstructorUsedError;
  @JsonKey(name: 'resource_id')
  int? get resourceId => throw _privateConstructorUsedError;
  @JsonKey(name: 'resource_name')
  String? get resourceName => throw _privateConstructorUsedError;

  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;
  @JsonKey(ignore: true)
  $TicketResourceCopyWith<TicketResource> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $TicketResourceCopyWith<$Res> {
  factory $TicketResourceCopyWith(
          TicketResource value, $Res Function(TicketResource) then) =
      _$TicketResourceCopyWithImpl<$Res, TicketResource>;
  @useResult
  $Res call(
      {@JsonKey(name: 'resource_type') String? resourceType,
      @JsonKey(name: 'resource_id') int? resourceId,
      @JsonKey(name: 'resource_name') String? resourceName});
}

/// @nodoc
class _$TicketResourceCopyWithImpl<$Res, $Val extends TicketResource>
    implements $TicketResourceCopyWith<$Res> {
  _$TicketResourceCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? resourceType = freezed,
    Object? resourceId = freezed,
    Object? resourceName = freezed,
  }) {
    return _then(_value.copyWith(
      resourceType: freezed == resourceType
          ? _value.resourceType
          : resourceType // ignore: cast_nullable_to_non_nullable
              as String?,
      resourceId: freezed == resourceId
          ? _value.resourceId
          : resourceId // ignore: cast_nullable_to_non_nullable
              as int?,
      resourceName: freezed == resourceName
          ? _value.resourceName
          : resourceName // ignore: cast_nullable_to_non_nullable
              as String?,
    ) as $Val);
  }
}

/// @nodoc
abstract class _$$TicketResourceImplCopyWith<$Res>
    implements $TicketResourceCopyWith<$Res> {
  factory _$$TicketResourceImplCopyWith(_$TicketResourceImpl value,
          $Res Function(_$TicketResourceImpl) then) =
      __$$TicketResourceImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call(
      {@JsonKey(name: 'resource_type') String? resourceType,
      @JsonKey(name: 'resource_id') int? resourceId,
      @JsonKey(name: 'resource_name') String? resourceName});
}

/// @nodoc
class __$$TicketResourceImplCopyWithImpl<$Res>
    extends _$TicketResourceCopyWithImpl<$Res, _$TicketResourceImpl>
    implements _$$TicketResourceImplCopyWith<$Res> {
  __$$TicketResourceImplCopyWithImpl(
      _$TicketResourceImpl _value, $Res Function(_$TicketResourceImpl) _then)
      : super(_value, _then);

  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? resourceType = freezed,
    Object? resourceId = freezed,
    Object? resourceName = freezed,
  }) {
    return _then(_$TicketResourceImpl(
      resourceType: freezed == resourceType
          ? _value.resourceType
          : resourceType // ignore: cast_nullable_to_non_nullable
              as String?,
      resourceId: freezed == resourceId
          ? _value.resourceId
          : resourceId // ignore: cast_nullable_to_non_nullable
              as int?,
      resourceName: freezed == resourceName
          ? _value.resourceName
          : resourceName // ignore: cast_nullable_to_non_nullable
              as String?,
    ));
  }
}

/// @nodoc
@JsonSerializable()
class _$TicketResourceImpl implements _TicketResource {
  const _$TicketResourceImpl(
      {@JsonKey(name: 'resource_type') this.resourceType,
      @JsonKey(name: 'resource_id') this.resourceId,
      @JsonKey(name: 'resource_name') this.resourceName});

  factory _$TicketResourceImpl.fromJson(Map<String, dynamic> json) =>
      _$$TicketResourceImplFromJson(json);

  @override
  @JsonKey(name: 'resource_type')
  final String? resourceType;
  @override
  @JsonKey(name: 'resource_id')
  final int? resourceId;
  @override
  @JsonKey(name: 'resource_name')
  final String? resourceName;

  @override
  String toString() {
    return 'TicketResource(resourceType: $resourceType, resourceId: $resourceId, resourceName: $resourceName)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$TicketResourceImpl &&
            (identical(other.resourceType, resourceType) ||
                other.resourceType == resourceType) &&
            (identical(other.resourceId, resourceId) ||
                other.resourceId == resourceId) &&
            (identical(other.resourceName, resourceName) ||
                other.resourceName == resourceName));
  }

  @JsonKey(ignore: true)
  @override
  int get hashCode =>
      Object.hash(runtimeType, resourceType, resourceId, resourceName);

  @JsonKey(ignore: true)
  @override
  @pragma('vm:prefer-inline')
  _$$TicketResourceImplCopyWith<_$TicketResourceImpl> get copyWith =>
      __$$TicketResourceImplCopyWithImpl<_$TicketResourceImpl>(
          this, _$identity);

  @override
  Map<String, dynamic> toJson() {
    return _$$TicketResourceImplToJson(
      this,
    );
  }
}

abstract class _TicketResource implements TicketResource {
  const factory _TicketResource(
          {@JsonKey(name: 'resource_type') final String? resourceType,
          @JsonKey(name: 'resource_id') final int? resourceId,
          @JsonKey(name: 'resource_name') final String? resourceName}) =
      _$TicketResourceImpl;

  factory _TicketResource.fromJson(Map<String, dynamic> json) =
      _$TicketResourceImpl.fromJson;

  @override
  @JsonKey(name: 'resource_type')
  String? get resourceType;
  @override
  @JsonKey(name: 'resource_id')
  int? get resourceId;
  @override
  @JsonKey(name: 'resource_name')
  String? get resourceName;
  @override
  @JsonKey(ignore: true)
  _$$TicketResourceImplCopyWith<_$TicketResourceImpl> get copyWith =>
      throw _privateConstructorUsedError;
}
