import 'package:flutter/material.dart';

/// 应用颜色主题
/// 参考Vue前端的Premium Blue主题
class AppColors {
  AppColors._();

  // 主色 - 蓝色
  static const Color primary = Color(0xFF0066FF);
  static const Color primaryLight = Color(0xFF3385FF);
  static const Color primaryDark = Color(0xFF0052CC);

  // 状态颜色
  static const Color success = Color(0xFF059669);
  static const Color warning = Color(0xFFD97706);
  static const Color danger = Color(0xFFDC2626);
  static const Color info = Color(0xFF0284C7);

  // 中性色
  static const Color white = Color(0xFFFFFFFF);
  static const Color black = Color(0xFF000000);

  // 灰色系
  static const Color gray50 = Color(0xFFF9FAFB);
  static const Color gray100 = Color(0xFFF3F4F6);
  static const Color gray200 = Color(0xFFE5E7EB);
  static const Color gray300 = Color(0xFFD1D5DB);
  static const Color gray400 = Color(0xFF9CA3AF);
  static const Color gray500 = Color(0xFF6B7280);
  static const Color gray600 = Color(0xFF4B5563);
  static const Color gray700 = Color(0xFF374151);
  static const Color gray800 = Color(0xFF1F2937);
  static const Color gray900 = Color(0xFF111827);

  // 背景色
  static const Color background = Color(0xFFF9FAFB);
  static const Color surface = Color(0xFFFFFFFF);
  static const Color surfaceVariant = Color(0xFFF3F4F6);

  // 暗色模式
  static const Color darkBackground = Color(0xFF0F1419);
  static const Color darkSurface = Color(0xFF1E2433);
  static const Color darkPrimary = Color(0xFF3B82F6);

  // VPS状态颜色
  static const Color vpsRunning = Color(0xFF059669);
  static const Color vpsStopped = Color(0xFFDC2626);
  static const Color vpsPending = Color(0xFFD97706);
  static const Color vpsSuspended = Color(0xFF6B7280);

  // 订单状态颜色
  static const Color orderPending = Color(0xFFD97706);
  static const Color orderPaid = Color(0xFF059669);
  static const Color orderCancelled = Color(0xFFDC2626);
  static const Color orderRefunded = Color(0xFF6B7280);
  static const Color orderCompleted = Color(0xFF0066FF);
}

/// 状态标签颜色扩展
extension StatusColorExtension on String {
  Color get vpsStatusColor {
    switch (toLowerCase()) {
      case 'running':
        return AppColors.vpsRunning;
      case 'stopped':
        return AppColors.vpsStopped;
      case 'pending':
        return AppColors.vpsPending;
      case 'suspended':
        return AppColors.vpsSuspended;
      default:
        return AppColors.gray500;
    }
  }

  Color get orderStatusColor {
    switch (toLowerCase()) {
      case 'pending':
        return AppColors.orderPending;
      case 'paid':
        return AppColors.orderPaid;
      case 'cancelled':
        return AppColors.orderCancelled;
      case 'refunded':
        return AppColors.orderRefunded;
      case 'completed':
        return AppColors.orderCompleted;
      default:
        return AppColors.gray500;
    }
  }
}
