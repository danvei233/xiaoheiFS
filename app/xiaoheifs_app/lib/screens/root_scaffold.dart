import 'dart:async';
import 'dart:io';

import 'package:firebase_messaging/firebase_messaging.dart';
import 'package:flutter/material.dart';
import 'package:flutter_local_notifications/flutter_local_notifications.dart';
import 'package:provider/provider.dart';

import '../app_state.dart';
import '../services/api_client.dart';
import 'home_screen.dart';
import 'orders_screen.dart';
import 'servers_screen.dart';
import 'users_screen.dart';
import 'tickets_screen.dart';

class RootScaffold extends StatefulWidget {
  const RootScaffold({super.key});

  @override
  State<RootScaffold> createState() => _RootScaffoldState();
}

class _RootScaffoldState extends State<RootScaffold> {
  int _index = 0;
  StreamSubscription<String>? _tokenSub;
  StreamSubscription<RemoteMessage>? _messageSub;
  StreamSubscription<RemoteMessage>? _openMessageSub;
  final FlutterLocalNotificationsPlugin _localNotifications =
      FlutterLocalNotificationsPlugin();

  final List<_TabItem> _tabs = const [
    _TabItem(title: '主页', icon: Icons.dashboard, widget: HomeScreen()),
    _TabItem(title: '订单', icon: Icons.receipt_long, widget: OrdersScreen()),
    _TabItem(title: '用户管理', icon: Icons.people_alt, widget: UsersScreen()),
    _TabItem(title: '服务器管理', icon: Icons.dns, widget: ServersScreen()),
    _TabItem(title: '工单', icon: Icons.support_agent, widget: TicketsScreen()),
  ];

  @override
  void initState() {
    super.initState();
    _initLocalNotifications();
    _initPush();
  }

  @override
  void dispose() {
    _tokenSub?.cancel();
    _messageSub?.cancel();
    _openMessageSub?.cancel();
    super.dispose();
  }

  Future<void> _initLocalNotifications() async {
    if (!Platform.isAndroid) return;
    const android = AndroidInitializationSettings('ic_launcher');
    await _localNotifications.initialize(
      const InitializationSettings(android: android),
      onDidReceiveNotificationResponse: (response) {
        _openByPayload(response.payload);
      },
    );
    await _localNotifications
        .resolvePlatformSpecificImplementation<
          AndroidFlutterLocalNotificationsPlugin
        >()
        ?.requestNotificationsPermission();
  }

  Future<void> _initPush() async {
    // Firebase push disabled for open-source build.
    return;
  }

  Future<void> _registerToken(ApiClient client, String token) async {
    try {
      await client.postJson(
        '/admin/api/v1/push-tokens',
        body: {'token': token, 'platform': 'android'},
      );
    } catch (_) {}
  }

  Future<void> _handleForegroundMessage(RemoteMessage message) async {
    if (!mounted) return;
    final title = message.notification?.title ?? '新消息';
    final body = message.notification?.body ?? '你有一条新通知';
    final payload = _routePayloadFromMessage(message);

    ScaffoldMessenger.of(context).showSnackBar(
      SnackBar(
        content: Text('$title：$body'),
        action: SnackBarAction(
          label: '查看',
          onPressed: () => _openByPayload(payload),
        ),
      ),
    );

    await _localNotifications.show(
      message.messageId.hashCode,
      title,
      body,
      const NotificationDetails(
        android: AndroidNotificationDetails(
          'new_orders_channel',
          '管理端通知',
          channelDescription: '用于提醒管理员有新消息',
          importance: Importance.high,
          priority: Priority.high,
        ),
      ),
      payload: payload,
    );
  }

  void _handleOpenedMessage(RemoteMessage message) {
    _openByPayload(_routePayloadFromMessage(message));
  }

  String _routePayloadFromMessage(RemoteMessage message) {
    final route = message.data['route']?.toString();
    if (route != null && route.isNotEmpty) return route;
    final type = message.data['type']?.toString().toLowerCase() ?? '';
    if (type.contains('order')) return 'orders';
    return 'home';
  }

  void _openByPayload(String? payload) {
    if (!mounted) return;
    if (payload == null || payload.isEmpty || payload == 'home') {
      setState(() => _index = 0);
      return;
    }
    if (payload == 'orders' || payload.contains('order')) {
      setState(() => _index = 1);
      return;
    }
    setState(() => _index = 0);
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      body: IndexedStack(
        index: _index,
        children: _tabs.map((tab) => tab.widget).toList(),
      ),
      bottomNavigationBar: NavigationBarTheme(
        data: NavigationBarThemeData(
          height: 72,
          backgroundColor: Colors.white,
          indicatorColor: Theme.of(
            context,
          ).colorScheme.primaryContainer.withOpacity(0.7),
          labelTextStyle: const WidgetStatePropertyAll(
            TextStyle(fontSize: 12, height: 1.1, fontWeight: FontWeight.w600),
          ),
          iconTheme: WidgetStateProperty.resolveWith((states) {
            if (states.contains(WidgetState.selected)) {
              return IconThemeData(
                size: 24,
                color: Theme.of(context).colorScheme.primary,
              );
            }
            return IconThemeData(
              size: 22,
              color: Theme.of(context).colorScheme.onSurfaceVariant,
            );
          }),
        ),
        child: SafeArea(
          top: false,
          child: Container(
            margin: const EdgeInsets.fromLTRB(10, 0, 10, 8),
            decoration: BoxDecoration(
              color: Colors.white,
              borderRadius: BorderRadius.circular(18),
              border: Border.all(color: const Color(0xFFE2E8F0)),
              boxShadow: const [
                BoxShadow(
                  color: Color(0x220F172A),
                  blurRadius: 18,
                  offset: Offset(0, 6),
                ),
              ],
            ),
            child: NavigationBar(
              selectedIndex: _index,
              labelBehavior: NavigationDestinationLabelBehavior.alwaysShow,
              onDestinationSelected: (value) => setState(() => _index = value),
              destinations: _tabs
                  .map(
                    (tab) => NavigationDestination(
                      icon: Icon(tab.icon),
                      label: tab.title,
                    ),
                  )
                  .toList(),
            ),
          ),
        ),
      ),
    );
  }
}

class _TabItem {
  final String title;
  final IconData icon;
  final Widget widget;

  const _TabItem({
    required this.title,
    required this.icon,
    required this.widget,
  });
}
