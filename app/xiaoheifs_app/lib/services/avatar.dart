Map<String, String> avatarHeaders({String? token, String? apiKey}) {
  final auth = token?.isNotEmpty == true
      ? token
      : (apiKey?.isNotEmpty == true ? apiKey : null);
  if (auth == null) return {};
  return {'Authorization': 'Bearer $auth'};
}

String resolveAvatarUrl({
  required String baseUrl,
  String? qq,
  String? avatarUrl,
}) {
  final qqValue = (qq ?? '').trim();
  var trimmedBase = baseUrl.trim();
  if (trimmedBase.endsWith('/')) {
    trimmedBase = trimmedBase.substring(0, trimmedBase.length - 1);
  }
  // Admin avatar API is rooted at /admin/api; strip trailing /api if present.
  if (trimmedBase.endsWith('/api')) {
    trimmedBase = trimmedBase.substring(0, trimmedBase.length - 4);
  }

  if (qqValue.isNotEmpty && trimmedBase.isNotEmpty) {
    return '$trimmedBase/admin/api/v1/avatar/qq/${Uri.encodeComponent(qqValue)}';
  }

  final direct = (avatarUrl ?? '').trim();
  if (direct.isNotEmpty) {
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

  return '';
}
