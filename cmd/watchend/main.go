package main

import (
	"context"
	"errors"
	"log/slog"

	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/cusox/watchend/internal/auth"
	"github.com/cusox/watchend/internal/config"
	"github.com/cusox/watchend/internal/github"
	watchhttp "github.com/cusox/watchend/internal/http"
	"github.com/cusox/watchend/internal/store"
)

type readiness struct {
	db *store.Store
}

func (r readiness) Ready(ctx context.Context) error {
	return r.db.Ping(ctx)
}

func main() {
	if err := run(); err != nil {
		slog.Error("watchend stopped", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	db, err := store.Open(ctx, cfg.DatabasePath)
	if err != nil {
		return err
	}
	defer db.Close()

	if err = ensureAdmin(ctx, db, cfg.AdminUsername, cfg.AdminPassword); err != nil {
		return err
	}

	secret, err := auth.LoadOrCreateSessionSecret(cfg.DatabasePath)
	if err != nil {
		return err
	}

	syncer := github.New(cfg.GitHubToken, db)
	handler, err := watchhttp.New(watchhttp.Options{
		Store:         db,
		SessionSecret: string(secret),
		SessionTTL:    cfg.SessionTTL,
		SecureCookies: cfg.SecureCookies,
		Logger:        slog.Default(),
		Readiness:     readiness{db},
		Syncer:        syncer,
	})
	if err != nil {
		return err
	}

	server := &http.Server{
		Addr:              cfg.Addr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		ticker := time.NewTicker(time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if err := db.DeleteExpiredSessions(ctx); err != nil {
					slog.Error("session cleanup failed", "error", err)
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	go func() {
		ticker := time.NewTicker(cfg.SyncInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if err := syncer.Sync(ctx); err != nil && !errors.Is(err, github.ErrAlreadyRunning) {
					slog.Error("automatic sync failed", "error", err)
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	go func() {
		<-ctx.Done()

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		_ = server.Shutdown(shutdownCtx)
	}()

	slog.Info("watchend listening", "address", cfg.Addr)

	err = server.ListenAndServe()
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}

	return err
}
