package testutil

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"xiaoheiplay/internal/adapter/repo"
	"xiaoheiplay/internal/adapter/seed"
	"xiaoheiplay/internal/domain"
	"xiaoheiplay/internal/pkg/config"
	"xiaoheiplay/internal/pkg/db"
)

func NewTestDB(t *testing.T, withCMS bool) (*sql.DB, *repo.SQLiteRepo) {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	conn, err := db.Open(config.Config{DBType: "sqlite", DBPath: dbPath})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() {
		_ = conn.SQL.Close()
	})
	if err := repo.Migrate(conn.Gorm); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if err := seed.EnsureSettings(conn.SQL, conn.Dialect); err != nil {
		t.Fatalf("seed settings: %v", err)
	}
	if err := seed.EnsurePermissionDefaults(conn.SQL, conn.Dialect); err != nil {
		t.Fatalf("seed permissions: %v", err)
	}
	if err := seed.EnsurePermissionGroups(conn.SQL, conn.Dialect); err != nil {
		t.Fatalf("seed permission groups: %v", err)
	}
	if withCMS {
		if err := seed.EnsureCMSDefaults(conn.SQL, conn.Dialect); err != nil {
			t.Fatalf("seed cms: %v", err)
		}
	}
	return conn.SQL, repo.NewSQLiteRepo(conn.Gorm)
}

func CreateUser(t *testing.T, repo *repo.SQLiteRepo, username, email, password string) domain.User {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	user := domain.User{
		Username:     username,
		Email:        email,
		PasswordHash: string(hash),
		Role:         domain.UserRoleUser,
		Status:       domain.UserStatusActive,
	}
	if err := repo.CreateUser(context.Background(), &user); err != nil {
		t.Fatalf("create user: %v", err)
	}
	return user
}

func CreateAdmin(t *testing.T, repo *repo.SQLiteRepo, username, email, password string, groupID int64) domain.User {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	user := domain.User{
		Username:          username,
		Email:             email,
		PasswordHash:      string(hash),
		Role:              domain.UserRoleAdmin,
		Status:            domain.UserStatusActive,
		PermissionGroupID: &groupID,
	}
	if err := repo.CreateUser(context.Background(), &user); err != nil {
		t.Fatalf("create admin: %v", err)
	}
	return user
}

func IssueJWT(t *testing.T, secret string, userID int64, role string, ttl time.Duration) string {
	t.Helper()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": userID,
		"role":    role,
		"exp":     time.Now().Add(ttl).Unix(),
	})
	signed, err := token.SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("sign jwt: %v", err)
	}
	return signed
}

func DoJSON(t *testing.T, router http.Handler, method, path string, body any, token string) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			t.Fatalf("encode body: %v", err)
		}
	}
	req := httptest.NewRequest(method, path, &buf)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func Itoa(v int64) string {
	return strconv.FormatInt(v, 10)
}
