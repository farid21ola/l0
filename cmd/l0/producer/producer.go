package main

import (
	"context"
	"fmt"
	"l0/config"
	"l0/internal/broker"
	"l0/internal/models"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"
)

func main() {
	cfg, err := config.NewConfig()
	if err != nil {
		log.Fatalf("failed to initialize config: %v", err)
	}

	// Создаем producer
	producer, err := broker.NewProducer(cfg.KafkaConfig)
	if err != nil {
		log.Fatalf("failed to create producer: %v", err)
	}
	defer producer.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Graceful shutdown
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		log.Print("shutting down producer...")
		cancel()
	}()

	// Отправляем тестовые заказы
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		orderCounter := 1
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				order := createTestOrder(orderCounter)
				if err := producer.SendOrder(ctx, order); err != nil {
					zap.S().Errorf("failed to send order: %v", err)
				}
				log.Printf("message №%d sent", orderCounter)
				orderCounter++
			}
		}
	}()

	// Ждем завершения
	<-ctx.Done()
	log.Print("producer stopped")
}

// createTestOrder создает тестовый заказ
func createTestOrder(counter int) *models.Order {
	return &models.Order{
		OrderUID:          fmt.Sprintf("test-order-%d", counter),
		TrackNumber:       fmt.Sprintf("WBILMTESTTRACK%d", counter),
		Entry:             "WBIL",
		InternalSignature: "internal-sig",
		CustomerID:        fmt.Sprintf("customer-%d", counter),
		Shardkey:          "shard-1",
		SmID:              99,
		OofShard:          "oof-1",
		Delivery: models.Delivery{
			Name:    "Test Testov",
			Phone:   "+9720000000",
			Zip:     "2639809",
			City:    "Kiryat Mozkin",
			Address: "Ploshad Mira 15",
			Region:  "Kraiot",
			Email:   "test@gmail.com",
		},
		Payment: models.Payment{
			Transaction:  fmt.Sprintf("b563feb7b2b84b6test%d", counter),
			RequestID:    fmt.Sprintf("internal-request-id-%d", counter),
			Currency:     "USD",
			Provider:     "wbpay",
			Amount:       1817 + counter*100,
			PaymentDt:    int(time.Now().Unix()),
			Bank:         "alpha",
			DeliveryCost: 1500,
			GoodsTotal:   317 + counter*100,
			CustomFee:    0,
		},
		Items: models.Items{
			{
				ChrtID:      9934930 + counter,
				TrackNumber: fmt.Sprintf("WBILMTESTTRACK%d", counter),
				Price:       453 + counter*10,
				Rid:         fmt.Sprintf("ab4219087a764ae0btest%d", counter),
				Name:        fmt.Sprintf("Product %d", counter),
				Sale:        30,
				Size:        "0",
				TotalPrice:  317 + counter*100,
				NmID:        2389212 + counter,
				Brand:       "Test Brand",
				Status:      202,
			},
		},
		Locale:          "en",
		DeliveryService: "meest",
		DateCreated:     time.Now(),
	}
}
