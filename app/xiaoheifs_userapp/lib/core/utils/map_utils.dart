T? pick<T>(Map<String, dynamic> map, List<String> keys) {
  for (final key in keys) {
    if (map.containsKey(key) && map[key] != null) {
      return map[key] as T;
    }
  }
  return null;
}

Map<String, dynamic> ensureMap(dynamic value) {
  if (value is Map<String, dynamic>) return value;
  if (value is Map) {
    return value.map((key, val) => MapEntry(key.toString(), val));
  }
  return {};
}
