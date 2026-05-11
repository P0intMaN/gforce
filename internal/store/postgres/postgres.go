// Package postgres implements store.Store backed by PostgreSQL via pgx/v5.
package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/gforce/gforce/internal/models"
	"github.com/gforce/gforce/internal/store"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// compile-time assertions: both concrete types must satisfy store.Store.
var (
	_ store.Store = (*DB)(nil)
	_ store.Store = (*txStore)(nil)
)

// pgxQuerier is the minimal interface satisfied by both *pgxpool.Pool and pgx.Tx,
// allowing query methods to be shared between the pool-backed and tx-backed stores.
type pgxQuerier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

// --- SQL constants ----------------------------------------------------------

const (
	sqlCreateUser = `
		INSERT INTO users (username, email, password_hash, display_name, avatar_url, bio)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, username, email, password_hash, display_name, avatar_url, bio,
		          is_admin, is_active, created_at, updated_at`

	sqlGetUserByID = `
		SELECT id, username, email, password_hash, display_name, avatar_url, bio,
		       is_admin, is_active, created_at, updated_at
		FROM users WHERE id = $1`

	sqlGetUserByUsername = `
		SELECT id, username, email, password_hash, display_name, avatar_url, bio,
		       is_admin, is_active, created_at, updated_at
		FROM users WHERE username = $1`

	sqlGetUserByEmail = `
		SELECT id, username, email, password_hash, display_name, avatar_url, bio,
		       is_admin, is_active, created_at, updated_at
		FROM users WHERE email = $1`

	sqlUpdateUser = `
		UPDATE users
		SET display_name = COALESCE($2, display_name),
		    avatar_url   = COALESCE($3, avatar_url),
		    bio          = COALESCE($4, bio)
		WHERE id = $1
		RETURNING id, username, email, password_hash, display_name, avatar_url, bio,
		          is_admin, is_active, created_at, updated_at`

	sqlListUsers = `
		SELECT id, username, email, password_hash, display_name, avatar_url, bio,
		       is_admin, is_active, created_at, updated_at
		FROM users ORDER BY created_at DESC LIMIT $1 OFFSET $2`

	sqlCreateRepo = `
		INSERT INTO repositories (owner_id, name, description, is_private, default_branch, disk_path, fork_of)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, owner_id, name, description, is_private, default_branch,
		          disk_path, fork_of, star_count, fork_count, created_at, updated_at`

	sqlGetRepoByID = `
		SELECT id, owner_id, name, description, is_private, default_branch,
		       disk_path, fork_of, star_count, fork_count, created_at, updated_at
		FROM repositories WHERE id = $1`

	sqlGetRepoByOwnerAndName = `
		SELECT id, owner_id, name, description, is_private, default_branch,
		       disk_path, fork_of, star_count, fork_count, created_at, updated_at
		FROM repositories WHERE owner_id = $1 AND name = $2`

	sqlListReposByOwner = `
		SELECT id, owner_id, name, description, is_private, default_branch,
		       disk_path, fork_of, star_count, fork_count, created_at, updated_at
		FROM repositories WHERE owner_id = $1
		ORDER BY created_at DESC LIMIT $2 OFFSET $3`

	sqlListPublicRepos = `
		SELECT id, owner_id, name, description, is_private, default_branch,
		       disk_path, fork_of, star_count, fork_count, created_at, updated_at
		FROM repositories WHERE is_private = false
		ORDER BY created_at DESC LIMIT $1 OFFSET $2`

	sqlListPublicReposByOwner = `
		SELECT id, owner_id, name, description, is_private, default_branch,
		       disk_path, fork_of, star_count, fork_count, created_at, updated_at
		FROM repositories WHERE owner_id = $1 AND is_private = false
		ORDER BY created_at DESC LIMIT $2 OFFSET $3`

	sqlUpdateRepo = `
		UPDATE repositories
		SET name           = COALESCE($2, name),
		    description    = COALESCE($3, description),
		    is_private     = COALESCE($4, is_private),
		    default_branch = COALESCE($5, default_branch)
		WHERE id = $1
		RETURNING id, owner_id, name, description, is_private, default_branch,
		          disk_path, fork_of, star_count, fork_count, created_at, updated_at`

	sqlDeleteRepo = `DELETE FROM repositories WHERE id = $1`

	sqlIncrementStarCount = `
		UPDATE repositories SET star_count = star_count + $2 WHERE id = $1`

	sqlCreateSSHKey = `
		INSERT INTO ssh_keys (user_id, title, public_key, fingerprint)
		VALUES ($1, $2, $3, $4)
		RETURNING id, user_id, title, public_key, fingerprint, last_used_at, created_at`

	sqlGetSSHKeyByFingerprint = `
		SELECT id, user_id, title, public_key, fingerprint, last_used_at, created_at
		FROM ssh_keys WHERE fingerprint = $1`

	sqlListSSHKeysByUser = `
		SELECT id, user_id, title, public_key, fingerprint, last_used_at, created_at
		FROM ssh_keys WHERE user_id = $1 ORDER BY created_at DESC`

	sqlDeleteSSHKey = `DELETE FROM ssh_keys WHERE id = $1 AND user_id = $2`

	sqlRecordEvent = `
		INSERT INTO activity_events (actor_id, event_type, repo_id, payload)
		VALUES ($1, $2, $3, $4)`

	sqlListUserActivity = `
		SELECT id, actor_id, event_type, repo_id, payload, created_at
		FROM activity_events
		WHERE actor_id = $1
		ORDER BY created_at DESC
		LIMIT $2`
)

