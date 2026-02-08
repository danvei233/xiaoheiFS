// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint
// ignore_for_file: unused_element, deprecated_member_use, deprecated_member_use_from_same_package, use_function_type_syntax_for_parameters, unnecessary_const, avoid_init_to_null, invalid_override_different_default_values_named, prefer_expression_function_bodies, annotate_overrides, invalid_annotation_target, unnecessary_question_mark

part of 'order.dart';

// **************************************************************************
// FreezedGenerator
// **************************************************************************

T _$identity<T>(T value) => value;

final _privateConstructorUsedError = UnsupportedError(
    'It seems like you constructed your class using `MyClass._()`. This constructor is only meant to be used by freezed and you are not supposed to need it nor use it.\nPlease check the documentation here for more information: https://github.com/rrousselGit/freezed#adding-getters-and-methods-to-our-models');

Order _$OrderFromJson(Map<String, dynamic> json) {
  return _Order.fromJson(json);
}

/// @nodoc
mixin _$Order {
  int? get id => throw _privateConstructorUsedError;
  @JsonKey(name: 'order_no')
  String? get orderNo => throw _privateConstructorUsedError;
  String? get status => throw _privateConstructorUsedError;
  @JsonKey(name: 'total_amount')
  double? get totalAmount => throw _privateConstructorUsedError;
  String? get currency => throw _privateConstructorUsedError;
  @JsonKey(name: 'paid_amount')
  double? get paidAmount => throw _privateConstructorUsedError;
  @JsonKey(name: 'created_at')
  String? get createdAt => throw _privateConstructorUsedError;
  @JsonKey(name: 'updated_at')
  String? get updatedAt => throw _privateConstructorUsedError;
  @JsonKey(name: 'paid_at')
  String? get paidAt => throw _privateConstructorUsedError;
  List<OrderItem>? get items => throw _privateConstructorUsedError;

  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;
  @JsonKey(ignore: true)
  $OrderCopyWith<Order> get copyWith => throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $OrderCopyWith<$Res> {
  factory $OrderCopyWith(Order value, $Res Function(Order) then) =
      _$OrderCopyWithImpl<$Res, Order>;
  @useResult
  $Res call(
      {int? id,
      @JsonKey(name: 'order_no') String? orderNo,
      String? status,
      @JsonKey(name: 'total_amount') double? totalAmount,
      String? currency,
      @JsonKey(name: 'paid_amount') double? paidAmount,
      @JsonKey(name: 'created_at') String? createdAt,
      @JsonKey(name: 'updated_at') String? updatedAt,
      @JsonKey(name: 'paid_at') String? paidAt,
      List<OrderItem>? items});
}

/// @nodoc
class _$OrderCopyWithImpl<$Res, $Val extends Order>
    implements $OrderCopyWith<$Res> {
  _$OrderCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? id = freezed,
    Object? orderNo = freezed,
    Object? status = freezed,
    Object? totalAmount = freezed,
    Object? currency = freezed,
    Object? paidAmount = freezed,
    Object? createdAt = freezed,
    Object? updatedAt = freezed,
    Object? paidAt = freezed,
    Object? items = freezed,
  }) {
    return _then(_value.copyWith(
      id: freezed == id
          ? _value.id
          : id // ignore: cast_nullable_to_non_nullable
              as int?,
      orderNo: freezed == orderNo
          ? _value.orderNo
          : orderNo // ignore: cast_nullable_to_non_nullable
              as String?,
      status: freezed == status
          ? _value.status
          : status // ignore: cast_nullable_to_non_nullable
              as String?,
      totalAmount: freezed == totalAmount
          ? _value.totalAmount
          : totalAmount // ignore: cast_nullable_to_non_nullable
              as double?,
      currency: freezed == currency
          ? _value.currency
          : currency // ignore: cast_nullable_to_non_nullable
              as String?,
      paidAmount: freezed == paidAmount
          ? _value.paidAmount
          : paidAmount // ignore: cast_nullable_to_non_nullable
              as double?,
      createdAt: freezed == createdAt
          ? _value.createdAt
          : createdAt // ignore: cast_nullable_to_non_nullable
              as String?,
      updatedAt: freezed == updatedAt
          ? _value.updatedAt
          : updatedAt // ignore: cast_nullable_to_non_nullable
              as String?,
      paidAt: freezed == paidAt
          ? _value.paidAt
          : paidAt // ignore: cast_nullable_to_non_nullable
              as String?,
      items: freezed == items
          ? _value.items
          : items // ignore: cast_nullable_to_non_nullable
              as List<OrderItem>?,
    ) as $Val);
  }
}

