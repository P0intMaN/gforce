// Package controllers implements the Kubernetes reconciliation loops for gforce CRDs.
package controllers

import (
	"context"
	"fmt"
	"path/filepath"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	gforcev1alpha1 "github.com/gforce/gforce/operator/api/v1alpha1"
	"github.com/gforce/gforce/pkg/gitutil"
)

const (
	conditionTypeReady = "Ready"
	gitRootPath        = "/var/lib/gforce/repos"
)

// RepositoryReconciler reconciles Repository custom resources.
//
// +kubebuilder:rbac:groups=gforce.dev,resources=repositories,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=gforce.dev,resources=repositories/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=gforce.dev,resources=repositories/finalizers,verbs=update
type RepositoryReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// SetupWithManager registers the reconciler with the controller manager.
func (r *RepositoryReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&gforcev1alpha1.Repository{}).
		Complete(r)
}

// Reconcile is called for every change to a Repository resource.
// It ensures that the bare git repository exists on disk and that the
// status reflects the current state of that resource.
func (r *RepositoryReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var repo gforcev1alpha1.Repository
	if err := r.Get(ctx, req.NamespacedName, &repo); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	diskPath := filepath.Join(gitRootPath, repo.Spec.OwnerRef, repo.Name+".git")

	if err := gitutil.InitBare(diskPath); err != nil {
		logger.Error(err, "initialising bare repository", "diskPath", diskPath)
		return ctrl.Result{}, r.setFailed(ctx, &repo, fmt.Sprintf("init bare repo: %v", err))
	}

	logger.Info("repository initialised", "diskPath", diskPath)
	return ctrl.Result{}, r.setReady(ctx, &repo, diskPath)
}

func (r *RepositoryReconciler) setReady(ctx context.Context, repo *gforcev1alpha1.Repository, diskPath string) error {
	repo.Status.Phase = gforcev1alpha1.RepositoryPhaseReady
	repo.Status.DiskPath = diskPath
	meta.SetStatusCondition(&repo.Status.Conditions, metav1.Condition{
		Type:               conditionTypeReady,
		Status:             metav1.ConditionTrue,
		ObservedGeneration: repo.Generation,
		Reason:             "Initialised",
		Message:            "bare git repository is ready",
	})
	if err := r.Status().Update(ctx, repo); err != nil {
		return fmt.Errorf("updating repository status to Ready: %w", err)
	}
	return nil
}

func (r *RepositoryReconciler) setFailed(ctx context.Context, repo *gforcev1alpha1.Repository, msg string) error {
	repo.Status.Phase = gforcev1alpha1.RepositoryPhaseFailed
	meta.SetStatusCondition(&repo.Status.Conditions, metav1.Condition{
		Type:               conditionTypeReady,
		Status:             metav1.ConditionFalse,
		ObservedGeneration: repo.Generation,
		Reason:             "InitFailed",
		Message:            msg,
	})
	if err := r.Status().Update(ctx, repo); err != nil {
		return fmt.Errorf("updating repository status to Failed: %w", err)
	}
	return nil
}
