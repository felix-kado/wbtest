package consumer

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"sync"
	"wbstorage/internal/db"
	"wbstorage/internal/models"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"golang.org/x/sync/errgroup"
)

type consumer struct {
	consumer jetstream.Consumer
	db       *db.CachedClient
}

func NewConsumer(ctx context.Context, db *db.CachedClient, natsUrl string) (*consumer, error) {
	nc, err := nats.Connect(natsUrl)
	if err != nil {
		return nil, fmt.Errorf("error connecting to NATS: %w", err)
	}

	js, err := jetstream.New(nc)
	if err != nil {
		return nil, fmt.Errorf("error creating a new JetStream instance: %w", err)
	}

	// TODO: Вынести строки в переменные откружения или конфиг
	streamConfig := jetstream.StreamConfig{
		Name:     "ORDERS",
		Subjects: []string{"ORDERS.*"},
	}
	st, err := js.CreateStream(ctx, streamConfig)
	if err != nil {
		return nil, fmt.Errorf("error creating stream: %w", err)
	}

	cs, err := st.CreateOrUpdateConsumer(ctx, jetstream.ConsumerConfig{
		Durable:   "CONS",
		AckPolicy: jetstream.AckExplicitPolicy,
	})
	if err != nil {
		return nil, fmt.Errorf("error creating consumer: %w", err)
	}

	return &consumer{
		consumer: cs,
		db:       db,
	}, nil
}

type job struct {
	Msg jetstream.Msg
}

// // Учечка горутины? Добавить контекст?
// func (c *consumer) StartListening(ctx context.Context, jobChan chan Job) error {
// 	it, err := c.consumer.Messages()
// 	if err != nil {
// 		return fmt.Errorf("error getting message iterator: %w", err)
// 	}
// 	defer it.Stop()

// 	for {
// 		msg, err := it.Next()
// 		if err != nil {
// 			log.Printf("error retrieving message: %v", err)
// 			continue
// 		}
// 		job := &Job{
// 			Msg: msg,
// 		}

// 		log.Print("Received a JetStream message")
// 		jobChan <- *job
// 	}
// }

// intentionally blocking. if you want nonblocking, run start inside errgroup
func (c *consumer) Start(ctx context.Context, g *errgroup.Group, workers int) error {
	jobs := make(chan job, workers)
	c.startWorkers(ctx, jobs, workers, g)
	return c.publishJobs(ctx, jobs)
}

func (c *consumer) startWorkers(ctx context.Context, jobs <-chan job, workers int, g *errgroup.Group) {
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range jobs {
				c.processJob(ctx, job)
			}
		}()
	}
	// this is for graceful shutdown, app errgroup should wait until all workers die peacefully
	g.Go(func() error {
		wg.Wait()
		return nil
	})
}

func (c *consumer) publishJobs(ctx context.Context, jobs chan<- job) error {
	// here you can publish jobs to the channel
	it, err := c.consumer.Messages()
	if err != nil {
		return err
	}
	for {
		select {
		case <-ctx.Done():
			close(jobs) // very important as it lets workers to stop
			return nil
		default:
			msg, err := it.Next()
			if err != nil {
				slog.Error("logs here")
				continue
			}
			job := job{Msg: msg}
			jobs <- job
		}
	}
}

func (c *consumer) processJob(ctx context.Context, job job) {
	var order models.Order
	if err := json.Unmarshal(job.Msg.Data(), &order); err != nil {
		log.Printf("error parsing message: %v", err)
		job.Msg.Ack()
		return
	}

	if err := c.db.InsertOrder(ctx, order); err != nil {
		log.Printf("error writing into db: %v", err)
	} else {
		job.Msg.Ack()
		log.Printf("Successfully inserted into DB: %s", order.OrderUID)
	}

	// Для мгновенной проверки, что все можно достать
	if orderDB, err := c.db.SelectOrder(ctx, order.OrderUID); err != nil {
		log.Printf("error selecting into db: %v", err)
	} else {
		log.Printf("Cheked: http://localhost:8080/%s", orderDB.OrderUID)
	}

}