// --- DB (pool-backed) -------------------------------------------------------

// DB is the production Store backed by a *pgxpool.Pool.
type DB struct {
	pool *pgxpool.Pool
}

// NewDB creates a DB from an already-configured pool. The caller owns the
// pool's lifecycle — call pool.Close() when done.
func NewDB(pool *pgxpool.Pool) *DB {
	return &DB{pool: pool}
}

// NewPool opens and validates a connection pool. Callers may pass the returned
// pool to NewDB and are responsible for calling pool.Close().
func NewPool(ctx context.Context, dsn string, maxConns, minConns int32) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("postgres.NewPool: parsing dsn: %w", err)
	}
	cfg.MaxConns = maxConns
	cfg.MinConns = minConns
	cfg.HealthCheckPeriod = 30 * time.Second

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("postgres.NewPool: creating pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("postgres.NewPool: pinging database: %w", err)
	}
	return pool, nil
}

// BeginTx starts a transaction and returns a Store whose methods all run
// within that transaction. Callers must call Commit or Rollback on the result.
func (db *DB) BeginTx(ctx context.Context) (store.Store, error) {
	tx, err := db.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("store.BeginTx: %w", err)
	}
	return &txStore{tx: tx}, nil
}

// Commit is invalid on a pool-backed store.
func (db *DB) Commit() error {
	return errors.New("store.Commit: called on non-transactional store")
}

// Rollback is invalid on a pool-backed store.
func (db *DB) Rollback() error {
	return errors.New("store.Rollback: called on non-transactional store")
}

// Ping checks that the database is reachable.
func (db *DB) Ping(ctx context.Context) error {
	if err := db.pool.Ping(ctx); err != nil {
		return fmt.Errorf("store.Ping: %w", err)
	}
	return nil
}

// --- txStore (transaction-backed) -------------------------------------------

// txStore wraps an active pgx.Tx and implements store.Store.
type txStore struct {
	tx pgx.Tx
}

// BeginTx returns an error — nested transactions are not supported.
func (t *txStore) BeginTx(_ context.Context) (store.Store, error) {
	return nil, errors.New("store.BeginTx: nested transactions are not supported")
}

// Commit commits the transaction.
func (t *txStore) Commit() error {
	if err := t.tx.Commit(context.Background()); err != nil {
		return fmt.Errorf("store.Commit: %w", err)
	}
	return nil
}

