import 'package:dio/dio.dart';
import 'package:pretty_dio_logger/pretty_dio_logger.dart';
import '../config/api_config.dart';
import '../storage/storage_service.dart';
import 'api_interceptors.dart';

/// API客户端
/// 封装Dio实例，配置基础设置和拦截器
class ApiClient {
  late final Dio _dio;
  static ApiClient? _instance;

  ApiClient._internal() {
    _dio = Dio(_buildBaseOptions());
    _setupInterceptors();
  }

  /// 获取单例实例
  static ApiClient get instance {
    _instance ??= ApiClient._internal();
    return _instance!;
  }

  /// 获取Dio实例
  Dio get dio => _dio;

  /// 构建基础配置
  static BaseOptions _buildBaseOptions() {
    // 从存储中读取用户自定义的API URL
    final customUrl = StorageService.instance.getApiBaseUrl();
    final baseUrl = customUrl ?? ApiConfig.defaultUrl;

    return BaseOptions(
      baseUrl: baseUrl,
      connectTimeout: const Duration(seconds: 15),
      receiveTimeout: const Duration(seconds: 15),
      sendTimeout: const Duration(seconds: 15),
      headers: {
        'Content-Type': 'application/json',
        'Accept': 'application/json',
      },
    );
  }

  /// 设置拦截器
  void _setupInterceptors() {
    // 认证拦截器 - 添加Token
    _dio.interceptors.add(AuthInterceptor());

    // 错误处理拦截器
    _dio.interceptors.add(ErrorInterceptor());

    // 日志拦截器 - 仅在Debug模式
    _dio.interceptors.add(PrettyDioLogger(
      requestHeader: true,
      requestBody: true,
      responseBody: true,
      responseHeader: false,
      error: true,
      compact: true,
      maxWidth: 90,
    ));
  }

  /// 更新基础URL（用于用户切换服务器）
  void updateBaseUrl(String newUrl) {
    _dio.options.baseUrl = newUrl;
    ApiConfig.setBaseUrl(newUrl);
  }

  /// 重置为默认URL
  void resetBaseUrl() {
    _dio.options.baseUrl = ApiConfig.defaultUrl;
    ApiConfig.reset();
  }

  /// GET请求
  Future<Response<T>> get<T>(
    String path, {
    Map<String, dynamic>? queryParameters,
    Options? options,
    CancelToken? cancelToken,
    ProgressCallback? onReceiveProgress,
  }) async {
    return await _dio.get<T>(
      path,
      queryParameters: queryParameters,
      options: options,
      cancelToken: cancelToken,
      onReceiveProgress: onReceiveProgress,
    );
  }

  /// POST请求
  Future<Response<T>> post<T>(
    String path, {
    dynamic data,
    Map<String, dynamic>? queryParameters,
    Options? options,
    CancelToken? cancelToken,
    ProgressCallback? onSendProgress,
    ProgressCallback? onReceiveProgress,
  }) async {
    return await _dio.post<T>(
      path,
      data: data,
      queryParameters: queryParameters,
      options: options,
      cancelToken: cancelToken,
      onSendProgress: onSendProgress,
      onReceiveProgress: onReceiveProgress,
    );
  }

  /// PUT请求
  Future<Response<T>> put<T>(
    String path, {
    dynamic data,
    Map<String, dynamic>? queryParameters,
    Options? options,
    CancelToken? cancelToken,
    ProgressCallback? onSendProgress,
    ProgressCallback? onReceiveProgress,
  }) async {
    return await _dio.put<T>(
      path,
      data: data,
      queryParameters: queryParameters,
      options: options,
      cancelToken: cancelToken,
      onSendProgress: onSendProgress,
      onReceiveProgress: onReceiveProgress,
    );
  }

  /// PATCH请求
  Future<Response<T>> patch<T>(
    String path, {
    dynamic data,
    Map<String, dynamic>? queryParameters,
    Options? options,
    CancelToken? cancelToken,
    ProgressCallback? onSendProgress,
    ProgressCallback? onReceiveProgress,
  }) async {
    return await _dio.patch<T>(
      path,
      data: data,
      queryParameters: queryParameters,
      options: options,
      cancelToken: cancelToken,
      onSendProgress: onSendProgress,
      onReceiveProgress: onReceiveProgress,
    );
  }

  /// DELETE请求
  Future<Response<T>> delete<T>(
    String path, {
    dynamic data,
    Map<String, dynamic>? queryParameters,
    Options? options,
    CancelToken? cancelToken,
  }) async {
    return await _dio.delete<T>(
      path,
      data: data,
      queryParameters: queryParameters,
      options: options,
      cancelToken: cancelToken,
    );
  }
}
