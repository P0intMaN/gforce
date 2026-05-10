// Package controllers implements the Kubernetes reconciliation loops for GForce CRDs.
package controllers

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/google/uuid"
	gforcev1alpha1 "github.com/gforce/gforce/operator/api/v1alpha1"
	"github.com/gforce/gforce/internal/gitserver"
	"github.com/gforce/gforce/internal/models"
	"github.com/gforce/gforce/internal/store"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"go.uber.org/zap"
)

const (
	repositoryFinalizer = "gforce.io/repository-cleanup"
	requeueDelay        = 30 * time.Second
)

// RepositoryReconciler reconciles Repository custom resources.
//
// +kubebuilder:rbac:groups=gforce.io,resources=repositories,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=gforce.io,resources=repositories/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=gforce.io,resources=repositories/finalizers,verbs=update
type RepositoryReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Store    store.Store
	RepoRoot string
	Logger   *zap.Logger
}

// SetupWithManager registers the reconciler with the controller manager.
func (r *RepositoryReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&gforcev1alpha1.Repository{}).
		WithOptions(controller.Options{MaxConcurrentReconciles: 5}).
		Complete(r)
}

// Reconcile is called for every change to a Repository resource.
func (r *RepositoryReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Logger.With(zap.String("name", req.Name), zap.String("namespace", req.Namespace))

	// ── 1. Fetch ─────────────────────────────────────────────────────────────
	var repo gforcev1alpha1.Repository
	if err := r.Get(ctx, req.NamespacedName, &repo); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, fmt.Errorf("fetching Repository: %w", err)
	}

	// ── 2. Deletion ───────────────────────────────────────────────────────────
	if !repo.DeletionTimestamp.IsZero() {
		return r.handleDeletion(ctx, &repo, log)
	}

	// ── 3. Finalizer ──────────────────────────────────────────────────────────
	if !controllerutil.ContainsFinalizer(&repo, repositoryFinalizer) {
		controllerutil.AddFinalizer(&repo, repositoryFinalizer)
		if err := r.Update(ctx, &repo); err != nil {
			return ctrl.Result{}, fmt.Errorf("adding finalizer: %w", err)
		}
		return ctrl.Result{Requeue: true}, nil
	}

	// ── 4. Initial status ─────────────────────────────────────────────────────
	if repo.Status.Phase == "" {
		repo.Status.Phase = gforcev1alpha1.RepositoryPhasePending
		setCondition(&repo.Status.Conditions, gforcev1alpha1.ConditionDiskReady,
			metav1.ConditionFalse, "Initializing", "Disk not yet initialized", repo.Generation)
		setCondition(&repo.Status.Conditions, gforcev1alpha1.ConditionDatabaseSynced,
			metav1.ConditionFalse, "Initializing", "Database not yet synced", repo.Generation)
		setCondition(&repo.Status.Conditions, gforcev1alpha1.ConditionReady,
			metav1.ConditionFalse, "Initializing", "Repository is being initialized", repo.Generation)
		if err := r.Status().Update(ctx, &repo); err != nil {
			return ctrl.Result{}, fmt.Errorf("setting Pending status: %w", err)
		}
		return ctrl.Result{Requeue: true}, nil
	}

	// ── 5. Ensure disk ────────────────────────────────────────────────────────
	diskPath := gitserver.GetRepoPath(r.RepoRoot, repo.Spec.OwnerRef.Username, repo.Spec.Name)
	if !gitserver.RepoExists(diskPath) {
		if err := gitserver.InitBareRepo(diskPath); err != nil {
			log.Error("initialising bare repository", zap.String("path", diskPath), zap.Error(err))
			setCondition(&repo.Status.Conditions, gforcev1alpha1.ConditionDiskReady,
				metav1.ConditionFalse, "InitFailed", err.Error(), repo.Generation)
			repo.Status.Phase = gforcev1alpha1.RepositoryPhaseFailed
			_ = r.Status().Update(ctx, &repo)
			return ctrl.Result{RequeueAfter: requeueDelay}, nil
		}
	}
	repo.Status.DiskPath = diskPath
	setCondition(&repo.Status.Conditions, gforcev1alpha1.ConditionDiskReady,
		metav1.ConditionTrue, "Initialized", "Bare git repository is ready", repo.Generation)

	// ── 6. Ensure database ────────────────────────────────────────────────────
	ownerID, err := uuid.Parse(repo.Spec.OwnerRef.UserID)
	if err != nil {
		setCondition(&repo.Status.Conditions, gforcev1alpha1.ConditionDatabaseSynced,
			metav1.ConditionFalse, "InvalidOwnerID", "ownerRef.userID is not a valid UUID", repo.Generation)
		repo.Status.Phase = gforcev1alpha1.RepositoryPhaseFailed
		_ = r.Status().Update(ctx, &repo)
		return ctrl.Result{}, nil // don't requeue; spec is invalid
	}

	dbRepo, err := r.Store.GetRepoByOwnerAndName(ctx, ownerID, repo.Spec.Name)
	switch {
	case errors.Is(err, store.ErrNotFound):
		dbRepo, err = r.createDBRecord(ctx, &repo, ownerID, diskPath)
		if err != nil {
			log.Error("creating DB record", zap.Error(err))
			setCondition(&repo.Status.Conditions, gforcev1alpha1.ConditionDatabaseSynced,
				metav1.ConditionFalse, "CreateFailed", err.Error(), repo.Generation)
			_ = r.Status().Update(ctx, &repo)
			return ctrl.Result{RequeueAfter: requeueDelay}, err
		}

	case err != nil:
		log.Error("querying DB record", zap.Error(err))
		return ctrl.Result{RequeueAfter: requeueDelay}, err

	default:
		// Record exists — sync mutable fields.
		if err := r.syncDBRecord(ctx, dbRepo, &repo); err != nil {
			log.Warn("syncing DB record", zap.Error(err))
			// Non-fatal: status will reflect partial sync, but we continue to Ready.
		}
	}

	if dbRepo != nil {
		repo.Status.DatabaseID = dbRepo.ID.String()
	}
	setCondition(&repo.Status.Conditions, gforcev1alpha1.ConditionDatabaseSynced,
		metav1.ConditionTrue, "Synced", "Database record is up to date", repo.Generation)

	// ── 7. Ready ──────────────────────────────────────────────────────────────
	repo.Status.Phase = gforcev1alpha1.RepositoryPhaseReady
	repo.Status.ObservedGeneration = repo.Generation
	setCondition(&repo.Status.Conditions, gforcev1alpha1.ConditionReady,
		metav1.ConditionTrue, "Ready", "Repository is ready", repo.Generation)

	if err := r.Status().Update(ctx, &repo); err != nil {
		return ctrl.Result{}, fmt.Errorf("updating status to Ready: %w", err)
	}

	log.Info("repository reconciled", zap.String("diskPath", diskPath))
	return ctrl.Result{}, nil
}

