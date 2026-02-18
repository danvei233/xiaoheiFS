package admin_test

import (
	"testing"

	"xiaoheiplay/internal/adapter/repo/core"
	"xiaoheiplay/internal/testutil"
)

func newTestRepo(t *testing.T) *repo.GormRepo {
	t.Helper()
	_, r := testutil.NewTestDB(t, false)
	return r
}
