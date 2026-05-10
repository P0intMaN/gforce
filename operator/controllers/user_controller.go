package controllers

import (
	"context"
	"errors"
	"fmt"

	gforcev1alpha1 "github.com/gforce/gforce/operator/api/v1alpha1"
	"github.com/gforce/gforce/internal/models"
	"github.com/gforce/gforce/internal/store"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"go.uber.org/zap"
)

const userFinalizer = "gforce.io/user-cleanup"

// UserReconciler reconciles GForceUser custom resources.
//
// +kubebuilder:rbac:groups=gforce.io,resources=gforceusers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=gforce.io,resources=gforceusers/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=gforce.io,resources=gforceusers/finalizers,verbs=update
type UserReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Store  store.Store
	Logger *zap.Logger
}

// SetupWithManager registers the reconciler with the controller manager.
func (r *UserReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&gforcev1alpha1.GForceUser{}).
		Complete(r)
}

// Reconcile reconciles a GForceUser resource to desired state.
func (r *UserReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Logger.With(zap.String("name", req.Name), zap.String("namespace", req.Namespace))

	// ── 1. Fetch ─────────────────────────────────────────────────────────────
	var user gforcev1alpha1.GForceUser
	if err := r.Get(ctx, req.NamespacedName, &user); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, fmt.Errorf("fetching GForceUser: %w", err)
	}

	// ── 2. Deletion ───────────────────────────────────────────────────────────
	if !user.DeletionTimestamp.IsZero() {
		return r.handleUserDeletion(ctx, &user, log)
	}

	// ── 3. Finalizer ──────────────────────────────────────────────────────────
	if !controllerutil.ContainsFinalizer(&user, userFinalizer) {
		controllerutil.AddFinalizer(&user, userFinalizer)
		if err := r.Update(ctx, &user); err != nil {
			return ctrl.Result{}, fmt.Errorf("adding finalizer: %w", err)
		}
		return ctrl.Result{Requeue: true}, nil
	}

	// ── 4. Ensure DB record ───────────────────────────────────────────────────
	dbUser, err := r.Store.GetUserByUsername(ctx, user.Spec.Username)
	if errors.Is(err, store.ErrNotFound) {
		dbUser, err = r.Store.CreateUser(ctx, models.CreateUserParams{
			Username: user.Spec.Username,
			Email:    user.Spec.Email,
			// Password hash left empty — operator-managed users authenticate via SSO/token.
			PasswordHash: "!operator-managed",
		})
		if errors.Is(err, store.ErrConflict) {
			dbUser, err = r.Store.GetUserByUsername(ctx, user.Spec.Username)
		}
		if err != nil {
			log.Error("creating DB user record", zap.Error(err))
			setCondition(&user.Status.Conditions, "DatabaseSynced",
				metav1.ConditionFalse, "CreateFailed", err.Error(), user.Generation)
			_ = r.Status().Update(ctx, &user)
			return ctrl.Result{RequeueAfter: requeueDelay}, err
		}
	} else if err != nil {
		log.Error("querying DB user record", zap.Error(err))
		return ctrl.Result{RequeueAfter: requeueDelay}, err
	} else {
		// Sync mutable fields.
		var displayName *string
		if user.Spec.DisplayName != "" {
			d := user.Spec.DisplayName
			displayName = &d
		}
		if _, err := r.Store.UpdateUser(ctx, dbUser.ID, models.UpdateUserParams{
			DisplayName: displayName,
		}); err != nil {
			log.Warn("syncing DB user record", zap.Error(err))
		}
	}

	// ── 5. Status ─────────────────────────────────────────────────────────────
	user.Status.Phase = gforcev1alpha1.GForceUserPhaseActive
	user.Status.DatabaseID = dbUser.ID.String()
	user.Status.ObservedGeneration = user.Generation
	setCondition(&user.Status.Conditions, "DatabaseSynced",
		metav1.ConditionTrue, "Synced", "Database record is up to date", user.Generation)
	setCondition(&user.Status.Conditions, "Ready",
		metav1.ConditionTrue, "Ready", "User is active", user.Generation)

	if err := r.Status().Update(ctx, &user); err != nil {
		return ctrl.Result{}, fmt.Errorf("updating user status: %w", err)
	}

	log.Info("user reconciled", zap.String("username", user.Spec.Username))
	return ctrl.Result{}, nil
}

// handleUserDeletion removes the DB record and strips the finalizer.
func (r *UserReconciler) handleUserDeletion(ctx context.Context, user *gforcev1alpha1.GForceUser, log *zap.Logger) (ctrl.Result, error) {
	log.Info("handling user deletion", zap.String("username", user.Spec.Username))

	if user.Status.DatabaseID != "" {
		dbUser, err := r.Store.GetUserByUsername(ctx, user.Spec.Username)
		if err == nil {
			if _, err := r.Store.UpdateUser(ctx, dbUser.ID, models.UpdateUserParams{}); err != nil {
				log.Warn("could not deactivate DB user", zap.Error(err))
			}
		}
	}

	controllerutil.RemoveFinalizer(user, userFinalizer)
	if err := r.Update(ctx, user); err != nil {
		return ctrl.Result{}, fmt.Errorf("removing user finalizer: %w", err)
	}

	log.Info("user deleted from operator")
	return ctrl.Result{}, nil
}
