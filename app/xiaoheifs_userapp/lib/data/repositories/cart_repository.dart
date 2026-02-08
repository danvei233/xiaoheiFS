import 'package:dio/dio.dart';
import '../../core/constants/api_endpoints.dart';
import '../../core/network/api_client.dart';
import '../../core/utils/map_utils.dart';

class CartRepository {
  final Dio _dio = ApiClient.instance.dio;

  Future<Map<String, dynamic>> listCart() async {
    final response = await _dio.get(ApiEndpoints.cart);
    return ensureMap(response.data);
  }

  Future<void> addItem(Map<String, dynamic> payload) async {
    await _dio.post(ApiEndpoints.cart, data: payload);
  }

  Future<void> updateItem(int id, Map<String, dynamic> payload) async {
    await _dio.patch(ApiEndpoints.cartItem(id), data: payload);
  }

  Future<void> deleteItem(int id) async {
    await _dio.delete(ApiEndpoints.cartItem(id));
  }

  Future<void> clear() async {
    await _dio.delete(ApiEndpoints.cart);
  }
}
