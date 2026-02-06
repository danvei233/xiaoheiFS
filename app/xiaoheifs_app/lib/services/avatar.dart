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
  final trimmedBase = baseUrl.endsWith('/')
      ? baseUrl.substring(0, baseUrl.length - 1)
      : baseUrl;
  if (qqValue.isNotEmpty && trimmedBase.isNotEmpty) {
    return '$trimmedBase/admin/api/v1/avatar/qq/${Uri.encodeComponent(qqValue)}';
  }
  final direct = (avatarUrl ?? '').trim();
  if (direct.isNotEmpty) return direct;
  return '';
}