/// @nodoc
abstract class _$$OrderImplCopyWith<$Res> implements $OrderCopyWith<$Res> {
  factory _$$OrderImplCopyWith(
          _$OrderImpl value, $Res Function(_$OrderImpl) then) =
      __$$OrderImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call(
      {int? id,
      @JsonKey(name: 'order_no') String? orderNo,
      String? status,
      @JsonKey(name: 'total_amount') double? totalAmount,
      String? currency,
      @JsonKey(name: 'paid_amount') double? paidAmount,
      @JsonKey(name: 'created_at') String? createdAt,
      @JsonKey(name: 'updated_at') String? updatedAt,
      @JsonKey(name: 'paid_at') String? paidAt,
      List<OrderItem>? items});
}

/// @nodoc
class __$$OrderImplCopyWithImpl<$Res>
    extends _$OrderCopyWithImpl<$Res, _$OrderImpl>
    implements _$$OrderImplCopyWith<$Res> {
  __$$OrderImplCopyWithImpl(
      _$OrderImpl _value, $Res Function(_$OrderImpl) _then)
      : super(_value, _then);

  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? id = freezed,
    Object? orderNo = freezed,
    Object? status = freezed,
    Object? totalAmount = freezed,
    Object? currency = freezed,
    Object? paidAmount = freezed,
    Object? createdAt = freezed,
    Object? updatedAt = freezed,
    Object? paidAt = freezed,
    Object? items = freezed,
  }) {
    return _then(_$OrderImpl(
      id: freezed == id
          ? _value.id
          : id // ignore: cast_nullable_to_non_nullable
              as int?,
      orderNo: freezed == orderNo
          ? _value.orderNo
          : orderNo // ignore: cast_nullable_to_non_nullable
              as String?,
      status: freezed == status
          ? _value.status
          : status // ignore: cast_nullable_to_non_nullable
              as String?,
      totalAmount: freezed == totalAmount
          ? _value.totalAmount
          : totalAmount // ignore: cast_nullable_to_non_nullable
              as double?,
      currency: freezed == currency
          ? _value.currency
          : currency // ignore: cast_nullable_to_non_nullable
              as String?,
      paidAmount: freezed == paidAmount
          ? _value.paidAmount
          : paidAmount // ignore: cast_nullable_to_non_nullable
              as double?,
      createdAt: freezed == createdAt
          ? _value.createdAt
          : createdAt // ignore: cast_nullable_to_non_nullable
              as String?,
      updatedAt: freezed == updatedAt
          ? _value.updatedAt
          : updatedAt // ignore: cast_nullable_to_non_nullable
              as String?,
      paidAt: freezed == paidAt
          ? _value.paidAt
          : paidAt // ignore: cast_nullable_to_non_nullable
              as String?,
      items: freezed == items
          ? _value._items
          : items // ignore: cast_nullable_to_non_nullable
              as List<OrderItem>?,
    ));
  }
}

/// @nodoc
@JsonSerializable()
class _$OrderImpl implements _Order {
  const _$OrderImpl(
      {this.id,
      @JsonKey(name: 'order_no') this.orderNo,
      this.status,
      @JsonKey(name: 'total_amount') this.totalAmount,
      this.currency,
      @JsonKey(name: 'paid_amount') this.paidAmount,
      @JsonKey(name: 'created_at') this.createdAt,
      @JsonKey(name: 'updated_at') this.updatedAt,
      @JsonKey(name: 'paid_at') this.paidAt,
      final List<OrderItem>? items})
      : _items = items;

  factory _$OrderImpl.fromJson(Map<String, dynamic> json) =>
      _$$OrderImplFromJson(json);

  @override
  final int? id;
  @override
  @JsonKey(name: 'order_no')
  final String? orderNo;
  @override
  final String? status;
  @override
  @JsonKey(name: 'total_amount')
  final double? totalAmount;
  @override
  final String? currency;
  @override
  @JsonKey(name: 'paid_amount')
  final double? paidAmount;
  @override
  @JsonKey(name: 'created_at')
  final String? createdAt;
  @override
  @JsonKey(name: 'updated_at')
  final String? updatedAt;
  @override
  @JsonKey(name: 'paid_at')
  final String? paidAt;
  final List<OrderItem>? _items;
  @override
  List<OrderItem>? get items {
    final value = _items;
    if (value == null) return null;
    if (_items is EqualUnmodifiableListView) return _items;
    // ignore: implicit_dynamic_type
    return EqualUnmodifiableListView(value);
  }

