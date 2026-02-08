import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import '../../core/constants/app_strings.dart';
import '../pages/auth/login_page.dart';
import '../pages/home/main_layout.dart';
import '../pages/dashboard/dashboard_page.dart';
import '../pages/vps/vps_list_page.dart';
import '../pages/vps/vps_detail_page.dart';
import '../pages/orders/orders_list_page.dart';
import 'package:xiaoheifs_userapp/presentation/pages/orders/order_detail_page.dart';
import '../pages/cart/cart_page.dart';
import 'package:xiaoheifs_userapp/presentation/pages/buy/buy_vps_page.dart';
import '../pages/billing/billing_page.dart';
import '../pages/tickets/tickets_list_page.dart';
import '../pages/tickets/ticket_detail_page.dart';
import '../pages/realname/realname_page.dart';
import '../pages/profile/profile_page.dart';
import '../pages/more/more_page.dart';
import '../pages/notifications/notifications_page.dart';
import '../../core/navigation/app_navigator.dart';
import '../providers/auth_provider.dart';

/// 路由配置 Provider
final routerProvider = Provider<GoRouter>((ref) {
  final authState = ref.watch(authProvider);

  return GoRouter(
    navigatorKey: AppNavigator.navigatorKey,
    initialLocation: '/login',
    redirect: (context, state) {
      final isAuthenticated = authState.isAuthenticated;
      final isLoginRoute = state.matchedLocation == '/login';

      if (!isAuthenticated && !isLoginRoute) {
        return '/login';
      }

      if (isAuthenticated && isLoginRoute) {
        return '/console';
      }

      return null;
    },
    routes: [
      GoRoute(
        path: '/login',
        name: 'login',
        pageBuilder: (context, state) => MaterialPage(
          key: state.pageKey,
          child: const LoginPage(),
        ),
      ),
      ShellRoute(
        builder: (context, state, child) => MainLayout(child: child),
        routes: [
          GoRoute(
            path: '/console',
            name: 'dashboard',
            pageBuilder: (context, state) => MaterialPage(
              key: state.pageKey,
              child: const DashboardPage(),
            ),
          ),
          GoRoute(
            path: '/console/vps',
            name: 'vps_list',
            pageBuilder: (context, state) => MaterialPage(
              key: state.pageKey,
              child: const VpsListPage(),
            ),
          ),
          GoRoute(
            path: '/console/vps/:id',
            name: 'vps_detail',
            pageBuilder: (context, state) {
              final id = int.tryParse(state.pathParameters['id'] ?? '');
              return MaterialPage(
                key: state.pageKey,
                child: VpsDetailPage(id: id ?? 0),
              );
            },
          ),
          GoRoute(
            path: '/console/buy',
            name: 'buy_vps',
            pageBuilder: (context, state) => MaterialPage(
              key: state.pageKey,
              child: const BuyVpsPage(),
            ),
          ),
          GoRoute(
            path: '/console/cart',
            name: 'cart',
            pageBuilder: (context, state) => MaterialPage(
              key: state.pageKey,
              child: const CartPage(),
            ),
          ),
          GoRoute(
            path: '/console/orders',
            name: 'orders',
            pageBuilder: (context, state) => MaterialPage(
              key: state.pageKey,
              child: const OrdersListPage(),
            ),
          ),
          GoRoute(
            path: '/console/orders/:id',
            name: 'order_detail',
            pageBuilder: (context, state) {
              final id = int.tryParse(state.pathParameters['id'] ?? '');
              return MaterialPage(
                key: state.pageKey,
                child: OrderDetailPage(id: id ?? 0),
              );
            },
          ),
          GoRoute(
            path: '/console/billing',
            name: 'billing',
            pageBuilder: (context, state) => MaterialPage(
              key: state.pageKey,
              child: const BillingPage(),
            ),
          ),
          GoRoute(
            path: '/console/tickets',
            name: 'tickets',
            pageBuilder: (context, state) => MaterialPage(
              key: state.pageKey,
              child: const TicketsListPage(),
            ),
          ),
          GoRoute(
            path: '/console/tickets/:id',
            name: 'ticket_detail',
            pageBuilder: (context, state) {
              final id = int.tryParse(state.pathParameters['id'] ?? '');
              return MaterialPage(
                key: state.pageKey,
                child: TicketDetailPage(id: id ?? 0),
              );
            },
          ),
          GoRoute(
            path: '/console/realname',
            name: 'realname',
            pageBuilder: (context, state) => MaterialPage(
              key: state.pageKey,
              child: const RealnamePage(),
            ),
          ),
          GoRoute(
            path: '/console/profile',
            name: 'profile',
            pageBuilder: (context, state) => MaterialPage(
              key: state.pageKey,
              child: const ProfilePage(),
            ),
          ),
          GoRoute(
            path: '/console/more',
            name: 'more',
            pageBuilder: (context, state) => MaterialPage(
              key: state.pageKey,
              child: const MorePage(),
            ),
          ),
          GoRoute(
            path: '/console/notifications',
            name: 'notifications',
            pageBuilder: (context, state) => MaterialPage(
              key: state.pageKey,
              child: const NotificationsPage(),
            ),
          ),
        ],
      ),
    ],
    errorBuilder: (context, state) => Scaffold(
      body: Center(
        child: Column(
          mainAxisAlignment: MainAxisAlignment.center,
          children: [
            const Icon(Icons.error_outline, size: 64, color: Colors.red),
            const SizedBox(height: 16),
            Text(
              '页面未找到: ${state.matchedLocation}',
              style: const TextStyle(fontSize: 16),
            ),
            const SizedBox(height: 16),
            ElevatedButton(
              onPressed: () => context.go('/console'),
              child: const Text(AppStrings.back),
            ),
          ],
        ),
      ),
    ),
  );
});
