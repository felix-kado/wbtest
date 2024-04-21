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

const nWorkers = 8

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
	cachedDb := db.NewCachedClient(ctx, dbConn, 100)

	if err != nil {
		log.Println("Cache warmup fails")
	} else {
		log.Println("Successful cache warm-up")
	}

	consumer, err := consumer.NewConsumer(ctx, cachedDb, cfg.NATSUrl)
	if err != nil {
		log.Fatalf("error initializing consumer: %v", err)
	}
	go consumer.Start(ctx, group, 8)

	log.Println("Preparation successful, waiting for request")
	serv := server.NewServer(cachedDb)
	router := server.NewRouter(serv)
	http.ListenAndServe(":8080", router)

	if err != nil {
		log.Fatalf("error initializing server: %v", err)
	}
}
