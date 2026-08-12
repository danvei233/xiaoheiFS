package repo

import (
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// TestEscapeLikePattern_LiteralWildcards ensures user keywords containing LIKE
// wildcards are matched literally rather than as pattern characters.
func TestEscapeLikePattern_LiteralWildcards(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"plain", "plain"},
		{"100%", `100\%`},
		{"a_b", `a\_b`},
		{`a\b`, `a\\b`},
		{`50%_o\ff`, `50\%\_o\\ff`},
	}
	for _, tc := range cases {
		got := escapeLikePattern(tc.in)
		if got != tc.want {
			t.Errorf("escapeLikePattern(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

// TestLikeEscapeClause_DialectAware guards the MySQL regression: the coupon
// keyword search previously emitted `ESCAPE '\'` which is a syntax error on
// MySQL (a backslash escapes the closing quote). SQLite/PostgreSQL want a
// single backslash; MySQL wants a doubled one.
func TestLikeEscapeClause_DialectAware(t *testing.T) {
	mysqlDB := &gorm.DB{Config: &gorm.Config{Dialector: mysql.New(mysql.Config{})}}
	sqliteDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	mysqlClause := likeEscapeClause(mysqlDB)
	if !strings.HasPrefix(mysqlClause, " ESCAPE '") {
		t.Fatalf("unexpected mysql clause: %q", mysqlClause)
	}
	// The SQL text emitted for MySQL must contain two backslashes inside the
	// string literal so MySQL parses it as a single escape backslash.
	if strings.Contains(mysqlClause, `ESCAPE '\'`) {
		t.Fatalf("mysql clause %q would be a syntax error on MySQL", mysqlClause)
	}
	if !strings.Contains(mysqlClause, `ESCAPE '\\'`) {
		t.Fatalf("mysql clause %q must double the backslash", mysqlClause)
	}

	otherClause := likeEscapeClause(sqliteDB)
	if !strings.Contains(otherClause, `ESCAPE '\'`) {
		t.Fatalf("sqlite/postgres clause %q must use a single backslash", otherClause)
	}
	if strings.Contains(otherClause, `ESCAPE '\\'`) {
		t.Fatalf("sqlite/postgres clause %q must not double the backslash", otherClause)
	}
}
