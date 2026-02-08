// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'order.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

_$OrderImpl _$$OrderImplFromJson(Map<String, dynamic> json) => _$OrderImpl(
      id: (json['id'] as num?)?.toInt(),
      orderNo: json['order_no'] as String?,
      status: json['status'] as String?,
      totalAmount: (json['total_amount'] as num?)?.toDouble(),
      currency: json['currency'] as String?,
      paidAmount: (json['paid_amount'] as num?)?.toDouble(),
      createdAt: json['created_at'] as String?,
      updatedAt: json['updated_at'] as String?,
      paidAt: json['paid_at'] as String?,
      items: (json['items'] as List<dynamic>?)
          ?.map((e) => OrderItem.fromJson(e as Map<String, dynamic>))
          .toList(),
    );

Map<String, dynamic> _$$OrderImplToJson(_$OrderImpl instance) =>
    <String, dynamic>{
      'id': instance.id,
      'order_no': instance.orderNo,
      'status': instance.status,
      'total_amount': instance.totalAmount,
      'currency': instance.currency,
      'paid_amount': instance.paidAmount,
      'created_at': instance.createdAt,
      'updated_at': instance.updatedAt,
      'paid_at': instance.paidAt,
      'items': instance.items,
    };

_$OrderItemImpl _$$OrderItemImplFromJson(Map<String, dynamic> json) =>
    _$OrderItemImpl(
      id: (json['id'] as num?)?.toInt(),
      type: json['type'] as String?,
      name: json['name'] as String?,
      specText: json['spec_text'] as String?,
      price: (json['price'] as num?)?.toDouble(),
      billingCycle: json['billing_cycle'] as String?,
      quantity: (json['quantity'] as num?)?.toInt(),
    );

Map<String, dynamic> _$$OrderItemImplToJson(_$OrderItemImpl instance) =>
    <String, dynamic>{
      'id': instance.id,
      'type': instance.type,
      'name': instance.name,
      'spec_text': instance.specText,
      'price': instance.price,
      'billing_cycle': instance.billingCycle,
      'quantity': instance.quantity,
    };

_$CartItemImpl _$$CartItemImplFromJson(Map<String, dynamic> json) =>
    _$CartItemImpl(
      id: (json['id'] as num?)?.toInt(),
      type: json['type'] as String?,
      goodsId: (json['goods_id'] as num?)?.toInt(),
      name: json['name'] as String?,
      specText: json['spec_text'] as String?,
      price: (json['price'] as num?)?.toDouble(),
      billingCycle: json['billing_cycle'] as String?,
      billingCycleDisplay: json['billing_cycle_display'] as String?,
      quantity: (json['quantity'] as num?)?.toInt(),
      regionName: json['region_name'] as String?,
      imageName: json['image_name'] as String?,
    );

Map<String, dynamic> _$$CartItemImplToJson(_$CartItemImpl instance) =>
    <String, dynamic>{
      'id': instance.id,
      'type': instance.type,
      'goods_id': instance.goodsId,
      'name': instance.name,
      'spec_text': instance.specText,
      'price': instance.price,
      'billing_cycle': instance.billingCycle,
      'billing_cycle_display': instance.billingCycleDisplay,
      'quantity': instance.quantity,
      'region_name': instance.regionName,
      'image_name': instance.imageName,
    };

_$CartSummaryImpl _$$CartSummaryImplFromJson(Map<String, dynamic> json) =>
    _$CartSummaryImpl(
      totalAmount: (json['total_amount'] as num?)?.toDouble(),
      itemCount: (json['item_count'] as num?)?.toInt(),
      currency: json['currency'] as String?,
    );

Map<String, dynamic> _$$CartSummaryImplToJson(_$CartSummaryImpl instance) =>
    <String, dynamic>{
      'total_amount': instance.totalAmount,
      'item_count': instance.itemCount,
      'currency': instance.currency,
    };
