import 'package:intl/intl.dart';

/// 金额格式化工具类
class MoneyFormatter {
  MoneyFormatter._();

  /// 格式化金额
  /// [amount] 金额数值
  /// [currency] 货币代码，默认 CNY
  /// [decimalDigits] 小数位数，默认 2 位
  static String format(
    double? amount, {
    String currency = '¥',
    int decimalDigits = 2,
  }) {
    if (amount == null) return '${currency}0.00';

    final formatter = NumberFormat.currency(
      symbol: currency,
      decimalDigits: decimalDigits,
      customPattern: '#,##0.00',
    );

    return formatter.format(amount);
  }

  /// 简洁格式化金额（不带货币符号）
  static String formatSimple(double? amount, {int decimalDigits = 2}) {
    if (amount == null) return '0.00';

    final formatter = NumberFormat('#,##0.00', 'en_US');
    return formatter.format(amount);
  }

  /// 格式化为千分位（如：1.2K, 1.5M）
  static String formatCompact(double? amount) {
    if (amount == null) return '0';

    if (amount >= 1000000) {
      return '${(amount / 1000000).toStringAsFixed(1)}M';
    } else if (amount >= 1000) {
      return '${(amount / 1000).toStringAsFixed(1)}K';
    } else {
      return amount.toStringAsFixed(0);
    }
  }

  /// 解析金额字符串
  static double? parse(String? value) {
    if (value == null || value.isEmpty) return null;
    try {
      final cleanValue = value.replaceAll(RegExp(r'[¥$,\s]'), '');
      return double.parse(cleanValue);
    } catch (e) {
      return null;
    }
  }

  /// 计算折扣后的价格
  static double calculateDiscount(
    double originalPrice,
    double discountPercent,
  ) {
    return originalPrice * (1 - discountPercent / 100);
  }

  /// 计算折扣百分比
  static double calculateDiscountPercent(
    double originalPrice,
    double discountedPrice,
  ) {
    if (originalPrice == 0) return 0;
    return ((originalPrice - discountedPrice) / originalPrice) * 100;
  }

  /// 格式化折扣显示
  static String formatDiscount(double percent) {
    return '${percent.toStringAsFixed(0)}% OFF';
  }
}
