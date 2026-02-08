import 'package:dio/dio.dart';
import '../../../core/network/api_client.dart';
import '../../../core/constants/api_endpoints.dart';
import '../../models/vps_instance.dart';

/// VPS API服务
class VpsApi {
  final Dio _dio = ApiClient.instance.dio;

  /// 获取VPS列表
  Future<List<VpsInstance>> getVpsList() async {
    final response = await _dio.get(ApiEndpoints.vps);
    final list = response.data as List;
    return list.map((e) => VpsInstance.fromJson(e)).toList();
  }

  /// 获取VPS详情
  Future<VpsInstance> getVpsDetail(int id) async {
    final response = await _dio.get(ApiEndpoints.vpsDetail(id));
    return VpsInstance.fromJson(response.data);
  }

  /// 刷新VPS状态
  Future<VpsInstance> refreshVps(int id) async {
    final response = await _dio.post(ApiEndpoints.vpsRefresh(id));
    return VpsInstance.fromJson(response.data);
  }

  /// 开机
  Future<void> startVps(int id) async {
    await _dio.post(ApiEndpoints.vpsStart(id));
  }

  /// 关机
  Future<void> shutdownVps(int id) async {
    await _dio.post(ApiEndpoints.vpsShutdown(id));
  }

  /// 重启
  Future<void> rebootVps(int id) async {
    await _dio.post(ApiEndpoints.vpsReboot(id));
  }

  /// 重装系统
  Future<void> reinstallVps(int id, {
    required int imageId,
    String? password,
  }) async {
    await _dio.post(
      ApiEndpoints.vpsResetOs(id),
      data: {
        'image_id': imageId,
        if (password != null) 'password': password,
      },
    );
  }

  /// 获取监控数据
  Future<VpsMetrics> getVpsMetrics(int id) async {
    final response = await _dio.get(ApiEndpoints.vpsMonitor(id));
    return VpsMetrics.fromJson(response.data);
  }

  /// 获取商品目录
  Future<Catalog> getCatalog() async {
    final response = await _dio.get(ApiEndpoints.catalog);
    return Catalog.fromJson(response.data);
  }
}
