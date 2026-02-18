package plugins

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
)

type EntryInfo struct {
	Platform           string   `json:"platform"`
	EntryPath          string   `json:"entry_path"`
	EntrySupported     bool     `json:"entry_supported"`
	SupportedPlatforms []string `json:"supported_platforms"`
}

func CurrentPlatformKey() string {
	return runtime.GOOS + "_" + runtime.GOARCH
}

func ResolveEntry(pluginDir string, manifest Manifest) (EntryInfo, error) {
	platform := CurrentPlatformKey()
	supported := supportedPlatformsFromManifest(manifest)

	if len(manifest.Binaries) > 0 {
		rel, ok := manifest.Binaries[platform]
		if !ok || strings.TrimSpace(rel) == "" {
			return EntryInfo{
				Platform:           platform,
				EntryPath:          "",
				EntrySupported:     false,
				SupportedPlatforms: supported,
			}, fmt.Errorf("plugin binary not available for current platform")
		}
		full, err := safeJoin(pluginDir, rel)
		if err != nil {
			return EntryInfo{Platform: platform, SupportedPlatforms: supported}, err
		}
		if fileExists(full) {
			return EntryInfo{
				Platform:           platform,
				EntryPath:          full,
				EntrySupported:     true,
				SupportedPlatforms: supported,
			}, nil
		}
		return EntryInfo{
			Platform:           platform,
			EntryPath:          full,
			EntrySupported:     false,
			SupportedPlatforms: supported,
		}, fmt.Errorf("plugin binary not found for current platform")
	}

	// Backward-compatible fallback: root plugin(.exe)
	if runtime.GOOS == "windows" {
		p := filepath.Join(pluginDir, "plugin.exe")
		if fileExists(p) {
			return EntryInfo{Platform: platform, EntryPath: p, EntrySupported: true, SupportedPlatforms: []string{platform}}, nil
		}
	}
	p := filepath.Join(pluginDir, "plugin")
	if fileExists(p) {
		return EntryInfo{Platform: platform, EntryPath: p, EntrySupported: true, SupportedPlatforms: []string{platform}}, nil
	}
	return EntryInfo{Platform: platform, EntrySupported: false, SupportedPlatforms: supported}, fmt.Errorf("plugin binary not found")
}

func supportedPlatformsFromManifest(manifest Manifest) []string {
	if len(manifest.Binaries) == 0 {
		return nil
	}
	keys := make([]string, 0, len(manifest.Binaries))
	for k := range manifest.Binaries {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func safeJoin(rootDir string, relSlash string) (string, error) {
	rel := filepath.FromSlash(filepath.ToSlash(strings.TrimSpace(relSlash)))
	if rel == "" {
		return "", fmt.Errorf("invalid entry path")
	}
	if strings.HasPrefix(filepath.ToSlash(rel), "/") || strings.Contains(filepath.ToSlash(rel), "..") || strings.Contains(filepath.ToSlash(rel), ":") {
		return "", fmt.Errorf("invalid entry path")
	}
	full := filepath.Join(rootDir, rel)
	fullAbs, err1 := filepath.Abs(full)
	rootAbs, err2 := filepath.Abs(rootDir)
	if err1 == nil && err2 == nil {
		ra := filepath.Clean(rootAbs) + string(os.PathSeparator)
		fa := filepath.Clean(fullAbs)
		if !strings.HasPrefix(fa, ra) && fa != strings.TrimSuffix(ra, string(os.PathSeparator)) {
			return "", fmt.Errorf("invalid entry path")
		}
	}
	return full, nil
}
