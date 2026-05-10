// Command gforce is the main entry point for the gforce Git platform server.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	gforcev1alpha1 "github.com/gforce/gforce/operator/api/v1alpha1"
	"github.com/gforce/gforce/internal/api"
	"github.com/gforce/gforce/internal/auth"
	"github.com/gforce/gforce/internal/config"
	"github.com/gforce/gforce/internal/server"
	"github.com/gforce/gforce/internal/store"
	"github.com/gforce/gforce/internal/store/postgres"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func main() {
	if err := rootCmd().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func rootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "gforce",
		Short: "gforce — Kubernetes-native Git platform",
	}
	root.AddCommand(serveCmd())
	return root
}

func serveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "serve",
		Short: "Start the gforce HTTP API server",
		RunE:  runServe,
	}
}

func runServe(_ *cobra.Command, _ []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	logger, err := buildLogger(cfg.Log.Level)
	if err != nil {
		return fmt.Errorf("building logger: %w", err)
	}
	defer logger.Sync() //nolint:errcheck

	ctx := context.Background()

	pool, err := postgres.NewPool(ctx, cfg.DB.DSN, int32(cfg.DB.MaxOpenConns), int32(cfg.DB.MaxIdleConns))
	if err != nil {
		return fmt.Errorf("connecting to database: %w", err)
	}
	defer pool.Close()

	if err := store.RunMigrations(ctx, pool, "internal/store/migrations"); err != nil {
		return fmt.Errorf("running migrations: %w", err)
	}
	logger.Info("migrations applied")

	db := postgres.NewDB(pool)
	if err := db.Ping(ctx); err != nil {
		return fmt.Errorf("database not reachable: %w", err)
	}
	logger.Info("database connected")

	authSvc, err := auth.NewService(cfg.Auth.JWTSecret, time.Duration(cfg.Auth.TokenTTLMinutes)*time.Minute)
	if err != nil {
		return fmt.Errorf("initialising auth service: %w", err)
	}

	// Optionally wire up the Kubernetes client for CR sync.
	// If running outside a cluster (e.g. local dev), this is skipped gracefully.
	var k8s k8sclient.Client
	k8sNamespace := cfg.Kubernetes.Namespace
	if k8sCfg, err := ctrl.GetConfig(); err == nil {
		k8sScheme := buildK8sScheme()
		if k8s, err = k8sclient.New(k8sCfg, k8sclient.Options{Scheme: k8sScheme}); err != nil {
			logger.Warn("could not create Kubernetes client; CR sync disabled", zap.Error(err))
			k8s = nil
		} else {
			logger.Info("kubernetes client ready", zap.String("namespace", k8sNamespace))
		}
	} else {
		logger.Info("kubernetes not available; CR sync disabled")
	}

	handler := api.NewRouter(api.RouterConfig{
		Store:          db,
		AuthService:    authSvc,
		GitRootPath:    cfg.Git.StoragePath,
		BaseURL:        cfg.Server.BaseURL,
		AllowedOrigins: cfg.Server.AllowedOrigins,
		Logger:         logger,
		K8sClient:      k8s,
		K8sNamespace:   k8sNamespace,
	})

	srv := server.New(server.Config{
		Port:             cfg.Server.Port,
		Handler:          handler,
		ReadTimeoutSecs:  cfg.Server.ReadTimeoutSecs,
		WriteTimeoutSecs: cfg.Server.WriteTimeoutSecs,
		Logger:           logger,
	})

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	serverErr := make(chan error, 1)
	go func() { serverErr <- srv.Start() }()

	select {
	case err := <-serverErr:
		return fmt.Errorf("server error: %w", err)
	case sig := <-quit:
		logger.Info("received shutdown signal", zap.String("signal", sig.String()))
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("graceful shutdown: %w", err)
	}

	logger.Info("server stopped cleanly")
	return nil
}

func buildK8sScheme() *runtime.Scheme {
	s := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(s))
	utilruntime.Must(gforcev1alpha1.AddToScheme(s))
	return s
}

func buildLogger(level string) (*zap.Logger, error) {
	lvl, err := zapcore.ParseLevel(level)
	if err != nil {
		return nil, fmt.Errorf("parsing log level %q: %w", level, err)
	}
	cfg := zap.NewProductionConfig()
	cfg.Level = zap.NewAtomicLevelAt(lvl)
	cfg.EncoderConfig.TimeKey = "ts"
	cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	return cfg.Build()
}
