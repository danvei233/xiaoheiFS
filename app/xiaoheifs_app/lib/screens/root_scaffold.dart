import 'package:flutter/material.dart';

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

  final List<_TabItem> _tabs = const [
    _TabItem(title: '主页', icon: Icons.dashboard, widget: HomeScreen()),
    _TabItem(title: '订单', icon: Icons.receipt_long, widget: OrdersScreen()),
    _TabItem(title: '用户管理', icon: Icons.people_alt, widget: UsersScreen()),
    _TabItem(title: '服务器管理', icon: Icons.dns, widget: ServersScreen()),
    _TabItem(title: '工单', icon: Icons.support_agent, widget: TicketsScreen()),
  ];

  @override
  Widget build(BuildContext context) {
    final active = _tabs[_index];
    return Scaffold(
      // 主页不显示外层 AppBar，因为 HomeScreen 有自己的 SliverAppBar
      appBar: _index == 0 ? null : AppBar(title: Text(active.title)),
      body: IndexedStack(
        index: _index,
        children: _tabs.map((tab) => tab.widget).toList(),
      ),
      bottomNavigationBar: NavigationBar(
        selectedIndex: _index,
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
