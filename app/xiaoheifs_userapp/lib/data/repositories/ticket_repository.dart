import 'package:dio/dio.dart';
import '../../core/constants/api_endpoints.dart';
import '../../core/network/api_client.dart';
import '../../core/utils/map_utils.dart';

class TicketRepository {
  final Dio _dio = ApiClient.instance.dio;

  Future<Map<String, dynamic>> listTickets({
    String? status,
    int? limit,
    int? offset,
  }) async {
    final response = await _dio.get(
      ApiEndpoints.tickets,
      queryParameters: {
        if (status != null && status.isNotEmpty) 'status': status,
        if (limit != null) 'limit': limit,
        if (offset != null) 'offset': offset,
      },
    );
    return ensureMap(response.data);
  }

  Future<Map<String, dynamic>> createTicket(Map<String, dynamic> payload) async {
    final response = await _dio.post(ApiEndpoints.tickets, data: payload);
    return ensureMap(response.data);
  }

  Future<Map<String, dynamic>> getDetail(int id) async {
    final response = await _dio.get(ApiEndpoints.ticketDetail(id));
    return ensureMap(response.data);
  }

  Future<void> addMessage(int id, Map<String, dynamic> payload) async {
    await _dio.post(ApiEndpoints.ticketMessages(id), data: payload);
  }

  Future<void> closeTicket(int id) async {
    await _dio.post(ApiEndpoints.ticketClose(id));
  }
}