// Rollback aborts the transaction.
func (t *txStore) Rollback() error {
	if err := t.tx.Rollback(context.Background()); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
		return fmt.Errorf("store.Rollback: %w", err)
	}
	return nil
}

// Ping is a no-op within a transaction (connection is already held).
func (t *txStore) Ping(_ context.Context) error { return nil }

// --- shared query helpers ----------------------------------------------------

func scanUser(row pgx.Row) (*models.User, error) {
	var u models.User
	err := row.Scan(
		&u.ID, &u.Username, &u.Email, &u.PasswordHash,
		&u.DisplayName, &u.AvatarURL, &u.Bio,
		&u.IsAdmin, &u.IsActive,
		&u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		return nil, mapErr(err)
	}
	return &u, nil
}

func scanRepo(row pgx.Row) (*models.Repository, error) {
	var r models.Repository
	err := row.Scan(
		&r.ID, &r.OwnerID, &r.Name, &r.Description,
		&r.IsPrivate, &r.DefaultBranch, &r.DiskPath,
		&r.ForkOf, &r.StarCount, &r.ForkCount,
		&r.CreatedAt, &r.UpdatedAt,
	)
	if err != nil {
		return nil, mapErr(err)
	}
	return &r, nil
}

func scanSSHKey(row pgx.Row) (*models.SSHKey, error) {
	var k models.SSHKey
	err := row.Scan(
		&k.ID, &k.UserID, &k.Title, &k.PublicKey,
		&k.Fingerprint, &k.LastUsedAt, &k.CreatedAt,
	)
	if err != nil {
		return nil, mapErr(err)
	}
	return &k, nil
}

// mapErr converts pgx-level errors to typed store sentinels.
func mapErr(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return store.ErrNotFound
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return store.ErrConflict
	}
	return err
}

// --- UserStore methods (DB) --------------------------------------------------

// CreateUser implements store.UserStore.
func (db *DB) CreateUser(ctx context.Context, p models.CreateUserParams) (*models.User, error) {
	u, err := scanUser(db.pool.QueryRow(ctx, sqlCreateUser,
		p.Username, p.Email, p.PasswordHash, p.DisplayName, p.AvatarURL, p.Bio))
	if err != nil {
		return nil, fmt.Errorf("store.CreateUser: %w", err)
	}
	return u, nil
}

// GetUserByID implements store.UserStore.
func (db *DB) GetUserByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	u, err := scanUser(db.pool.QueryRow(ctx, sqlGetUserByID, id))
	if err != nil {
		return nil, fmt.Errorf("store.GetUserByID: %w", err)
	}
	return u, nil
}

// GetUserByUsername implements store.UserStore.
func (db *DB) GetUserByUsername(ctx context.Context, username string) (*models.User, error) {
	u, err := scanUser(db.pool.QueryRow(ctx, sqlGetUserByUsername, username))
	if err != nil {
		return nil, fmt.Errorf("store.GetUserByUsername: %w", err)
	}
	return u, nil
}

// GetUserByEmail implements store.UserStore.
func (db *DB) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	u, err := scanUser(db.pool.QueryRow(ctx, sqlGetUserByEmail, email))
	if err != nil {
		return nil, fmt.Errorf("store.GetUserByEmail: %w", err)
	}
	return u, nil
}

// UpdateUser implements store.UserStore.
func (db *DB) UpdateUser(ctx context.Context, id uuid.UUID, p models.UpdateUserParams) (*models.User, error) {
	u, err := scanUser(db.pool.QueryRow(ctx, sqlUpdateUser, id, p.DisplayName, p.AvatarURL, p.Bio))
	if err != nil {
		return nil, fmt.Errorf("store.UpdateUser: %w", err)
	}
	return u, nil
}

// ListUsers implements store.UserStore.
func (db *DB) ListUsers(ctx context.Context, limit, offset int) ([]*models.User, error) {
	return queryUsers(ctx, db.pool, limit, offset)
}

// --- RepoStore methods (DB) -------------------------------------------------

