Map<String, String> avatarHeaders({
  String? token,
  String? apiKey,
  String? baseUrl,
  String? url,
}) {
  final auth = token?.isNotEmpty == true
      ? token
      : (apiKey?.isNotEmpty == true ? apiKey : null);
  if (auth == null) return {};
  // On native platforms Image.network does not enforce CORS, so a
  // user-controlled absolute avatar_url would otherwise exfiltrate the admin
  // token to an arbitrary host. Only same-origin avatars (relative paths or
  // the backend /admin/api/v1/avatar/qq proxy) may carry credentials.
  if (baseUrl == null || baseUrl.isEmpty) return {};
  if (url == null || url.isEmpty) return {};
  if (!_isSameOrigin(baseUrl, url)) return {};
  return {'Authorization': 'Bearer $auth'};
}

bool _isSameOrigin(String baseUrl, String url) {
  final base = Uri.tryParse(baseUrl);
  final target = Uri.tryParse(url);
  if (base == null || target == null) return false;
  if (!base.hasScheme || !base.hasAuthority) return false;
  if (!target.hasScheme || !target.hasAuthority) return false;
  return base.scheme == target.scheme && base.authority == target.authority;
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
