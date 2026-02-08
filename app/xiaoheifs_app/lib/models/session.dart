class Session {
  final String apiUrl;
  final String username;
  final String? token;
  final String? refreshToken;
  final DateTime? tokenExpiresAt;
  final String? apiKey;
  final String? email;
  final String authType;

  const Session({
    required this.apiUrl,
    required this.username,
    required this.authType,
    this.token,
    this.refreshToken,
    this.tokenExpiresAt,
    this.apiKey,
    this.email,
  });

  Session copyWith({
    String? apiUrl,
    String? username,
    String? token,
    String? refreshToken,
    DateTime? tokenExpiresAt,
    String? apiKey,
    String? email,
    String? authType,
  }) {
    return Session(
      apiUrl: apiUrl ?? this.apiUrl,
      username: username ?? this.username,
      token: token ?? this.token,
      refreshToken: refreshToken ?? this.refreshToken,
      tokenExpiresAt: tokenExpiresAt ?? this.tokenExpiresAt,
      apiKey: apiKey ?? this.apiKey,
      email: email ?? this.email,
      authType: authType ?? this.authType,
    );
  }
}
