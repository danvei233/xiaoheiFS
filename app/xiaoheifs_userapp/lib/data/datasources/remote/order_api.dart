import 'package:dio/dio.dart';
import '../../../core/network/api_client.dart';
import '../../../core/constants/api_endpoints.dart';
import '../../models/order.dart';

/// 订单API服务
class OrderApi {
  final Dio _dio = ApiClient.instance.dio;

  /// 获取订单列表
  Future<List<Order>> getOrders({
    String? status,
    int? page,
    int? pageSize,
  }) async {
    final response = await _dio.get(
      ApiEndpoints.orders,
      queryParameters: {
        if (status != null) 'status': status,
        if (page != null) 'page': page,
        if (pageSize != null) 'page_size': pageSize,
      },
    );
    final list = response.data as List;
    return list.map((e) => Order.fromJson(e)).toList();
  }

  /// 获取订单详情
  Future<Order> getOrderDetail(int id) async {
    final response = await _dio.get(ApiEndpoints.orderDetail(id));
    return Order.fromJson(response.data);
  }

  /// 创建订单（从购物车）
  Future<Order> createOrder() async {
    final response = await _dio.post(ApiEndpoints.orders);
    return Order.fromJson(response.data);
  }

  /// 取消订单
  Future<void> cancelOrder(int id) async {
    await _dio.post(ApiEndpoints.orderCancel(id));
  }

  /// 支付订单
  Future<Order> payOrder(int id, {
    required String paymentMethod,
  }) async {
    final response = await _dio.post(
      ApiEndpoints.orderPay(id),
      data: {'payment_method': paymentMethod},
    );
    return Order.fromJson(response.data);
  }
}

/// 购物车API服务
class CartApi {
  final Dio _dio = ApiClient.instance.dio;

  /// 获取购物车列表
  Future<List<CartItem>> getCart() async {
    final response = await _dio.get(ApiEndpoints.cart);
    final list = response.data as List;
    return list.map((e) => CartItem.fromJson(e)).toList();
  }

  /// 添加到购物车
  Future<CartItem> addToCart({
    required String type,
    required int goodsId,
    String? billingCycle,
    int? quantity,
    Map<String, dynamic>? config,
  }) async {
    final response = await _dio.post(
      ApiEndpoints.cart,
      data: {
        'type': type,
        'goods_id': goodsId,
        if (billingCycle != null) 'billing_cycle': billingCycle,
        if (quantity != null) 'quantity': quantity,
        if (config != null) ...config,
      },
    );
    return CartItem.fromJson(response.data);
  }

  /// 更新购物车项
  Future<CartItem> updateCartItem(int id, {
    int? quantity,
    String? billingCycle,
  }) async {
    final response = await _dio.patch(
      ApiEndpoints.cartItem(id),
      data: {
        if (quantity != null) 'quantity': quantity,
        if (billingCycle != null) 'billing_cycle': billingCycle,
      },
    );
    return CartItem.fromJson(response.data);
  }

  /// 删除购物车项
  Future<void> removeCartItem(int id) async {
    await _dio.delete(ApiEndpoints.cartItem(id));
  }

  /// 清空购物车
  Future<void> clearCart() async {
    await _dio.delete(ApiEndpoints.cart);
  }
}
