// Package pluginsign implements the single canonical file manifest used by both
// the plugin signing tool (cmd/tools/pluginsign) and the server-side verifier
// (internal/adapter/plugins/core). Sharing one walk means the signer and the
// verifier can never disagree about which files a signature covers, closing the
// TOCTOU / platform-divergence gap where a plugin could add payload the verifier
// does not check.
package pluginsign

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// FileEntry describes one regular file inside a plugin directory together with
// its sha256 digest. Rel is slash-separated and relative to the plugin root.
type FileEntry struct {
	Rel string
	Sum string
}

// IsBookkeepingFile reports whether name is one of the signing bookkeeping files
// (checksums.json / signature.sig). Note that only the two ledger files at the
// plugin ROOT are excluded from the signed payload (see walk); this predicate
// matches the bare file name, so callers that need the root-only rule must use
// isRootBookkeepingFile with the slash-separated relative path.
func IsBookkeepingFile(name string) bool {
	n := strings.TrimSpace(name)
	return strings.EqualFold(n, "checksums.json") || strings.EqualFold(n, "signature.sig")
}

// isRootBookkeepingFile reports whether rel is exactly the plugin-root
// checksums.json or signature.sig ledger file. A same-named file deeper in the
// tree is ordinary payload: it must be signed and covered by the content digest,
// or an attacker could hide a nested checksums.json/signature.sig past both.
func isRootBookkeepingFile(rel string) bool {
	if strings.Contains(rel, "/") {
		return false
	}
	return IsBookkeepingFile(rel)
}

// CanonicalManifest walks pluginDir and returns every regular file it contains
// (except the plugin-root checksums.json / signature.sig bookkeeping files) with
// its sha256, sorted by slash-separated relative path. It refuses non-regular
// files (FIFO, socket, device, symlink), hardlinked files, and case-folding path
// collisions so a package cannot hide payload behind a link, reuse an inode to
// slip an extra file past the checksummed set, or smuggle two paths that would
// alias each other on a case-insensitive filesystem (Windows NTFS, macOS HFS+/APFS).
func CanonicalManifest(pluginDir string) ([]FileEntry, error) {
	entries, err := walk(pluginDir)
	if err != nil {
		return nil, err
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Rel < entries[j].Rel })
	return entries, nil
}

func walk(pluginDir string) ([]FileEntry, error) {
	var entries []FileEntry
	// caseFolded tracks rel -> first-seen rel so two paths that differ only by
	// letter case (or by other case-folding) are rejected instead of being
	// verified as distinct files on Linux and silently aliasing on Windows/macOS.
	caseFolded := map[string]string{}
	err := filepath.WalkDir(pluginDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, rerr := filepath.Rel(pluginDir, path)
		if rerr != nil {
			return rerr
		}
		rel = filepath.ToSlash(rel)
		if rel == "" || strings.HasPrefix(rel, "/") || strings.Contains(rel, "..") {
			return fmt.Errorf("invalid canonical path %q", rel)
		}
		// A literal backslash in a relative path is a single filename on Linux but
		// a directory separator on Windows. Refuse it so a package cannot contain
		// a path that means one thing where it is signed and another where it is
		// extracted/executed (verifier/signer platform divergence).
		if strings.Contains(rel, `\`) {
			return fmt.Errorf("plugin contains a path with backslash separators: %q", rel)
		}
		// Only the plugin-root ledger files are excluded from the payload; a
		// nested checksums.json/signature.sig must be covered.
		if isRootBookkeepingFile(rel) {
			return nil
		}
		folded := foldKey(rel)
		if prev, dup := caseFolded[folded]; dup && prev != rel {
			return fmt.Errorf("plugin contains case-colliding paths: %q and %q", prev, rel)
		}
		caseFolded[folded] = rel

		info, err := d.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("plugin contains a non-regular file: %s", rel)
		}
		if isHardlinked(info) {
			return fmt.Errorf("plugin contains a hardlinked file: %s", rel)
		}
		sum, err := fileSHA256(path)
		if err != nil {
			return err
		}
		entries = append(entries, FileEntry{Rel: rel, Sum: sum})
		return nil
	})
	if err != nil {
		return nil, err
	}
	return entries, nil
}

// foldKey normalizes a slash-separated relative path for cross-platform
// collision detection. It models the case-insensitive, trailing-dot/space
// folding of Windows NTFS (and macOS HFS+/APFS case-insensitivity): each path
// segment is lowercased and trailing dots/spaces are trimmed, so two genuinely
// distinct paths on a case-sensitive system that would alias on a
// case-insensitive one are rejected instead of verified as distinct files. This
// is a conservative over-approximation — the same key on a case-sensitive
// system is rejected too, which is safe and keeps a single manifest verifiable
// everywhere.
func foldKey(rel string) string {
	rel = strings.ToLower(rel)
	parts := strings.Split(rel, "/")
	for i, p := range parts {
		p = strings.TrimRight(p, ". ")
		parts[i] = p
	}
	return strings.Join(parts, "/")
}

// FilesMap converts a canonical manifest into a rel -> sha256 map.
func FilesMap(entries []FileEntry) map[string]string {
	m := make(map[string]string, len(entries))
	for _, e := range entries {
		m[e.Rel] = e.Sum
	}
	return m
}

// ContentDigest hashes the canonical manifest (per file: relative path, NUL,
// sha256) so any change to a file's content, the file set, or a file's path
// changes the digest. The server persists it at install time as the tamper
// anchor for unsigned/untrusted plugins and re-checks it on every start.
func ContentDigest(pluginDir string) (string, error) {
	entries, err := CanonicalManifest(pluginDir)
	if err != nil {
		return "", err
	}
	h := sha256.New()
	for _, e := range entries {
		io.WriteString(h, e.Rel)
		h.Write([]byte{0})
		io.WriteString(h, e.Sum)
		h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func fileSHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
