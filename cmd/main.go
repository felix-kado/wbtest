package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"

	"wbstorage/internal/configs"
	"wbstorage/internal/consumer"
	"wbstorage/internal/db"
	"wbstorage/internal/server"

	"golang.org/x/sync/errgroup"
)

func main() {
	cfg, err := configs.LoadConfig()
	if err != nil {
		log.Fatalf("failed to config: %v", err)
	}

	ctx := context.Background()
	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt)
	defer cancel()
	group, ctx := errgroup.WithContext(ctx)
	log.Println(cfg.ConnString)

	dbConn, err := db.NewDB(cfg.ConnString)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	cachedDb, err := db.NewCachedClient(ctx, dbConn, 100)

	if err != nil {
		log.Printf("Cache warmup fails: %v\n", err)
	} else {
		log.Println("Successful cache warm-up")
	}

	consumer, err := consumer.NewConsumer(ctx, cachedDb, cfg.NATSUrl)
	if err != nil {
		log.Fatalf("error initializing consumer: %v", err)
	}
	go consumer.Start(ctx, group, 8)

	log.Println("Preparation successful, waiting for request")
	serv, err := server.NewServer(cachedDb)

	if err != nil {
		log.Fatalf("error initializing server: %v", err)
	}
	router := server.NewRouter(serv)
	http.ListenAndServe(cfg.ServerPort, router)
}