// CreateRepo implements store.RepoStore.
func (db *DB) CreateRepo(ctx context.Context, p models.CreateRepoParams) (*models.Repository, error) {
	r, err := scanRepo(db.pool.QueryRow(ctx, sqlCreateRepo,
		p.OwnerID, p.Name, p.Description, p.IsPrivate, p.DefaultBranch, p.DiskPath, p.ForkOf))
	if err != nil {
		return nil, fmt.Errorf("store.CreateRepo: %w", err)
	}
	return r, nil
}

// GetRepoByID implements store.RepoStore.
func (db *DB) GetRepoByID(ctx context.Context, id uuid.UUID) (*models.Repository, error) {
	r, err := scanRepo(db.pool.QueryRow(ctx, sqlGetRepoByID, id))
	if err != nil {
		return nil, fmt.Errorf("store.GetRepoByID: %w", err)
	}
	return r, nil
}

// GetRepoByOwnerAndName implements store.RepoStore.
func (db *DB) GetRepoByOwnerAndName(ctx context.Context, ownerID uuid.UUID, name string) (*models.Repository, error) {
	r, err := scanRepo(db.pool.QueryRow(ctx, sqlGetRepoByOwnerAndName, ownerID, name))
	if err != nil {
		return nil, fmt.Errorf("store.GetRepoByOwnerAndName: %w", err)
	}
	return r, nil
}

// ListReposByOwner implements store.RepoStore.
func (db *DB) ListReposByOwner(ctx context.Context, ownerID uuid.UUID, limit, offset int) ([]*models.Repository, error) {
	return queryRepos(ctx, db.pool, sqlListReposByOwner, ownerID, limit, offset)
}

// ListPublicRepos implements store.RepoStore.
func (db *DB) ListPublicRepos(ctx context.Context, limit, offset int) ([]*models.Repository, error) {
	return queryRepos(ctx, db.pool, sqlListPublicRepos, limit, offset)
}

// ListPublicReposByOwner implements store.RepoStore.
func (db *DB) ListPublicReposByOwner(ctx context.Context, ownerID uuid.UUID, limit, offset int) ([]*models.Repository, error) {
	return queryRepos(ctx, db.pool, sqlListPublicReposByOwner, ownerID, limit, offset)
}

// UpdateRepo implements store.RepoStore.
func (db *DB) UpdateRepo(ctx context.Context, id uuid.UUID, p models.UpdateRepoParams) (*models.Repository, error) {
	r, err := scanRepo(db.pool.QueryRow(ctx, sqlUpdateRepo,
		id, p.Name, p.Description, p.IsPrivate, p.DefaultBranch))
	if err != nil {
		return nil, fmt.Errorf("store.UpdateRepo: %w", err)
	}
	return r, nil
}

