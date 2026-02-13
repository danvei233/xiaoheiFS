import 'package:intl/intl.dart';

/// 日期格式化工具类
class DateFormatter {
  DateFormatter._();

  static DateTime _toDisplayTime(DateTime dateTime) {
    return dateTime.isUtc ? dateTime.toLocal() : dateTime;
  }

  /// 常用日期格式
  static const String formatFull = 'yyyy-MM-dd HH:mm:ss';
  static const String formatDate = 'yyyy-MM-dd';
  static const String formatTime = 'HH:mm:ss';
  static const String formatMonth = 'yyyy-MM';
  static const String formatYear = 'yyyy';
  static const String formatCompact = 'yyyy/MM/dd HH:mm';

  /// 格式化日期时间
  static String format(DateTime? dateTime, [String pattern = formatFull]) {
    if (dateTime == null) return '-';
    return DateFormat(pattern).format(_toDisplayTime(dateTime));
  }

  /// 格式化时间戳（秒）
  static String formatTimestamp(int? timestamp, [String pattern = formatFull]) {
    if (timestamp == null) return '-';
    final dateTime = DateTime.fromMillisecondsSinceEpoch(timestamp * 1000);
    return DateFormat(pattern).format(dateTime);
  }

  /// 格式化时间戳（毫秒）
  static String formatTimestampMs(int? timestamp, [String pattern = formatFull]) {
    if (timestamp == null) return '-';
    final dateTime = DateTime.fromMillisecondsSinceEpoch(timestamp);
    return DateFormat(pattern).format(dateTime);
  }

  /// 格式化 ISO 8601 字符串
  static String formatIso(String? isoString, [String pattern = formatFull]) {
    if (isoString == null || isoString.isEmpty) return '-';
    try {
      final dateTime = DateTime.parse(isoString);
      return DateFormat(pattern).format(_toDisplayTime(dateTime));
    } catch (e) {
      return isoString;
    }
  }

  /// 相对时间格式化（如：3分钟前，2小时前）
  static String formatRelative(DateTime? dateTime) {
    if (dateTime == null) return '-';

    final now = DateTime.now();
    final difference = now.difference(dateTime);

    if (difference.inSeconds < 60) {
      return '刚刚';
    } else if (difference.inMinutes < 60) {
      return '${difference.inMinutes}分钟前';
    } else if (difference.inHours < 24) {
      return '${difference.inHours}小时前';
    } else if (difference.inDays < 7) {
      return '${difference.inDays}天前';
    } else if (difference.inDays < 30) {
      return '${(difference.inDays / 7).floor()}周前';
    } else if (difference.inDays < 365) {
      return '${(difference.inDays / 30).floor()}个月前';
    } else {
      return '${(difference.inDays / 365).floor()}年前';
    }
  }

  /// 解析多种日期格式
  static DateTime? parse(dynamic value) {
    if (value == null) return null;

    if (value is DateTime) {
      return _toDisplayTime(value);
    }

    if (value is String) {
      try {
        return _toDisplayTime(DateTime.parse(value));
      } catch (e) {
        return null;
      }
    }

    if (value is int) {
      if (value > 1000000000000) {
        return DateTime.fromMillisecondsSinceEpoch(value);
      } else {
        return DateTime.fromMillisecondsSinceEpoch(value * 1000);
      }
    }

    return null;
  }

  /// 计算剩余时间
  static String timeRemaining(DateTime? expireDate) {
    if (expireDate == null) return '-';

    final now = DateTime.now();
    final difference = expireDate.difference(now);

    if (difference.isNegative) {
      return '已过期';
    }

    if (difference.inDays > 0) {
      return '${difference.inDays}天后过期';
    } else if (difference.inHours > 0) {
      return '${difference.inHours}小时后过期';
    } else if (difference.inMinutes > 0) {
      return '${difference.inMinutes}分钟后过期';
    } else {
      return '即将过期';
    }
  }

  /// 判断是否即将过期（默认7天内）
  static bool isExpiringSoon(DateTime? date, {int days = 7}) {
    if (date == null) return false;
    final now = DateTime.now();
    final difference = date.difference(now);
    return difference.inDays <= days && difference.inDays >= 0;
  }
}
