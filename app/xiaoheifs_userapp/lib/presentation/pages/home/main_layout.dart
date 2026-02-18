import 'dart:ui';

import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import '../../../core/constants/app_colors.dart';
import '../../../core/constants/app_strings.dart';
import '../../../core/network/api_client.dart';
import '../../../core/utils/avatar_url.dart';
import '../../providers/auth_provider.dart';
import '../../providers/nav_history_provider.dart';
import '../../providers/notification_provider.dart';
import '../../providers/refresh_provider.dart';
import '../../providers/site_provider.dart';

/// 主布局组件
/// 包含顶部栏、侧边/底部导航和用户菜单
class MainLayout extends ConsumerStatefulWidget {
  final Widget child;

  const MainLayout({super.key, required this.child});

  @override
  ConsumerState<MainLayout> createState() => _MainLayoutState();
}

class _MainLayoutState extends ConsumerState<MainLayout> {
  DateTime? _lastBackPressTime;
  String? _lastSyncedLocation;
  bool _pendingResetHistory = false;
  bool _navigatingFromHistoryBack = false;

  BoxDecoration _glassDecoration(
    ColorScheme colorScheme, {
    bool top = false,
    bool bottom = false,
  }) {
    return BoxDecoration(
      gradient: LinearGradient(
        begin: Alignment.topCenter,
        end: Alignment.bottomCenter,
        colors: [
          colorScheme.surface.withOpacity(0.56),
          colorScheme.surface.withOpacity(0.36),
        ],
      ),
      boxShadow: [
        BoxShadow(
          color: Colors.black.withOpacity(0.28),
          blurRadius: 20,
          offset: const Offset(0, 6),
        ),
        BoxShadow(
          color: Colors.white.withOpacity(0.05),
          blurRadius: 10,
          offset: const Offset(0, -1),
        ),
      ],
    );
  }

  @override
  void initState() {
    super.initState();
    Future.microtask(
      () => ref.read(notificationProvider.notifier).fetchUnreadCount(),
    );
    Future.microtask(() => ref.read(siteProvider.notifier).fetchSettings());
  }

  static const List<NavigationItem> _desktopItems = [
    NavigationItem(
      icon: Icons.dashboard_outlined,
      selectedIcon: Icons.dashboard,
      label: AppStrings.navDashboard,
      route: '/console',
    ),
    NavigationItem(
      icon: Icons.cloud_outlined,
      selectedIcon: Icons.cloud,
      label: AppStrings.navVps,
      route: '/console/vps',
    ),
    NavigationItem(
      icon: Icons.shopping_cart_outlined,
      selectedIcon: Icons.shopping_cart,
      label: AppStrings.navCart,
      route: '/console/cart',
    ),
    NavigationItem(
      icon: Icons.receipt_long_outlined,
      selectedIcon: Icons.receipt_long,
      label: AppStrings.navOrders,
      route: '/console/orders',
    ),
    NavigationItem(
      icon: Icons.account_balance_wallet_outlined,
      selectedIcon: Icons.account_balance_wallet,
      label: AppStrings.navWallet,
      route: '/console/billing',
    ),
    NavigationItem(
      icon: Icons.support_agent_outlined,
      selectedIcon: Icons.support_agent,
      label: AppStrings.navTickets,
      route: '/console/tickets',
    ),
    NavigationItem(
      icon: Icons.verified_user_outlined,
      selectedIcon: Icons.verified_user,
      label: AppStrings.navRealname,
      route: '/console/realname',
    ),
    NavigationItem(
      icon: Icons.settings_outlined,
      selectedIcon: Icons.settings,
      label: AppStrings.navProfile,
      route: '/console/profile',
    ),
  ];

  static const List<NavigationItem> _mobilePrimaryItems = [
    NavigationItem(
      icon: Icons.dashboard_outlined,
      selectedIcon: Icons.dashboard,
      label: AppStrings.navDashboard,
      route: '/console',
    ),
    NavigationItem(
      icon: Icons.cloud_outlined,
      selectedIcon: Icons.cloud,
      label: AppStrings.navVps,
      route: '/console/vps',
    ),
    NavigationItem(
      icon: Icons.shopping_cart_outlined,
      selectedIcon: Icons.shopping_cart,
      label: AppStrings.navCart,
      route: '/console/cart',
    ),
    NavigationItem(
      icon: Icons.receipt_long_outlined,
      selectedIcon: Icons.receipt_long,
      label: AppStrings.navOrders,
      route: '/console/orders',
    ),
    NavigationItem(
      icon: Icons.menu,
      selectedIcon: Icons.menu,
      label: AppStrings.navMore,
      route: '/console/more',
    ),
  ];

