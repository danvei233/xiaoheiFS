import 'dart:async';
import 'dart:io';

import 'package:firebase_messaging/firebase_messaging.dart';
import 'package:flutter/material.dart';
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
    _initPush();
  }

  @override
  void dispose() {
    _tokenSub?.cancel();
    super.dispose();
  }

  Future<void> _initPush() async {
    if (!Platform.isAndroid) return;
    final client = context.read<AppState>().apiClient;
    if (client == null) return;
    try {
      await FirebaseMessaging.instance.requestPermission();
      final token = await FirebaseMessaging.instance.getToken();
      if (token != null && token.isNotEmpty) {
        await _registerToken(client, token);
      }
      _tokenSub = FirebaseMessaging.instance.onTokenRefresh.listen((token) {
        final latest = context.read<AppState>().apiClient;
        if (latest != null && token.isNotEmpty) {
          _registerToken(latest, token);
        }
      });
    } catch (_) {}
  }

  Future<void> _registerToken(ApiClient client, String token) async {
    try {
      await client.postJson('/admin/api/v1/push-tokens', body: {
        'token': token,
        'platform': 'android',
      });
    } catch (_) {}
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      // 由各页面自行提供顶栏，底栏切换时不使用外层 AppBar
      body: IndexedStack(
        index: _index,
        children: _tabs.map((tab) => tab.widget).toList(),
      ),
      bottomNavigationBar: NavigationBarTheme(
        data: const NavigationBarThemeData(
          height: 48,
          labelTextStyle: WidgetStatePropertyAll(
            TextStyle(fontSize: 10, height: 1.0),
          ),
          iconTheme: WidgetStatePropertyAll(
            IconThemeData(size: 18),
          ),
        ),
        child: NavigationBar(
          selectedIndex: _index,
          labelBehavior: NavigationDestinationLabelBehavior.alwaysHide,
          onDestinationSelected: (value) {
            setState(() {
              _index = value;
            });
          },
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
