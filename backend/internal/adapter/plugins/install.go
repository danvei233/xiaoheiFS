package plugins

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"crypto/ed25519"
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
}

func InstallPackage(baseDir string, filename string, r io.Reader, officialKeys []ed25519.PublicKey) (InstallResult, error) {
	baseDir = strings.TrimSpace(baseDir)
	if baseDir == "" {
		return InstallResult{}, fmt.Errorf("missing base dir")
	}

	tmpRoot, err := os.MkdirTemp("", "xiaoheiplay-plugin-install-*")
	if err != nil {
		return InstallResult{}, err
	}
	defer os.RemoveAll(tmpRoot)

	ext := strings.ToLower(filename)
	switch {
	case strings.HasSuffix(ext, ".zip"):
		if err := extractZip(tmpRoot, r); err != nil {
			return InstallResult{}, err
		}
	case strings.HasSuffix(ext, ".tar.gz") || strings.HasSuffix(ext, ".tgz"):
		if err := extractTarGz(tmpRoot, r); err != nil {
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
	if err := os.MkdirAll(filepath.Dir(finalDir), 0o755); err != nil {
		return InstallResult{}, err
	}
	if err := copyDir(pluginDir, finalDir); err != nil {
		_ = os.RemoveAll(finalDir)
		return InstallResult{}, err
	}
	return InstallResult{
		Category:        category,
		PluginID:        pluginID,
		PluginDir:       finalDir,
		SignatureStatus: sigStatus,
		Manifest:        m,
	}, nil
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
	b, err := io.ReadAll(r)
	if err != nil {
		return err
	}
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
			if err := extractZipFile(dst, f); err != nil {
				return err
			}
		}
		return nil
	}
	for _, f := range zr.File {
		if err := extractZipFile(dst, f); err != nil {
			return err
		}
	}
	return nil
}

func extractZipFile(dst string, f *zip.File) error {
	name := filepath.ToSlash(f.Name)
	if name == "" || strings.HasPrefix(name, "/") || strings.Contains(name, "..") || strings.Contains(name, ":") {
		return fmt.Errorf("invalid zip entry path")
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
	defer w.Close()
	_, err = io.Copy(w, rc)
	return err
}

func extractTarGz(dst string, r io.Reader) error {
	gr, err := gzip.NewReader(r)
	if err != nil {
		return err
	}
	defer gr.Close()
	tr := tar.NewReader(gr)
	for {
		h, err := tr.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		name := filepath.ToSlash(h.Name)
		if name == "" || strings.HasPrefix(name, "/") || strings.Contains(name, "..") || strings.Contains(name, ":") {
			return fmt.Errorf("invalid tar entry path")
		}
		target := filepath.Join(dst, filepath.FromSlash(name))
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
			if _, err := io.Copy(w, tr); err != nil {
				_ = w.Close()
				return err
			}
			_ = w.Close()
		default:
			// ignore other types
		}
	}
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
