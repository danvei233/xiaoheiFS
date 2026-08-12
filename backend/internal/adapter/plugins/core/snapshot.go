package plugins

import (
	"context"
	"crypto/ed25519"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"xiaoheiplay/internal/domain"
	"xiaoheiplay/pkg/pluginsign"
)

// snapshotRootName is the hidden root under baseDir that holds verified plugin
// snapshots. The leading dot keeps it out of scanDiskPlugins and the bootstrap
// scan, so snapshots are never treated as installed plugins.
const snapshotRootName = ".snapshots"

// runDirFor returns the immutable verified snapshot directory to execute a
// plugin instance from, or "" when the plugin is already running (no snapshot
// needed because the live process is the trusted one). The snapshot is created
// from the live plugin dir, copied first, and only the SNAPSHOT's own content
// digest and signature are verified — so a swap in the live dir between
// verification and spawn cannot change what is executed.
func (m *Manager) runDirFor(ctx context.Context, inst domain.PluginInstallation) (string, error) {
	// Only an entry whose process is still alive means "already running, no new
	// snapshot needed". A stale entry whose process has exited must fall through
	// to a fresh verified snapshot so the respawn executes verified bytes.
	if rp, ok := m.runtime.GetRunning(inst.Category, inst.PluginID, inst.InstanceID); ok {
		if rp.client == nil || !rp.client.Exited() {
			return "", nil
		}
	}
	return m.verifiedSnapshot(ctx, inst)
}

// verifiedSnapshot materializes an immutable, content-addressed snapshot of the
// plugin directory and verifies the snapshot itself before it is returned. The
// caller executes the plugin from this directory, never from the live plugin
// dir, so a path swap after verification cannot alter the payload.
func (m *Manager) verifiedSnapshot(ctx context.Context, inst domain.PluginInstallation) (string, error) {
	liveDir := filepath.Join(m.baseDir, inst.Category, inst.PluginID)
	if _, err := os.Stat(liveDir); err != nil {
		return "", fmt.Errorf("plugin dir not found: %w", err)
	}

	snapRoot := filepath.Join(m.baseDir, snapshotRootName, inst.Category, inst.PluginID)
	if err := os.MkdirAll(snapRoot, 0o700); err != nil {
		return "", err
	}

	// Build into a fresh temp dir under the snapshot root so a crash mid-copy
	// never leaves a half-verified snapshot that later runs pick up.
	tmp, err := os.MkdirTemp(snapRoot, ".snap-*")
	if err != nil {
		return "", err
	}

	if err := copyDir(liveDir, tmp); err != nil {
		_ = removeAllForce(tmp)
		return "", err
	}
	if err := makeTreeReadOnly(tmp); err != nil {
		_ = removeAllForce(tmp)
		return "", err
	}

	// Verify the SNAPSHOT, not the live dir: these are the exact bytes that will
	// be executed, so a live-dir swap during the copy is caught here.
	digest, err := pluginsign.ContentDigest(tmp)
	if err != nil {
		_ = removeAllForce(tmp)
		return "", fmt.Errorf("plugin %s/%s snapshot verification failed: %w", inst.Category, inst.PluginID, err)
	}
	if inst.ContentDigest == "" {
		_ = removeAllForce(tmp)
		return "", fmt.Errorf("plugin %s/%s has no recorded content digest; refusing to start", inst.Category, inst.PluginID)
	}
	if digest != inst.ContentDigest {
		_ = removeAllForce(tmp)
		return "", fmt.Errorf("plugin %s/%s content changed on disk; refusing to start", inst.Category, inst.PluginID)
	}
	if m.officialKeys != nil {
		status, verr := VerifySignature(tmp, m.officialKeys)
		if verr != nil {
			_ = removeAllForce(tmp)
			return "", fmt.Errorf("plugin %s/%s signature re-verification failed: %w", inst.Category, inst.PluginID, verr)
		}
		if status != inst.SignatureStatus {
			_ = removeAllForce(tmp)
			return "", fmt.Errorf("plugin %s/%s signature changed: installed %q now verifies as %q; refusing to start",
				inst.Category, inst.PluginID, inst.SignatureStatus, status)
		}
	}

	// Content-addressed final location: the digest is part of the path, so the
	// same verified content is never copied twice and a different digest is
	// never mixed up.
	final := filepath.Join(snapRoot, digest)
	if _, err := os.Stat(final); err == nil {
		// Reuse path: NEVER trust a pre-existing snapshot without re-verifying
		// it. The snapshot tree is read-only at the file level, but the
		// directories stay owner-writable so the tree can be removed, which means
		// a same-user attacker can unlink/replace files between startups. Recompute
		// the content digest and re-verify the signature before handing the
		// snapshot to exec; a tampered snapshot is refused, not executed.
		if verr := verifySnapshotTree(final, digest, m.officialKeys, inst); verr != nil {
			_ = removeAllForce(tmp)
			return "", fmt.Errorf("plugin %s/%s snapshot re-verification failed: %w", inst.Category, inst.PluginID, verr)
		}
		_ = removeAllForce(tmp)
		return final, nil
	}
	if m.beforeRename != nil {
		m.beforeRename()
	}
	if err := os.Rename(tmp, final); err != nil {
		if _, statErr := os.Stat(final); statErr == nil {
			// A concurrent caller may have won the race, OR an attacker may have
			// planted a directory at final between the earlier os.Stat and this
			// rename. Either way the tree now at final was never verified by THIS
			// call, so re-verify it exactly like the main reuse path before handing
			// it to exec. A tampered tree is refused, never silently executed.
			if verr := verifySnapshotTree(final, digest, m.officialKeys, inst); verr != nil {
				_ = removeAllForce(tmp)
				return "", fmt.Errorf("plugin %s/%s snapshot re-verification failed: %w", inst.Category, inst.PluginID, verr)
			}
			_ = removeAllForce(tmp)
			return final, nil
		}
		_ = removeAllForce(tmp)
		return "", err
	}
	return final, nil
}

