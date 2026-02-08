import 'package:dio/dio.dart';
import '../../core/constants/api_endpoints.dart';
import '../../core/network/api_client.dart';
import '../../core/utils/map_utils.dart';

class SiteRepository {
  final Dio _dio = ApiClient.instance.dio;

  Future<Map<String, dynamic>> getSiteSettings() async {
    final response = await _dio.get(ApiEndpoints.siteSettings);
    return ensureMap(response.data);
  }
}
