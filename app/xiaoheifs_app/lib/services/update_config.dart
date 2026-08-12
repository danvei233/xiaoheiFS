class UpdateConfig {
  const UpdateConfig._();

  /// Base URL of the self-hosted update manifest server.
  ///
  /// Build-time injectable via `--dart-define=UPDATE_SERVER_URL=...` (the
  /// release workflow passes the operator's real URL so the placeholder is
  /// never shipped). Defaults to a reserved placeholder domain so the update
  /// check fails closed. Mutable so it can be configured at runtime /
  /// overridden in tests.
  static String serverBaseUrl = String.fromEnvironment(
    'UPDATE_SERVER_URL',
    defaultValue: 'https://pkg.example.com',
  );
  static const String checkPath = '/api/v1/update/check';
}
