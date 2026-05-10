package controllers_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	gforcev1alpha1 "github.com/gforce/gforce/operator/api/v1alpha1"
	"github.com/gforce/gforce/operator/controllers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"go.uber.org/zap"
)

const testNamespace = "default"

// newReconciler creates a RepositoryReconciler backed by an in-memory store
// and a temporary repo root directory.
func newReconciler(t *testing.T) (*controllers.RepositoryReconciler, *memStore, string) {
	t.Helper()
	ms := newMemStore()
	repoRoot := t.TempDir()
	r := &controllers.RepositoryReconciler{
		Client:   k8sClient,
		Scheme:   testScheme,
		Store:    ms,
		RepoRoot: repoRoot,
		Logger:   zap.NewNop(),
	}
	return r, ms, repoRoot
}

// reconcileUntilDone calls Reconcile repeatedly until it returns no requeue,
// no requeue-after, and no error — or until the deadline is exceeded.
func reconcileUntilDone(t *testing.T, r *controllers.RepositoryReconciler, name string) {
	t.Helper()
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		res, err := r.Reconcile(context.Background(), ctrl.Request{
			NamespacedName: types.NamespacedName{Name: name, Namespace: testNamespace},
		})
		require.NoError(t, err)
		if !res.Requeue && res.RequeueAfter == 0 {
			return
		}
	}
}

// getRepo fetches the current state of a Repository CR.
func getRepo(t *testing.T, name string) gforcev1alpha1.Repository {
	t.Helper()
	var repo gforcev1alpha1.Repository
	require.NoError(t, k8sClient.Get(context.Background(),
		types.NamespacedName{Name: name, Namespace: testNamespace}, &repo))
	return repo
}

// dirExists reports whether a directory exists at path.
func dirExists(path string) bool {
	fi, err := os.Stat(path)
	return err == nil && fi.IsDir()
}

// ── Tests ────────────────────────────────────────────────────────────────────

// TestReconcile_NewRepo_CreatesDiskAndDB verifies that a fresh Repository CR
// transitions to Ready, a bare git repo appears on disk, and one DB record is created.
func TestReconcile_NewRepo_CreatesDiskAndDB(t *testing.T) {
	r, ms, repoRoot := newReconciler(t)

	cr := &gforcev1alpha1.Repository{
		ObjectMeta: metav1.ObjectMeta{Name: "test-new-repo", Namespace: testNamespace},
		Spec: gforcev1alpha1.RepositorySpec{
			OwnerRef:      gforcev1alpha1.OwnerReference{Username: "alice", UserID: "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"},
			Name:          "my-repo",
			DefaultBranch: "main",
		},
	}
	require.NoError(t, k8sClient.Create(context.Background(), cr))
	t.Cleanup(func() { _ = k8sClient.Delete(context.Background(), cr) })

	reconcileUntilDone(t, r, cr.Name)

	updated := getRepo(t, cr.Name)
	assert.Equal(t, gforcev1alpha1.RepositoryPhaseReady, updated.Status.Phase)
	assert.NotEmpty(t, updated.Status.DiskPath)
	assert.NotEmpty(t, updated.Status.DatabaseID)

	assert.True(t, dirExists(filepath.Join(repoRoot, "alice", "my-repo.git")),
		"bare repo must exist on disk")
	assert.Equal(t, 1, ms.repoCount(), "exactly one DB record must be created")
}

// TestReconcile_IdempotentReconcile verifies that running Reconcile 10 times
// creates exactly one DB record and one disk directory.
func TestReconcile_IdempotentReconcile(t *testing.T) {
	r, ms, repoRoot := newReconciler(t)

	cr := &gforcev1alpha1.Repository{
		ObjectMeta: metav1.ObjectMeta{Name: "test-idem-repo", Namespace: testNamespace},
		Spec: gforcev1alpha1.RepositorySpec{
			OwnerRef:      gforcev1alpha1.OwnerReference{Username: "bob", UserID: "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"},
			Name:          "idem-repo",
			DefaultBranch: "main",
		},
	}
	require.NoError(t, k8sClient.Create(context.Background(), cr))
	t.Cleanup(func() { _ = k8sClient.Delete(context.Background(), cr) })

	// Run 10 reconcile cycles.
	for i := 0; i < 10; i++ {
		_, err := r.Reconcile(context.Background(), ctrl.Request{
			NamespacedName: types.NamespacedName{Name: cr.Name, Namespace: testNamespace},
		})
		require.NoError(t, err)
	}

	assert.Equal(t, 1, ms.repoCount(), "idempotent: exactly one DB record")
	assert.True(t, dirExists(filepath.Join(repoRoot, "bob", "idem-repo.git")))
}

// TestReconcile_DeletedRepo_Cleanup verifies that the deletion workflow removes
// the disk directory, DB record, and the finalizer.
func TestReconcile_DeletedRepo_Cleanup(t *testing.T) {
	r, ms, repoRoot := newReconciler(t)

	cr := &gforcev1alpha1.Repository{
		ObjectMeta: metav1.ObjectMeta{Name: "test-del-repo", Namespace: testNamespace},
		Spec: gforcev1alpha1.RepositorySpec{
			OwnerRef:      gforcev1alpha1.OwnerReference{Username: "carol", UserID: "cccccccc-cccc-cccc-cccc-cccccccccccc"},
			Name:          "del-repo",
			DefaultBranch: "main",
		},
	}
	require.NoError(t, k8sClient.Create(context.Background(), cr))

	// Bring to Ready.
	reconcileUntilDone(t, r, cr.Name)
	assert.Equal(t, 1, ms.repoCount())
	diskPath := filepath.Join(repoRoot, "carol", "del-repo.git")
	assert.True(t, dirExists(diskPath))

	// Trigger deletion.
	require.NoError(t, k8sClient.Delete(context.Background(), cr))

	// Reconcile handles the deletion workflow.
	reconcileUntilDone(t, r, cr.Name)

	assert.Equal(t, 0, ms.repoCount(), "DB record must be removed")
	assert.False(t, dirExists(diskPath), "disk directory must be removed")
}

// TestReconcile_InvalidOwnerID_SetsFailed verifies that a non-UUID ownerRef.userID
// results in a Failed phase and no requeue (the spec is invalid, not transient).
func TestReconcile_InvalidOwnerID_SetsFailed(t *testing.T) {
	r, _, _ := newReconciler(t)

	cr := &gforcev1alpha1.Repository{
		ObjectMeta: metav1.ObjectMeta{Name: "test-bad-id-repo", Namespace: testNamespace},
		Spec: gforcev1alpha1.RepositorySpec{
			OwnerRef:      gforcev1alpha1.OwnerReference{Username: "dave", UserID: "not-a-uuid"},
			Name:          "bad-id-repo",
			DefaultBranch: "main",
		},
	}
	require.NoError(t, k8sClient.Create(context.Background(), cr))
	t.Cleanup(func() { _ = k8sClient.Delete(context.Background(), cr) })

	reconcileUntilDone(t, r, cr.Name)

	updated := getRepo(t, cr.Name)
	assert.Equal(t, gforcev1alpha1.RepositoryPhaseFailed, updated.Status.Phase)
}

// --- memStore helpers -------------------------------------------------------

func (s *memStore) repoCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.byID)
}
