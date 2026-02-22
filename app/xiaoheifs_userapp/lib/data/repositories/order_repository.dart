import 'package:dio/dio.dart';
import '../../core/constants/api_endpoints.dart';
import '../../core/network/api_client.dart';
import '../../core/utils/map_utils.dart';

class OrderRepository {
  final Dio _dio = ApiClient.instance.dio;

  Future<Map<String, dynamic>> listOrders({
    String? status,
    int? limit,
    int? offset,
  }) async {
    final response = await _dio.get(
      ApiEndpoints.orders,
      queryParameters: {
        if (status != null && status.isNotEmpty) 'status': status,
        if (limit != null) 'limit': limit,
        if (offset != null) 'offset': offset,
      },
    );
    return ensureMap(response.data);
  }

  Future<Map<String, dynamic>> getOrderDetail(int id) async {
    final response = await _dio.get(ApiEndpoints.orderDetail(id));
    return ensureMap(response.data);
  }

  Future<Map<String, dynamic>> createOrderFromCart({
    String? idempotencyKey,
  }) async {
    final response = await _dio.post(
      ApiEndpoints.orders,
      options: Options(
        headers: {
          if (idempotencyKey != null) 'Idempotency-Key': idempotencyKey,
        },
      ),
    );
    return ensureMap(response.data);
  }

  Future<Map<String, dynamic>> createOrderItems(
    Map<String, dynamic> payload, {
    String? idempotencyKey,
  }) async {
    final response = await _dio.post(
      ApiEndpoints.ordersItems,
      data: payload,
      options: Options(
        headers: {
          if (idempotencyKey != null) 'Idempotency-Key': idempotencyKey,
        },
      ),
    );
    return ensureMap(response.data);
  }

  Future<void> cancelOrder(int id) async {
    await _dio.post(ApiEndpoints.orderCancel(id));
  }

  Future<void> refreshOrder(int id) async {
    await _dio.post(ApiEndpoints.orderRefresh(id));
  }

  Future<Map<String, dynamic>> submitOrderPayment(
    int id,
    Map<String, dynamic> payload, {
    String? idempotencyKey,
  }) async {
    final response = await _dio.post(
      ApiEndpoints.orderPayments(id),
      data: payload,
      options: Options(
        headers: {
          if (idempotencyKey != null) 'Idempotency-Key': idempotencyKey,
        },
      ),
    );
    return ensureMap(response.data);
  }

  Future<Map<String, dynamic>> createOrderPayment(
    int id,
    Map<String, dynamic> payload,
  ) async {
    final response = await _dio.post(ApiEndpoints.orderPay(id), data: payload);
    return ensureMap(response.data);
  }

  Future<Map<String, dynamic>> previewCoupon(
    Map<String, dynamic> payload,
  ) async {
    final response = await _dio.post(
      ApiEndpoints.couponsPreview,
      data: payload,
    );
    return ensureMap(response.data);
  }

  Future<Map<String, dynamic>> listPaymentProviders() async {
    final response = await _dio.get(ApiEndpoints.paymentProviders);
    return ensureMap(response.data);
  }
}
