package postgres

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/gforce/gforce/internal/models"
	"github.com/gforce/gforce/internal/store"
	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"
)

const (
	sqlCreatePAT = `
		INSERT INTO personal_access_tokens (user_id, name, token_hash, prefix, scopes, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, user_id, name, token_hash, prefix, scopes, last_used_at, expires_at, created_at`

	sqlGetPATsByPrefix = `
		SELECT id, user_id, name, token_hash, prefix, scopes, last_used_at, expires_at, created_at
		FROM personal_access_tokens
		WHERE prefix = $1 AND (expires_at IS NULL OR expires_at > NOW())`

	sqlListPATs = `
		SELECT id, user_id, name, prefix, scopes, last_used_at, expires_at, created_at
		FROM personal_access_tokens
		WHERE user_id = $1 ORDER BY created_at DESC`

	sqlRevokePAT = `DELETE FROM personal_access_tokens WHERE id = $1 AND user_id = $2`

	sqlUpdatePATLastUsed = `UPDATE personal_access_tokens SET last_used_at = NOW() WHERE id = $1`
)

// --- token generation -------------------------------------------------------

func generateRawToken() (raw, prefix string, err error) {
	buf := make([]byte, 32)
	if _, err = rand.Read(buf); err != nil {
		return "", "", fmt.Errorf("generating token entropy: %w", err)
	}
	raw = "gf_" + base64.RawURLEncoding.EncodeToString(buf)
	prefix = raw[:12] // "gf_" + first 9 base64 chars
	return raw, prefix, nil
}

// --- scan helpers ------------------------------------------------------------

// scanPATWithHash scans a row that includes token_hash (used during validation).
func scanPATWithHash(rows pgx.Rows) (*models.PersonalAccessToken, error) {
	var p models.PersonalAccessToken
	err := rows.Scan(
		&p.ID, &p.UserID, &p.Name, &p.TokenHash, &p.Prefix,
		&p.Scopes, &p.LastUsedAt, &p.ExpiresAt, &p.CreatedAt,
	)
	if err != nil {
		return nil, mapErr(err)
	}
	return &p, nil
}

// scanPATPublic scans a row without token_hash (safe for list responses).
func scanPATPublic(row pgx.Row) (*models.PersonalAccessToken, error) {
	var p models.PersonalAccessToken
	err := row.Scan(
		&p.ID, &p.UserID, &p.Name, &p.Prefix,
		&p.Scopes, &p.LastUsedAt, &p.ExpiresAt, &p.CreatedAt,
	)
	if err != nil {
		return nil, mapErr(err)
	}
	return &p, nil
}

// --- DB (pool) implementations ----------------------------------------------

// CreatePAT implements store.PATStore.
func (db *DB) CreatePAT(ctx context.Context, p store.CreatePATParams) (*models.PersonalAccessToken, string, error) {
	raw, prefix, err := generateRawToken()
	if err != nil {
		return nil, "", fmt.Errorf("store.CreatePAT: %w", err)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(raw), 12)
	if err != nil {
		return nil, "", fmt.Errorf("store.CreatePAT: hashing token: %w", err)
	}

	scopes := p.Scopes
	if len(scopes) == 0 {
		scopes = []string{"repo:read", "repo:write"}
	}

	var pat models.PersonalAccessToken
	err = db.pool.QueryRow(ctx, sqlCreatePAT,
		p.UserID, p.Name, string(hash), prefix, scopes, p.ExpiresAt,
	).Scan(
		&pat.ID, &pat.UserID, &pat.Name, &pat.TokenHash, &pat.Prefix,
		&pat.Scopes, &pat.LastUsedAt, &pat.ExpiresAt, &pat.CreatedAt,
	)
	if err != nil {
		return nil, "", fmt.Errorf("store.CreatePAT: inserting: %w", err)
	}

	return &pat, raw, nil
}

