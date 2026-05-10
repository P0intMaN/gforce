package postgres_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/gforce/gforce/internal/models"
	"github.com/gforce/gforce/internal/store"
	"github.com/gforce/gforce/internal/store/postgres"
	pgcontainer "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	tcontainers "github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// newTestDB spins up a throwaway Postgres container, runs migrations, and
// returns a ready-to-use *postgres.DB. It registers cleanup via t.Cleanup.
func newTestDB(t *testing.T) *postgres.DB {
	t.Helper()
	ctx := context.Background()

	ctr, err := pgcontainer.Run(ctx,
		"postgres:16-alpine",
		pgcontainer.WithDatabase("gforce_test"),
		pgcontainer.WithUsername("gforce"),
		pgcontainer.WithPassword("gforce"),
		tcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second),
		),
	)
	require.NoError(t, err, "starting postgres container")

	t.Cleanup(func() {
		_ = ctr.Terminate(ctx)
	})

	dsn, err := ctr.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	pool, err := postgres.NewPool(ctx, dsn, 5, 1)
	require.NoError(t, err, "connecting to test postgres")

	t.Cleanup(func() { pool.Close() })

	require.NoError(t,
		store.RunMigrations(ctx, pool, "../migrations"),
		"running migrations",
	)

	return postgres.NewDB(pool)
}

// --- helper ------------------------------------------------------------------

func ptr[T any](v T) *T { return &v }

func newPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	ctx := context.Background()

	ctr, err := pgcontainer.Run(ctx,
		"postgres:16-alpine",
		pgcontainer.WithDatabase("gforce_test"),
		pgcontainer.WithUsername("gforce"),
		pgcontainer.WithPassword("gforce"),
		tcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second),
		),
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = ctr.Terminate(ctx) })

	dsn, err := ctr.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	pool, err := postgres.NewPool(ctx, dsn, 5, 1)
	require.NoError(t, err)
	t.Cleanup(pool.Close)

	require.NoError(t, store.RunMigrations(ctx, pool, "../migrations"))
	return pool
}

// --- User tests --------------------------------------------------------------

func TestCreateUser(t *testing.T) {
	t.Parallel()
	db := newTestDB(t)
	ctx := context.Background()

	u, err := db.CreateUser(ctx, models.CreateUserParams{
		Username:     "alice",
		Email:        "alice@example.com",
		PasswordHash: "$2a$10$placeholder",
		DisplayName:  ptr("Alice"),
	})

	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, u.ID)
	assert.Equal(t, "alice", u.Username)
	assert.Equal(t, "alice@example.com", u.Email)
	assert.Equal(t, ptr("Alice"), u.DisplayName)
	assert.False(t, u.IsAdmin)
	assert.True(t, u.IsActive)
}

func TestGetUserByUsername(t *testing.T) {
	t.Parallel()
	db := newTestDB(t)
	ctx := context.Background()

	_, err := db.CreateUser(ctx, models.CreateUserParams{
		Username:     "bob",
		Email:        "bob@example.com",
		PasswordHash: "hash",
	})
	require.NoError(t, err)

	got, err := db.GetUserByUsername(ctx, "bob")
	require.NoError(t, err)
	assert.Equal(t, "bob", got.Username)
}

func TestGetUserByUsername_NotFound(t *testing.T) {
	t.Parallel()
	db := newTestDB(t)

	_, err := db.GetUserByUsername(context.Background(), "nonexistent")
	assert.ErrorIs(t, err, store.ErrNotFound)
}

func TestCreateUser_DuplicateUsername(t *testing.T) {
	t.Parallel()
	db := newTestDB(t)
	ctx := context.Background()

	params := models.CreateUserParams{
		Username:     "carol",
		Email:        "carol@example.com",
		PasswordHash: "hash",
	}
	_, err := db.CreateUser(ctx, params)
	require.NoError(t, err)

	params.Email = "carol2@example.com" // different email, same username
	_, err = db.CreateUser(ctx, params)
	assert.ErrorIs(t, err, store.ErrConflict)
}

