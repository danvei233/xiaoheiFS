import 'package:dio/dio.dart';
import '../../core/constants/api_endpoints.dart';
import '../../core/network/api_client.dart';
import '../../core/utils/map_utils.dart';

class WalletRepository {
  final Dio _dio = ApiClient.instance.dio;

  Future<Map<String, dynamic>> getWallet() async {
    final response = await _dio.get(ApiEndpoints.wallet);
    return ensureMap(response.data);
  }

  Future<Map<String, dynamic>> listWalletOrders({
    int? limit,
    int? offset,
  }) async {
    final response = await _dio.get(
      ApiEndpoints.walletOrders,
      queryParameters: {
        if (limit != null) 'limit': limit,
        if (offset != null) 'offset': offset,
      },
    );
    return ensureMap(response.data);
  }

  Future<Map<String, dynamic>> listWalletTransactions({
    int? limit,
    int? offset,
  }) async {
    final response = await _dio.get(
      ApiEndpoints.walletTransactions,
      queryParameters: {
        if (limit != null) 'limit': limit,
        if (offset != null) 'offset': offset,
      },
    );
    return ensureMap(response.data);
  }

  Future<Map<String, dynamic>> createRecharge(Map<String, dynamic> payload) async {
    final response = await _dio.post(ApiEndpoints.walletRecharge, data: payload);
    return ensureMap(response.data);
  }

  Future<Map<String, dynamic>> createWithdraw(Map<String, dynamic> payload) async {
    final response = await _dio.post(ApiEndpoints.walletWithdraw, data: payload);
    return ensureMap(response.data);
  }
}
