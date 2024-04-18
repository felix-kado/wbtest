package consumer

import (
	"context"
	"fmt"
	"log"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

type consumer struct {
	ctx         context.Context
	natsConnect *nats.Conn
	jetStream   jetstream.JetStream
	stream      jetstream.Stream
	consumer    jetstream.Consumer
}

func NewConsumer(ctx context.Context, natsUrl string) (*consumer, error) {
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
		ctx:         ctx,
		natsConnect: nc,
		jetStream:   js,
		stream:      st,
		consumer:    cs,
	}, nil
}

type Job struct {
	ctx context.Context
	Msg jetstream.Msg
}

func (c *consumer) StartListening(jobChan chan Job) error {
	it, err := c.consumer.Messages()
	if err != nil {
		return fmt.Errorf("error getting message iterator: %w", err)
	}
	defer it.Stop()

	for {
		msg, err := it.Next()
		if err != nil {
			log.Printf("error retrieving message: %v", err)
			continue
		}
		job := &Job{
			ctx: c.ctx,
			Msg: msg,
		}

		log.Printf("Received a JetStream message: %s", string(msg.Data())[:30])
		jobChan <- *job
	}
}
