import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:shared_preferences/shared_preferences.dart';

import 'package:opensource_userapp/core/storage/storage_service.dart';
import 'package:opensource_userapp/data/repositories/dashboard_repository.dart';
import 'package:opensource_userapp/presentation/providers/dashboard_provider.dart';

/// F13a 回归测试：dashboard 指标字段以字符串返回时必须被容忍解析，
/// 而不是 `as int` 直接强转抛错导致整个 Future.wait 失败。
class _FakeDashboardRepo extends DashboardRepository {
  final Map<String, dynamic> dash;

  _FakeDashboardRepo(this.dash);

  @override
  Future<Map<String, dynamic>> getDashboard() async => dash;

  @override
  Future<Map<String, dynamic>> listOrders({int? limit, int? offset}) async {
    return {
      'items': <Map<String, dynamic>>[],
    };
  }

  @override
  Future<Map<String, dynamic>> listVps() async {
    return {
      'items': <Map<String, dynamic>>[],
    };
  }

  @override
  Future<Map<String, dynamic>> getWallet() async {
    return {
      'wallet': {'balance': '12.34', 'currency': 'CNY'},
    };
  }

  @override
  Future<Map<String, dynamic>> getRealnameStatus() async {
    return {'verified': true};
  }
}

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  setUp(() async {
    SharedPreferences.setMockInitialValues({});
    FlutterSecureStorage.setMockInitialValues({});
    await StorageService.init();
  });

  group('dashboard 字符串数字指标解析 (F13a)', () {
    test('字符串类型指标不会抛错，正确解析为数值', () async {
      final notifier = DashboardNotifier(
        _FakeDashboardRepo({
          'vps': '5',
          'expiring': '2',
          'orders': '3',
          'pending_review': '1',
          'spend_30d': '100.5',
          'cart_items': '7',
          'pending_payment': '0',
        }),
      );
      await pumpEventQueue();

      expect(notifier.state.error, isNull);
      expect(notifier.state.metrics.vpsTotal, 5);
      expect(notifier.state.metrics.expiring, 2);
      expect(notifier.state.metrics.ordersTotal, 3);
      expect(notifier.state.metrics.pendingOrders, 1);
      expect(notifier.state.metrics.spend30d, 100.5);
      expect(notifier.state.metrics.cartItems, 7);
    });

    test('缺失字段使用兜底值，不抛错', () async {
      final notifier = DashboardNotifier(_FakeDashboardRepo({}));
      await pumpEventQueue();

      expect(notifier.state.error, isNull);
      expect(notifier.state.metrics.vpsTotal, 0);
      expect(notifier.state.metrics.spend30d, 0);
    });
  });
}
