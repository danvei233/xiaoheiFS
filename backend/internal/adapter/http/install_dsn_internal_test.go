package http

import (
	"context"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"

	mysqlDriver "github.com/go-sql-driver/mysql"
)

// TestValidateInstallMySQLDSN guards P0-8: the pre-auth install endpoints must
// not accept arbitrary DSNs that turn them into free-form outbound dialers.
// Unix-socket, empty-host and unparseable DSNs are rejected up front, while a
// normal TCP host (the legitimate install target) is accepted.
func TestValidateInstallMySQLDSN(t *testing.T) {
	cases := []struct {
		name string
		dsn  string
		want bool
	}{
		{name: "valid public IPv4", dsn: "user:pass@tcp(8.8.8.8:3306)/xiaohei", want: true},
		{name: "local file read rejected", dsn: "user:pass@tcp(8.8.8.8:3306)/xiaohei?allowAllFiles=true", want: false},
		{name: "cleartext password rejected", dsn: "user:pass@tcp(8.8.8.8:3306)/xiaohei?allowCleartextPasswords=true", want: false},
		{name: "plaintext fallback rejected", dsn: "user:pass@tcp(8.8.8.8:3306)/xiaohei?allowFallbackToPlaintext=true", want: false},
		{name: "old password rejected", dsn: "user:pass@tcp(8.8.8.8:3306)/xiaohei?allowOldPasswords=true", want: false},
		{name: "private IPv4 rejected", dsn: "user:pass@tcp(10.0.0.1:3306)/xiaohei", want: false},
		{name: "loopback IPv4 rejected", dsn: "user:pass@tcp(127.0.0.1:3306)/xiaohei", want: false},
		{name: "link-local IPv4 rejected", dsn: "user:pass@tcp(169.254.169.254:3306)/xiaohei", want: false},
		{name: "loopback IPv6 rejected", dsn: "user:pass@tcp6([::1]:3306)/xiaohei", want: false},
		{name: "unspecified IPv6 rejected", dsn: "user:pass@tcp6([::]:3306)/xiaohei", want: false},
		{name: "shared CGNAT rejected", dsn: "user:pass@tcp(100.100.100.200:3306)/xiaohei", want: false},
		{name: "documentation IPv4 rejected", dsn: "user:pass@tcp(203.0.113.10:3306)/xiaohei", want: false},
		{name: "documentation IPv6 rejected", dsn: "user:pass@tcp6([2001:db8::10]:3306)/xiaohei", want: false},
		{name: "multicast rejected", dsn: "user:pass@tcp(224.0.0.1:3306)/xiaohei", want: false},
		{name: "localhost rejected", dsn: "user:pass@tcp(localhost:3306)/xiaohei", want: false},
		{name: "trailing dot localhost rejected", dsn: "user:pass@tcp(localhost.:3306)/xiaohei", want: false},
		{name: "unix socket rejected", dsn: "user:pass@unix(/var/run/mysqld/mysqld.sock)/xiaohei", want: false},
		{name: "unparseable rejected", dsn: "user:pass@tcp(:3306)/xiaohei", want: false},
		{name: "tcp() defaults to localhost", dsn: "user:pass@tcp()/xiaohei", want: false},
		{name: "not a dsn rejected", dsn: "just-a-string", want: false},
		{name: "empty rejected", dsn: "", want: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := validateInstallMySQLDSN(tc.dsn); got != tc.want {
				t.Fatalf("validateInstallMySQLDSN(%q) = %v, want %v", tc.dsn, got, tc.want)
			}
		})
	}
}

func TestWriteInstallConfigAtomic(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "app.config.yaml")
	if err := writeInstallConfigAtomic(path, []byte("db:\n  type: sqlite\n")); err != nil {
		t.Fatalf("write install config: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil || string(data) != "db:\n  type: sqlite\n" {
		t.Fatalf("read persisted config: data=%q err=%v", data, err)
	}
	info, err := os.Stat(path)
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("config permissions=%v err=%v", info.Mode().Perm(), err)
	}
	if err := writeInstallConfigAtomic(filepath.Join(path, "child"), []byte("x")); err == nil {
		t.Fatal("invalid config target must fail instead of reporting success")
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".xiaohei-install-") {
			t.Fatalf("failed atomic write leaked temporary file %q", entry.Name())
		}
	}
}

func TestPinInstallMySQLDSNResolvesAndPinsPublicAddress(t *testing.T) {
	original := installLookupIP
	t.Cleanup(func() { installLookupIP = original })
	installLookupIP = func(context.Context, string) ([]net.IPAddr, error) {
		return []net.IPAddr{{IP: net.ParseIP("8.8.8.8")}}, nil
	}
	pinned, cleanup, ok := pinInstallMySQLDSN(context.Background(), "user:pass@tcp(db.example.test:3306)/xiaohei?tls=true")
	defer cleanup()
	if !ok || !strings.Contains(pinned, "xiaohei_install_") || !strings.Contains(pinned, "db.example.test:3306") {
		t.Fatalf("expected a custom dialer with the original TLS hostname, got ok=%v dsn=%q", ok, pinned)
	}
	parsed, err := mysqlDriver.ParseDSN(pinned)
	if err != nil || parsed.TLS == nil || parsed.TLS.ServerName != "db.example.test" {
		t.Fatalf("expected TLS to retain the original server name, parsed=%#v err=%v", parsed, err)
	}
}

func TestPinInstallMySQLDSNRejectsPrivateOrMixedDNSAnswers(t *testing.T) {
	original := installLookupIP
	t.Cleanup(func() { installLookupIP = original })
	for name, addresses := range map[string][]net.IPAddr{
		"private": {{IP: net.ParseIP("10.0.0.8")}},
		"mixed":   {{IP: net.ParseIP("8.8.8.8")}, {IP: net.ParseIP("127.0.0.1")}},
	} {
		t.Run(name, func(t *testing.T) {
			installLookupIP = func(context.Context, string) ([]net.IPAddr, error) { return addresses, nil }
			_, cleanup, ok := pinInstallMySQLDSN(context.Background(), "user:pass@tcp(db.example.test:3306)/xiaohei")
			cleanup()
			if ok {
				t.Fatal("private DNS answers must be rejected")
			}
		})
	}
}

// TestDedupeAndBoundIDs guards P2-16 bulk-delete hardening: duplicate ids are
// removed, the batch is bounded, and zero/invalid ids are rejected.
func TestDedupeAndBoundIDs(t *testing.T) {
	ids, err := dedupeAndBoundIDs([]int64{1, 2, 2, 3, 1})
	if err != nil {
		t.Fatalf("dedupe: %v", err)
	}
	if len(ids) != 3 {
		t.Fatalf("expected 3 unique ids, got %v", ids)
	}
	if _, err := dedupeAndBoundIDs([]int64{}); err == nil {
		t.Fatal("empty id list must be rejected")
	}
	big := make([]int64, 201)
	for i := range big {
		big[i] = int64(i + 1)
	}
	if _, err := dedupeAndBoundIDs(big); err == nil {
		t.Fatal("over-limit id list must be rejected")
	}
	if _, err := dedupeAndBoundIDs([]int64{0, -1, 5}); err == nil {
		t.Fatal("non-positive ids must be rejected")
	}
}
