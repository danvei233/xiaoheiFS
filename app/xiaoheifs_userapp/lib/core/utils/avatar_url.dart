String resolveUserAvatarUrl({
  required String baseUrl,
  String? qq,
  String? avatarUrl,
  String? avatar,
}) {
  var trimmedBase = baseUrl.trim();
  if (trimmedBase.endsWith('/')) {
    trimmedBase = trimmedBase.substring(0, trimmedBase.length - 1);
  }
  if (trimmedBase.endsWith('/api')) {
    trimmedBase = trimmedBase.substring(0, trimmedBase.length - 4);
  }

  final qqValue = (qq ?? '').trim();
  if (qqValue.isNotEmpty) {
    // 使用 QQ 官方头像 API
    return 'http://q1.qlogo.cn/g?b=qq&nk=$qqValue&s=100';
  }

  final direct = (avatarUrl ?? avatar ?? '').trim();
  if (direct.isEmpty) return '';
  if (direct.startsWith('http://') || direct.startsWith('https://')) {
    return direct;
  }
  if (direct.startsWith('//')) {
    return 'https:$direct';
  }
  if (trimmedBase.isEmpty) {
    return direct;
  }
  if (direct.startsWith('/')) {
    return '$trimmedBase$direct';
  }
  return '$trimmedBase/$direct';
}