  @override
  String toString() {
    return 'Order(id: $id, orderNo: $orderNo, status: $status, totalAmount: $totalAmount, currency: $currency, paidAmount: $paidAmount, createdAt: $createdAt, updatedAt: $updatedAt, paidAt: $paidAt, items: $items)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$OrderImpl &&
            (identical(other.id, id) || other.id == id) &&
            (identical(other.orderNo, orderNo) || other.orderNo == orderNo) &&
            (identical(other.status, status) || other.status == status) &&
            (identical(other.totalAmount, totalAmount) ||
                other.totalAmount == totalAmount) &&
            (identical(other.currency, currency) ||
                other.currency == currency) &&
            (identical(other.paidAmount, paidAmount) ||
                other.paidAmount == paidAmount) &&
            (identical(other.createdAt, createdAt) ||
                other.createdAt == createdAt) &&
            (identical(other.updatedAt, updatedAt) ||
                other.updatedAt == updatedAt) &&
            (identical(other.paidAt, paidAt) || other.paidAt == paidAt) &&
            const DeepCollectionEquality().equals(other._items, _items));
  }

  @JsonKey(ignore: true)
  @override
  int get hashCode => Object.hash(
      runtimeType,
      id,
      orderNo,
      status,
      totalAmount,
      currency,
      paidAmount,
      createdAt,
      updatedAt,
      paidAt,
      const DeepCollectionEquality().hash(_items));

  @JsonKey(ignore: true)
  @override
  @pragma('vm:prefer-inline')
  _$$OrderImplCopyWith<_$OrderImpl> get copyWith =>
      __$$OrderImplCopyWithImpl<_$OrderImpl>(this, _$identity);

  @override
  Map<String, dynamic> toJson() {
    return _$$OrderImplToJson(
      this,
    );
  }
}

abstract class _Order implements Order {
  const factory _Order(
      {final int? id,
      @JsonKey(name: 'order_no') final String? orderNo,
      final String? status,
      @JsonKey(name: 'total_amount') final double? totalAmount,
      final String? currency,
      @JsonKey(name: 'paid_amount') final double? paidAmount,
      @JsonKey(name: 'created_at') final String? createdAt,
      @JsonKey(name: 'updated_at') final String? updatedAt,
      @JsonKey(name: 'paid_at') final String? paidAt,
      final List<OrderItem>? items}) = _$OrderImpl;

  factory _Order.fromJson(Map<String, dynamic> json) = _$OrderImpl.fromJson;

  @override
  int? get id;
  @override
  @JsonKey(name: 'order_no')
  String? get orderNo;
  @override
  String? get status;
  @override
  @JsonKey(name: 'total_amount')
  double? get totalAmount;
  @override
  String? get currency;
  @override
  @JsonKey(name: 'paid_amount')
  double? get paidAmount;
  @override
  @JsonKey(name: 'created_at')
  String? get createdAt;
  @override
  @JsonKey(name: 'updated_at')
  String? get updatedAt;
  @override
  @JsonKey(name: 'paid_at')
  String? get paidAt;
  @override
  List<OrderItem>? get items;
  @override
  @JsonKey(ignore: true)
  _$$OrderImplCopyWith<_$OrderImpl> get copyWith =>
      throw _privateConstructorUsedError;
}

OrderItem _$OrderItemFromJson(Map<String, dynamic> json) {
  return _OrderItem.fromJson(json);
}

/// @nodoc
mixin _$OrderItem {
  int? get id => throw _privateConstructorUsedError;
  String? get type => throw _privateConstructorUsedError;
  String? get name => throw _privateConstructorUsedError;
  @JsonKey(name: 'spec_text')
  String? get specText => throw _privateConstructorUsedError;
  double? get price => throw _privateConstructorUsedError;
  @JsonKey(name: 'billing_cycle')
  String? get billingCycle => throw _privateConstructorUsedError;
  int? get quantity => throw _privateConstructorUsedError;

  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;
  @JsonKey(ignore: true)
  $OrderItemCopyWith<OrderItem> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $OrderItemCopyWith<$Res> {
  factory $OrderItemCopyWith(OrderItem value, $Res Function(OrderItem) then) =
      _$OrderItemCopyWithImpl<$Res, OrderItem>;
  @useResult
  $Res call(
      {int? id,
      String? type,
      String? name,
      @JsonKey(name: 'spec_text') String? specText,
      double? price,
      @JsonKey(name: 'billing_cycle') String? billingCycle,
      int? quantity});
}

/// @nodoc
class _$OrderItemCopyWithImpl<$Res, $Val extends OrderItem>
    implements $OrderItemCopyWith<$Res> {
  _$OrderItemCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? id = freezed,
    Object? type = freezed,
    Object? name = freezed,
    Object? specText = freezed,
    Object? price = freezed,
    Object? billingCycle = freezed,
    Object? quantity = freezed,
  }) {
    return _then(_value.copyWith(
      id: freezed == id
          ? _value.id
          : id // ignore: cast_nullable_to_non_nullable
              as int?,
      type: freezed == type
          ? _value.type
          : type // ignore: cast_nullable_to_non_nullable
              as String?,
      name: freezed == name
          ? _value.name
          : name // ignore: cast_nullable_to_non_nullable
              as String?,
      specText: freezed == specText
          ? _value.specText
          : specText // ignore: cast_nullable_to_non_nullable
              as String?,
      price: freezed == price
          ? _value.price
          : price // ignore: cast_nullable_to_non_nullable
              as double?,
      billingCycle: freezed == billingCycle
          ? _value.billingCycle
          : billingCycle // ignore: cast_nullable_to_non_nullable
              as String?,
      quantity: freezed == quantity
          ? _value.quantity
          : quantity // ignore: cast_nullable_to_non_nullable
              as int?,
    ) as $Val);
  }
}

