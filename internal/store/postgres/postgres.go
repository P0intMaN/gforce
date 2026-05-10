// Package postgres implements the store.Store interface backed by PostgreSQL.
package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/gforce/gforce/internal/models"
	"github.com/gforce/gforce/internal/store"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DB wraps a pgxpool.Pool and implements store.Store.
type DB struct {
	pool *pgxpool.Pool
}

// compile-time assertion: *DB must satisfy store.Store.
var _ store.Store = (*DB)(nil)

// New opens a connection pool to the PostgreSQL instance identified by dsn.
func New(ctx context.Context, dsn string, maxConns, minConns int32) (*DB, error) {
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("parsing dsn: %w", err)
	}

	cfg.MaxConns = maxConns
	cfg.MinConns = minConns
	cfg.HealthCheckPeriod = 30 * time.Second

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("creating pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("pinging database: %w", err)
	}

	return &DB{pool: pool}, nil
}

// Users returns the UserStore implementation backed by this pool.
func (db *DB) Users() store.UserStore { return &userStore{pool: db.pool} }

// Repos returns the RepoStore implementation backed by this pool.
func (db *DB) Repos() store.RepoStore { return &repoStore{pool: db.pool} }

// SSHKeys returns the SSHKeyStore implementation backed by this pool.
func (db *DB) SSHKeys() store.SSHKeyStore { return &sshKeyStore{pool: db.pool} }

// Close releases all connections in the pool.
func (db *DB) Close() error {
	db.pool.Close()
	return nil
}

// userStore implements store.UserStore.
type userStore struct{ pool *pgxpool.Pool }

func (s *userStore) Create(ctx context.Context, u *models.User) error {
	const q = `
		INSERT INTO users (id, username, email, password_hash, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)`
	_, err := s.pool.Exec(ctx, q, u.ID, u.Username, u.Email, u.PasswordHash, u.CreatedAt, u.UpdatedAt)
	if err != nil {
		return fmt.Errorf("inserting user: %w", err)
	}
	return nil
}

func (s *userStore) GetByID(ctx context.Context, id string) (*models.User, error) {
	const q = `SELECT id, username, email, password_hash, created_at, updated_at FROM users WHERE id = $1`
	return scanUser(s.pool.QueryRow(ctx, q, id))
}

func (s *userStore) GetByUsername(ctx context.Context, username string) (*models.User, error) {
	const q = `SELECT id, username, email, password_hash, created_at, updated_at FROM users WHERE username = $1`
	return scanUser(s.pool.QueryRow(ctx, q, username))
}

func (s *userStore) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	const q = `SELECT id, username, email, password_hash, created_at, updated_at FROM users WHERE email = $1`
	return scanUser(s.pool.QueryRow(ctx, q, email))
}

