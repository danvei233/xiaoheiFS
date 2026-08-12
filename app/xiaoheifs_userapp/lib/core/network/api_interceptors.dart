import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import '../constants/api_endpoints.dart';
import '../navigation/app_navigator.dart';
import '../storage/storage_service.dart';
import 'api_client.dart';

class AuthInterceptor extends Interceptor {
  final StorageService _storage = StorageService.instance;

  /// 并发 401 刷新单飞（singleflight）：多个请求同时 401 只发起一次 refresh。
  ///
  /// `_refreshingFor` 记录该 in-flight refresh 所属的会话身份（refresh token）。
  /// 会话切换（logout/login）后 refresh token 变化，新会话不会复用旧会话的
  /// in-flight future，旧会话的刷新结果也不会写入新会话的 storage/state。
  static Future<String?>? _refreshing;
  static String? _refreshingFor;
  static bool _unauthorizedHandled = false;

  /// 401 强制登出后的本地状态清理回调（由 App 启动时注册）。
  /// 用于清空业务 provider，防止跨账号数据串台。
  static Future<void> Function()? onSessionExpired;

  @override
  void onRequest(
    RequestOptions options,
    RequestInterceptorHandler handler,
  ) async {
    final token = await _storage.getAccessToken();
    if (token != null && token.isNotEmpty) {
      options.headers['Authorization'] = 'Bearer $token';
      _unauthorizedHandled = false;
    }

    if (options.headers.containsKey('X-Use-Api-Key')) {
      options.headers.remove('X-Use-Api-Key');
    }

    handler.next(options);
  }

  @override
  void onError(DioException err, ErrorInterceptorHandler handler) async {
    if (err.response?.statusCode == 401 &&
        err.requestOptions.extra['retried'] != true) {
      if (_unauthorizedHandled) {
        // 已有并发请求完成登出，此错误交由后续拦截器透传，不再重复提示。
        err.requestOptions.extra['authHandled'] = true;
        handler.next(err);
        return;
      }
      // 捕获发起本次请求时的会话身份（access token）：强制登出只能作用于
      // 「仍处在该会话」的情况。若请求在途期间发生了 logout/login（换号），
      // 存储里的 access token 已变化——旧会话的 401 不得清空新会话的数据。
      final sessionAccess = await _storage.getAccessToken();
      final path = err.requestOptions.path;
      final isAuthEndpoint =
          path.contains(ApiEndpoints.authLogin) ||
          path.contains(ApiEndpoints.authRefresh);

      if (!isAuthEndpoint) {
        final refreshToken = await _storage.getRefreshToken();
        if (refreshToken != null && refreshToken.isNotEmpty) {
          try {
            // 会话绑定的单飞：只有同一会话（同一 refresh token）的并发 401
            // 才共享同一个 in-flight future；logout/login 换号后必须重新发起，
            // 且旧会话的 waiter 不能清掉新会话发起的新一代 future。
            if (_refreshing == null || _refreshingFor != refreshToken) {
              final future = _refreshToken(refreshToken);
              _refreshing = future;
              _refreshingFor = refreshToken;
              future.whenComplete(() {
                if (identical(_refreshing, future)) {
                  _refreshing = null;
                  _refreshingFor = null;
                }
              });
            }
            final newToken = await _refreshing;

            if (newToken != null && newToken.isNotEmpty) {
              _unauthorizedHandled = false;
              final options = err.requestOptions;
              options.headers['Authorization'] = 'Bearer $newToken';
              options.extra['retried'] = true;
              try {
                final response = await ApiClient.instance.dio.fetch(options);
                handler.resolve(response);
                return;
              } catch (_) {}
            }
          } finally {
            // 不在此清 _refreshing：由 whenComplete 中 identical 守卫清，避免
            // 慢 waiter 清掉更新一代 future。
          }
        }
      }

      // 会话代次守卫：仅当存储里的 access token 仍是发起本请求时的那一个才
      // 强制登出。请求在途期间 logout/login 换号（sessionAccess 已变）则跳过
      // ——账号 A 的过期响应绝不能把账号 B 踢下线或清空 B 的 provider 状态。
      final nowAccess = await _storage.getAccessToken();
      if (nowAccess == sessionAccess) {
        // 已在此处完成登出与提示，标记给 ErrorInterceptor 避免二次 SnackBar。
        err.requestOptions.extra['authHandled'] = true;
        _unauthorizedHandled = true;
        await _storage.clearAuthData();
        // 清空业务 provider 状态，避免下一账号在缓存窗口内看到上一账号的数据。
        if (onSessionExpired != null) {
          try {
            await onSessionExpired!();
          } catch (_) {}
        }
        AppNavigator.showSnackBar('鉴权失败，请重新登录');
        AppNavigator.goToLogin();
      }
    }

    handler.next(err);
  }

  Future<String?> _refreshToken(String refreshToken) async {
    try {
      final baseUrl = ApiClient.instance.dio.options.baseUrl;
      final dio = Dio(BaseOptions(baseUrl: baseUrl));
      final response = await dio.post(
        ApiEndpoints.authRefresh,
        data: {'refresh_token': refreshToken},
      );
      final data = response.data is Map ? response.data as Map : {};
      final access =
          data['access_token'] ?? data['accessToken'] ?? data['token'];
      final newRefresh = data['refresh_token'] ?? data['refreshToken'];
      if (access is String && access.isNotEmpty) {
        // 会话代次守卫：refresh 结果只能写回「发起它的那个会话」。若期间
        // 发生 logout/login（换号），存储里的 refresh token 已不等于发起时
        // 的 refreshToken——丢弃结果，账号 A 的 refresh 永不覆盖账号 B 的
        // storage/state。
        final current = await _storage.getRefreshToken();
        if (current != refreshToken) {
          return null;
        }
        await _storage.setAccessToken(access);
        if (newRefresh is String && newRefresh.isNotEmpty) {
          await _storage.setRefreshToken(newRefresh);
        }
        return access;
      }
    } catch (_) {}
    return null;
  }
}

class ErrorInterceptor extends Interceptor {
  static bool _realnameDialogOpen = false;

  @override
  void onError(DioException err, ErrorInterceptorHandler handler) {
    // AuthInterceptor 已完成登出并提示，这里直接透传，避免重复 SnackBar。
    if (err.requestOptions.extra['authHandled'] == true) {
      handler.next(err);
      return;
    }
    final status = err.response?.statusCode;
    // 401 由 AuthInterceptor 全权处理（刷新或登出）。Dio 的 onError 按注册
    // 逆序派发，ErrorInterceptor 先于 AuthInterceptor 执行，因此这里必须
    // 主动跳过 401，否则会与 AuthInterceptor 的“鉴权失败”提示重复。
    if (status == 401) {
      handler.next(err);
      return;
    }
    final data = err.response?.data;
    final message = _extractMessage(data) ?? err.message ?? 'Request failed';
    final url = err.requestOptions.path;

    if (status == 403 &&
        url.startsWith('/v1') &&
        message.toLowerCase().contains('real name required')) {
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

    handler.next(
      DioException(
        requestOptions: err.requestOptions,
        response: err.response,
        type: err.type,
        error: message,
        message: message,
      ),
    );
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

  RetryInterceptor({required this.dio, this.maxRetries = 3});

  @override
  void onError(DioException err, ErrorInterceptorHandler handler) async {
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
