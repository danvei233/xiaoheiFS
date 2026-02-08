/// 验证器工具类
class Validators {
  Validators._();

  /// 验证用户名
  /// 长度3-20字符，只能包含字母、数字、下划线
  static String? validateUsername(String? value) {
    if (value == null || value.isEmpty) {
      return '请输入用户名';
    }
    if (value.length < 3 || value.length > 20) {
      return '用户名长度应为3-20个字符';
    }
    if (!RegExp(r'^[a-zA-Z0-9_]+$').hasMatch(value)) {
      return '用户名只能包含字母、数字和下划线';
    }
    return null;
  }

  /// 验证密码
  /// 最小长度6位
  static String? validatePassword(String? value) {
    if (value == null || value.isEmpty) {
      return '请输入密码';
    }
    if (value.length < 6) {
      return '密码长度至少6位';
    }
    return null;
  }

  /// 验证邮箱
  static String? validateEmail(String? value) {
    if (value == null || value.isEmpty) {
      return '请输入邮箱';
    }
    if (!RegExp(r'^[\w-\.]+@([\w-]+\.)+[\w-]{2,4}$').hasMatch(value)) {
      return '请输入有效的邮箱地址';
    }
    return null;
  }

  /// 验证手机号（中国大陆）
  static String? validatePhone(String? value) {
    if (value == null || value.isEmpty) {
      return '请输入手机号';
    }
    if (!RegExp(r'^1[3-9]\d{9}$').hasMatch(value)) {
      return '请输入有效的手机号';
    }
    return null;
  }

  /// 验证QQ号
  static String? validateQQ(String? value) {
    if (value == null || value.isEmpty) {
      return '请输入QQ号';
    }
    if (!RegExp(r'^[1-9]\d{4,10}$').hasMatch(value)) {
      return '请输入有效的QQ号';
    }
    return null;
  }

  /// 验证身份证号
  static String? validateIdNumber(String? value) {
    if (value == null || value.isEmpty) {
      return '请输入身份证号';
    }
    if (value.length != 18) {
      return '身份证号应为18位';
    }
    // 简单验证格式
    if (!RegExp(r'^\d{17}[\dXx]$').hasMatch(value)) {
      return '请输入有效的身份证号';
    }
    return null;
  }

  /// 验证URL
  static String? validateUrl(String? value) {
    if (value == null || value.isEmpty) {
      return '请输入URL';
    }
    if (!RegExp(r'^https?://.+').hasMatch(value)) {
      return '请输入有效的URL';
    }
    return null;
  }

  /// 验证非空
  static String? validateRequired(String? value, [String fieldName = '此项']) {
    if (value == null || value.isEmpty) {
      return '$fieldName不能为空';
    }
    return null;
  }

  /// 验证最小长度
  static String? validateMinLength(String? value, int minLength) {
    if (value == null || value.isEmpty) {
      return '此项不能为空';
    }
    if (value.length < minLength) {
      return '长度至少为$minLength个字符';
    }
    return null;
  }

  /// 验证最大长度
  static String? validateMaxLength(String? value, int maxLength) {
    if (value != null && value.length > maxLength) {
      return '长度最多为$maxLength个字符';
    }
    return null;
  }

  /// 验证数字范围
  static String? validateRange(
    String? value,
    num min,
    num max, {
    String? fieldName,
  }) {
    final number = num.tryParse(value ?? '');
    if (number == null) {
      return '${fieldName ?? '此项'}必须是数字';
    }
    if (number < min || number > max) {
      return '${fieldName ?? '此项'}应在$min到$max之间';
    }
    return null;
  }
}
