// Package store provides utilities for managing database schema migrations.
package store

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

const createMigrationsTable = `
CREATE TABLE IF NOT EXISTS schema_migrations (
    filename   TEXT        PRIMARY KEY,
    applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
)`

// RunMigrations applies all unapplied .sql files from migrationsDir in
// lexicographic order. It is idempotent: files already recorded in the
// schema_migrations table are skipped. Safe to call on every startup.
func RunMigrations(ctx context.Context, pool *pgxpool.Pool, migrationsDir string) error {
	if _, err := pool.Exec(ctx, createMigrationsTable); err != nil {
		return fmt.Errorf("store.RunMigrations: creating schema_migrations table: %w", err)
	}

	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		return fmt.Errorf("store.RunMigrations: reading migrations dir %q: %w", migrationsDir, err)
	}

	var files []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".sql") {
			files = append(files, e.Name())
		}
	}
	sort.Strings(files)

	applied, err := appliedMigrations(ctx, pool)
	if err != nil {
		return fmt.Errorf("store.RunMigrations: querying applied migrations: %w", err)
	}

	for _, name := range files {
		if applied[name] {
			continue
		}

		sql, err := os.ReadFile(filepath.Join(migrationsDir, name))
		if err != nil {
			return fmt.Errorf("store.RunMigrations: reading %q: %w", name, err)
		}

		tx, err := pool.Begin(ctx)
		if err != nil {
			return fmt.Errorf("store.RunMigrations: beginning tx for %q: %w", name, err)
		}

		if _, err := tx.Exec(ctx, string(sql)); err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("store.RunMigrations: executing %q: %w", name, err)
		}

		if _, err := tx.Exec(ctx, `INSERT INTO schema_migrations (filename) VALUES ($1)`, name); err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("store.RunMigrations: recording %q: %w", name, err)
		}

		if err := tx.Commit(ctx); err != nil {
			return fmt.Errorf("store.RunMigrations: committing %q: %w", name, err)
		}
	}

	return nil
}

func appliedMigrations(ctx context.Context, pool *pgxpool.Pool) (map[string]bool, error) {
	rows, err := pool.Query(ctx, `SELECT filename FROM schema_migrations`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	applied := make(map[string]bool)
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		applied[name] = true
	}
	return applied, rows.Err()
}
