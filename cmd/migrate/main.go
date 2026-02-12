package main

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

const (
	cmdUp      = "up"
	cmdDown    = "down"
	cmdVersion = "version"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

var (
	loadEnvFunc = godotenv.Load
	openPool    = pgxpool.New
)

type migration struct {
	Version int64
	Name    string
	UpSQL   string
	DownSQL string
}

func main() {
	loadEnvFunc()

	if len(os.Args) < 2 {
		log.Fatalf("usage: go run ./cmd/migrate [up|down|version] [steps]")
	}

	dsn := os.Getenv("DATABASE_URL")
	if strings.TrimSpace(dsn) == "" {
		log.Fatal("DATABASE_URL is required")
	}

	ctx := context.Background()
	pool, err := openPool(ctx, dsn)
	if err != nil {
		log.Fatalf("connect to postgres: %v", err)
	}
	defer pool.Close()

	if err := ensureMigrationTable(ctx, pool); err != nil {
		log.Fatalf("ensure schema_migrations table: %v", err)
	}

	migrations, err := loadMigrations(migrationsFS)
	if err != nil {
		log.Fatalf("load migrations: %v", err)
	}

	switch os.Args[1] {
	case cmdUp:
		applied, err := applyUp(ctx, pool, migrations)
		if err != nil {
			log.Fatalf("apply migrations up: %v", err)
		}
		log.Printf("migrations up complete (%d applied)", applied)
	case cmdDown:
		steps := 1
		if len(os.Args) > 2 {
			n, err := strconv.Atoi(os.Args[2])
			if err != nil || n <= 0 {
				log.Fatalf("invalid down steps: %q", os.Args[2])
			}
			steps = n
		}
		rolledBack, err := applyDown(ctx, pool, migrations, steps)
		if err != nil {
			log.Fatalf("apply migrations down: %v", err)
		}
		log.Printf("migrations down complete (%d rolled back)", rolledBack)
	case cmdVersion:
		version, name, err := currentVersion(ctx, pool)
		if err != nil {
			log.Fatalf("read current version: %v", err)
		}
		if version == 0 {
			log.Println("no migrations applied")
			return
		}
		log.Printf("current version: %d (%s)", version, name)
	default:
		log.Fatalf("unknown command %q. usage: go run ./cmd/migrate [up|down|version] [steps]", os.Args[1])
	}
}

func ensureMigrationTable(ctx context.Context, pool *pgxpool.Pool) error {
	_, err := pool.Exec(ctx, `
CREATE TABLE IF NOT EXISTS schema_migrations (
    version     BIGINT PRIMARY KEY,
    name        TEXT NOT NULL,
    applied_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);`)
	return err
}

func loadMigrations(fsys fs.FS) ([]migration, error) {
	paths, err := fs.Glob(fsys, "migrations/*.sql")
	if err != nil {
		return nil, err
	}
	if len(paths) == 0 {
		return nil, errors.New("no migration files found")
	}

	re := regexp.MustCompile(`^migrations/([0-9]+)_([a-z0-9_]+)\.(up|down)\.sql$`)
	index := make(map[int64]*migration)

	for _, p := range paths {
		matches := re.FindStringSubmatch(p)
		if matches == nil {
			return nil, fmt.Errorf("invalid migration filename: %s", p)
		}

		version, err := strconv.ParseInt(matches[1], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("parse version in %s: %w", p, err)
		}
		name := matches[2]
		direction := matches[3]

		sqlBytes, err := fs.ReadFile(fsys, p)
		if err != nil {
			return nil, fmt.Errorf("read migration %s: %w", p, err)
		}
		sqlText := strings.TrimSpace(string(sqlBytes))
		if sqlText == "" {
			return nil, fmt.Errorf("empty migration file: %s", p)
		}

		m, ok := index[version]
		if !ok {
			m = &migration{Version: version, Name: name}
			index[version] = m
		} else if m.Name != name {
			return nil, fmt.Errorf("conflicting names for version %d: %s vs %s", version, m.Name, name)
		}

		switch direction {
		case "up":
			if m.UpSQL != "" {
				return nil, fmt.Errorf("duplicate up migration for version %d", version)
			}
			m.UpSQL = sqlText
		case "down":
			if m.DownSQL != "" {
				return nil, fmt.Errorf("duplicate down migration for version %d", version)
			}
			m.DownSQL = sqlText
		default:
			return nil, fmt.Errorf("invalid direction in migration: %s", p)
		}
	}

	migrations := make([]migration, 0, len(index))
	for _, m := range index {
		if m.UpSQL == "" || m.DownSQL == "" {
			return nil, fmt.Errorf("migration version %d must include both up and down files", m.Version)
		}
		migrations = append(migrations, *m)
	}
	sort.Slice(migrations, func(i, j int) bool { return migrations[i].Version < migrations[j].Version })
	return migrations, nil
}

func loadAppliedVersions(ctx context.Context, pool *pgxpool.Pool) (map[int64]struct{}, error) {
	rows, err := pool.Query(ctx, `SELECT version FROM schema_migrations`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	applied := make(map[int64]struct{})
	for rows.Next() {
		var version int64
		if err := rows.Scan(&version); err != nil {
			return nil, err
		}
		applied[version] = struct{}{}
	}
	return applied, rows.Err()
}

func applyUp(ctx context.Context, pool *pgxpool.Pool, migrations []migration) (int, error) {
	appliedSet, err := loadAppliedVersions(ctx, pool)
	if err != nil {
		return 0, err
	}

	appliedCount := 0
	for _, m := range migrations {
		if _, ok := appliedSet[m.Version]; ok {
			continue
		}

		tx, err := pool.Begin(ctx)
		if err != nil {
			return appliedCount, err
		}

		if _, err := tx.Exec(ctx, m.UpSQL); err != nil {
			tx.Rollback(ctx)
			return appliedCount, fmt.Errorf("version %d up failed: %w", m.Version, err)
		}
		if _, err := tx.Exec(ctx, `INSERT INTO schema_migrations (version, name) VALUES ($1, $2)`, m.Version, m.Name); err != nil {
			tx.Rollback(ctx)
			return appliedCount, fmt.Errorf("record version %d failed: %w", m.Version, err)
		}
		if err := tx.Commit(ctx); err != nil {
			return appliedCount, err
		}

		appliedCount++
	}
	return appliedCount, nil
}

func applyDown(ctx context.Context, pool *pgxpool.Pool, migrations []migration, steps int) (int, error) {
	if steps <= 0 {
		return 0, fmt.Errorf("steps must be > 0")
	}

	migrationByVersion := make(map[int64]migration, len(migrations))
	for _, m := range migrations {
		migrationByVersion[m.Version] = m
	}

	rows, err := pool.Query(ctx, `SELECT version FROM schema_migrations ORDER BY version DESC LIMIT $1`, steps)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	versions := make([]int64, 0, steps)
	for rows.Next() {
		var version int64
		if err := rows.Scan(&version); err != nil {
			return 0, err
		}
		versions = append(versions, version)
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}

	rolledBack := 0
	for _, version := range versions {
		m, ok := migrationByVersion[version]
		if !ok {
			return rolledBack, fmt.Errorf("cannot find migration source for applied version %d", version)
		}

		tx, err := pool.Begin(ctx)
		if err != nil {
			return rolledBack, err
		}

		if _, err := tx.Exec(ctx, m.DownSQL); err != nil {
			tx.Rollback(ctx)
			return rolledBack, fmt.Errorf("version %d down failed: %w", m.Version, err)
		}
		if _, err := tx.Exec(ctx, `DELETE FROM schema_migrations WHERE version = $1`, m.Version); err != nil {
			tx.Rollback(ctx)
			return rolledBack, fmt.Errorf("delete version %d failed: %w", m.Version, err)
		}
		if err := tx.Commit(ctx); err != nil {
			return rolledBack, err
		}

		rolledBack++
	}

	return rolledBack, nil
}

func currentVersion(ctx context.Context, pool *pgxpool.Pool) (int64, string, error) {
	var version int64
	var name string
	err := pool.QueryRow(ctx, `SELECT version, name FROM schema_migrations ORDER BY version DESC LIMIT 1`).Scan(&version, &name)
	if err == nil {
		return version, name, nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, "", nil
	}
	return 0, "", err
}
