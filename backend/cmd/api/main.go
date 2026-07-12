package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rozy/backend/internal/platform/config"
	"github.com/rozy/backend/internal/platform/db"
	redisclient "github.com/rozy/backend/internal/platform/redis"
	"github.com/rozy/backend/internal/platform/server"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	ctx := context.Background()

	if cfg.AutoMigrate {
		log.Println("running database migrations...")
		if err := db.RunMigrations(cfg.DatabaseURL); err != nil {
			log.Fatalf("migrate: %v", err)
		}
		log.Println("migrations up to date")
	}

	pool, err := db.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("db: %v", err)
	}
	defer pool.Close()

	var redis *redisclient.Client
	if r, err := redisclient.New(cfg.RedisURL); err != nil {
		log.Printf("redis: %v (continuing without redis)", err)
	} else {
		redis = r
		defer redis.Close()
	}

	srv := server.New(cfg, pool, redis)
	httpServer := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      srv.Handler(),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Printf("rozy api listening on :%s (env=%s)", cfg.Port, cfg.Env)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("shutdown: %v", err)
	}
	log.Println("rozy api stopped")
}
