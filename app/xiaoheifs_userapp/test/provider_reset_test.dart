import 'dart:async';

import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:shared_preferences/shared_preferences.dart';

import 'package:opensource_userapp/core/storage/storage_service.dart';
import 'package:opensource_userapp/data/models/user.dart';
import 'package:opensource_userapp/data/repositories/auth_repository.dart';
import 'package:opensource_userapp/data/repositories/order_repository.dart';
import 'package:opensource_userapp/data/repositories/vps_repository.dart';
import 'package:opensource_userapp/presentation/providers/auth_provider.dart';
import 'package:opensource_userapp/presentation/providers/order_provider.dart';
import 'package:opensource_userapp/presentation/providers/vps_provider.dart';

/// F1 回归测试：登出时业务 provider 必须清空状态与缓存，
/// 防止用户 A 退出登录后，用户 B 在 30s TTL 缓存窗口内看到 A 的数据。
class _CountingOrderRepo extends OrderRepository {
  int listCalls = 0;

  @override
  Future<Map<String, dynamic>> listOrders({
    String? status,
    int? limit,
    int? offset,
  }) async {
    listCalls++;
    return {
      'items': [
        {'id': 1, 'order_no': 'A-ORDER'},
      ],
      'total': 1,
    };
  }
}

/// F13b 回归测试：vpsDetailProvider 是单例 notifier（非 family），
/// 打开新实例时必须先清空上个实例的 detail，避免新实例加载完成前
/// 短暂展示上一个实例的 access_info 密码。
class _FakeVpsRepo extends VpsRepository {
  @override
  Future<Map<String, dynamic>> getDetail(int id) async {
    return {
      'id': id,
      'name': 'vps-$id',
      'access_info': {'os_password': 'pw$id'},
    };
  }
}

/// Round-5 回归：可控完成时序的仓库，用于模拟乱序/跨账号慢响应。
class _OrderedVpsRepo extends VpsRepository {
  _OrderedVpsRepo({required this.slowForId, required this.slowCompleter});

  final int slowForId;
  final Completer<Map<String, dynamic>> slowCompleter;

  @override
  Future<Map<String, dynamic>> getDetail(int id) async {
    if (id == slowForId) return slowCompleter.future;
    return {
      'id': id,
      'name': 'vps-$id',
      'access_info': {'os_password': 'pw$id'},
    };
  }
}

/// Round-5 回归：family provider 失效测试用的假认证仓储（无网络）。
class _FakeAuthRepo extends AuthRepository {
  @override
  Future<bool> isLoggedIn() async => false;

  @override
  Future<void> logout() async {}

  @override
  Future<User> getMe() async => User(id: 1);
}

/// Round-5 回归：统计 listFirewallRules 调用次数，验证失效后重新拉取。
class _FamilyVpsRepo extends VpsRepository {
  int firewallCalls = 0;

