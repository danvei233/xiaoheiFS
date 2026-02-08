import 'package:freezed_annotation/freezed_annotation.dart';

part 'order.freezed.dart';
part 'order.g.dart';

/// 订单模型
@freezed
class Order with _$Order {
  const factory Order({
    int? id,
    @JsonKey(name: 'order_no') String? orderNo,
    String? status,
    @JsonKey(name: 'total_amount') double? totalAmount,
    String? currency,
    @JsonKey(name: 'paid_amount') double? paidAmount,
    @JsonKey(name: 'created_at') String? createdAt,
    @JsonKey(name: 'updated_at') String? updatedAt,
    @JsonKey(name: 'paid_at') String? paidAt,
    List<OrderItem>? items,
  }) = _Order;

  factory Order.fromJson(Map<String, dynamic> json) => _$OrderFromJson(json);
}

/// 订单项模型
@freezed
class OrderItem with _$OrderItem {
  const factory OrderItem({
    int? id,
    String? type,
    String? name,
    @JsonKey(name: 'spec_text') String? specText,
    double? price,
    @JsonKey(name: 'billing_cycle') String? billingCycle,
    int? quantity,
  }) = _OrderItem;

  factory OrderItem.fromJson(Map<String, dynamic> json) =>
      _$OrderItemFromJson(json);
}

/// 购物车项模型
@freezed
class CartItem with _$CartItem {
  const factory CartItem({
    int? id,
    String? type,
    @JsonKey(name: 'goods_id') int? goodsId,
    String? name,
    @JsonKey(name: 'spec_text') String? specText,
    double? price,
    @JsonKey(name: 'billing_cycle') String? billingCycle,
    @JsonKey(name: 'billing_cycle_display') String? billingCycleDisplay,
    int? quantity,
    @JsonKey(name: 'region_name') String? regionName,
    @JsonKey(name: 'image_name') String? imageName,
  }) = _CartItem;

  factory CartItem.fromJson(Map<String, dynamic> json) =>
      _$CartItemFromJson(json);
}

/// 购物车汇总信息
@freezed
class CartSummary with _$CartSummary {
  const factory CartSummary({
    @JsonKey(name: 'total_amount') double? totalAmount,
    @JsonKey(name: 'item_count') int? itemCount,
    String? currency,
  }) = _CartSummary;

  factory CartSummary.fromJson(Map<String, dynamic> json) =>
      _$CartSummaryFromJson(json);
}
