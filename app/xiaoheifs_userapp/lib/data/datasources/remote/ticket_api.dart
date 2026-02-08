import 'package:dio/dio.dart';
import '../../../core/network/api_client.dart';
import '../../../core/constants/api_endpoints.dart';
import '../../models/ticket.dart';

/// 工单API服务
class TicketApi {
  final Dio _dio = ApiClient.instance.dio;

  /// 获取工单列表
  Future<List<Ticket>> getTickets({
    String? status,
    int? page,
    int? pageSize,
  }) async {
    final response = await _dio.get(
      ApiEndpoints.tickets,
      queryParameters: {
        if (status != null) 'status': status,
        if (page != null) 'page': page,
        if (pageSize != null) 'page_size': pageSize,
      },
    );
    final list = response.data as List;
    return list.map((e) => Ticket.fromJson(e)).toList();
  }

  /// 获取工单详情
  Future<Ticket> getTicketDetail(int id) async {
    final response = await _dio.get(ApiEndpoints.ticketDetail(id));
    return Ticket.fromJson(response.data);
  }

  /// 创建工单
  Future<Ticket> createTicket({
    required String subject,
    required String content,
    List<TicketResource>? resources,
  }) async {
    final response = await _dio.post(
      ApiEndpoints.tickets,
      data: {
        'subject': subject,
        'content': content,
        if (resources != null && resources.isNotEmpty)
          'resources': resources.map((e) => e.toJson()).toList(),
      },
    );
    return Ticket.fromJson(response.data);
  }

  /// 发送工单消息
  Future<TicketMessage> sendMessage({
    required int ticketId,
    required String content,
  }) async {
    final response = await _dio.post(
      ApiEndpoints.ticketMessages(ticketId),
      data: {'content': content},
    );
    return TicketMessage.fromJson(response.data);
  }

  /// 关闭工单
  Future<void> closeTicket(int id) async {
    await _dio.post(ApiEndpoints.ticketClose(id));
  }
}