  @override
  Future<List<Map<String, dynamic>>> listFirewallRules(int id) async {
    firewallCalls++;
    return [
      {'id': id, 'rule': 'r$id'},
    ];
  }
}

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  setUp(() async {
    SharedPreferences.setMockInitialValues({});
    FlutterSecureStorage.setMockInitialValues({});
    await StorageService.init();
  });

  group('登出后业务 provider 状态重置 (F1)', () {
    test('OrderListNotifier.reset 清空 items/total 与 30s 缓存', () async {
      final repo = _CountingOrderRepo();
      final notifier = OrderListNotifier(repo);

      await notifier.fetchOrders();
      expect(repo.listCalls, 1);
      expect(notifier.state.items, hasLength(1));
      expect(notifier.state.total, 1);

      // 命中 30s TTL 缓存，不重新请求
      await notifier.fetchOrders();
      expect(repo.listCalls, 1);

      notifier.reset();
      expect(notifier.state.items, isEmpty);
      expect(notifier.state.total, 0);

      // reset 必须清空缓存：再次 fetch 应重新请求，而不是命中 A 留下的缓存
      await notifier.fetchOrders();
      expect(repo.listCalls, 2);
      expect(notifier.state.items, hasLength(1));
    });
  });

  group('切换 VPS 实例时清空上个实例 detail (F13b)', () {
    test('fetch 新实例时同步清空旧 detail，加载完成前不展示旧数据', () async {
      final notifier = VpsDetailNotifier(_FakeVpsRepo());

      await notifier.fetch(1);
      expect(notifier.state.detail?['id'], 1);

      // 发起 fetch(2) 后、加载完成前，detail 必须立即被清空
      final loading = notifier.fetch(2);
      expect(notifier.state.detail, isNull);
      await loading;
      expect(notifier.state.detail?['id'], 2);
    });

    test('刷新同一实例时保留当前 detail（不闪空）', () async {
      final notifier = VpsDetailNotifier(_FakeVpsRepo());
      await notifier.fetch(1);

      final loading = notifier.fetch(1);
      expect(notifier.state.detail?['id'], 1);
      await loading;
      expect(notifier.state.detail?['id'], 1);
    });
  });

  group('VPS 请求代次守卫：乱序与跨账号慢响应 (Round-5)', () {
    test('乱序响应不覆盖新实例：慢的旧请求晚到即丢弃', () async {
      final slow1 = Completer<Map<String, dynamic>>();
      final repo = _OrderedVpsRepo(slowForId: 1, slowCompleter: slow1);
      final notifier = VpsDetailNotifier(repo);

      // 先请求 VPS 1（挂起），随后立即请求 VPS 2（快速返回）。
      final f1 = notifier.fetch(1);
      final f2 = notifier.fetch(2);
      await f2;
      expect(notifier.state.detail?['id'], 2);

      // VPS 1 的慢响应晚到：必须被代次守卫丢弃，不得覆盖 VPS 2。
      slow1.complete({
        'id': 1,
        'name': 'vps-1',
        'access_info': {'os_password': 'pw1'},
      });
      await f1;
      expect(notifier.state.detail?['id'], 2);
      expect(notifier.state.detail?['name'], 'vps-2');
    });

    test('跨账号：登出后旧账号慢响应不写入新会话', () async {
      final slow1 = Completer<Map<String, dynamic>>();
      final repo = _OrderedVpsRepo(slowForId: 1, slowCompleter: slow1);
      final notifier = VpsDetailNotifier(repo);

      // 账号 A 请求 VPS 1 时挂起；登出（reset）；账号 B 请求 VPS 2 快速返回。
      final f1 = notifier.fetch(1);
      notifier.reset();
      final f2 = notifier.fetch(2);
      await f2;
      expect(notifier.state.detail?['id'], 2);

      // A 的慢响应到达：代次已递增，必须丢弃，不得污染 B 的会话。
      slow1.complete({'id': 1, 'name': 'vps-1-old-account'});
      await f1;
      expect(notifier.state.detail?['id'], 2);
      expect(notifier.state.detail?['name'], 'vps-2');
    });

    test('登出清理失效 non-autoDispose family providers（跨账号不串台）', () async {
      final familyRepo = _FamilyVpsRepo();
      final container = ProviderContainer(
        overrides: [
          authRepositoryProvider.overrideWithValue(_FakeAuthRepo()),
          vpsRepositoryProvider.overrideWithValue(familyRepo),
        ],
      );
      addTearDown(container.dispose);

      // 首次读取防火墙规则（FutureProvider.family）→ 缓存。
      final first = await container.read(vpsFirewallProvider(1).future);
      expect(first, hasLength(1));
      expect(familyRepo.firewallCalls, 1);

      // 会话失效清理（logout / 401 登出共用 _resetBusinessState）。
      await container.read(authProvider.notifier).handleSessionExpired();

      // 再次读取：缓存必须已失效并重新请求，不能复用上一账号的缓存。
      final second = await container.read(vpsFirewallProvider(1).future);
      expect(second, hasLength(1));
      expect(
        familyRepo.firewallCalls,
        2,
        reason: 'family provider cache must be invalidated on session boundary',
      );
    });
  });
}