/// @nodoc
abstract class _$$OrderItemImplCopyWith<$Res>
    implements $OrderItemCopyWith<$Res> {
  factory _$$OrderItemImplCopyWith(
          _$OrderItemImpl value, $Res Function(_$OrderItemImpl) then) =
      __$$OrderItemImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call(
      {int? id,
      String? type,
      String? name,
      @JsonKey(name: 'spec_text') String? specText,
      double? price,
      @JsonKey(name: 'billing_cycle') String? billingCycle,
      int? quantity});
}

/// @nodoc
class __$$OrderItemImplCopyWithImpl<$Res>
    extends _$OrderItemCopyWithImpl<$Res, _$OrderItemImpl>
    implements _$$OrderItemImplCopyWith<$Res> {
  __$$OrderItemImplCopyWithImpl(
      _$OrderItemImpl _value, $Res Function(_$OrderItemImpl) _then)
      : super(_value, _then);

  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? id = freezed,
    Object? type = freezed,
    Object? name = freezed,
    Object? specText = freezed,
    Object? price = freezed,
    Object? billingCycle = freezed,
    Object? quantity = freezed,
  }) {
    return _then(_$OrderItemImpl(
      id: freezed == id
          ? _value.id
          : id // ignore: cast_nullable_to_non_nullable
              as int?,
      type: freezed == type
          ? _value.type
          : type // ignore: cast_nullable_to_non_nullable
              as String?,
      name: freezed == name
          ? _value.name
          : name // ignore: cast_nullable_to_non_nullable
              as String?,
      specText: freezed == specText
          ? _value.specText
          : specText // ignore: cast_nullable_to_non_nullable
              as String?,
      price: freezed == price
          ? _value.price
          : price // ignore: cast_nullable_to_non_nullable
              as double?,
      billingCycle: freezed == billingCycle
          ? _value.billingCycle
          : billingCycle // ignore: cast_nullable_to_non_nullable
              as String?,
      quantity: freezed == quantity
          ? _value.quantity
          : quantity // ignore: cast_nullable_to_non_nullable
              as int?,
    ));
  }
}

/// @nodoc
@JsonSerializable()
class _$OrderItemImpl implements _OrderItem {
  const _$OrderItemImpl(
      {this.id,
      this.type,
      this.name,
      @JsonKey(name: 'spec_text') this.specText,
      this.price,
      @JsonKey(name: 'billing_cycle') this.billingCycle,
      this.quantity});

  factory _$OrderItemImpl.fromJson(Map<String, dynamic> json) =>
      _$$OrderItemImplFromJson(json);

  @override
  final int? id;
  @override
  final String? type;
  @override
  final String? name;
  @override
  @JsonKey(name: 'spec_text')
  final String? specText;
  @override
  final double? price;
  @override
  @JsonKey(name: 'billing_cycle')
  final String? billingCycle;
  @override
  final int? quantity;

  @override
  String toString() {
    return 'OrderItem(id: $id, type: $type, name: $name, specText: $specText, price: $price, billingCycle: $billingCycle, quantity: $quantity)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$OrderItemImpl &&
            (identical(other.id, id) || other.id == id) &&
            (identical(other.type, type) || other.type == type) &&
            (identical(other.name, name) || other.name == name) &&
            (identical(other.specText, specText) ||
                other.specText == specText) &&
            (identical(other.price, price) || other.price == price) &&
            (identical(other.billingCycle, billingCycle) ||
                other.billingCycle == billingCycle) &&
            (identical(other.quantity, quantity) ||
                other.quantity == quantity));
  }

  @JsonKey(ignore: true)
  @override
  int get hashCode => Object.hash(
      runtimeType, id, type, name, specText, price, billingCycle, quantity);

  @JsonKey(ignore: true)
  @override
  @pragma('vm:prefer-inline')
  _$$OrderItemImplCopyWith<_$OrderItemImpl> get copyWith =>
      __$$OrderItemImplCopyWithImpl<_$OrderItemImpl>(this, _$identity);

  @override
  Map<String, dynamic> toJson() {
    return _$$OrderItemImplToJson(
      this,
    );
  }
}

