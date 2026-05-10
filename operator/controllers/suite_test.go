package controllers_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	gforcev1alpha1 "github.com/gforce/gforce/operator/api/v1alpha1"
	"github.com/gforce/gforce/internal/models"
	"github.com/gforce/gforce/internal/store"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"go.uber.org/zap/zapcore"
)

var (
	testScheme = runtime.NewScheme()
	k8sClient  client.Client
	testEnv    *envtest.Environment
	cancelMgr  context.CancelFunc
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(testScheme))
	utilruntime.Must(gforcev1alpha1.AddToScheme(testScheme))
}

// resolveEnvtestAssets returns KUBEBUILDER_ASSETS or tries setup-envtest to find them.
// Returns "" if binaries are unavailable — callers should skip tests in that case.
func resolveEnvtestAssets() string {
	if dir := os.Getenv("KUBEBUILDER_ASSETS"); dir != "" {
		return dir
	}
	cmd := exec.Command("go", "run",
		"sigs.k8s.io/controller-runtime/tools/setup-envtest@latest",
		"use", "1.29.x", "-p", "path")
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func TestMain(m *testing.M) {
	assetsDir := resolveEnvtestAssets()
	if assetsDir == "" {
		// No envtest binaries — skip all integration tests gracefully.
		os.Exit(0)
	}
	os.Setenv("KUBEBUILDER_ASSETS", assetsDir)

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&zap.Options{
		Development: true,
		Level:       zapcore.InfoLevel,
	})))

	testEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "..", "config", "crd", "bases")},
		ErrorIfCRDPathMissing: true,
		Scheme:                testScheme,
	}

	cfg, err := testEnv.Start()
	if err != nil {
		panic("starting envtest: " + err.Error())
	}

	k8sClient, err = client.New(cfg, client.Options{Scheme: testScheme})
	if err != nil {
		panic("creating client: " + err.Error())
	}

	code := m.Run()

	_ = testEnv.Stop()
	os.Exit(code)
}

// --- in-memory store for controller tests -----------------------------------

type memStore struct {
	mu    sync.Mutex
	users map[string]*models.User            // username → User
	repos map[string]*models.Repository      // "ownerID/name" → Repository
	byID  map[uuid.UUID]*models.Repository   // repoID → Repository
}

func newMemStore() *memStore {
	return &memStore{
		users: make(map[string]*models.User),
		repos: make(map[string]*models.Repository),
		byID:  make(map[uuid.UUID]*models.Repository),
	}
}

func (s *memStore) repoKey(ownerID uuid.UUID, name string) string {
	return ownerID.String() + "/" + name
}

