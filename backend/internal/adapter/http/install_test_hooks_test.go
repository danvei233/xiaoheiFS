package http

// SetInstallLockPathForTest overrides the install lock path so installer-gating
// tests stay isolated without relying on env vars or the real app.config.yaml
// location. It is compiled only into the test binary, never into the production
// server, so the exported symbol cannot be reached from production code.
func SetInstallLockPathForTest(path string) {
	installLockPathOverride = path
}

// ResetInstallRateLimitersForTest clears the shared install rate limiter state so
// tests that deliberately exhaust a limit do not leak state into later tests.
func ResetInstallRateLimitersForTest() {
	installRateLimiter = newRateLimiter()
	checkAdminPathLimiter = newRateLimiter()
}
