package plugins

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"fmt"
	"xiaoheiplay/internal/domain"
)

type InstallResult struct {
	Category        string
	PluginID        string
	PluginDir       string
	SignatureStatus domain.PluginSignatureStatus
	Manifest        Manifest
	// Digest is the sha256 of the raw archive bytes. It binds the pre-commit
	// trust gate to the exact package being committed: the admin second factor
	// is granted for this digest, and the commit re-hashes its stream and
	// refuses to extract anything that does not match.
	Digest string
}

// Extraction safety limits. The compressed upload is read into memory (zip) or
// streamed (tar.gz); these bounds prevent zip/gzip bombs and archives with a
// pathological number of entries from exhausting memory or disk.
//
// They are vars (not consts) so tests can lower them to verify the limits
// without materializing hundreds of megabytes; production uses the defaults.
var (
	maxPluginUploadBytes    = int64(256 << 20) // 256 MiB compressed upload
	maxPluginExtractBytes   = int64(512 << 20) // 512 MiB total uncompressed payload
	maxPluginExtractPerFile = int64(256 << 20) // 256 MiB single file
	maxPluginExtractFiles   = 10000            // max archive entries
)

// InstallPackage extracts and validates a plugin archive and, when commit is
// true, atomically copies it into baseDir/<category>/<pluginID>. When commit is
// false it only validates (manifest, entry, signature, bounds) and returns the
// result without writing anything under baseDir — this is the pre-commit trust
// gate that lets the caller require a second factor for untrusted plugins
// before any files land in the plugin base directory. The returned digest lets
// a caller bind the trust confirmation to the exact package being committed.
func InstallPackage(baseDir string, filename string, r io.Reader, officialKeys []ed25519.PublicKey, commit bool) (InstallResult, error) {
	return installPackage(baseDir, filename, r, officialKeys, commit, "")
}

