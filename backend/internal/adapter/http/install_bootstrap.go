package http

// NewInstallBootstrapServer starts HTTP routes required for first-time install
// without requiring a configured database connection.
func NewInstallBootstrapServer(jwtSecret string) *Server {
	handler := NewHandlerWithServices(
		nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
		nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
		jwtSecret,
		nil, nil, nil,
	)
	middleware := NewMiddleware(jwtSecret, nil, nil)
	return NewServer(handler, middleware)
}
