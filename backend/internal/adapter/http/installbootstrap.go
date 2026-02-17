package http

// NewInstallBootstrapServer starts HTTP routes required for first-time install
// without requiring a configured database connection.
func NewInstallBootstrapServer(jwtSecret string) *Server {
	handler := NewHandler(HandlerDeps{JWTSecret: jwtSecret})
	middleware := NewMiddleware(jwtSecret, nil, nil)
	return NewServer(handler, middleware)
}
