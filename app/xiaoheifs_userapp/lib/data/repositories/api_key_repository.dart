import 'package:dio/dio.dart';

import '../../core/network/api_client.dart';
import '../../core/utils/map_utils.dart';

class ApiKeyRepository {
  final Dio _dio = ApiClient.instance.dio;
  static const String _apiKeysPath = '/v1/open/me/api-keys';
  static String _apiKeyDetailPath(int id) => '/v1/open/me/api-keys/$id';

  Future<Map<String, dynamic>> listApiKeys({int? limit, int? offset}) async {
    final queryParameters = <String, dynamic>{};
    if (limit != null) {
      queryParameters['limit'] = limit;
    }
    if (offset != null) {
      queryParameters['offset'] = offset;
    }

    final response = await _dio.get(
      _apiKeysPath,
      queryParameters: queryParameters,
    );
    return _unwrapData(response.data, wrapList: true);
  }

  Future<Map<String, dynamic>> createApiKey({
    required String name,
    List<String>? scopes,
  }) async {
    final response = await _dio.post(
      _apiKeysPath,
      data: {'name': name, 'scopes': scopes ?? <String>[]},
    );
    return _unwrapData(response.data);
  }

  Future<void> updateApiKeyStatus(int id, String status) async {
    await _dio.patch(_apiKeyDetailPath(id), data: {'status': status});
  }

  Future<void> deleteApiKey(int id) async {
    await _dio.delete(_apiKeyDetailPath(id));
  }

  Map<String, dynamic> _unwrapData(dynamic data, {bool wrapList = false}) {
    if (data is List) {
      return wrapList ? {'items': data.map((e) => ensureMap(e)).toList()} : {};
    }

    final map = ensureMap(data);
    if (map.containsKey('data')) {
      final inner = map['data'];
      if (inner is List) {
        return wrapList
            ? {'items': inner.map((e) => ensureMap(e)).toList()}
            : {};
      }
      if (inner is Map) {
        return ensureMap(inner);
      }
    }
    return map;
  }
}