abstract class _OrderItem implements OrderItem {
  const factory _OrderItem(
      {final int? id,
      final String? type,
      final String? name,
      @JsonKey(name: 'spec_text') final String? specText,
      final double? price,
      @JsonKey(name: 'billing_cycle') final String? billingCycle,
      final int? quantity}) = _$OrderItemImpl;

  factory _OrderItem.fromJson(Map<String, dynamic> json) =
      _$OrderItemImpl.fromJson;

  @override
  int? get id;
  @override
  String? get type;
  @override
  String? get name;
  @override
  @JsonKey(name: 'spec_text')
  String? get specText;
  @override
  double? get price;
  @override
  @JsonKey(name: 'billing_cycle')
  String? get billingCycle;
  @override
  int? get quantity;
  @override
  @JsonKey(ignore: true)
  _$$OrderItemImplCopyWith<_$OrderItemImpl> get copyWith =>
      throw _privateConstructorUsedError;
}

CartItem _$CartItemFromJson(Map<String, dynamic> json) {
  return _CartItem.fromJson(json);
}

/// @nodoc
mixin _$CartItem {
  int? get id => throw _privateConstructorUsedError;
  String? get type => throw _privateConstructorUsedError;
  @JsonKey(name: 'goods_id')
  int? get goodsId => throw _privateConstructorUsedError;
  String? get name => throw _privateConstructorUsedError;
  @JsonKey(name: 'spec_text')
  String? get specText => throw _privateConstructorUsedError;
  double? get price => throw _privateConstructorUsedError;
  @JsonKey(name: 'billing_cycle')
  String? get billingCycle => throw _privateConstructorUsedError;
  @JsonKey(name: 'billing_cycle_display')
  String? get billingCycleDisplay => throw _privateConstructorUsedError;
  int? get quantity => throw _privateConstructorUsedError;
  @JsonKey(name: 'region_name')
  String? get regionName => throw _privateConstructorUsedError;
  @JsonKey(name: 'image_name')
  String? get imageName => throw _privateConstructorUsedError;

  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;
  @JsonKey(ignore: true)
  $CartItemCopyWith<CartItem> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $CartItemCopyWith<$Res> {
  factory $CartItemCopyWith(CartItem value, $Res Function(CartItem) then) =
      _$CartItemCopyWithImpl<$Res, CartItem>;
  @useResult
  $Res call(
      {int? id,
      String? type,
      @JsonKey(name: 'goods_id') int? goodsId,
      String? name,
      @JsonKey(name: 'spec_text') String? specText,
      double? price,
      @JsonKey(name: 'billing_cycle') String? billingCycle,
      @JsonKey(name: 'billing_cycle_display') String? billingCycleDisplay,
      int? quantity,
      @JsonKey(name: 'region_name') String? regionName,
      @JsonKey(name: 'image_name') String? imageName});
}

/// @nodoc
class _$CartItemCopyWithImpl<$Res, $Val extends CartItem>
    implements $CartItemCopyWith<$Res> {
  _$CartItemCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? id = freezed,
    Object? type = freezed,
    Object? goodsId = freezed,
    Object? name = freezed,
    Object? specText = freezed,
    Object? price = freezed,
    Object? billingCycle = freezed,
    Object? billingCycleDisplay = freezed,
    Object? quantity = freezed,
    Object? regionName = freezed,
    Object? imageName = freezed,
  }) {
    return _then(_value.copyWith(
      id: freezed == id
          ? _value.id
          : id // ignore: cast_nullable_to_non_nullable
              as int?,
      type: freezed == type
          ? _value.type
          : type // ignore: cast_nullable_to_non_nullable
              as String?,
      goodsId: freezed == goodsId
          ? _value.goodsId
          : goodsId // ignore: cast_nullable_to_non_nullable
              as int?,
      name: freezed == name
          ? _value.name
          : name // ignore: cast_nullable_to_non_nullable
              as String?,
      specText: freezed == specText
          ? _value.specText
          : specText // ignore: cast_nullable_to_non_nullable
              as String?,
      price: freezed == price
          ? _value.price
          : price // ignore: cast_nullable_to_non_nullable
              as double?,
      billingCycle: freezed == billingCycle
          ? _value.billingCycle
          : billingCycle // ignore: cast_nullable_to_non_nullable
              as String?,
      billingCycleDisplay: freezed == billingCycleDisplay
          ? _value.billingCycleDisplay
          : billingCycleDisplay // ignore: cast_nullable_to_non_nullable
              as String?,
      quantity: freezed == quantity
          ? _value.quantity
          : quantity // ignore: cast_nullable_to_non_nullable
              as int?,
      regionName: freezed == regionName
          ? _value.regionName
          : regionName // ignore: cast_nullable_to_non_nullable
              as String?,
      imageName: freezed == imageName
          ? _value.imageName
          : imageName // ignore: cast_nullable_to_non_nullable
              as String?,
    ) as $Val);
  }
}

