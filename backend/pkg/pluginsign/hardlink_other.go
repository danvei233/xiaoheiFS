//go:build !unix

package pluginsign

import "os"

// isHardlinked reports whether info describes a file with more than one hard
// link. On platforms without a portable link count the walk treats every file
// as a single link; the symlink / device / socket rejection still applies.
func isHardlinked(os.FileInfo) bool {
	return false
}