// installPackage is InstallPackage with an optional expectedDigest: when set,
// the incoming stream must hash to that value before any extraction, so the
// commit can never process a different package than the one a trust
// confirmation was granted for (validate-A/commit-B swap).
func installPackage(baseDir string, filename string, r io.Reader, officialKeys []ed25519.PublicKey, commit bool, expectedDigest string) (InstallResult, error) {
	baseDir = strings.TrimSpace(baseDir)
	if baseDir == "" {
		return InstallResult{}, fmt.Errorf("missing base dir")
	}

	tmpRoot, err := os.MkdirTemp("", "xiaoheiplay-plugin-install-*")
	if err != nil {
		return InstallResult{}, err
	}
	defer os.RemoveAll(tmpRoot)

	// Spool the archive to a temp file while hashing it. The digest is computed
	// over exactly the bytes that are later extracted, so there is no re-read
	// window in which the payload could differ from what was approved.
	uploadPath, digest, err := spoolArchive(tmpRoot, r)
	if err != nil {
		return InstallResult{}, err
	}
	if expectedDigest != "" && digest != expectedDigest {
		return InstallResult{}, fmt.Errorf("package digest mismatch")
	}
	uf, err := os.Open(uploadPath)
	if err != nil {
		return InstallResult{}, err
	}
	defer uf.Close()

	ext := strings.ToLower(filename)
	switch {
	case strings.HasSuffix(ext, ".zip"):
		if err := extractZip(tmpRoot, uf); err != nil {
			return InstallResult{}, err
		}
	case strings.HasSuffix(ext, ".tar.gz") || strings.HasSuffix(ext, ".tgz"):
		if err := extractTarGz(tmpRoot, uf); err != nil {
			return InstallResult{}, err
		}
	default:
		return InstallResult{}, fmt.Errorf("unsupported package type")
	}

	manifestPath, err := findSingleManifest(tmpRoot)
	if err != nil {
		return InstallResult{}, err
	}
	pluginDir := filepath.Dir(manifestPath)
	rel, _ := filepath.Rel(tmpRoot, pluginDir)
	rel = filepath.ToSlash(rel)
	category, pluginID, err := parsePluginDirFromRel(rel)
	if err != nil {
		return InstallResult{}, err
	}

	m, err := ReadManifest(pluginDir)
	if err != nil {
		return InstallResult{}, err
	}
	if m.PluginID != pluginID {
		return InstallResult{}, fmt.Errorf("manifest plugin_id mismatch")
	}
	// test_only plugins (e.g. mockpay) must never be installed through the
	// admin upload path in production: the packaging scripts already exclude
	// them from releases, this is the runtime backstop.
	if err := m.RejectTestOnly(false); err != nil {
		return InstallResult{}, err
	}
	entry, err := ResolveEntry(pluginDir, m)
	if err != nil {
		if len(entry.SupportedPlatforms) > 0 {
			return InstallResult{}, fmt.Errorf("%s", "unsupported platform "+entry.Platform+", supported: "+strings.Join(entry.SupportedPlatforms, ", "))
		}
		return InstallResult{}, err
	}

	sigStatus, err := VerifySignature(pluginDir, officialKeys)
	if err != nil {
		return InstallResult{}, err
	}

	finalDir := filepath.Join(baseDir, category, pluginID)
	if fileExists(finalDir) {
		return InstallResult{}, fmt.Errorf("plugin already installed")
	}
	if !commit {
		// Pre-commit trust gate: no files are written under baseDir. The temp
		// extraction is removed by the defer above.
		return InstallResult{
			Category:        category,
			PluginID:        pluginID,
			SignatureStatus: sigStatus,
			Manifest:        m,
			Digest:          digest,
		}, nil
	}
	if err := os.MkdirAll(filepath.Dir(finalDir), 0o755); err != nil {
		return InstallResult{}, err
	}
	// Copy into a sibling temp dir and atomically rename into place. A crash
	// mid-copy then never leaves a partial final dir that blocks reinstall or is
	// imported as a damaged plugin by the next bootstrap scan. The hidden
	// ".install-" prefix keeps a leftover temp dir out of scanDiskPlugins.
	tmpDir, err := os.MkdirTemp(filepath.Dir(finalDir), "."+pluginID+".install-*")
	if err != nil {
		return InstallResult{}, err
	}
	defer os.RemoveAll(tmpDir)
	if err := os.Chmod(tmpDir, 0o755); err != nil {
		return InstallResult{}, err
	}
	if err := copyDir(pluginDir, tmpDir); err != nil {
		return InstallResult{}, err
	}
	if err := os.Rename(tmpDir, finalDir); err != nil {
		// A concurrent install may have won the race; report that consistently.
		if fileExists(finalDir) {
			return InstallResult{}, fmt.Errorf("plugin already installed")
		}
		return InstallResult{}, err
	}
	return InstallResult{
		Category:        category,
		PluginID:        pluginID,
		PluginDir:       finalDir,
		SignatureStatus: sigStatus,
		Manifest:        m,
		Digest:          digest,
	}, nil
}

// spoolArchive writes the incoming archive to a uniquely named temp file under
// tmpRoot while computing its sha256, enforcing the compressed upload bound. It
// returns the file path and the hex digest.
func spoolArchive(tmpRoot string, r io.Reader) (string, string, error) {
	uf, err := os.CreateTemp(tmpRoot, ".archive-*")
	if err != nil {
		return "", "", err
	}
	path := uf.Name()
	defer uf.Close()
	h := sha256.New()
	n, err := io.Copy(io.MultiWriter(uf, h), io.LimitReader(r, maxPluginUploadBytes+1))
	if err != nil {
		_ = os.Remove(path)
		return "", "", err
	}
	if n > maxPluginUploadBytes {
		_ = os.Remove(path)
		return "", "", fmt.Errorf("plugin archive exceeds size limit")
	}
	return path, hex.EncodeToString(h.Sum(nil)), nil
}

func parsePluginDirFromRel(rel string) (category, pluginID string, err error) {
	rel = strings.Trim(rel, "/")
	parts := strings.Split(rel, "/")
	// Support:
	// - plugins/<category>/<plugin_id>/*
	// - <category>/<plugin_id>/*
	if len(parts) >= 3 && parts[0] == "plugins" {
		category = strings.TrimSpace(parts[1])
		pluginID = strings.TrimSpace(parts[2])
	} else if len(parts) >= 2 {
		category = strings.TrimSpace(parts[0])
		pluginID = strings.TrimSpace(parts[1])
	}
	if category == "" || pluginID == "" {
		return "", "", fmt.Errorf("invalid plugin directory")
	}
	if strings.Contains(category, "..") || strings.Contains(pluginID, "..") {
		return "", "", fmt.Errorf("invalid plugin directory")
	}
	if strings.Contains(category, ":") || strings.Contains(pluginID, ":") {
		return "", "", fmt.Errorf("invalid plugin directory")
	}
	return category, pluginID, nil
}

