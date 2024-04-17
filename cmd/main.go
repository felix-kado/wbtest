package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"wbstorage/internal/consumer"
	"wbstorage/internal/db"
	"wbstorage/internal/models"
)

const nWorkers = 8

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 360*time.Second)
	defer cancel()

	// TODO: Вынести строки в переменные откружения или конфиг
	dbConn, err := db.NewDB("user=felix dbname=wbstore sslmode=disable password=12345678 host=localhost")
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}

	defer dbConn.Db.Close()

	cachedDb := db.NewCachedClient(dbConn)
	jobChan := make(chan consumer.Job, nWorkers)

	for i := 0; i < nWorkers; i++ {
		go worker(ctx, cachedDb, jobChan)
	}
	consumer, err := consumer.NewConsumer(ctx)
	if err != nil {
		log.Fatalf("error initializing consumer: %v", err)
	}

	go consumer.StartListening(jobChan)

	for i := 0; i < nWorkers; i++ {
		go worker(ctx, cachedDb, jobChan)
	}

	<-ctx.Done()
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
		fmt.Println("Successfully inserted")
	}

	// Для мгновенной проверки, что все можно достать
	if orderDB, err := cachedDb.GetOrderFromCache(order.OrderUID); err != nil {
		log.Printf("error selecting into db: %v", err)
	} else {
		fmt.Println("Successfully selected")
		fmt.Println(orderDB)
	}

}
