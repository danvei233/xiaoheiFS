import 'package:dio/dio.dart';
import '../../core/constants/api_endpoints.dart';
import '../../core/network/api_client.dart';
import '../../core/utils/map_utils.dart';

class RealnameRepository {
  final Dio _dio = ApiClient.instance.dio;

  Future<Map<String, dynamic>> getStatus() async {
    final response = await _dio.get(ApiEndpoints.realnameStatus);
    return ensureMap(response.data);
  }

  Future<Map<String, dynamic>> submit(Map<String, dynamic> payload) async {
    final response = await _dio.post(ApiEndpoints.realnameVerify, data: payload);
    return ensureMap(response.data);
  }
}
