import 'package:dio/dio.dart';
import '../../core/constants/api_endpoints.dart';
import '../../core/network/api_client.dart';
import '../../core/utils/map_utils.dart';

class DashboardRepository {
  final Dio _dio = ApiClient.instance.dio;

  Future<Map<String, dynamic>> getDashboard() async {
    final response = await _dio.get(ApiEndpoints.dashboard);
    return _unwrapData(response.data);
  }

  Future<Map<String, dynamic>> listOrders({int? limit, int? offset}) async {
    final response = await _dio.get(
      ApiEndpoints.orders,
      queryParameters: {
        if (limit != null) 'limit': limit,
        if (offset != null) 'offset': offset,
      },
    );
    return _unwrapData(response.data, wrapList: true);
  }

  Future<Map<String, dynamic>> listVps() async {
    final response = await _dio.get(ApiEndpoints.vps);
    return _unwrapData(response.data, wrapList: true);
  }

  Future<Map<String, dynamic>> getWallet() async {
    final response = await _dio.get(ApiEndpoints.wallet);
    return _unwrapData(response.data);
  }

  Future<Map<String, dynamic>> getRealnameStatus() async {
    final response = await _dio.get(ApiEndpoints.realnameStatus);
    return _unwrapData(response.data);
  }

  Map<String, dynamic> _unwrapData(dynamic data, {bool wrapList = false}) {
    if (data is List) {
      return wrapList
          ? {'items': data.map((e) => ensureMap(e)).toList()}
          : {};
    }
    final map = ensureMap(data);
    if (map.containsKey('data')) {
      final inner = map['data'];
      if (inner is List) {
        return wrapList
            ? {'items': inner.map((e) => ensureMap(e)).toList()}
            : {};
      }
      if (inner is Map) return ensureMap(inner);
    }
    return map;
  }
}