func TestUpdateUser(t *testing.T) {
	t.Parallel()
	db := newTestDB(t)
	ctx := context.Background()

	u, err := db.CreateUser(ctx, models.CreateUserParams{
		Username:     "dave",
		Email:        "dave@example.com",
		PasswordHash: "hash",
	})
	require.NoError(t, err)

	updated, err := db.UpdateUser(ctx, u.ID, models.UpdateUserParams{
		DisplayName: ptr("Dave Updated"),
		Bio:         ptr("I write Go"),
	})
	require.NoError(t, err)
	assert.Equal(t, ptr("Dave Updated"), updated.DisplayName)
	assert.Equal(t, ptr("I write Go"), updated.Bio)
}

// --- Repository tests --------------------------------------------------------

func TestCreateRepo(t *testing.T) {
	t.Parallel()
	db := newTestDB(t)
	ctx := context.Background()

	owner, err := db.CreateUser(ctx, models.CreateUserParams{
		Username: "eve", Email: "eve@example.com", PasswordHash: "hash",
	})
	require.NoError(t, err)

	repo, err := db.CreateRepo(ctx, models.CreateRepoParams{
		OwnerID:       owner.ID,
		Name:          "my-repo",
		Description:   ptr("A test repo"),
		IsPrivate:     false,
		DefaultBranch: "main",
		DiskPath:      "/repos/eve/my-repo.git",
	})
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, repo.ID)
	assert.Equal(t, "my-repo", repo.Name)
	assert.Equal(t, owner.ID, repo.OwnerID)
	assert.Equal(t, 0, repo.StarCount)
}

func TestGetRepoByOwnerAndName(t *testing.T) {
	t.Parallel()
	db := newTestDB(t)
	ctx := context.Background()

	owner, _ := db.CreateUser(ctx, models.CreateUserParams{
		Username: "frank", Email: "frank@example.com", PasswordHash: "hash",
	})
	_, err := db.CreateRepo(ctx, models.CreateRepoParams{
		OwnerID: owner.ID, Name: "proj", DefaultBranch: "main", DiskPath: "/repos/frank/proj.git",
	})
	require.NoError(t, err)

	got, err := db.GetRepoByOwnerAndName(ctx, owner.ID, "proj")
	require.NoError(t, err)
	assert.Equal(t, "proj", got.Name)
}

func TestCreateRepo_UniqueViolation(t *testing.T) {
	t.Parallel()
	db := newTestDB(t)
	ctx := context.Background()

	owner, _ := db.CreateUser(ctx, models.CreateUserParams{
		Username: "grace", Email: "grace@example.com", PasswordHash: "hash",
	})
	p := models.CreateRepoParams{
		OwnerID: owner.ID, Name: "dup", DefaultBranch: "main", DiskPath: "/repos/grace/dup.git",
	}
	_, err := db.CreateRepo(ctx, p)
	require.NoError(t, err)

	_, err = db.CreateRepo(ctx, p)
	assert.ErrorIs(t, err, store.ErrConflict)
}

func TestIncrementStarCount(t *testing.T) {
	t.Parallel()
	db := newTestDB(t)
	ctx := context.Background()

	owner, _ := db.CreateUser(ctx, models.CreateUserParams{
		Username: "henry", Email: "henry@example.com", PasswordHash: "hash",
	})
	repo, _ := db.CreateRepo(ctx, models.CreateRepoParams{
		OwnerID: owner.ID, Name: "starred", DefaultBranch: "main", DiskPath: "/repos/henry/starred.git",
	})

	require.NoError(t, db.IncrementStarCount(ctx, repo.ID, 5))
	got, err := db.GetRepoByID(ctx, repo.ID)
	require.NoError(t, err)
	assert.Equal(t, 5, got.StarCount)

	require.NoError(t, db.IncrementStarCount(ctx, repo.ID, -2))
	got, err = db.GetRepoByID(ctx, repo.ID)
	require.NoError(t, err)
	assert.Equal(t, 3, got.StarCount)
}

// --- Transaction tests -------------------------------------------------------

