import 'package:dio/dio.dart';
import '../../core/constants/api_endpoints.dart';
import '../../core/network/api_client.dart';
import '../../core/utils/map_utils.dart';

class NotificationsRepository {
  final Dio _dio = ApiClient.instance.dio;

  Future<Map<String, dynamic>> listNotifications({
    String? status,
    int? limit,
    int? offset,
  }) async {
    final response = await _dio.get(
      ApiEndpoints.notifications,
      queryParameters: {
        if (status != null && status.isNotEmpty) 'status': status,
        if (limit != null) 'limit': limit,
        if (offset != null) 'offset': offset,
      },
    );
    return ensureMap(response.data);
  }

  Future<int> getUnreadCount() async {
    final response = await _dio.get(ApiEndpoints.notificationsUnreadCount);
    final data = ensureMap(response.data);
    return parseUnreadCount(data['unread']);
  }

  /// 容错解析未读数量：服务端可能返回字符串数字（如 `"3"`），裸 `as int`
  /// 会抛类型错误导致消息中心崩溃。
  static int parseUnreadCount(dynamic value) {
    if (value is int) return value;
    if (value is num) return value.toInt();
    if (value is String) return int.tryParse(value.trim()) ?? 0;
    return 0;
  }

  Future<void> markRead(int id) async {
    await _dio.post(ApiEndpoints.notificationRead(id));
  }

  Future<void> markAllRead() async {
    await _dio.post(ApiEndpoints.notificationsReadAll);
  }
}