func extractZip(dst string, r io.Reader) error {
	b, err := io.ReadAll(io.LimitReader(r, maxPluginUploadBytes+1))
	if err != nil {
		return err
	}
	if int64(len(b)) > maxPluginUploadBytes {
		return fmt.Errorf("plugin archive exceeds size limit")
	}
	var totalDeclared int64
	var totalWritten int64
	fileCount := 0
	zr, err := zip.NewReader(bytes.NewReader(b), int64(len(b)))
	if err != nil {
		// Fallback: write to tmp file then OpenReader (more robust for binary data).
		tmp := filepath.Join(dst, "__upload.zip")
		if werr := os.WriteFile(tmp, b, 0o600); werr != nil {
			return err
		}
		zf, oerr := zip.OpenReader(tmp)
		if oerr != nil {
			return err
		}
		defer zf.Close()
		for _, f := range zf.File {
			if err := extractZipFile(dst, f, &totalDeclared, &totalWritten, &fileCount); err != nil {
				return err
			}
		}
		return nil
	}
	for _, f := range zr.File {
		if err := extractZipFile(dst, f, &totalDeclared, &totalWritten, &fileCount); err != nil {
			return err
		}
	}
	return nil
}

// extractZipFile writes one zip entry to disk. Per-file and global limits are
// enforced against the ACTUAL bytes written (via copyBounded), never just the
// central-directory declared sizes: a forged UncompressedSize64 or a decompressor
// that emits more data than it declares must fail before it can overrun the disk.
// A declared-total fast-fail is also kept so a manifest that claims a huge total
// is rejected before any decompression work. The caller's temp root is removed by
// installPackage's defer on any error, and a partial entry file is removed here.
func extractZipFile(dst string, f *zip.File, totalDeclared, totalWritten *int64, fileCount *int) error {
	name := filepath.ToSlash(f.Name)
	if name == "" || strings.HasPrefix(name, "/") || strings.Contains(name, "..") || strings.Contains(name, ":") {
		return fmt.Errorf("invalid zip entry path")
	}
	*fileCount++
	if *fileCount > maxPluginExtractFiles {
		return fmt.Errorf("plugin archive contains too many files")
	}
	if f.UncompressedSize64 > uint64(maxPluginExtractPerFile) {
		return fmt.Errorf("plugin archive entry exceeds size limit")
	}
	*totalDeclared += int64(f.UncompressedSize64)
	if *totalDeclared > maxPluginExtractBytes {
		return fmt.Errorf("plugin archive exceeds total size limit")
	}
	target := filepath.Join(dst, filepath.FromSlash(name))
	if f.FileInfo().IsDir() {
		return os.MkdirAll(target, 0o755)
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	rc, err := f.Open()
	if err != nil {
		return err
	}
	defer rc.Close()
	w, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o755)
	if err != nil {
		return err
	}
	n, cerr := copyBounded(w, rc, maxPluginExtractPerFile)
	closeErr := w.Close()
	if cerr != nil {
		_ = os.Remove(target)
		return cerr
	}
	if closeErr != nil {
		_ = os.Remove(target)
		return closeErr
	}
	*totalWritten += n
	if *totalWritten > maxPluginExtractBytes {
		return fmt.Errorf("plugin archive exceeds total size limit")
	}
	return nil
}

