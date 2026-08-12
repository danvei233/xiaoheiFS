// Command update-sign signs a xiaoheiFS update manifest with an Ed25519
// private key, producing the manifest fields the Flutter clients verify
// (see docs/update-signing-contract.md).
//
// The canonical payload it signs MUST stay byte-for-byte identical to the
// Dart verifier in app/*/manifest_verifier.dart. The Go unit test
// (canonical_test.go) pins the expected payload, and the Dart test
// (script/manifest-verifier-test) pins the same string.
//
// Secret handling:
//   - The private key is NEVER accepted on the command line (it would leak via
//     the process list / shell history / CI logs). It is read from exactly one
//     of: the protected environment secret UPDATE_SIGN_PRIVATE_KEY, a 0600
//     private-key file (-key-file), or stdin (-key-stdin).
//   - The private key is never printed, and secret-bearing input is never
//     echoed back.
package main

import (
	"bufio"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

// Manifest is the subset of the update manifest that participates in the
// signature. All fields must be present and typed exactly as documented.
type Manifest struct {
	HasUpdate         bool   `json:"hasUpdate"`
	KeyEpoch          int    `json:"keyEpoch"`
	LatestVersion     string `json:"latestVersion"`
	LatestVersionCode int    `json:"latestVersionCode"`
	ForceUpdate       bool   `json:"forceUpdate"`
	ApkURL            string `json:"apkUrl"`
	SHA256            string `json:"sha256"`
	Changelog         string `json:"changelog"`
	IssuedAt          int64  `json:"issuedAt"`
	Algorithm         string `json:"algorithm,omitempty"`
	Signature         string `json:"signature,omitempty"`
}

// BuildCanonicalPayload reproduces the Dart verifier's canonical payload.
// Field order, separators and escaping MUST NOT change without updating both
// sides and their pinned tests. Every field that can influence the client's
// display/download/force/replay decision is covered here (hasUpdate, keyEpoch,
// version, versionCode, force, url, sha256, changelog, issuedAt) so no
// unsigned field can alter the update decision.
func BuildCanonicalPayload(packageName string, m Manifest) string {
	var b strings.Builder
	writeLine(&b, "pkg", packageName)
	hasUpdate := "0"
	if m.HasUpdate {
		hasUpdate = "1"
	}
	fmt.Fprintf(&b, "hasUpdate=%s\n", hasUpdate)
	fmt.Fprintf(&b, "epoch=%d\n", m.KeyEpoch)
	writeLine(&b, "version", m.LatestVersion)
	fmt.Fprintf(&b, "versionCode=%d\n", m.LatestVersionCode)
	force := "0"
	if m.ForceUpdate {
		force = "1"
	}
	fmt.Fprintf(&b, "force=%s\n", force)
	writeLine(&b, "url", m.ApkURL)
	writeLine(&b, "sha256", m.SHA256)
	writeLine(&b, "notes", m.Changelog)
	// No trailing newline: the Dart verifier builds the payload with
	// lines.join('\n'), which must match byte-for-byte.
	fmt.Fprintf(&b, "issuedAt=%d", m.IssuedAt)
	return b.String()
}

func writeLine(b *strings.Builder, key, value string) {
	b.WriteString(key)
	b.WriteByte('=')
	b.WriteString(escape(value))
	b.WriteByte('\n')
}

func escape(s string) string {
	var sb strings.Builder
	for _, r := range s {
		switch r {
		case '\\':
			sb.WriteString(`\\`)
		case '\n':
			sb.WriteString(`\n`)
		case '\r':
			sb.WriteString(`\r`)
		case '\t':
			sb.WriteString(`\t`)
		default:
			sb.WriteRune(r)
		}
	}
	return sb.String()
}

func main() {
	keyFile := flag.String("key-file", "", "path to a 0600 file containing the Ed25519 private key (seed or full key, hex or base64)")
	keyStdin := flag.Bool("key-stdin", false, "read the Ed25519 private key from stdin (e.g. piped from a secret manager)")
	packageName := flag.String("pkg", "", "package name the manifest is bound to (must equal the client's pkg query param)")
	manifestPath := flag.String("manifest", "", "path to a JSON manifest (without algorithm/signature)")
	keyEpoch := flag.Int("epoch", 1, "key epoch the manifest is signed with (must match a key the clients embed for that epoch)")
	issuedAt := flag.Int64("issued-at", time.Now().Unix(), "issuedAt Unix seconds (defaults to now)")
	flag.Parse()

	if *packageName == "" {
		fatal("missing -pkg")
	}
	if *manifestPath == "" {
		fatal("missing -manifest")
	}
	if *keyEpoch <= 0 {
		fatal("-epoch must be a positive integer")
	}

	priv, err := loadPrivateKey(*keyFile, *keyStdin)
	if err != nil {
		fatal("private key: %v", err)
	}

	raw, err := os.ReadFile(*manifestPath)
	if err != nil {
		fatal("read manifest: %v", err)
	}
	var m Manifest
	if err := json.Unmarshal(raw, &m); err != nil {
		fatal("parse manifest: %v", err)
	}
	m.IssuedAt = *issuedAt
	m.KeyEpoch = *keyEpoch

	if m.SHA256 == "" {
		fatal("manifest is missing sha256 (the artifact hash must be signed)")
	}
	if m.ApkURL == "" {
		fatal("manifest is missing apkUrl")
	}
	if m.KeyEpoch <= 0 {
		fatal("keyEpoch must be a positive integer")
	}

	canonical := BuildCanonicalPayload(*packageName, m)
	sig := ed25519.Sign(priv, []byte(canonical))
	m.Algorithm = "ed25519"
	m.Signature = base64.StdEncoding.EncodeToString(sig)

	out, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		fatal("encode output: %v", err)
	}
	fmt.Println(string(out))
	fmt.Fprintf(os.Stderr, "signed canonical payload (%d bytes):\n%s\n", len(canonical), canonical)
}

