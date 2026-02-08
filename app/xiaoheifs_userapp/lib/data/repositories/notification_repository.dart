import 'package:dio/dio.dart';
import '../../core/constants/api_endpoints.dart';
import '../../core/network/api_client.dart';
import '../../core/utils/map_utils.dart';

class NotificationRepository {
  final Dio _dio = ApiClient.instance.dio;

  Future<Map<String, dynamic>> listNotifications({String? status, int? limit, int? offset}) async {
    final response = await _dio.get(
      ApiEndpoints.notifications,
      queryParameters: {
        if (status != null && status.isNotEmpty && status != 'all') 'status': status,
        if (limit != null) 'limit': limit,
        if (offset != null) 'offset': offset,
      },
    );
    return _unwrapData(response.data, wrapList: true);
  }

  Future<Map<String, dynamic>> getUnreadCount() async {
    final response = await _dio.get(ApiEndpoints.notificationsUnreadCount);
    return _unwrapData(response.data);
  }

  Future<void> markRead(int id) async {
    await _dio.post(ApiEndpoints.notificationRead(id));
  }

  Future<void> markAllRead() async {
    await _dio.post(ApiEndpoints.notificationsReadAll);
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