// DeleteRepo implements store.RepoStore.
func (db *DB) DeleteRepo(ctx context.Context, id uuid.UUID) error {
	tag, err := db.pool.Exec(ctx, sqlDeleteRepo, id)
	if err != nil {
		return fmt.Errorf("store.DeleteRepo: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("store.DeleteRepo: %w", store.ErrNotFound)
	}
	return nil
}

// IncrementStarCount implements store.RepoStore.
func (db *DB) IncrementStarCount(ctx context.Context, id uuid.UUID, delta int) error {
	if _, err := db.pool.Exec(ctx, sqlIncrementStarCount, id, delta); err != nil {
		return fmt.Errorf("store.IncrementStarCount: %w", err)
	}
	return nil
}

// --- SSHKeyStore methods (DB) ------------------------------------------------

// CreateSSHKey implements store.SSHKeyStore.
func (db *DB) CreateSSHKey(ctx context.Context, p models.CreateSSHKeyParams) (*models.SSHKey, error) {
	k, err := scanSSHKey(db.pool.QueryRow(ctx, sqlCreateSSHKey,
		p.UserID, p.Title, p.PublicKey, p.Fingerprint))
	if err != nil {
		return nil, fmt.Errorf("store.CreateSSHKey: %w", err)
	}
	return k, nil
}

// GetSSHKeyByFingerprint implements store.SSHKeyStore.
func (db *DB) GetSSHKeyByFingerprint(ctx context.Context, fingerprint string) (*models.SSHKey, error) {
	k, err := scanSSHKey(db.pool.QueryRow(ctx, sqlGetSSHKeyByFingerprint, fingerprint))
	if err != nil {
		return nil, fmt.Errorf("store.GetSSHKeyByFingerprint: %w", err)
	}
	return k, nil
}

// ListSSHKeysByUser implements store.SSHKeyStore.
func (db *DB) ListSSHKeysByUser(ctx context.Context, userID uuid.UUID) ([]*models.SSHKey, error) {
	rows, err := db.pool.Query(ctx, sqlListSSHKeysByUser, userID)
	if err != nil {
		return nil, fmt.Errorf("store.ListSSHKeysByUser: %w", err)
	}
	defer rows.Close()
	return collectSSHKeys(rows)
}

// DeleteSSHKey implements store.SSHKeyStore.
func (db *DB) DeleteSSHKey(ctx context.Context, id, userID uuid.UUID) error {
	tag, err := db.pool.Exec(ctx, sqlDeleteSSHKey, id, userID)
	if err != nil {
		return fmt.Errorf("store.DeleteSSHKey: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("store.DeleteSSHKey: %w", store.ErrNotFound)
	}
	return nil
}

// --- UserStore methods (txStore) --------------------------------------------

func (t *txStore) CreateUser(ctx context.Context, p models.CreateUserParams) (*models.User, error) {
	u, err := scanUser(t.tx.QueryRow(ctx, sqlCreateUser,
		p.Username, p.Email, p.PasswordHash, p.DisplayName, p.AvatarURL, p.Bio))
	if err != nil {
		return nil, fmt.Errorf("store.CreateUser: %w", err)
	}
	return u, nil
}

func (t *txStore) GetUserByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	u, err := scanUser(t.tx.QueryRow(ctx, sqlGetUserByID, id))
	if err != nil {
		return nil, fmt.Errorf("store.GetUserByID: %w", err)
	}
	return u, nil
}

func (t *txStore) GetUserByUsername(ctx context.Context, username string) (*models.User, error) {
	u, err := scanUser(t.tx.QueryRow(ctx, sqlGetUserByUsername, username))
	if err != nil {
		return nil, fmt.Errorf("store.GetUserByUsername: %w", err)
	}
	return u, nil
}

func (t *txStore) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	u, err := scanUser(t.tx.QueryRow(ctx, sqlGetUserByEmail, email))
	if err != nil {
		return nil, fmt.Errorf("store.GetUserByEmail: %w", err)
	}
	return u, nil
}

func (t *txStore) UpdateUser(ctx context.Context, id uuid.UUID, p models.UpdateUserParams) (*models.User, error) {
	u, err := scanUser(t.tx.QueryRow(ctx, sqlUpdateUser, id, p.DisplayName, p.AvatarURL, p.Bio))
	if err != nil {
		return nil, fmt.Errorf("store.UpdateUser: %w", err)
	}
	return u, nil
}

func (t *txStore) ListUsers(ctx context.Context, limit, offset int) ([]*models.User, error) {
	return queryUsers(ctx, t.tx, limit, offset)
}

// --- RepoStore methods (txStore) --------------------------------------------

func (t *txStore) CreateRepo(ctx context.Context, p models.CreateRepoParams) (*models.Repository, error) {
	r, err := scanRepo(t.tx.QueryRow(ctx, sqlCreateRepo,
		p.OwnerID, p.Name, p.Description, p.IsPrivate, p.DefaultBranch, p.DiskPath, p.ForkOf))
	if err != nil {
		return nil, fmt.Errorf("store.CreateRepo: %w", err)
	}
	return r, nil
}

func (t *txStore) GetRepoByID(ctx context.Context, id uuid.UUID) (*models.Repository, error) {
	r, err := scanRepo(t.tx.QueryRow(ctx, sqlGetRepoByID, id))
	if err != nil {
		return nil, fmt.Errorf("store.GetRepoByID: %w", err)
	}
	return r, nil
}

