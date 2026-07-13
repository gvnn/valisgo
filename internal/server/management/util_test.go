package management

import (
	"net/http"
	"testing"

	"valisgo/internal/testutil"

	"gorm.io/gorm"
)

func runAPITest(t *testing.T, testFunc func(t *testing.T, tx *gorm.DB, router http.Handler)) {
	t.Helper()
	testutil.RunInTransaction(t, func(tx *gorm.DB) {
		api := NewAPI(tx)
		testFunc(t, tx, api.MountRoutes())
	})
}
