import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';

class AppNavigator {
  AppNavigator._();

  static final GlobalKey<NavigatorState> navigatorKey =
      GlobalKey<NavigatorState>();
  static final GlobalKey<ScaffoldMessengerState> messengerKey =
      GlobalKey<ScaffoldMessengerState>();

  static BuildContext? get context => navigatorKey.currentContext;

  static void showSnackBar(String message, {Color? backgroundColor}) {
    final messenger = messengerKey.currentState;
    if (messenger == null) return;
    messenger.showSnackBar(
      SnackBar(
        content: Text(message),
        backgroundColor: backgroundColor,
      ),
    );
  }

  static Future<bool?> showConfirmDialog({
    required String title,
    required String content,
    String confirmText = '确认',
    String cancelText = '取消',
  }) {
    final ctx = context;
    if (ctx == null) return Future.value(false);
    return showDialog<bool>(
      context: ctx,
      builder: (context) => AlertDialog(
        title: Text(title),
        content: Text(content),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(context, false),
            child: Text(cancelText),
          ),
          TextButton(
            onPressed: () => Navigator.pop(context, true),
            child: Text(confirmText),
          ),
        ],
      ),
    );
  }

  static void goToLogin() {
    final ctx = context;
    if (ctx == null) return;
    final router = GoRouter.of(ctx);
    final current = router.routeInformationProvider.value.uri.toString();
    final redirect = Uri.encodeComponent(current);
    router.go('/login?redirect=$redirect');
  }

  static void goToRealname() {
    final ctx = context;
    if (ctx == null) return;
    GoRouter.of(ctx).go('/console/realname');
  }
}
