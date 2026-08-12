//go:build unix

package pluginsign

import (
	"os"
	"syscall"
)

// isHardlinked reports whether info describes a file with more than one hard
// link (nlink > 1). Multiple names for the same inode can smuggle an extra,
// un-checksummed path past a naive walk on filesystems that do not deduplicate.
func isHardlinked(info os.FileInfo) bool {
	if st, ok := info.Sys().(*syscall.Stat_t); ok {
		return st.Nlink > 1
	}
	return false
}
