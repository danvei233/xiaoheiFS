import 'package:dio/dio.dart';
import '../../../core/network/api_client.dart';
import '../../../core/constants/api_endpoints.dart';
import '../../models/dashboard.dart';
import '../../models/realname.dart';

/// Dashboard API服务
class DashboardApi {
  final Dio _dio = ApiClient.instance.dio;

  /// 获取Dashboard数据
  Future<DashboardData> getDashboard() async {
    final response = await _dio.get(ApiEndpoints.dashboard);
    return DashboardData.fromJson(response.data);
  }
}

/// 实名认证API服务
class RealnameApi {
  final Dio _dio = ApiClient.instance.dio;

  /// 获取实名认证状态
  Future<RealnameStatus> getRealnameStatus() async {
    final response = await _dio.get(ApiEndpoints.realnameStatus);
    return RealnameStatus.fromJson(response.data);
  }

  /// 提交实名认证
  Future<RealnameVerification> submitRealname({
    required String realName,
    required String idNumber,
  }) async {
    final response = await _dio.post(
      ApiEndpoints.realnameVerify,
      data: {
        'real_name': realName,
        'id_number': idNumber,
      },
    );
    return RealnameVerification.fromJson(response.data);
  }
}
