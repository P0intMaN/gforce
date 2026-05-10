// Command operator is the Kubernetes operator entrypoint for gforce.
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	gforcev1alpha1 "github.com/gforce/gforce/operator/api/v1alpha1"
	"github.com/gforce/gforce/operator/controllers"
	"github.com/gforce/gforce/internal/store/postgres"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	zaplogger "sigs.k8s.io/controller-runtime/pkg/log/zap"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var scheme = runtime.NewScheme()

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(gforcev1alpha1.AddToScheme(scheme))
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	ctrl.SetLogger(zaplogger.New(zaplogger.UseDevMode(false)))

	logger, err := buildLogger()
	if err != nil {
		return fmt.Errorf("building logger: %w", err)
	}
	defer logger.Sync() //nolint:errcheck

	// Connect to the GForce database.
	dsn := os.Getenv("GFORCE_DB_DSN")
	if dsn == "" {
		return fmt.Errorf("GFORCE_DB_DSN environment variable must be set")
	}

	repoRoot := os.Getenv("GFORCE_GIT_STORAGE_PATH")
	if repoRoot == "" {
		repoRoot = "/var/lib/gforce/repos"
	}

	ctx := context.Background()
	pool, err := postgres.NewPool(ctx, dsn, 10, 2)
	if err != nil {
		return fmt.Errorf("connecting to database: %w", err)
	}
	defer pool.Close()

	db := postgres.NewDB(pool)
	logger.Info("database connected")

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme: scheme,
		Metrics: metricsserver.Options{
			BindAddress: ":8081",
		},
		HealthProbeBindAddress:  ":8082",
		LeaderElection:          true,
		LeaderElectionID:        "gforce-operator-leader.gforce.io",
		LeaderElectionNamespace: "gforce-system",
	})
	if err != nil {
		return fmt.Errorf("creating manager: %w", err)
	}

	if err := (&controllers.RepositoryReconciler{
		Client:   mgr.GetClient(),
		Scheme:   mgr.GetScheme(),
		Store:    db,
		RepoRoot: repoRoot,
		Logger:   logger,
	}).SetupWithManager(mgr); err != nil {
		return fmt.Errorf("setting up Repository controller: %w", err)
	}

	if err := (&controllers.UserReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
		Store:  db,
		Logger: logger,
	}).SetupWithManager(mgr); err != nil {
		return fmt.Errorf("setting up User controller: %w", err)
	}

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		return fmt.Errorf("adding healthz check: %w", err)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		return fmt.Errorf("adding readyz check: %w", err)
	}

	logger.Info("starting operator",
		zap.String("repoRoot", repoRoot),
		zap.String("metricsAddr", ":8081"),
	)
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		return fmt.Errorf("running manager: %w", err)
	}
	return nil
}

func buildLogger() (*zap.Logger, error) {
	level := os.Getenv("GFORCE_LOG_LEVEL")
	if level == "" {
		level = "info"
	}
	lvl, err := zapcore.ParseLevel(level)
	if err != nil {
		return nil, fmt.Errorf("invalid log level %q: %w", level, err)
	}
	cfg := zap.NewProductionConfig()
	cfg.Level = zap.NewAtomicLevelAt(lvl)
	cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	return cfg.Build()
}

// keep time import happy
var _ = time.Second
