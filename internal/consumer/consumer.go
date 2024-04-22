package consumer

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"
	"wbstorage/internal/db"
	"wbstorage/internal/models"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"golang.org/x/sync/errgroup"
)

type consumer struct {
	consumer jetstream.Consumer
	db       db.Database
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

func (c *consumer) Start(ctx context.Context, g *errgroup.Group, workers int) {
	jobs := make(chan job, workers)
	c.startWorkers(ctx, jobs, workers, g)
	c.publishJobs(ctx, jobs)
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

func (c *consumer) publishJobs(ctx context.Context, jobs chan<- job) {
	// here you can publish jobs to the channel
	it, err := c.consumer.Messages()
	if err != nil {
		slog.Error("Failed to retrieve messages", "error", err)
		return
	}
	for {
		select {
		case <-ctx.Done():
			close(jobs) // very important as it lets workers to stop
			slog.Info("Context done, jobs channel closed")
		default:
			msg, err := it.Next()
			if err != nil {
				slog.Error("Failed to get next message", "error", err)
				continue
			}
			job := job{Msg: msg}
			jobs <- job
			slog.Info("Job published", "job", job)
		}
	}
}

func (c *consumer) processJob(ctx context.Context, job job) {
	var order models.Order
	if err := json.Unmarshal(job.Msg.Data(), &order); err != nil {
		slog.Error("Error parsing message", "error", err)
		ackErr := job.Msg.Ack()
		if ackErr != nil {
			slog.Error("Error acknowledges a message", "error", ackErr)
		}
		return
	}
	ctxInsert, cancel := context.WithTimeout(ctx, time.Second*15)
	defer cancel()

	if err := c.db.InsertOrder(ctxInsert, order); err != nil {
		slog.Error("Error writing into DB", "error", err)
		return
	} else {
		ackErr := job.Msg.Ack()
		if ackErr != nil {
			slog.Error("Error acknowledges a message", "error", ackErr)
		}
		slog.Info("Successfully inserted into DB", "orderUID", order.OrderUID)
	}

	slog.Info("Success", "url", "http://localhost:8080/"+order.OrderUID)

}