func (t *txStore) GetRepoByOwnerAndName(ctx context.Context, ownerID uuid.UUID, name string) (*models.Repository, error) {
	r, err := scanRepo(t.tx.QueryRow(ctx, sqlGetRepoByOwnerAndName, ownerID, name))
	if err != nil {
		return nil, fmt.Errorf("store.GetRepoByOwnerAndName: %w", err)
	}
	return r, nil
}

func (t *txStore) ListReposByOwner(ctx context.Context, ownerID uuid.UUID, limit, offset int) ([]*models.Repository, error) {
	return queryRepos(ctx, t.tx, sqlListReposByOwner, ownerID, limit, offset)
}

func (t *txStore) ListPublicRepos(ctx context.Context, limit, offset int) ([]*models.Repository, error) {
	return queryRepos(ctx, t.tx, sqlListPublicRepos, limit, offset)
}

func (t *txStore) ListPublicReposByOwner(ctx context.Context, ownerID uuid.UUID, limit, offset int) ([]*models.Repository, error) {
	return queryRepos(ctx, t.tx, sqlListPublicReposByOwner, ownerID, limit, offset)
}

func (t *txStore) UpdateRepo(ctx context.Context, id uuid.UUID, p models.UpdateRepoParams) (*models.Repository, error) {
	r, err := scanRepo(t.tx.QueryRow(ctx, sqlUpdateRepo,
		id, p.Name, p.Description, p.IsPrivate, p.DefaultBranch))
	if err != nil {
		return nil, fmt.Errorf("store.UpdateRepo: %w", err)
	}
	return r, nil
}