// ValidatePAT implements store.PATStore.
// Filters by prefix (indexed) then bcrypt-compares. Never full table scan.
func (db *DB) ValidatePAT(ctx context.Context, rawToken string) (*models.User, *models.PersonalAccessToken, error) {
	if len(rawToken) < 12 {
		return nil, nil, store.ErrNotFound
	}
	prefix := rawToken[:12]

	rows, err := db.pool.Query(ctx, sqlGetPATsByPrefix, prefix)
	if err != nil {
		return nil, nil, fmt.Errorf("store.ValidatePAT: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		pat, err := scanPATWithHash(rows)
		if err != nil {
			continue
		}
		if bcrypt.CompareHashAndPassword([]byte(pat.TokenHash), []byte(rawToken)) != nil {
			continue // wrong hash — try next candidate
		}

		// Match found — update last_used_at asynchronously.
		patID := pat.ID
		go func() {
			_, _ = db.pool.Exec(context.Background(), sqlUpdatePATLastUsed, patID)
		}()

		user, err := db.GetUserByID(ctx, pat.UserID)
		if err != nil {
			return nil, nil, fmt.Errorf("store.ValidatePAT: loading user: %w", err)
		}
		return user, pat, nil
	}

	return nil, nil, store.ErrNotFound
}

// ListPATs implements store.PATStore.
func (db *DB) ListPATs(ctx context.Context, userID uuid.UUID) ([]*models.PersonalAccessToken, error) {
	rows, err := db.pool.Query(ctx, sqlListPATs, userID)
	if err != nil {
		return nil, fmt.Errorf("store.ListPATs: %w", err)
	}
	defer rows.Close()
	return collectPATs(rows)
}

// RevokePAT implements store.PATStore.
func (db *DB) RevokePAT(ctx context.Context, id, userID uuid.UUID) error {
	tag, err := db.pool.Exec(ctx, sqlRevokePAT, id, userID)
	if err != nil {
		return fmt.Errorf("store.RevokePAT: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return store.ErrNotFound
	}
	return nil
}

// --- txStore implementations ------------------------------------------------

func (t *txStore) CreatePAT(ctx context.Context, p store.CreatePATParams) (*models.PersonalAccessToken, string, error) {
	raw, prefix, err := generateRawToken()
	if err != nil {
		return nil, "", fmt.Errorf("store.CreatePAT: %w", err)
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(raw), 12)
	if err != nil {
		return nil, "", fmt.Errorf("store.CreatePAT: %w", err)
	}
	scopes := p.Scopes
	if len(scopes) == 0 {
		scopes = []string{"repo:read", "repo:write"}
	}
	var pat models.PersonalAccessToken
	err = t.tx.QueryRow(ctx, sqlCreatePAT,
		p.UserID, p.Name, string(hash), prefix, scopes, p.ExpiresAt,
	).Scan(
		&pat.ID, &pat.UserID, &pat.Name, &pat.TokenHash, &pat.Prefix,
		&pat.Scopes, &pat.LastUsedAt, &pat.ExpiresAt, &pat.CreatedAt,
	)
	if err != nil {
		return nil, "", fmt.Errorf("store.CreatePAT: %w", err)
	}
	return &pat, raw, nil
}

func (t *txStore) ValidatePAT(ctx context.Context, rawToken string) (*models.User, *models.PersonalAccessToken, error) {
	if len(rawToken) < 12 {
		return nil, nil, store.ErrNotFound
	}
	prefix := rawToken[:12]
	rows, err := t.tx.Query(ctx, sqlGetPATsByPrefix, prefix)
	if err != nil {
		return nil, nil, fmt.Errorf("store.ValidatePAT: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		pat, err := scanPATWithHash(rows)
		if err != nil {
			continue
		}
		if bcrypt.CompareHashAndPassword([]byte(pat.TokenHash), []byte(rawToken)) != nil {
			continue
		}
		patID := pat.ID
		go func() {
			_, _ = t.tx.Exec(context.Background(), sqlUpdatePATLastUsed, patID)
		}()
		user, err := t.GetUserByID(ctx, pat.UserID)
		if err != nil {
			return nil, nil, fmt.Errorf("store.ValidatePAT: %w", err)
		}
		return user, pat, nil
	}
	return nil, nil, store.ErrNotFound
}

func (t *txStore) ListPATs(ctx context.Context, userID uuid.UUID) ([]*models.PersonalAccessToken, error) {
	rows, err := t.tx.Query(ctx, sqlListPATs, userID)
	if err != nil {
		return nil, fmt.Errorf("store.ListPATs: %w", err)
	}
	defer rows.Close()
	return collectPATs(rows)
}

func (t *txStore) RevokePAT(ctx context.Context, id, userID uuid.UUID) error {
	tag, err := t.tx.Exec(ctx, sqlRevokePAT, id, userID)
	if err != nil {
		return fmt.Errorf("store.RevokePAT: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return store.ErrNotFound
	}
	return nil
}

// --- collection helper ------------------------------------------------------

func collectPATs(rows pgx.Rows) ([]*models.PersonalAccessToken, error) {
	var pats []*models.PersonalAccessToken
	for rows.Next() {
		var p models.PersonalAccessToken
		if err := rows.Scan(
			&p.ID, &p.UserID, &p.Name, &p.Prefix,
			&p.Scopes, &p.LastUsedAt, &p.ExpiresAt, &p.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("store: scanning PAT row: %w", err)
		}
		pats = append(pats, &p)
	}
	return pats, rows.Err()
}

// keep time import used
var _ = time.Now
