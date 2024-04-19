package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"wbstorage/internal/consumer"
	"wbstorage/internal/db"
	"wbstorage/internal/models"
	"wbstorage/internal/server"

	"github.com/nats-io/nats.go"
)

const nWorkers = 8

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}

func main() {
	dbHost := getEnv("DB_HOST", "localhost")
	dbUser := getEnv("DB_USER", "user")
	dbPassword := getEnv("DB_PASSWORD", "password")
	dbName := getEnv("DB_NAME", "dbname")
	natsUrl := getEnv("NATS_URL", nats.DefaultURL)

	ctx, cancel := context.WithTimeout(context.Background(), 24*time.Hour)
	defer cancel()
	dbConnString := fmt.Sprintf("user=%s dbname=%s sslmode=disable password=%s host=%s", dbUser, dbName, dbPassword, dbHost)

	dbConn, err := db.NewDB(dbConnString)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}

	defer dbConn.Db.Close()

	cachedDb := db.NewCachedClient(dbConn)
	err = cachedDb.CacheWarming(1000)
	if err != nil {
		log.Println("Cache warmup fails")
	} else {
		log.Println("Successful cache warm-up")
	}

	jobChan := make(chan consumer.Job, nWorkers)

	for i := 0; i < nWorkers; i++ {
		go worker(ctx, cachedDb, jobChan)
	}
	consumer, err := consumer.NewConsumer(ctx, natsUrl)
	if err != nil {
		log.Fatalf("error initializing consumer: %v", err)
	}

	go consumer.StartListening(jobChan)

	for i := 0; i < nWorkers; i++ {
		go worker(ctx, cachedDb, jobChan)
	}
	log.Println("Preparation successful, waiting for request")

	serv := server.NewServer(cachedDb)
	err = serv.Start(":8080")
	if err != nil {
		log.Fatalf("error initializing server: %v", err)
	}
}

func worker(ctx context.Context, cachedDb *db.CachedClient, jobChan chan consumer.Job) {
	for {
		select {
		case job, ok := <-jobChan:
			if !ok {
				return
			}
			processJob(cachedDb, job)
		case <-ctx.Done():
			return
		}
	}
}

func processJob(cachedDb *db.CachedClient, job consumer.Job) {
	var order models.Order
	if err := json.Unmarshal(job.Msg.Data(), &order); err != nil {
		log.Printf("error parsing message: %v", err)
		job.Msg.Ack()
		return
	}

	if err := cachedDb.PutOrderIntoDbAndCache(order); err != nil {
		log.Printf("error writing into db: %v", err)
	} else {
		job.Msg.Ack()
		log.Printf("Successfully inserted into DB: %s", order.OrderUID)
	}

	// Для мгновенной проверки, что все можно достать
	if orderDB, err := cachedDb.GetOrderFromCache(order.OrderUID); err != nil {
		log.Printf("error selecting into db: %v", err)
	} else {
		// fmt.Println("Successfully selected")
		log.Printf("Cheked: http://localhost:8080/%s", orderDB.OrderUID)
	}

}