// loadPrivateKey reads the signing key from exactly one protected source. It
// deliberately does NOT accept a -key argument: command-line arguments leak
// into the process list, shell history and CI logs.
func loadPrivateKey(keyFile string, keyStdin bool) (ed25519.PrivateKey, error) {
	sources := 0
	if os.Getenv("UPDATE_SIGN_PRIVATE_KEY") != "" {
		sources++
	}
	if keyFile != "" {
		sources++
	}
	if keyStdin {
		sources++
	}
	if sources != 1 {
		return nil, fmt.Errorf("exactly one key source is required: UPDATE_SIGN_PRIVATE_KEY env, -key-file, or -key-stdin (got %d)", sources)
	}

	var raw string
	switch {
	case os.Getenv("UPDATE_SIGN_PRIVATE_KEY") != "":
		raw = os.Getenv("UPDATE_SIGN_PRIVATE_KEY")
	case keyFile != "":
		info, err := os.Stat(keyFile)
		if err != nil {
			return nil, fmt.Errorf("stat key file: %w", err)
		}
		if info.Mode().Perm()&0o077 != 0 {
			return nil, fmt.Errorf("key file %s is group/world readable; chmod 0600 (mode %o)", keyFile, info.Mode().Perm())
		}
		b, err := os.ReadFile(keyFile)
		if err != nil {
			return nil, fmt.Errorf("read key file: %w", err)
		}
		raw = string(b)
	case keyStdin:
		reader := bufio.NewReader(os.Stdin)
		b, err := io.ReadAll(reader)
		if err != nil {
			return nil, fmt.Errorf("read key from stdin: %w", err)
		}
		raw = string(b)
	}

	return decodePrivateKey(raw)
}

func decodePrivateKey(s string) (ed25519.PrivateKey, error) {
	s = strings.TrimSpace(s)
	if hexKey, err := hex.DecodeString(s); err == nil {
		switch len(hexKey) {
		case ed25519.SeedSize:
			return ed25519.NewKeyFromSeed(hexKey), nil
		case ed25519.PrivateKeySize:
			return ed25519.PrivateKey(hexKey), nil
		}
	}
	if b64, err := base64.StdEncoding.DecodeString(s); err == nil {
		switch len(b64) {
		case ed25519.SeedSize:
			return ed25519.NewKeyFromSeed(b64), nil
		case ed25519.PrivateKeySize:
			return ed25519.PrivateKey(b64), nil
		}
	}
	return nil, fmt.Errorf("expected hex/base64 32-byte seed or 64-byte private key")
}

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "update-sign: "+format+"\n", args...)
	os.Exit(1)
}
