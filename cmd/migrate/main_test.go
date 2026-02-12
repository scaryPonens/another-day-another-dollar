package main

import "testing"

func TestLoadMigrations(t *testing.T) {
	migrations, err := loadMigrations(migrationsFS)
	if err != nil {
		t.Fatalf("unexpected error loading embedded migrations: %v", err)
	}
	if len(migrations) < 2 {
		t.Fatalf("expected at least 2 migrations, got %d", len(migrations))
	}
	if migrations[0].Version != 1 {
		t.Fatalf("expected first migration version 1, got %d", migrations[0].Version)
	}
	if migrations[1].Version != 2 {
		t.Fatalf("expected second migration version 2, got %d", migrations[1].Version)
	}
	if migrations[0].UpSQL == "" || migrations[0].DownSQL == "" {
		t.Fatal("expected non-empty up/down sql for first migration")
	}
}