func extractTarGz(dst string, r io.Reader) error {
	gr, err := gzip.NewReader(io.LimitReader(r, maxPluginUploadBytes+1))
	if err != nil {
		return err
	}
	defer gr.Close()
	tr := tar.NewReader(gr)
	var totalDeclared int64
	var totalWritten int64
	fileCount := 0
	// lastTarget tracks the most recently written regular file so a partial
	// entry is removed if a later header parse fails or the stream is truncated.
	lastTarget := ""
	for {
		h, err := tr.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			if lastTarget != "" {
				_ = os.Remove(lastTarget)
			}
			return err
		}
		name := filepath.ToSlash(h.Name)
		if name == "" || strings.HasPrefix(name, "/") || strings.Contains(name, "..") || strings.Contains(name, ":") {
			if lastTarget != "" {
				_ = os.Remove(lastTarget)
			}
			return fmt.Errorf("invalid tar entry path")
		}
		fileCount++
		if fileCount > maxPluginExtractFiles {
			if lastTarget != "" {
				_ = os.Remove(lastTarget)
			}
			return fmt.Errorf("plugin archive contains too many files")
		}
		if h.Size > maxPluginExtractPerFile {
			if lastTarget != "" {
				_ = os.Remove(lastTarget)
			}
			return fmt.Errorf("plugin archive entry exceeds size limit")
		}
		totalDeclared += h.Size
		if totalDeclared > maxPluginExtractBytes {
			if lastTarget != "" {
				_ = os.Remove(lastTarget)
			}
			return fmt.Errorf("plugin archive exceeds total size limit")
		}
		target := filepath.Join(dst, filepath.FromSlash(name))
		lastTarget = ""
		switch h.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			w, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o755)
			if err != nil {
				return err
			}
			// Bounds are enforced on actual written bytes, not just the tar header
			// size, so a mis-sized entry cannot exhaust disk before the reader
			// reports a mismatch.
			n, cerr := copyBounded(w, tr, maxPluginExtractPerFile)
			closeErr := w.Close()
			if cerr != nil {
				_ = os.Remove(target)
				return cerr
			}
			if closeErr != nil {
				_ = os.Remove(target)
				return closeErr
			}
			totalWritten += n
			if totalWritten > maxPluginExtractBytes {
				_ = os.Remove(target)
				return fmt.Errorf("plugin archive exceeds total size limit")
			}
			lastTarget = target
		default:
			// ignore other types
		}
	}
}

// copyBounded copies from r to w, refusing to write more than limit bytes. It
// returns the number of bytes actually written so callers can enforce global
// totals against real output instead of archive-declared sizes.
func copyBounded(w io.Writer, r io.Reader, limit int64) (int64, error) {
	lw := &limitedWriter{w: w, remaining: limit}
	n, err := io.Copy(lw, r)
	if lw.exceeded {
		if err == nil {
			err = fmt.Errorf("plugin archive entry exceeds size limit")
		}
		return n, err
	}
	return n, err
}

// limitedWriter fails once more than limit bytes would be written, truncating the
// write that crosses the boundary and surfacing a size-limit error to io.Copy.
type limitedWriter struct {
	w         io.Writer
	remaining int64
	exceeded  bool
}

func (l *limitedWriter) Write(p []byte) (int, error) {
	if l.remaining <= 0 {
		l.exceeded = true
		return 0, fmt.Errorf("plugin archive entry exceeds size limit")
	}
	if int64(len(p)) > l.remaining {
		p = p[:l.remaining]
		l.exceeded = true
	}
	n, err := l.w.Write(p)
	l.remaining -= int64(n)
	if err != nil {
		return n, err
	}
	if l.exceeded {
		// The write crossed the boundary: report the size-limit error instead of
		// letting io.Copy turn the short write into a confusing io.ErrShortWrite.
		// n counts the partial bytes actually written so global totals are
		// accurate.
		return n, fmt.Errorf("plugin archive entry exceeds size limit")
	}
	return n, nil
}

func findSingleManifest(root string) (string, error) {
	var found []string
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if strings.EqualFold(d.Name(), "manifest.json") {
			found = append(found, path)
		}
		return nil
	})
	if len(found) != 1 {
		return "", fmt.Errorf("package must contain exactly one manifest.json")
	}
	return found[0], nil
}

func copyDir(src, dst string) error {
	return filepath.WalkDir(src, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, path)
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		in, err := os.Open(path)
		if err != nil {
			return err
		}
		defer in.Close()
		out, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o755)
		if err != nil {
			return err
		}
		if _, err := io.Copy(out, in); err != nil {
			_ = out.Close()
			return err
		}
		return out.Close()
	})
}

func pluginBinaryName() string {
	if runtime.GOOS == "windows" {
		return "plugin.exe"
	}
	// prefer linux/mac default name
	return "plugin"
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
