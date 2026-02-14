class UpdateConfig {
  const UpdateConfig._();

  // Accepts host notes like "pkg,example.com" and normalizes to valid domain.
  static const String _rawHost = 'pkg,example.com';
  static final String serverBaseUrl =
      'https://${_rawHost.replaceAll(',', '.')}';
  static const String checkPath = '/api/v1/update/check';
}
