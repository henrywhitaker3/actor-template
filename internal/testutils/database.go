package testutils

import (
	"path/filepath"
	"testing"

	"github.com/henrywhitaker3/actor-template/internal/bootstrap"
	"github.com/henrywhitaker3/actor-template/internal/database"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func Gorm(t testing.TB, r ...bootstrap.RegisterModel) *gorm.DB {
	t.Helper()

	db, err := database.Open("sqlite", filepath.Join(t.TempDir(), "db.sqlite"))
	require.Nil(t, err)

	for _, r := range r {
		require.Nil(t, r(db))
	}

	t.Cleanup(func() {
		ddb, err := db.DB()
		require.Nil(t, err)
		require.Nil(t, ddb.Close())
	})

	return db
}
