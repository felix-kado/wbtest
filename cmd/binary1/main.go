package main

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"time"

	"wbstorage/internal/consumer"
	"wbstorage/internal/db"
	"wbstorage/internal/server"

	"golang.org/x/sync/errgroup"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	cfg, err := LoadConfig()
	if err != nil {
		slog.Error("Failed to load configuration", "error", err)
		os.Exit(1)
	}

	ctx := context.Background()
	// for graceful shutdown
	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt)
	defer cancel()
	group, ctx := errgroup.WithContext(ctx)

	slog.Info("Database connection string loaded", "connString", cfg.ConnString)

	dbConn, err := db.NewDB(cfg.ConnString)
	if err != nil {
		slog.Error("Failed to connect to the database", "error", err)
		os.Exit(1)
	}
	cachedDb, err := db.NewCachedClient(ctx, dbConn, 100)

	if err != nil {
		slog.Error("Cache warmup failed", "error", err)
	} else {
		slog.Info("Successful cache warm-up")
	}

	consumer, err := consumer.NewConsumer(ctx, cachedDb, cfg.NATSUrl)
	if err != nil {
		slog.Error("Error initializing consumer", "error", err)
		os.Exit(1)
	}
	go consumer.Start(ctx, group, cfg.NWorkers)
	slog.Info("Consumer prepared and started successfully")

	runServer(ctx, cachedDb, cfg, group)

	if err := group.Wait(); err != nil {
		slog.Error("Error waiting for all goroutines to finish", "error", err)
	}

	slog.Info("Server shutdown successfully")
	time.Sleep(time.Second * 5)
}

func runServer(ctx context.Context, db db.Database, cfg Config, g *errgroup.Group) {
	echoServer, err := server.NewServer(db)
	if err != nil {
		slog.Error("Error initializing server", "error", err)
		os.Exit(1)
	}
	echoMux := server.NewRouter(echoServer)

	httpServer := http.Server{
		Addr:        cfg.ServerPort,                                                           // host:port pair from config
		Handler:     echoMux,                                                                  // http mux that was created
		BaseContext: func(net.Listener) context.Context { return context.WithoutCancel(ctx) }, // important to not have context cancelled mid-request on shutdown
	}

	g.Go(func() error {
		<-ctx.Done()

		slog.Info("Shutting down HTTP server...", "context", "server-shutdown")
		shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancel()
		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			slog.Error("Failed to shut down HTTP server", "error", err)
			return err
		}
		slog.Info("HTTP server shut down successfully")
		return nil
	})

	g.Go(func() error {
		if err := httpServer.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			slog.Error("HTTP server stopped unexpectedly", "error", err)
			return err
		}
		slog.Info("HTTP server stopped gracefully")
		return nil
	})
}
