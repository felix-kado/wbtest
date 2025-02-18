package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"
	"wbstorage/internal/models"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/nats-io/nats.go"
)

//go:embed "testdata/model.json"
var sampleOrderJSON []byte

func generateOrder() ([]byte, error) {
	gofakeit.Seed(0)
	order := models.Order{
		OrderUID:    gofakeit.UUID(),
		TrackNumber: "WBILMTESTTRACK",
		Entry:       "WBIL",
		Delivery: models.Delivery{
			Name:    gofakeit.Name(),
			Phone:   gofakeit.Phone(),
			Zip:     gofakeit.Zip(),
			City:    gofakeit.City(),
			Address: gofakeit.Address().Address,
			Region:  gofakeit.State(),
			Email:   gofakeit.Email(),
		},
		Payment: models.Payment{
			Transaction:  gofakeit.UUID(),
			RequestID:    "",
			Currency:     "USD",
			Provider:     "wbpay",
			Amount:       gofakeit.Number(100, 5000),
			PaymentDt:    int64(gofakeit.Date().Unix()),
			Bank:         "alpha",
			DeliveryCost: gofakeit.Number(1000, 2000),
			GoodsTotal:   gofakeit.Number(300, 400),
			CustomFee:    0,
		},
		Items: func() []models.Item {
			var items []models.Item
			count := gofakeit.Number(1, 15)
			for i := 0; i < count; i++ {
				items = append(items, models.Item{
					ChrtID:      gofakeit.Number(1000000, 9999999),
					TrackNumber: "WBILMTESTTRACK",
					Price:       gofakeit.Number(100, 1000),
					RID:         gofakeit.UUID(),
					Name:        gofakeit.ProductName(),
					Sale:        gofakeit.Number(10, 50),
					Size:        "0",
					TotalPrice:  gofakeit.Number(200, 500),
					NmID:        gofakeit.Number(100000, 999999),
					Brand:       gofakeit.Company(),
					Status:      gofakeit.Number(100, 300),
				})
			}
			return items
		}(),
		Locale:            "en",
		InternalSignature: "",
		CustomerID:        "test",
		DeliveryService:   "meest",
		ShardKey:          "9",
		SmID:              99,
		DateCreated:       gofakeit.Date(),
		OofShard:          "1",
	}

	jsonData, err := json.MarshalIndent(order, "", "  ")

	return jsonData, err
}

func main() {
	cfg, err := loadConfig()
	if err != nil {
		log.Panicln("Failed to load configuration", err)
		os.Exit(1)
	}
	gofakeit.Seed(0)

	js, err := JetStreamInit(cfg.NATSUrl)
	if err != nil {
		log.Fatalf("Cannot init Jetstream")
	}

	if err != nil {
		fmt.Println("Error encoding order to JSON:", err)
		return
	}
	_ = sampleOrderJSON
	// publishOrder(js, sampleOrderJSON)

	for i := 0; i < 5; i++ {
		orderJSON, err := generateOrder()

		if err != nil {
			fmt.Println("Error encoding order to JSON:", err)
			return
		}

		publishOrder(js, orderJSON)
		time.Sleep(time.Second * 1)

	}

}

const (
	StreamName     = "ORDERS"
	StreamSubjects = "ORDERS.*"
)

func JetStreamInit(natsURL string) (nats.JetStreamContext, error) {
	nc, err := nats.Connect(natsURL)
	if err != nil {
		return nil, err
	}

	js, err := nc.JetStream(nats.PublishAsyncMaxPending(256))
	if err != nil {
		return nil, err
	}

	return js, nil
}

func CreateStream(jetStream nats.JetStreamContext) error {
	stream, err := jetStream.StreamInfo(StreamName)
	if err != nil {
		log.Fatalf("Error getting info: %s\n", StreamName)
	}

	if stream == nil {
		log.Printf("Creating stream: %s\n", StreamName)

		_, err = jetStream.AddStream(&nats.StreamConfig{
			Name:     StreamName,
			Subjects: []string{StreamSubjects},
		})
		if err != nil {
			return err
		}
	}
	return nil
}

const (
	SubjectNameReviewCreated = "ORDERS.order"
)

func publishOrder(js nats.JetStreamContext, orderJSON []byte) {

	_, err := js.Publish(SubjectNameReviewCreated, orderJSON)
	if err != nil {
		log.Println(err)
	} else {
		log.Println("Publisher  =>  Message")
	}

}