/// @nodoc
abstract class _$$CartItemImplCopyWith<$Res>
    implements $CartItemCopyWith<$Res> {
  factory _$$CartItemImplCopyWith(
          _$CartItemImpl value, $Res Function(_$CartItemImpl) then) =
      __$$CartItemImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call(
      {int? id,
      String? type,
      @JsonKey(name: 'goods_id') int? goodsId,
      String? name,
      @JsonKey(name: 'spec_text') String? specText,
      double? price,
      @JsonKey(name: 'billing_cycle') String? billingCycle,
      @JsonKey(name: 'billing_cycle_display') String? billingCycleDisplay,
      int? quantity,
      @JsonKey(name: 'region_name') String? regionName,
      @JsonKey(name: 'image_name') String? imageName});
}

/// @nodoc
class __$$CartItemImplCopyWithImpl<$Res>
    extends _$CartItemCopyWithImpl<$Res, _$CartItemImpl>
    implements _$$CartItemImplCopyWith<$Res> {
  __$$CartItemImplCopyWithImpl(
      _$CartItemImpl _value, $Res Function(_$CartItemImpl) _then)
      : super(_value, _then);

  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? id = freezed,
    Object? type = freezed,
    Object? goodsId = freezed,
    Object? name = freezed,
    Object? specText = freezed,
    Object? price = freezed,
    Object? billingCycle = freezed,
    Object? billingCycleDisplay = freezed,
    Object? quantity = freezed,
    Object? regionName = freezed,
    Object? imageName = freezed,
  }) {
    return _then(_$CartItemImpl(
      id: freezed == id
          ? _value.id
          : id // ignore: cast_nullable_to_non_nullable
              as int?,
      type: freezed == type
          ? _value.type
          : type // ignore: cast_nullable_to_non_nullable
              as String?,
      goodsId: freezed == goodsId
          ? _value.goodsId
          : goodsId // ignore: cast_nullable_to_non_nullable
              as int?,
      name: freezed == name
          ? _value.name
          : name // ignore: cast_nullable_to_non_nullable
              as String?,
      specText: freezed == specText
          ? _value.specText
          : specText // ignore: cast_nullable_to_non_nullable
              as String?,
      price: freezed == price
          ? _value.price
          : price // ignore: cast_nullable_to_non_nullable
              as double?,
      billingCycle: freezed == billingCycle
          ? _value.billingCycle
          : billingCycle // ignore: cast_nullable_to_non_nullable
              as String?,
      billingCycleDisplay: freezed == billingCycleDisplay
          ? _value.billingCycleDisplay
          : billingCycleDisplay // ignore: cast_nullable_to_non_nullable
              as String?,
      quantity: freezed == quantity
          ? _value.quantity
          : quantity // ignore: cast_nullable_to_non_nullable
              as int?,
      regionName: freezed == regionName
          ? _value.regionName
          : regionName // ignore: cast_nullable_to_non_nullable
              as String?,
      imageName: freezed == imageName
          ? _value.imageName
          : imageName // ignore: cast_nullable_to_non_nullable
              as String?,
    ));
  }
}

/// @nodoc
@JsonSerializable()
class _$CartItemImpl implements _CartItem {
  const _$CartItemImpl(
      {this.id,
      this.type,
      @JsonKey(name: 'goods_id') this.goodsId,
      this.name,
      @JsonKey(name: 'spec_text') this.specText,
      this.price,
      @JsonKey(name: 'billing_cycle') this.billingCycle,
      @JsonKey(name: 'billing_cycle_display') this.billingCycleDisplay,
      this.quantity,
      @JsonKey(name: 'region_name') this.regionName,
      @JsonKey(name: 'image_name') this.imageName});

  factory _$CartItemImpl.fromJson(Map<String, dynamic> json) =>
      _$$CartItemImplFromJson(json);

