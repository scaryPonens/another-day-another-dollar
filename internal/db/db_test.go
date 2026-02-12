package db

import (
	"context"
	"os"
	"testing"
)

func TestInitPostgres_NoDSN(t *testing.T) {
	os.Setenv("DATABASE_URL", "")
	// Should not panic or fatal, just log and return
	InitPostgres(context.Background())
}

func TestInitPostgres_WithDSN(t *testing.T) {
	os.Setenv("DATABASE_URL", "postgres://user:pass@localhost:5432/dbname")
	// Don't actually connect to Postgres in unit test
}