// handleDeletion removes all external resources and strips the finalizer.
func (r *RepositoryReconciler) handleDeletion(ctx context.Context, repo *gforcev1alpha1.Repository, log *zap.Logger) (ctrl.Result, error) {
	log.Info("handling deletion")

	repo.Status.Phase = gforcev1alpha1.RepositoryPhaseDeleting
	_ = r.Status().Update(ctx, repo)

	// Remove disk artefacts.
	if repo.Status.DiskPath != "" {
		if err := os.RemoveAll(repo.Status.DiskPath); err != nil {
			log.Warn("removing repository disk", zap.String("path", repo.Status.DiskPath), zap.Error(err))
		}
	}

	// Remove DB record.
	if repo.Status.DatabaseID != "" {
		dbID, err := uuid.Parse(repo.Status.DatabaseID)
		if err == nil {
			if err := r.Store.DeleteRepo(ctx, dbID); err != nil && !errors.Is(err, store.ErrNotFound) {
				log.Error("deleting DB record", zap.String("id", repo.Status.DatabaseID), zap.Error(err))
			}
		}
	}

	controllerutil.RemoveFinalizer(repo, repositoryFinalizer)
	if err := r.Update(ctx, repo); err != nil {
		return ctrl.Result{}, fmt.Errorf("removing finalizer: %w", err)
	}

	log.Info("repository deleted")
	return ctrl.Result{}, nil
}

// createDBRecord creates a new repository record in the GForce database.
func (r *RepositoryReconciler) createDBRecord(ctx context.Context, repo *gforcev1alpha1.Repository, ownerID uuid.UUID, diskPath string) (*models.Repository, error) {
	var desc *string
	if repo.Spec.Description != "" {
		d := repo.Spec.Description
		desc = &d
	}

	dbRepo, err := r.Store.CreateRepo(ctx, models.CreateRepoParams{
		OwnerID:       ownerID,
		Name:          repo.Spec.Name,
		Description:   desc,
		IsPrivate:     repo.Spec.IsPrivate,
		DefaultBranch: repo.Spec.DefaultBranch,
		DiskPath:      diskPath,
	})
	if errors.Is(err, store.ErrConflict) {
		// Race with another reconcile or the API handler; re-read the existing record.
		return r.Store.GetRepoByOwnerAndName(ctx, ownerID, repo.Spec.Name)
	}
	return dbRepo, err
}

// syncDBRecord updates mutable fields on an existing DB record to match the CR spec.
func (r *RepositoryReconciler) syncDBRecord(ctx context.Context, dbRepo *models.Repository, repo *gforcev1alpha1.Repository) error {
	isPrivate := repo.Spec.IsPrivate
	defaultBranch := repo.Spec.DefaultBranch
	params := models.UpdateRepoParams{
		IsPrivate:     &isPrivate,
		DefaultBranch: &defaultBranch,
	}
	if repo.Spec.Description != "" {
		d := repo.Spec.Description
		params.Description = &d
	}
	_, err := r.Store.UpdateRepo(ctx, dbRepo.ID, params)
	return err
}

// setCondition sets or updates a condition on the conditions slice.
func setCondition(conditions *[]metav1.Condition, condType string, status metav1.ConditionStatus, reason, message string, generation int64) {
	meta.SetStatusCondition(conditions, metav1.Condition{
		Type:               condType,
		Status:             status,
		ObservedGeneration: generation,
		Reason:             reason,
		Message:            message,
	})
}