  void _onDestinationSelected(int index, List<NavigationItem> items) {
    _pendingResetHistory = true;
    context.go(items[index].route);
  }

  Future<void> _logout() async {
    final confirmed = await showDialog<bool>(
      context: context,
      builder: (context) => AlertDialog(
        title: const Text(AppStrings.logout),
        content: const Text(AppStrings.logoutConfirm),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(context, false),
            child: const Text(AppStrings.cancel),
          ),
          TextButton(
            onPressed: () => Navigator.pop(context, true),
            child: const Text(AppStrings.confirm),
          ),
        ],
      ),
    );

    if (confirmed == true && mounted) {
      await ref.read(authProvider.notifier).logout();
      if (mounted) {
        context.go('/login');
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    final authState = ref.watch(authProvider);
    final user = authState.user;
    final unreadCount = ref.watch(
      notificationProvider.select((state) => state.unreadCount),
    );
    final siteName = ref.watch(siteProvider.select((state) => state.siteName));
    final isDesktop = MediaQuery.of(context).size.width > 1024;
    final location = GoRouterState.of(context).uri.path;
    _syncNavHistory(location);

    if (isDesktop) {
      final selectedIndex = _desktopIndexForLocation(location);
      return _wrapBackHandler(
        isDesktop: true,
        child: _buildDesktopLayout(
          user,
          selectedIndex,
          unreadCount,
          location,
          siteName,
        ),
      );
    }

    final selectedIndex = _mobileIndexForLocation(location);
    return _wrapBackHandler(
      isDesktop: false,
      child: _buildMobileLayout(
        user,
        selectedIndex,
        unreadCount,
        location,
        siteName,
      ),
    );
  }

  Widget _wrapBackHandler({required bool isDesktop, required Widget child}) {
    return PopScope(
      canPop: false,
      onPopInvokedWithResult: (didPop, result) async {
        if (didPop) return;

        final previousRoute = ref.read(navHistoryProvider.notifier).pop();
        if (previousRoute != null) {
          _navigatingFromHistoryBack = true;
          context.go(previousRoute);
          return;
        }

        if (!isDesktop) {
          final now = DateTime.now();
          if (_lastBackPressTime != null &&
              now.difference(_lastBackPressTime!).inSeconds < 2) {
            SystemNavigator.pop();
          } else {
            _lastBackPressTime = now;
            ScaffoldMessenger.of(context).showSnackBar(
              const SnackBar(
                content: Text('再按一次返回键退出'),
                behavior: SnackBarBehavior.floating,
              ),
            );
          }
        }
      },
      child: child,
    );
  }

  void _syncNavHistory(String location) {
    if (_lastSyncedLocation == location) {
      return;
    }
    _lastSyncedLocation = location;

    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (!mounted) return;
      final navHistory = ref.read(navHistoryProvider.notifier);

      if (_pendingResetHistory) {
        navHistory.resetTo(location);
        _pendingResetHistory = false;
        _navigatingFromHistoryBack = false;
        return;
      }

      if (_navigatingFromHistoryBack) {
        _navigatingFromHistoryBack = false;
        return;
      }

      navHistory.push(location);
    });
  }

  int _desktopIndexForLocation(String location) {
    if (location.startsWith('/console/vps')) return 1;
    if (location.startsWith('/console/cart')) return 2;
    if (location.startsWith('/console/orders')) return 3;
    if (location.startsWith('/console/billing')) return 4;
    if (location.startsWith('/console/tickets')) return 5;
    if (location.startsWith('/console/realname')) return 6;
    if (location.startsWith('/console/profile')) return 7;
    return 0;
  }

  int _mobileIndexForLocation(String location) {
    if (location.startsWith('/console/vps') ||
        location.startsWith('/console/buy')) {
      return 1;
    }
    if (location.startsWith('/console/cart')) return 2;
    if (location.startsWith('/console/orders')) return 3;
    if (location.startsWith('/console/billing') ||
        location.startsWith('/console/tickets') ||
        location.startsWith('/console/realname') ||
        location.startsWith('/console/profile') ||
        location.startsWith('/console/more')) {
      return 4;
    }
    return 0;
  }

  Widget _buildDesktopLayout(
    dynamic user,
    int selectedIndex,
    int unreadCount,
    String route,
    String siteName,
  ) {
    return Scaffold(
      body: SafeArea(
        child: Row(
          children: [
            NavigationRail(
              selectedIndex: selectedIndex,
              onDestinationSelected: (index) =>
                  _onDestinationSelected(index, _desktopItems),
              labelType: NavigationRailLabelType.all,
              leading: Column(
                children: [
                  const SizedBox(height: 16),
                  _buildLogo(),
                  const SizedBox(height: 32),
                ],
              ),
              trailing: Column(
                children: [
                  const Divider(),
                  _buildUserMenu(user),
                  const SizedBox(height: 16),
                ],
              ),
              destinations: _desktopItems
                  .map(
                    (item) => NavigationRailDestination(
                      icon: Icon(item.icon),
                      selectedIcon: Icon(item.selectedIcon),
                      label: Text(item.label),
                    ),
                  )
                  .toList(),
            ),
            const VerticalDivider(thickness: 1, width: 1),
            Expanded(
              child: Column(
                children: [
                  _buildTopBar(user, siteName, unreadCount, route),
                  Expanded(child: widget.child),
                ],
              ),
            ),
          ],
        ),
      ),
    );
  }

  Widget _buildMobileLayout(
    dynamic user,
    int selectedIndex,
    int unreadCount,
    String route,
    String siteName,
  ) {
    final colorScheme = Theme.of(context).colorScheme;
    return Scaffold(
      body: SafeArea(
        child: Column(
          children: [
            _buildMobileTopBar(user, unreadCount, route, siteName),
            Expanded(child: widget.child),
          ],
        ),
      ),
      bottomNavigationBar: ClipRect(
        child: BackdropFilter(
          filter: ImageFilter.blur(sigmaX: 30, sigmaY: 30),
          child: Container(
            decoration: _glassDecoration(colorScheme, top: true),
            child: BottomNavigationBar(
              currentIndex: selectedIndex,
              onTap: (index) =>
                  _onDestinationSelected(index, _mobilePrimaryItems),
              type: BottomNavigationBarType.fixed,
              backgroundColor: Colors.transparent,
              elevation: 0,
              items: _mobilePrimaryItems
                  .map(
                    (item) => BottomNavigationBarItem(
                      icon: Icon(item.icon),
                      activeIcon: Icon(item.selectedIcon),
                      label: item.label,
                    ),
                  )
                  .toList(),
            ),
          ),
        ),
      ),
    );
  }

  Widget _buildLogo() {
    final siteName = ref.read(siteProvider).siteName;
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 16),
      child: Column(
        children: [
          Container(
            width: 48,
            height: 48,
            decoration: BoxDecoration(
              color: AppColors.primary,
              borderRadius: BorderRadius.circular(8),
            ),
            child: const Icon(
              Icons.cloud_outlined,
              size: 28,
              color: Colors.white,
            ),
          ),
          const SizedBox(height: 8),
          Text(
            siteName.isNotEmpty ? siteName : AppStrings.appTitle,
            style: const TextStyle(fontSize: 14, fontWeight: FontWeight.bold),
          ),
        ],
      ),
    );
  }

  Widget _buildTopBar(
    dynamic user,
    String title,
    int unreadCount,
    String route,
  ) {
    final colorScheme = Theme.of(context).colorScheme;
    final onSurface = colorScheme.onSurface;
    return ClipRect(
      child: BackdropFilter(
        filter: ImageFilter.blur(sigmaX: 28, sigmaY: 28),
        child: Container(
          padding: const EdgeInsets.symmetric(horizontal: 24, vertical: 16),
          decoration: _glassDecoration(colorScheme, bottom: true),
          child: Row(
            children: [
              Text(
                title.isNotEmpty ? title : AppStrings.appTitle,
                style: TextStyle(
                  fontSize: 20,
                  fontWeight: FontWeight.bold,
                  color: onSurface,
                ),
              ),
              const Spacer(),
              _buildHeaderActions(unreadCount, route),
              if (user != null) _buildUserMenu(user),
            ],
          ),
        ),
      ),
    );
  }

  Widget _buildMobileTopBar(
    dynamic user,
    int unreadCount,
    String route,
    String siteName,
  ) {
    final colorScheme = Theme.of(context).colorScheme;
    final onSurface = colorScheme.onSurface;
    final isNarrow = MediaQuery.of(context).size.width < 390;
    return ClipRect(
      child: BackdropFilter(
        filter: ImageFilter.blur(sigmaX: 28, sigmaY: 28),
        child: Container(
          padding: EdgeInsets.symmetric(
            horizontal: isNarrow ? 12 : 16,
            vertical: isNarrow ? 8 : 12,
          ),
          decoration: _glassDecoration(colorScheme, bottom: true),
          child: Row(
            children: [
              Text(
                siteName.isNotEmpty ? siteName : AppStrings.appTitle,
                style: TextStyle(
                  fontSize: isNarrow ? 16 : 18,
                  fontWeight: FontWeight.bold,
                  color: onSurface,
                ),
              ),
              const Spacer(),
              _buildHeaderActions(unreadCount, route),
              if (user != null) _buildUserMenu(user),
            ],
          ),
        ),
      ),
    );
  }

  Widget _buildHeaderActions(int unreadCount, String route) {
    return Row(
      children: [
        IconButton(
          onPressed: () =>
              ref.read(pageRefreshProvider.notifier).state = RefreshEvent(
                route: route,
                nonce: DateTime.now().millisecondsSinceEpoch,
              ),
          icon: const Icon(Icons.refresh),
          tooltip: AppStrings.refresh,
        ),
        const SizedBox(width: 4),
        IconButton(
          onPressed: () {
            ref.read(notificationProvider.notifier).fetchUnreadCount();
            context.go('/console/notifications');
          },
          icon: Stack(
            clipBehavior: Clip.none,
            children: [
              const Icon(Icons.notifications_none),
              if (unreadCount > 0)
                Positioned(
                  right: -2,
                  top: -2,
                  child: Container(
                    padding: const EdgeInsets.symmetric(
                      horizontal: 4,
                      vertical: 1,
                    ),
                    decoration: BoxDecoration(
                      color: AppColors.danger,
                      borderRadius: BorderRadius.circular(10),
                    ),
                    child: Text(
                      unreadCount > 99 ? '99+' : '$unreadCount',
                      style: const TextStyle(fontSize: 10, color: Colors.white),
                    ),
                  ),
                ),
            ],
          ),
          tooltip: AppStrings.notifications,
        ),
        const SizedBox(width: 8),
      ],
    );
  }

  Widget _buildUserMenu(dynamic user) {
    final avatarUrl = resolveUserAvatarUrl(
      baseUrl: ApiClient.instance.dio.options.baseUrl,
      qq: user?.qq?.toString(),
      avatarUrl: user?.avatarUrl?.toString(),
      avatar: user?.avatar?.toString(),
    );
    return PopupMenuButton<String>(
      icon: avatarUrl.isNotEmpty
          ? CircleAvatar(backgroundImage: NetworkImage(avatarUrl))
          : CircleAvatar(
              backgroundColor: AppColors.primaryLight,
              child: Text(
                (user?.username ?? 'U')[0].toUpperCase(),
                style: const TextStyle(
                  fontSize: 16,
                  fontWeight: FontWeight.bold,
                  color: Colors.white,
                ),
              ),
            ),
      onSelected: (value) {
        if (value == 'profile') {
          context.go('/console/profile');
        }
        if (value == 'security') {
          context.go('/console/profile/security');
        }
        if (value == 'logout') {
          _logout();
        }
      },
      itemBuilder: (context) => [
        PopupMenuItem(value: 'profile', child: Text(user?.username ?? 'User')),
        const PopupMenuItem(
          value: 'security',
          child: Row(
            children: [
              Icon(Icons.security_outlined, size: 20),
              SizedBox(width: 12),
              Text('安全中心'),
            ],
          ),
        ),
        const PopupMenuDivider(),
        const PopupMenuItem(
          value: 'logout',
          child: Row(
            children: [
              Icon(Icons.logout, size: 20),
              SizedBox(width: 12),
              Text(AppStrings.logout),
            ],
          ),
        ),
      ],
    );
  }
}

/// 导航项
class NavigationItem {
  final IconData icon;
  final IconData selectedIcon;
  final String label;
  final String route;

  const NavigationItem({
    required this.icon,
    required this.selectedIcon,
    required this.label,
    required this.route,
  });
}