  @override
  final int? id;
  @override
  final String? type;
  @override
  @JsonKey(name: 'goods_id')
  final int? goodsId;
  @override
  final String? name;
  @override
  @JsonKey(name: 'spec_text')
  final String? specText;
  @override
  final double? price;
  @override
  @JsonKey(name: 'billing_cycle')
  final String? billingCycle;
  @override
  @JsonKey(name: 'billing_cycle_display')
  final String? billingCycleDisplay;
  @override
  final int? quantity;
  @override
  @JsonKey(name: 'region_name')
  final String? regionName;
  @override
  @JsonKey(name: 'image_name')
  final String? imageName;

  @override
  String toString() {
    return 'CartItem(id: $id, type: $type, goodsId: $goodsId, name: $name, specText: $specText, price: $price, billingCycle: $billingCycle, billingCycleDisplay: $billingCycleDisplay, quantity: $quantity, regionName: $regionName, imageName: $imageName)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$CartItemImpl &&
            (identical(other.id, id) || other.id == id) &&
            (identical(other.type, type) || other.type == type) &&
            (identical(other.goodsId, goodsId) || other.goodsId == goodsId) &&
            (identical(other.name, name) || other.name == name) &&
            (identical(other.specText, specText) ||
                other.specText == specText) &&
            (identical(other.price, price) || other.price == price) &&
            (identical(other.billingCycle, billingCycle) ||
                other.billingCycle == billingCycle) &&
            (identical(other.billingCycleDisplay, billingCycleDisplay) ||
                other.billingCycleDisplay == billingCycleDisplay) &&
            (identical(other.quantity, quantity) ||
                other.quantity == quantity) &&
            (identical(other.regionName, regionName) ||
                other.regionName == regionName) &&
            (identical(other.imageName, imageName) ||
                other.imageName == imageName));
  }

  @JsonKey(ignore: true)
  @override
  int get hashCode => Object.hash(
      runtimeType,
      id,
      type,
      goodsId,
      name,
      specText,
      price,
      billingCycle,
      billingCycleDisplay,
      quantity,
      regionName,
      imageName);

  @JsonKey(ignore: true)
  @override
  @pragma('vm:prefer-inline')
  _$$CartItemImplCopyWith<_$CartItemImpl> get copyWith =>
      __$$CartItemImplCopyWithImpl<_$CartItemImpl>(this, _$identity);

  @override
  Map<String, dynamic> toJson() {
    return _$$CartItemImplToJson(
      this,
    );
  }
}

abstract class _CartItem implements CartItem {
  const factory _CartItem(
      {final int? id,
      final String? type,
      @JsonKey(name: 'goods_id') final int? goodsId,
      final String? name,
      @JsonKey(name: 'spec_text') final String? specText,
      final double? price,
      @JsonKey(name: 'billing_cycle') final String? billingCycle,
      @JsonKey(name: 'billing_cycle_display') final String? billingCycleDisplay,
      final int? quantity,
      @JsonKey(name: 'region_name') final String? regionName,
      @JsonKey(name: 'image_name') final String? imageName}) = _$CartItemImpl;

  factory _CartItem.fromJson(Map<String, dynamic> json) =
      _$CartItemImpl.fromJson;

  @override
  int? get id;
  @override
  String? get type;
  @override
  @JsonKey(name: 'goods_id')
  int? get goodsId;
  @override
  String? get name;
  @override
  @JsonKey(name: 'spec_text')
  String? get specText;
  @override
  double? get price;
  @override
  @JsonKey(name: 'billing_cycle')
  String? get billingCycle;
  @override
  @JsonKey(name: 'billing_cycle_display')
  String? get billingCycleDisplay;
  @override
  int? get quantity;
  @override
  @JsonKey(name: 'region_name')
  String? get regionName;
  @override
  @JsonKey(name: 'image_name')
  String? get imageName;
  @override
  @JsonKey(ignore: true)
  _$$CartItemImplCopyWith<_$CartItemImpl> get copyWith =>
      throw _privateConstructorUsedError;
}

CartSummary _$CartSummaryFromJson(Map<String, dynamic> json) {
  return _CartSummary.fromJson(json);
}

/// @nodoc
mixin _$CartSummary {
  @JsonKey(name: 'total_amount')
  double? get totalAmount => throw _privateConstructorUsedError;
  @JsonKey(name: 'item_count')
  int? get itemCount => throw _privateConstructorUsedError;
  String? get currency => throw _privateConstructorUsedError;

  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;
  @JsonKey(ignore: true)
  $CartSummaryCopyWith<CartSummary> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $CartSummaryCopyWith<$Res> {
  factory $CartSummaryCopyWith(
          CartSummary value, $Res Function(CartSummary) then) =
      _$CartSummaryCopyWithImpl<$Res, CartSummary>;
  @useResult
  $Res call(
      {@JsonKey(name: 'total_amount') double? totalAmount,
      @JsonKey(name: 'item_count') int? itemCount,
      String? currency});
}

/// @nodoc
class _$CartSummaryCopyWithImpl<$Res, $Val extends CartSummary>
    implements $CartSummaryCopyWith<$Res> {
  _$CartSummaryCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? totalAmount = freezed,
    Object? itemCount = freezed,
    Object? currency = freezed,
  }) {
    return _then(_value.copyWith(
      totalAmount: freezed == totalAmount
          ? _value.totalAmount
          : totalAmount // ignore: cast_nullable_to_non_nullable
              as double?,
      itemCount: freezed == itemCount
          ? _value.itemCount
          : itemCount // ignore: cast_nullable_to_non_nullable
              as int?,
      currency: freezed == currency
          ? _value.currency
          : currency // ignore: cast_nullable_to_non_nullable
              as String?,
    ) as $Val);
  }
}