func (t *txStore) DeleteRepo(ctx context.Context, id uuid.UUID) error {
	tag, err := t.tx.Exec(ctx, sqlDeleteRepo, id)
	if err != nil {
		return fmt.Errorf("store.DeleteRepo: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("store.DeleteRepo: %w", store.ErrNotFound)
	}
	return nil
}

func (t *txStore) IncrementStarCount(ctx context.Context, id uuid.UUID, delta int) error {
	if _, err := t.tx.Exec(ctx, sqlIncrementStarCount, id, delta); err != nil {
		return fmt.Errorf("store.IncrementStarCount: %w", err)
	}
	return nil
}

// --- SSHKeyStore methods (txStore) ------------------------------------------

func (t *txStore) CreateSSHKey(ctx context.Context, p models.CreateSSHKeyParams) (*models.SSHKey, error) {
	k, err := scanSSHKey(t.tx.QueryRow(ctx, sqlCreateSSHKey,
		p.UserID, p.Title, p.PublicKey, p.Fingerprint))
	if err != nil {
		return nil, fmt.Errorf("store.CreateSSHKey: %w", err)
	}
	return k, nil
}

func (t *txStore) GetSSHKeyByFingerprint(ctx context.Context, fingerprint string) (*models.SSHKey, error) {
	k, err := scanSSHKey(t.tx.QueryRow(ctx, sqlGetSSHKeyByFingerprint, fingerprint))
	if err != nil {
		return nil, fmt.Errorf("store.GetSSHKeyByFingerprint: %w", err)
	}
	return k, nil
}

func (t *txStore) ListSSHKeysByUser(ctx context.Context, userID uuid.UUID) ([]*models.SSHKey, error) {
	rows, err := t.tx.Query(ctx, sqlListSSHKeysByUser, userID)
	if err != nil {
		return nil, fmt.Errorf("store.ListSSHKeysByUser: %w", err)
	}
	defer rows.Close()
	return collectSSHKeys(rows)
}

func (t *txStore) DeleteSSHKey(ctx context.Context, id, userID uuid.UUID) error {
	tag, err := t.tx.Exec(ctx, sqlDeleteSSHKey, id, userID)
	if err != nil {
		return fmt.Errorf("store.DeleteSSHKey: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("store.DeleteSSHKey: %w", store.ErrNotFound)
	}
	return nil
}

// --- collection helpers -----------------------------------------------------

func queryUsers(ctx context.Context, q pgxQuerier, limit, offset int) ([]*models.User, error) {
	rows, err := q.Query(ctx, sqlListUsers, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("store.ListUsers: %w", err)
	}
	defer rows.Close()

	var users []*models.User
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, fmt.Errorf("store.ListUsers: scanning row: %w", err)
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

func queryRepos(ctx context.Context, q pgxQuerier, sql string, args ...any) ([]*models.Repository, error) {
	rows, err := q.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("store: listing repos: %w", err)
	}
	defer rows.Close()

	var repos []*models.Repository
	for rows.Next() {
		r, err := scanRepo(rows)
		if err != nil {
			return nil, fmt.Errorf("store: scanning repo row: %w", err)
		}
		repos = append(repos, r)
	}
	return repos, rows.Err()
}

func collectSSHKeys(rows pgx.Rows) ([]*models.SSHKey, error) {
	var keys []*models.SSHKey
	for rows.Next() {
		k, err := scanSSHKey(rows)
		if err != nil {
			return nil, fmt.Errorf("store: scanning ssh_key row: %w", err)
		}
		keys = append(keys, k)
	}
	return keys, rows.Err()
}

// --- ActivityStore methods (DB) ---------------------------------------------

// RecordEvent implements store.ActivityStore.
func (db *DB) RecordEvent(ctx context.Context, p store.RecordEventParams) error {
	payload, err := json.Marshal(p.Payload)
	if err != nil {
		return fmt.Errorf("store.RecordEvent: marshaling payload: %w", err)
	}
	_, err = db.pool.Exec(ctx, sqlRecordEvent, p.ActorID, p.EventType, p.RepoID, payload)
	if err != nil {
		return fmt.Errorf("store.RecordEvent: %w", err)
	}
	return nil
}

// ListUserActivity implements store.ActivityStore.
func (db *DB) ListUserActivity(ctx context.Context, actorID uuid.UUID, limit int) ([]*models.ActivityEvent, error) {
	rows, err := db.pool.Query(ctx, sqlListUserActivity, actorID, limit)
	if err != nil {
		return nil, fmt.Errorf("store.ListUserActivity: %w", err)
	}
	defer rows.Close()
	return collectActivityEvents(rows)
}

// --- ActivityStore methods (txStore) ----------------------------------------

func (t *txStore) RecordEvent(ctx context.Context, p store.RecordEventParams) error {
	payload, err := json.Marshal(p.Payload)
	if err != nil {
		return fmt.Errorf("store.RecordEvent: marshaling payload: %w", err)
	}
	_, err = t.tx.Exec(ctx, sqlRecordEvent, p.ActorID, p.EventType, p.RepoID, payload)
	if err != nil {
		return fmt.Errorf("store.RecordEvent: %w", err)
	}
	return nil
}

func (t *txStore) ListUserActivity(ctx context.Context, actorID uuid.UUID, limit int) ([]*models.ActivityEvent, error) {
	rows, err := t.tx.Query(ctx, sqlListUserActivity, actorID, limit)
	if err != nil {
		return nil, fmt.Errorf("store.ListUserActivity: %w", err)
	}
	defer rows.Close()
	return collectActivityEvents(rows)
}

// collectActivityEvents scans pgx.Rows into []*models.ActivityEvent.
func collectActivityEvents(rows pgx.Rows) ([]*models.ActivityEvent, error) {
	var events []*models.ActivityEvent
	for rows.Next() {
		var e models.ActivityEvent
		var payloadRaw []byte
		if err := rows.Scan(
			&e.ID, &e.ActorID, &e.EventType, &e.RepoID, &payloadRaw, &e.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("store: scanning activity_event: %w", err)
		}
		if len(payloadRaw) > 0 {
			if err := json.Unmarshal(payloadRaw, &e.Payload); err != nil {
				e.Payload = map[string]interface{}{}
			}
		} else {
			e.Payload = map[string]interface{}{}
		}
		events = append(events, &e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: iterating activity_events: %w", err)
	}
	return events, nil
}