func (s *memStore) CreateUser(_ context.Context, p models.CreateUserParams) (*models.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.users[p.Username]; ok {
		return nil, store.ErrConflict
	}
	u := &models.User{
		ID:           uuid.New(),
		Username:     p.Username,
		Email:        p.Email,
		PasswordHash: p.PasswordHash,
		IsActive:     true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	s.users[p.Username] = u
	return u, nil
}

func (s *memStore) GetUserByID(_ context.Context, id uuid.UUID) (*models.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, u := range s.users {
		if u.ID == id {
			cp := *u
			return &cp, nil
		}
	}
	return nil, store.ErrNotFound
}

func (s *memStore) GetUserByUsername(_ context.Context, username string) (*models.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	u, ok := s.users[username]
	if !ok {
		return nil, store.ErrNotFound
	}
	cp := *u
	return &cp, nil
}

func (s *memStore) GetUserByEmail(_ context.Context, email string) (*models.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, u := range s.users {
		if u.Email == email {
			cp := *u
			return &cp, nil
		}
	}
	return nil, store.ErrNotFound
}

func (s *memStore) UpdateUser(_ context.Context, id uuid.UUID, p models.UpdateUserParams) (*models.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, u := range s.users {
		if u.ID == id {
			if p.DisplayName != nil {
				u.DisplayName = p.DisplayName
			}
			if p.Bio != nil {
				u.Bio = p.Bio
			}
			if p.AvatarURL != nil {
				u.AvatarURL = p.AvatarURL
			}
			u.UpdatedAt = time.Now()
			cp := *u
			return &cp, nil
		}
	}
	return nil, store.ErrNotFound
}

func (s *memStore) ListUsers(_ context.Context, _, _ int) ([]*models.User, error) {
	return nil, nil
}

func (s *memStore) CreateRepo(_ context.Context, p models.CreateRepoParams) (*models.Repository, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := s.repoKey(p.OwnerID, p.Name)
	if _, ok := s.repos[key]; ok {
		return nil, store.ErrConflict
	}
	r := &models.Repository{
		ID:            uuid.New(),
		OwnerID:       p.OwnerID,
		Name:          p.Name,
		Description:   p.Description,
		IsPrivate:     p.IsPrivate,
		DefaultBranch: p.DefaultBranch,
		DiskPath:      p.DiskPath,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
	s.repos[key] = r
	s.byID[r.ID] = r
	return r, nil
}

func (s *memStore) GetRepoByID(_ context.Context, id uuid.UUID) (*models.Repository, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	r, ok := s.byID[id]
	if !ok {
		return nil, store.ErrNotFound
	}
	cp := *r
	return &cp, nil
}

func (s *memStore) GetRepoByOwnerAndName(_ context.Context, ownerID uuid.UUID, name string) (*models.Repository, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	r, ok := s.repos[s.repoKey(ownerID, name)]
	if !ok {
		return nil, store.ErrNotFound
	}
	cp := *r
	return &cp, nil
}

func (s *memStore) ListReposByOwner(_ context.Context, _ uuid.UUID, _, _ int) ([]*models.Repository, error) {
	return nil, nil
}

func (s *memStore) ListPublicRepos(_ context.Context, _, _ int) ([]*models.Repository, error) {
	return nil, nil
}

func (s *memStore) ListPublicReposByOwner(_ context.Context, _ uuid.UUID, _, _ int) ([]*models.Repository, error) {
	return nil, nil
}

func (s *memStore) UpdateRepo(_ context.Context, id uuid.UUID, p models.UpdateRepoParams) (*models.Repository, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	r, ok := s.byID[id]
	if !ok {
		return nil, store.ErrNotFound
	}
	if p.IsPrivate != nil {
		r.IsPrivate = *p.IsPrivate
	}
	if p.DefaultBranch != nil {
		r.DefaultBranch = *p.DefaultBranch
	}
	if p.Description != nil {
		r.Description = p.Description
	}
	r.UpdatedAt = time.Now()
	cp := *r
	return &cp, nil
}

func (s *memStore) DeleteRepo(_ context.Context, id uuid.UUID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	r, ok := s.byID[id]
	if !ok {
		return store.ErrNotFound
	}
	delete(s.repos, s.repoKey(r.OwnerID, r.Name))
	delete(s.byID, id)
	return nil
}

func (s *memStore) IncrementStarCount(_ context.Context, _ uuid.UUID, _ int) error { return nil }

func (s *memStore) CreateSSHKey(_ context.Context, _ models.CreateSSHKeyParams) (*models.SSHKey, error) {
	return nil, nil
}

func (s *memStore) GetSSHKeyByFingerprint(_ context.Context, _ string) (*models.SSHKey, error) {
	return nil, store.ErrNotFound
}

func (s *memStore) ListSSHKeysByUser(_ context.Context, _ uuid.UUID) ([]*models.SSHKey, error) {
	return nil, nil
}

func (s *memStore) DeleteSSHKey(_ context.Context, _, _ uuid.UUID) error { return nil }

func (s *memStore) BeginTx(_ context.Context) (store.Store, error) { return s, nil }
func (s *memStore) Commit() error                                    { return nil }
func (s *memStore) Rollback() error                                  { return nil }
func (s *memStore) Ping(_ context.Context) error                     { return nil }

// compile-time check
var _ store.Store = (*memStore)(nil)