/// @nodoc
abstract class _$$CartSummaryImplCopyWith<$Res>
    implements $CartSummaryCopyWith<$Res> {
  factory _$$CartSummaryImplCopyWith(
          _$CartSummaryImpl value, $Res Function(_$CartSummaryImpl) then) =
      __$$CartSummaryImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call(
      {@JsonKey(name: 'total_amount') double? totalAmount,
      @JsonKey(name: 'item_count') int? itemCount,
      String? currency});
}

/// @nodoc
class __$$CartSummaryImplCopyWithImpl<$Res>
    extends _$CartSummaryCopyWithImpl<$Res, _$CartSummaryImpl>
    implements _$$CartSummaryImplCopyWith<$Res> {
  __$$CartSummaryImplCopyWithImpl(
      _$CartSummaryImpl _value, $Res Function(_$CartSummaryImpl) _then)
      : super(_value, _then);

  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? totalAmount = freezed,
    Object? itemCount = freezed,
    Object? currency = freezed,
  }) {
    return _then(_$CartSummaryImpl(
      totalAmount: freezed == totalAmount
          ? _value.totalAmount
          : totalAmount // ignore: cast_nullable_to_non_nullable
              as double?,
      itemCount: freezed == itemCount
          ? _value.itemCount
          : itemCount // ignore: cast_nullable_to_non_nullable
              as int?,
      currency: freezed == currency
          ? _value.currency
          : currency // ignore: cast_nullable_to_non_nullable
              as String?,
    ));
  }
}

/// @nodoc
@JsonSerializable()
class _$CartSummaryImpl implements _CartSummary {
  const _$CartSummaryImpl(
      {@JsonKey(name: 'total_amount') this.totalAmount,
      @JsonKey(name: 'item_count') this.itemCount,
      this.currency});

  factory _$CartSummaryImpl.fromJson(Map<String, dynamic> json) =>
      _$$CartSummaryImplFromJson(json);

  @override
  @JsonKey(name: 'total_amount')
  final double? totalAmount;
  @override
  @JsonKey(name: 'item_count')
  final int? itemCount;
  @override
  final String? currency;

  @override
  String toString() {
    return 'CartSummary(totalAmount: $totalAmount, itemCount: $itemCount, currency: $currency)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$CartSummaryImpl &&
            (identical(other.totalAmount, totalAmount) ||
                other.totalAmount == totalAmount) &&
            (identical(other.itemCount, itemCount) ||
                other.itemCount == itemCount) &&
            (identical(other.currency, currency) ||
                other.currency == currency));
  }

  @JsonKey(ignore: true)
  @override
  int get hashCode =>
      Object.hash(runtimeType, totalAmount, itemCount, currency);

  @JsonKey(ignore: true)
  @override
  @pragma('vm:prefer-inline')
  _$$CartSummaryImplCopyWith<_$CartSummaryImpl> get copyWith =>
      __$$CartSummaryImplCopyWithImpl<_$CartSummaryImpl>(this, _$identity);

  @override
  Map<String, dynamic> toJson() {
    return _$$CartSummaryImplToJson(
      this,
    );
  }
}

abstract class _CartSummary implements CartSummary {
  const factory _CartSummary(
      {@JsonKey(name: 'total_amount') final double? totalAmount,
      @JsonKey(name: 'item_count') final int? itemCount,
      final String? currency}) = _$CartSummaryImpl;

  factory _CartSummary.fromJson(Map<String, dynamic> json) =
      _$CartSummaryImpl.fromJson;

  @override
  @JsonKey(name: 'total_amount')
  double? get totalAmount;
  @override
  @JsonKey(name: 'item_count')
  int? get itemCount;
  @override
  String? get currency;
  @override
  @JsonKey(ignore: true)
  _$$CartSummaryImplCopyWith<_$CartSummaryImpl> get copyWith =>
      throw _privateConstructorUsedError;
}