func (s *userStore) Update(ctx context.Context, u *models.User) error {
	const q = `
		UPDATE users SET username = $1, email = $2, password_hash = $3, updated_at = $4
		WHERE id = $5`
	tag, err := s.pool.Exec(ctx, q, u.Username, u.Email, u.PasswordHash, u.UpdatedAt, u.ID)
	if err != nil {
		return fmt.Errorf("updating user: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return store.ErrNotFound
	}
	return nil
}

func (s *userStore) Delete(ctx context.Context, id string) error {
	const q = `DELETE FROM users WHERE id = $1`
	tag, err := s.pool.Exec(ctx, q, id)
	if err != nil {
		return fmt.Errorf("deleting user: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return store.ErrNotFound
	}
	return nil
}

func scanUser(row pgx.Row) (*models.User, error) {
	var u models.User
	err := row.Scan(&u.ID, &u.Username, &u.Email, &u.PasswordHash, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, store.ErrNotFound
		}
		return nil, fmt.Errorf("scanning user: %w", err)
	}
	return &u, nil
}

// repoStore implements store.RepoStore.
type repoStore struct{ pool *pgxpool.Pool }

func (s *repoStore) Create(ctx context.Context, r *models.Repository) error {
	const q = `
		INSERT INTO repositories (id, owner_id, name, description, is_private, default_branch, disk_path, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`
	_, err := s.pool.Exec(ctx, q,
		r.ID, r.OwnerID, r.Name, r.Description,
		r.IsPrivate, r.DefaultBranch, r.DiskPath,
		r.CreatedAt, r.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("inserting repository: %w", err)
	}
	return nil
}

func (s *repoStore) GetByID(ctx context.Context, id string) (*models.Repository, error) {
	const q = `
		SELECT id, owner_id, name, description, is_private, default_branch, disk_path, created_at, updated_at
		FROM repositories WHERE id = $1`
	return scanRepo(s.pool.QueryRow(ctx, q, id))
}

func (s *repoStore) GetByOwnerAndName(ctx context.Context, ownerID, name string) (*models.Repository, error) {
	const q = `
		SELECT id, owner_id, name, description, is_private, default_branch, disk_path, created_at, updated_at
		FROM repositories WHERE owner_id = $1 AND name = $2`
	return scanRepo(s.pool.QueryRow(ctx, q, ownerID, name))
}

func (s *repoStore) ListByOwner(ctx context.Context, ownerID string) ([]*models.Repository, error) {
	const q = `
		SELECT id, owner_id, name, description, is_private, default_branch, disk_path, created_at, updated_at
		FROM repositories WHERE owner_id = $1 ORDER BY created_at DESC`
	rows, err := s.pool.Query(ctx, q, ownerID)
	if err != nil {
		return nil, fmt.Errorf("listing repositories: %w", err)
	}
	defer rows.Close()

	var repos []*models.Repository
	for rows.Next() {
		r, err := scanRepo(rows)
		if err != nil {
			return nil, err
		}
		repos = append(repos, r)
	}
	return repos, rows.Err()
}

func (s *repoStore) Update(ctx context.Context, r *models.Repository) error {
	const q = `
		UPDATE repositories
		SET name = $1, description = $2, is_private = $3, default_branch = $4, updated_at = $5
		WHERE id = $6`
	tag, err := s.pool.Exec(ctx, q, r.Name, r.Description, r.IsPrivate, r.DefaultBranch, r.UpdatedAt, r.ID)
	if err != nil {
		return fmt.Errorf("updating repository: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return store.ErrNotFound
	}
	return nil
}

func (s *repoStore) Delete(ctx context.Context, id string) error {
	const q = `DELETE FROM repositories WHERE id = $1`
	tag, err := s.pool.Exec(ctx, q, id)
	if err != nil {
		return fmt.Errorf("deleting repository: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return store.ErrNotFound
	}
	return nil
}

func scanRepo(row pgx.Row) (*models.Repository, error) {
	var r models.Repository
	err := row.Scan(
		&r.ID, &r.OwnerID, &r.Name, &r.Description,
		&r.IsPrivate, &r.DefaultBranch, &r.DiskPath,
		&r.CreatedAt, &r.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, store.ErrNotFound
		}
		return nil, fmt.Errorf("scanning repository: %w", err)
	}
	return &r, nil
}

// sshKeyStore implements store.SSHKeyStore.
type sshKeyStore struct{ pool *pgxpool.Pool }

func (s *sshKeyStore) Create(ctx context.Context, k *models.SSHKey) error {
	const q = `
		INSERT INTO ssh_keys (id, user_id, title, public_key, fingerprint, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)`
	_, err := s.pool.Exec(ctx, q, k.ID, k.UserID, k.Title, k.PublicKey, k.Fingerprint, k.CreatedAt)
	if err != nil {
		return fmt.Errorf("inserting ssh key: %w", err)
	}
	return nil
}

func (s *sshKeyStore) ListByUser(ctx context.Context, userID string) ([]*models.SSHKey, error) {
	const q = `
		SELECT id, user_id, title, public_key, fingerprint, created_at
		FROM ssh_keys WHERE user_id = $1 ORDER BY created_at DESC`
	rows, err := s.pool.Query(ctx, q, userID)
	if err != nil {
		return nil, fmt.Errorf("listing ssh keys: %w", err)
	}
	defer rows.Close()

	var keys []*models.SSHKey
	for rows.Next() {
		var k models.SSHKey
		if err := rows.Scan(&k.ID, &k.UserID, &k.Title, &k.PublicKey, &k.Fingerprint, &k.CreatedAt); err != nil {
			return nil, fmt.Errorf("scanning ssh key: %w", err)
		}
		keys = append(keys, &k)
	}
	return keys, rows.Err()
}

func (s *sshKeyStore) Delete(ctx context.Context, id string) error {
	const q = `DELETE FROM ssh_keys WHERE id = $1`
	tag, err := s.pool.Exec(ctx, q, id)
	if err != nil {
		return fmt.Errorf("deleting ssh key: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return store.ErrNotFound
	}
	return nil
}
