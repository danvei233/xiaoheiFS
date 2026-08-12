package main

import (
	"crypto/sha256"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"xiaoheiplay/pkg/pluginsign"
)

type checksums struct {
	Algo  string            `json:"algo"`
	Files map[string]string `json:"files"`
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr, os.Stdin))
}

func run(args []string, stdout, stderr io.Writer, stdin io.Reader) int {
	fs := flag.NewFlagSet("pluginsign", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dir := fs.String("dir", "", "plugin directory to sign (must contain manifest.json)")
	// No argv secret flag on purpose: a private key passed on the command line is
	// visible in the process list. Only protected sources are accepted.
	privFile := fs.String("ed25519-priv-file", "", "path to a 0600 file containing the base64 ed25519 private key, or '-' to read it from stdin")
	privFD := fs.Int("ed25519-priv-fd", -1, "read the base64 ed25519 private key from a file descriptor")
	privStdin := fs.Bool("ed25519-priv-stdin", false, "read the base64 ed25519 private key from stdin (one line)")
	verify := fs.Bool("verify", false, "verify checksums.json + signature.sig against -ed25519-pub instead of signing")
	pub := fs.String("ed25519-pub", "", "base64 ed25519 public key (used with -verify)")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	if strings.TrimSpace(*dir) == "" {
		fmt.Fprintln(stderr, "missing -dir")
		return 2
	}
	pluginDir := strings.TrimSpace(*dir)
	if _, err := os.Stat(filepath.Join(pluginDir, "manifest.json")); err != nil {
		fmt.Fprintln(stderr, "manifest.json not found:", err)
		return 2
	}
	if *verify {
		return verifyPlugin(pluginDir, strings.TrimSpace(*pub), stderr)
	}

	key, err := loadPrivateKey(*privFile, *privFD, *privStdin, stdin)
	if err != nil {
		fmt.Fprintln(stderr, "private key:", err)
		return 2
	}
	if key == nil {
		// No private key supplied: generate one, write it to disk (0600, outside the
		// packaged plugin dir), and refuse to emit a signature that the server would
		// never trust. The operator re-runs with the generated key to sign.
		return generateKeyAndExplain(pluginDir, stdout, stderr)
	}
	return signWithKey(pluginDir, key, stdout, stderr)
}

// loadPrivateKey resolves the signing key from exactly one protected source:
// a 0600 file, a file descriptor, stdin, or the PLUGIN_SIGN_PRIVATE_KEY env var.
// It returns (nil, nil) when no key source was supplied so the caller can fall
// back to generating a new keypair. Error messages never echo the key material.
func loadPrivateKey(privFile string, privFD int, privStdin bool, stdin io.Reader) (ed25519.PrivateKey, error) {
	envKey := strings.TrimSpace(os.Getenv("PLUGIN_SIGN_PRIVATE_KEY"))
	sources := 0
	if privFile != "" {
		sources++
	}
	if privFD >= 0 {
		sources++
	}
	if privStdin {
		sources++
	}
	if envKey != "" {
		sources++
	}
	if sources == 0 {
		return nil, nil
	}
	if sources > 1 {
		return nil, fmt.Errorf("specify exactly one private key source (-ed25519-priv-file, -ed25519-priv-fd, -ed25519-priv-stdin, or PLUGIN_SIGN_PRIVATE_KEY)")
	}

	switch {
	case envKey != "":
		return decodePrivateKey(envKey)
	case privFile != "":
		if strings.TrimSpace(privFile) == "-" {
			return decodePrivateKey(trimLine(io.ReadAll(stdin)))
		}
		info, err := os.Stat(privFile)
		if err != nil {
			return nil, err
		}
		// Only a 0600-or-stricter file is acceptable: a key readable by group or
		// others is a credential leak waiting to happen.
		if info.Mode().Perm()&0o077 != 0 {
			return nil, fmt.Errorf("private key file %s has permissions %04o; chmod 0600", privFile, info.Mode().Perm())
		}
		return decodePrivateKey(trimLine(os.ReadFile(privFile)))
	case privStdin:
		return decodePrivateKey(trimLine(io.ReadAll(stdin)))
	case privFD >= 0:
		f := os.NewFile(uintptr(privFD), "private-key-fd")
		if f == nil {
			return nil, fmt.Errorf("invalid private key fd %d", privFD)
		}
		return decodePrivateKey(trimLine(io.ReadAll(f)))
	}
	return nil, nil
}

// trimLine collapses the (b, err) pair of a file/stdin read into a single
// trimmed line. It returns "" on any read error so decodePrivateKey reports the
// length mismatch without echoing the read failure (which could contain the
// key material on exotic readers).
func trimLine(b []byte, err error) string {
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}

func decodePrivateKey(b64 string) (ed25519.PrivateKey, error) {
	b, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return nil, err
	}
	if len(b) != ed25519.PrivateKeySize {
		return nil, fmt.Errorf("invalid key length %d, want %d", len(b), ed25519.PrivateKeySize)
	}
	return ed25519.PrivateKey(b), nil
}

