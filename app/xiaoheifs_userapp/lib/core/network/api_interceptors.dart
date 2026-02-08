import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import '../navigation/app_navigator.dart';
import '../storage/storage_service.dart';

class AuthInterceptor extends Interceptor {
  final StorageService _storage = StorageService.instance;

  @override
  void onRequest(
    RequestOptions options,
    RequestInterceptorHandler handler,
  ) async {
    final token = _storage.getAccessToken();
    if (token != null && token.isNotEmpty) {
      options.headers['Authorization'] = 'Bearer $token';
    }

    if (options.headers.containsKey('X-Use-Api-Key')) {
      options.headers.remove('X-Use-Api-Key');
    }

    handler.next(options);
  }

  @override
  void onError(
    DioException err,
    ErrorInterceptorHandler handler,
  ) async {
    if (err.response?.statusCode == 401) {
      await _storage.clearAuthData();
      AppNavigator.showSnackBar('鉴权失败，请重新登录');
      AppNavigator.goToLogin();
    }

    handler.next(err);
  }
}

class ErrorInterceptor extends Interceptor {
  static bool _realnameDialogOpen = false;

  @override
  void onError(
    DioException err,
    ErrorInterceptorHandler handler,
  ) {
    final status = err.response?.statusCode;
    final data = err.response?.data;
    final message = _extractMessage(data) ?? err.message ?? 'Request failed';
    final url = err.requestOptions.path;

    if (status == 403 && url.startsWith('/v1') && message.toLowerCase().contains('real name required')) {
      if (!_realnameDialogOpen) {
        _realnameDialogOpen = true;
        AppNavigator.showConfirmDialog(
          title: '需要实名认证',
          content: '该操作需要完成实名认证，是否前往认证页面？',
          confirmText: '去认证',
          cancelText: '稍后再说',
        ).then((confirmed) {
          _realnameDialogOpen = false;
          if (confirmed == true) {
            AppNavigator.goToRealname();
          }
        });
      }
      handler.next(err);
      return;
    }

    if (status != null && status >= 500) {
      AppNavigator.showSnackBar('服务端错误：$message', backgroundColor: Colors.red);
    } else if (status != null) {
      AppNavigator.showSnackBar(message, backgroundColor: Colors.red);
    }

    handler.next(DioException(
      requestOptions: err.requestOptions,
      response: err.response,
      type: err.type,
      error: message,
      message: message,
    ));
  }

  String? _extractMessage(dynamic data) {
    if (data is Map<String, dynamic>) {
      return data['error']?.toString() ?? data['message']?.toString();
    }
    return null;
  }
}

class RetryInterceptor extends Interceptor {
  final int maxRetries;
  final Dio dio;
  int retryCount = 0;

  RetryInterceptor({
    required this.dio,
    this.maxRetries = 3,
  });

  @override
  void onError(
    DioException err,
    ErrorInterceptorHandler handler,
  ) async {
    if (_shouldRetry(err)) {
      if (retryCount < maxRetries) {
        retryCount++;
        try {
          final response = await dio.fetch(err.requestOptions);
          retryCount = 0;
          handler.resolve(response);
          return;
        } catch (_) {}
      }
      retryCount = 0;
    }

    handler.next(err);
  }

  bool _shouldRetry(DioException error) {
    return error.type == DioExceptionType.connectionError ||
        (error.type == DioExceptionType.badResponse &&
            error.response?.statusCode != null &&
            error.response!.statusCode! >= 500);
  }
}
