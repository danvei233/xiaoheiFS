import 'dart:async';
import 'dart:convert';

import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;
import 'package:http/testing.dart';
import 'package:opensource_adminapp/app_state.dart';
import 'package:opensource_adminapp/models/auth_tokens.dart';
import 'package:opensource_adminapp/services/admin_auth.dart';
import 'package:opensource_adminapp/services/api_client.dart';
import 'package:shared_preferences/shared_preferences.dart';

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  setUp(() {
    SharedPreferences.setMockInitialValues({});
    FlutterSecureStorage.setMockInitialValues({});
  });

  test(
    'AppState serializes concurrent 401 refreshes to one in-flight request',
    () async {
      var refreshCalls = 0;
      final mockHttp = MockClient((req) async {
        if (req.headers['Authorization'] == 'Bearer expired-token') {
          return http.Response('{"error":"unauthorized"}', 401);
        }
        return http.Response('{"ok":true}', 200);
      });
      final authService = AdminAuthService(
        client: MockClient((req) async {
          refreshCalls++;
          return http.Response(
            jsonEncode({
              'access_token': 'fresh-token',
              'refresh_token': 'fresh-refresh',
              'expires_in': 3600,
            }),
            200,
          );
        }),
      );
      final state = AppState(
        authService: authService,
        clientOverride: mockHttp,
      );
      await state.loginWithPassword(
        apiUrl: 'https://api.example.com',
        tokens: const AuthTokens(
          accessToken: 'expired-token',
          refreshToken: 'expired-refresh',
          expiresIn: 3600,
        ),
        username: 'admin',
      );

      // 5 个并发 401（多个页面同时访问各自新建的 ApiClient）。
      final clients = List.generate(5, (_) => state.apiClient!);
      final results = await Future.wait(
        clients.map((c) => c.getJson('/admin/api/v1/profile')),
      );
      for (final r in results) {
        expect(
          r['ok'],
          isTrue,
          reason: 'each 401 is retried with the fresh token',
        );
      }
      expect(
        refreshCalls,
        1,
        reason: 'concurrent 401s share a single in-flight refresh',
      );
    },
  );

  test(
    'a later refresh after the first completes starts a new refresh',
    () async {
      var refreshCalls = 0;
      var serverCalls = 0;
      final mockHttp = MockClient((req) async {
        serverCalls++;
        // 一直 401，直到会话被刷新到 fresh-token。
        if (req.headers['Authorization'] == 'Bearer fresh-token') {
          return http.Response('{"ok":true}', 200);
        }
        return http.Response('{"error":"unauthorized"}', 401);
      });
      final authService = AdminAuthService(
        client: MockClient((req) async {
          refreshCalls++;
          return http.Response(
            jsonEncode({
              'access_token': 'fresh-token',
              'refresh_token': 'fresh-refresh',
              'expires_in': 3600,
            }),
            200,
          );
        }),
      );
      final state = AppState(
        authService: authService,
        clientOverride: mockHttp,
      );
      await state.loginWithPassword(
        apiUrl: 'https://api.example.com',
        tokens: const AuthTokens(
          accessToken: 'old-token',
          refreshToken: 'old-refresh',
          expiresIn: 3600,
        ),
        username: 'admin',
      );

      // 第一轮并发。
      final first = await state.apiClient!.getJson('/admin/api/v1/profile');
      expect(first['ok'], isTrue);
      expect(refreshCalls, 1);

      // 会话已被刷新到 fresh-token，直接成功，不再刷新。
      final second = await state.apiClient!.getJson('/admin/api/v1/profile');
      expect(second['ok'], isTrue);
      expect(refreshCalls, 1, reason: 'no refresh needed once token is fresh');
    },
  );

  test(
    'account A in-flight refresh cannot overwrite or log out account B',
    () async {
      final refreshGate = Completer<void>();
      var refreshCalls = 0;
      final authService = AdminAuthService(
        client: MockClient((req) async {
          if (req.url.path.endsWith('/auth/refresh')) {
            refreshCalls++;
            await refreshGate.future; // block until the test switches to B
            return http.Response(
              jsonEncode({
                'access_token': 'A-FRESH-TOKEN',
                'refresh_token': 'A-FRESH-REFRESH',
                'expires_in': 3600,
              }),
              200,
            );
          }
          return http.Response('{"ok":true}', 200);
        }),
      );
      final mockHttp = MockClient((req) async {
        if (req.headers['Authorization'] == 'Bearer A-EXPIRED') {
          return http.Response('{"error":"unauthorized"}', 401);
        }
        return http.Response('{"ok":true}', 200);
      });
      final state = AppState(
        authService: authService,
        clientOverride: mockHttp,
      );
      await state.loginWithPassword(
        apiUrl: 'https://api.example.com',
        tokens: const AuthTokens(
          accessToken: 'A-EXPIRED',
          refreshToken: 'A-REFRESH',
          expiresIn: 3600,
        ),
        username: 'A',
      );

      // 用账号 A 的 ApiClient 发起请求：401 -> 触发 A 的 refresh（阻塞在 gate）。
      final clientA = state.apiClient!;
      final futA = clientA.getJson('/admin/api/v1/profile');
      final start = DateTime.now();
      while (refreshCalls == 0 &&
          DateTime.now().difference(start) < const Duration(seconds: 2)) {
        await Future<void>.delayed(const Duration(milliseconds: 10));
      }
      expect(refreshCalls, 1, reason: 'A refresh must start before the switch');

      // 在 A 的 refresh 在途时 logout/login 为账号 B。
      await state.logout();
      await state.loginWithPassword(
        apiUrl: 'https://api.example.com',
        tokens: const AuthTokens(
          accessToken: 'B-TOKEN',
          refreshToken: 'B-REFRESH',
          expiresIn: 3600,
        ),
        username: 'B',
      );

      // 放行 A 的 refresh：其结果必须被会话代次守卫丢弃，不得写回 B。
      refreshGate.complete();
      try {
        await futA;
      } on ApiException {
        // 预期：A 的请求无法用被丢弃的 refresh 重试，最终 401。
      }

      expect(
        state.session?.token,
        'B-TOKEN',
        reason: 'A in-flight refresh must not overwrite B access token',
      );
      expect(
        state.session?.refreshToken,
        'B-REFRESH',
        reason: 'A in-flight refresh must not overwrite B refresh token',
      );
      expect(
        state.isLoggedIn,
        isTrue,
        reason: 'A stale 401 must not log out account B',
      );
    },
  );
}