func signWithKey(pluginDir string, privKey ed25519.PrivateKey, stdout, stderr io.Writer) int {
	pubKey := privKey.Public().(ed25519.PublicKey)
	fmt.Fprintln(stdout, "PublicKey(base64):", base64.StdEncoding.EncodeToString(pubKey))

	cs, err := buildChecksums(pluginDir)
	if err != nil {
		fmt.Fprintln(stderr, "checksums:", err)
		return 1
	}
	csBytes, err := json.MarshalIndent(cs, "", "  ")
	if err != nil {
		fmt.Fprintln(stderr, "marshal checksums:", err)
		return 1
	}
	if err := os.WriteFile(filepath.Join(pluginDir, "checksums.json"), csBytes, 0o644); err != nil {
		fmt.Fprintln(stderr, "write checksums.json:", err)
		return 1
	}

	sig := ed25519.Sign(privKey, csBytes)
	if err := os.WriteFile(filepath.Join(pluginDir, "signature.sig"), sig, 0o644); err != nil {
		fmt.Fprintln(stderr, "write signature.sig:", err)
		return 1
	}

	fmt.Fprintln(stdout, "OK: wrote checksums.json + signature.sig")
	return 0
}

func generateKeyAndExplain(pluginDir string, stdout, stderr io.Writer) int {
	pubKey, privKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		fmt.Fprintln(stderr, "generate keypair:", err)
		return 1
	}
	// Sibling of the plugin dir (e.g. plugins/payment/ezpay.ed25519.private.key):
	// outside the packaged dir so it is never checksummed/uploaded with the plugin.
	// Trim a trailing separator first so a -dir ending in "/" cannot place the
	// private key inside the packaged plugin directory.
	keyPath := strings.TrimRight(pluginDir, `/\`) + ".ed25519.private.key"
	if err := os.WriteFile(keyPath, []byte(base64.StdEncoding.EncodeToString(privKey)+"\n"), 0o600); err != nil {
		fmt.Fprintln(stderr, "write private key:", err)
		return 1
	}
	// os.WriteFile leaves an EXISTING file's permissions unchanged, so an old
	// group/world-readable key would stay readable after being overwritten with
	// the fresh key. Tighten explicitly: a private key must never be group/other
	// readable regardless of how the path came to exist.
	if err := os.Chmod(keyPath, 0o600); err != nil {
		fmt.Fprintln(stderr, "chmod private key:", err)
		return 1
	}

	fmt.Fprintln(stdout, "Generated a new Ed25519 keypair for this plugin:")
	fmt.Fprintln(stdout, "PublicKey(base64):", base64.StdEncoding.EncodeToString(pubKey))
	fmt.Fprintln(stdout, "Private key written to", keyPath, "(mode 0600). Keep it secret and back it up.")
	fmt.Fprintln(stderr, "WARNING: no signature was produced. To sign the plugin, re-run with the generated key:")
	fmt.Fprintln(stderr, "  pluginsign -dir <plugin> -ed25519-priv-file", keyPath)
	fmt.Fprintln(stderr, "The server will only trust this signature after you add the public key to")
	fmt.Fprintln(stderr, "config.plugin_official_ed25519_pubkeys.")
	return 1
}

// buildChecksums uses the same canonical manifest the server verifier uses
// (pluginsign.CanonicalManifest), so the signer and the verifier can never
// disagree about the file set a signature covers.
func buildChecksums(pluginDir string) (checksums, error) {
	entries, err := pluginsign.CanonicalManifest(pluginDir)
	if err != nil {
		return checksums{}, err
	}
	files := pluginsign.FilesMap(entries)
	// stable ordering in output (for better diffs)
	keys := make([]string, 0, len(files))
	for k := range files {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	ordered := map[string]string{}
	for _, k := range keys {
		ordered[k] = files[k]
	}
	return checksums{Algo: "sha256", Files: ordered}, nil
}

// verifyPlugin mirrors the host-side VerifySignature (plugins core): it checks
// checksums.json + signature.sig against the given public key and requires every
// bin/** file to be covered by a matching checksum. Exit 0 on success.
func verifyPlugin(pluginDir, pubB64 string, stderr io.Writer) int {
	if pubB64 == "" {
		fmt.Fprintln(stderr, "-verify requires -ed25519-pub")
		return 2
	}
	pubBytes, err := base64.StdEncoding.DecodeString(pubB64)
	if err != nil || len(pubBytes) != ed25519.PublicKeySize {
		fmt.Fprintln(stderr, "invalid -ed25519-pub")
		return 2
	}
	csBytes, err := os.ReadFile(filepath.Join(pluginDir, "checksums.json"))
	if err != nil {
		fmt.Fprintln(stderr, "checksums.json:", err)
		return 1
	}
	sigBytes, err := os.ReadFile(filepath.Join(pluginDir, "signature.sig"))
	if err != nil {
		fmt.Fprintln(stderr, "signature.sig:", err)
		return 1
	}
	if len(sigBytes) != ed25519.SignatureSize {
		decoded, derr := base64.StdEncoding.DecodeString(strings.TrimSpace(string(sigBytes)))
		if derr != nil || len(decoded) != ed25519.SignatureSize {
			fmt.Fprintln(stderr, "invalid signature size")
			return 1
		}
		sigBytes = decoded
	}
	var cs checksums
	if err := json.Unmarshal(csBytes, &cs); err != nil {
		fmt.Fprintln(stderr, "checksums.json parse:", err)
		return 1
	}
	if !ed25519.Verify(ed25519.PublicKey(pubBytes), csBytes, sigBytes) {
		fmt.Fprintln(stderr, "signature verification failed")
		return 1
	}
	if len(cs.Files) == 0 {
		fmt.Fprintln(stderr, "empty checksums")
		return 1
	}
	// Every file under bin/** must be covered by a matching checksum.
	_ = filepath.WalkDir(filepath.Join(pluginDir, "bin"), func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		rel, _ := filepath.Rel(pluginDir, path)
		rel = filepath.ToSlash(rel)
		want, ok := cs.Files[rel]
		if !ok {
			fmt.Fprintln(stderr, "checksums missing for bin file:", rel)
			os.Exit(1)
			return nil
		}
		sum, err := sha256File(path)
		if err != nil || sum != strings.ToLower(strings.TrimSpace(want)) {
			fmt.Fprintln(stderr, "checksum mismatch for:", rel)
			os.Exit(1)
			return nil
		}
		return nil
	})
	fmt.Println("VERIFIED", pluginDir)
	return 0
}

// sha256File returns the lowercase hex sha256 of a file's contents.
func sha256File(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}