func TestTransaction_CreateUserAndRepo_Commit(t *testing.T) {
	t.Parallel()
	pool := newPool(t)
	db := postgres.NewDB(pool)
	ctx := context.Background()

	tx, err := db.BeginTx(ctx)
	require.NoError(t, err)

	user, err := tx.CreateUser(ctx, models.CreateUserParams{
		Username: "ivan", Email: "ivan@example.com", PasswordHash: "hash",
	})
	require.NoError(t, err)

	_, err = tx.CreateRepo(ctx, models.CreateRepoParams{
		OwnerID: user.ID, Name: "tx-repo", DefaultBranch: "main", DiskPath: "/repos/ivan/tx-repo.git",
	})
	require.NoError(t, err)

	require.NoError(t, tx.Commit())

	// Verify visible outside the transaction
	got, err := db.GetUserByUsername(ctx, "ivan")
	require.NoError(t, err)
	assert.Equal(t, "ivan", got.Username)

	gotRepo, err := db.GetRepoByOwnerAndName(ctx, user.ID, "tx-repo")
	require.NoError(t, err)
	assert.Equal(t, "tx-repo", gotRepo.Name)
}

func TestTransaction_Rollback(t *testing.T) {
	t.Parallel()
	pool := newPool(t)
	db := postgres.NewDB(pool)
	ctx := context.Background()

	tx, err := db.BeginTx(ctx)
	require.NoError(t, err)

	_, err = tx.CreateUser(ctx, models.CreateUserParams{
		Username: "judy", Email: "judy@example.com", PasswordHash: "hash",
	})
	require.NoError(t, err)

	require.NoError(t, tx.Rollback())

	// Rollback means the user must NOT be visible
	_, err = db.GetUserByUsername(ctx, "judy")
	assert.True(t, errors.Is(err, store.ErrNotFound),
		"expected ErrNotFound after rollback, got %v", err)
}

func TestBeginTx_NestedNotSupported(t *testing.T) {
	t.Parallel()
	db := newTestDB(t)
	ctx := context.Background()

	tx, err := db.BeginTx(ctx)
	require.NoError(t, err)
	defer tx.Rollback() //nolint:errcheck

	_, err = tx.BeginTx(ctx)
	assert.Error(t, err, "expected error for nested transaction")
}

// --- SSH key tests -----------------------------------------------------------

func TestCreateAndListSSHKeys(t *testing.T) {
	t.Parallel()
	db := newTestDB(t)
	ctx := context.Background()

	user, _ := db.CreateUser(ctx, models.CreateUserParams{
		Username: "kate", Email: "kate@example.com", PasswordHash: "hash",
	})

	_, err := db.CreateSSHKey(ctx, models.CreateSSHKeyParams{
		UserID:      user.ID,
		Title:       "laptop",
		PublicKey:   "ssh-ed25519 AAAAC3Nz laptop",
		Fingerprint: "SHA256:abc123",
	})
	require.NoError(t, err)

	keys, err := db.ListSSHKeysByUser(ctx, user.ID)
	require.NoError(t, err)
	require.Len(t, keys, 1)
	assert.Equal(t, "laptop", keys[0].Title)
}

func TestDeleteSSHKey_WrongUser(t *testing.T) {
	t.Parallel()
	db := newTestDB(t)
	ctx := context.Background()

	user, _ := db.CreateUser(ctx, models.CreateUserParams{
		Username: "leo", Email: "leo@example.com", PasswordHash: "hash",
	})
	key, err := db.CreateSSHKey(ctx, models.CreateSSHKeyParams{
		UserID:      user.ID,
		Title:       "work",
		PublicKey:   "ssh-ed25519 AAAAC3Nz work",
		Fingerprint: "SHA256:xyz789",
	})
	require.NoError(t, err)

	// Attempt deletion by a different (random) user ID — must fail with ErrNotFound
	err = db.DeleteSSHKey(ctx, key.ID, uuid.New())
	assert.ErrorIs(t, err, store.ErrNotFound)
}

func TestPing(t *testing.T) {
	t.Parallel()
	db := newTestDB(t)
	assert.NoError(t, db.Ping(context.Background()))
}
