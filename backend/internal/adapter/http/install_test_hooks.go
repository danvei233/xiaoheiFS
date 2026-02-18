package http

// SetInstallLockPathForTest overrides the install lock path.
// It exists to keep installer-gating tests isolated without relying on env vars.
func SetInstallLockPathForTest(path string) {
	installLockPathOverride = path
}
