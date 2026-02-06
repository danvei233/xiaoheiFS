class Session {
  final String apiUrl;
  final String username;
  final String? token;
  final String? apiKey;
  final String? email;
  final String authType;

  const Session({
    required this.apiUrl,
    required this.username,
    required this.authType,
    this.token,
    this.apiKey,
    this.email,
  });

  Session copyWith({
    String? apiUrl,
    String? username,
    String? token,
    String? apiKey,
    String? email,
    String? authType,
  }) {
    return Session(
      apiUrl: apiUrl ?? this.apiUrl,
      username: username ?? this.username,
      token: token ?? this.token,
      apiKey: apiKey ?? this.apiKey,
      email: email ?? this.email,
      authType: authType ?? this.authType,
    );
  }
}