// verifySnapshotTree re-verifies an existing content-addressed snapshot against
// the digest that names it and, when official keys are configured, against the
// recorded signature status. verifiedSnapshot calls it on every reuse so a
// same-user replacement of the (read-only but owner-removable) snapshot tree is
// detected before it can be executed.
func verifySnapshotTree(dir, expectedDigest string, keys []ed25519.PublicKey, inst domain.PluginInstallation) error {
	digest, err := pluginsign.ContentDigest(dir)
	if err != nil {
		return fmt.Errorf("content verification failed: %w", err)
	}
	if digest != expectedDigest {
		return fmt.Errorf("content digest %s does not match snapshot anchor %s", digest, expectedDigest)
	}
	if keys != nil {
		status, verr := VerifySignature(dir, keys)
		if verr != nil {
			return fmt.Errorf("signature re-verification failed: %w", verr)
		}
		if status != inst.SignatureStatus {
			return fmt.Errorf("signature changed: installed %q now verifies as %q", inst.SignatureStatus, status)
		}
	}
	return nil
}

// makeTreeReadOnly makes the verified snapshot immutable against in-place
// modification: every file loses its write bit (executables keep +x). Directories
// are kept owner-writable so the tree can still be removed by os.RemoveAll
// (removal needs write on the parent, not on the file); an attacker with write
// access could unlink a file, but the exec target is fixed at spawn time and the
// snapshot bytes were verified before spawn, which is the guarantee that closes
// the verify-to-spawn window.
func makeTreeReadOnly(root string) error {
	return filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return os.Chmod(path, 0o700)
		}
		info, ierr := d.Info()
		if ierr != nil {
			return ierr
		}
		if info.Mode().Perm()&0o111 != 0 {
			return os.Chmod(path, info.Mode().Perm()&0o555)
		}
		return os.Chmod(path, info.Mode().Perm()&0o444)
	})
}

// removeSnapshots deletes the verified snapshots for a plugin. It is invoked
// when the last instance is removed so stale snapshots do not accumulate.
func (m *Manager) removeSnapshots(category, pluginID string) {
	_ = removeAllForce(filepath.Join(m.baseDir, snapshotRootName, category, pluginID))
}

// removeAllForce removes a directory tree even when it was made read-only (the
// snapshot payload). os.RemoveAll alone fails on read-only parent directories,
// so every entry is made writable first.
func removeAllForce(dir string) error {
	if dir == "" {
		return nil
	}
	_ = filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			_ = os.Chmod(path, 0o700)
		} else {
			_ = os.Chmod(path, 0o600)
		}
		return nil
	})
	return os.RemoveAll(dir)
}
