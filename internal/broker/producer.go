package broker

import (
	"context"
	"encoding/json"
	"fmt"
	"l0/internal/models"
	"time"

	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"
)

type Producer struct {
	writer *kafka.Writer
	topic  string
}

// NewProducer создает новый Kafka producer
func NewProducer(config Config) (*Producer, error) {
	writer := kafka.NewWriter(kafka.WriterConfig{
		Brokers:      config.Brokers,
		Topic:        config.Topic,
		Balancer:     &kafka.Hash{},
		Async:        false,
		ErrorLogger:  kafka.LoggerFunc(zap.S().Errorf),
		RequiredAcks: int(kafka.RequireAll),
	})
	return &Producer{writer: writer, topic: config.Topic}, nil
}

// SendOrder сериализует заказ и отправляет его в Kafka
func (p *Producer) SendOrder(ctx context.Context, order *models.Order) error {
	value, err := json.Marshal(order)
	if err != nil {
		zap.S().Errorf("failed to marshal order: %v", err)
		return err
	}

	msg := kafka.Message{
		Key:   []byte(order.OrderUID),
		Value: value,
		Time:  time.Now(),
	}

	// Отправляем сообщение с таймаутом
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	err = p.writer.WriteMessages(ctx, msg)
	if err != nil {
		zap.S().Errorf("failed to send order %s: %v", order.OrderUID, err)
		return fmt.Errorf("failed to send message: %w", err)
	}

	zap.S().Infof("order sent to kafka: %s", order.OrderUID)
	return nil
}

// SendOrderBatch отправляет несколько заказов пакетом
func (p *Producer) SendOrderBatch(ctx context.Context, orders []*models.Order) error {
	if len(orders) == 0 {
		return nil
	}

	messages := make([]kafka.Message, len(orders))
	for i, order := range orders {
		value, err := json.Marshal(order)
		if err != nil {
			zap.S().Errorf("failed to marshal order %s: %v", order.OrderUID, err)
			return err
		}

		messages[i] = kafka.Message{
			Key:   []byte(order.OrderUID),
			Value: value,
			Time:  time.Now(),
		}
	}

	// Отправляем пакет с таймаутом
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	err := p.writer.WriteMessages(ctx, messages...)
	if err != nil {
		zap.S().Errorf("failed to send order batch: %v", err)
		return fmt.Errorf("failed to send message batch: %w", err)
	}

	zap.S().Infof("sent %d orders to kafka", len(orders))
	return nil
}

// Close закрывает producer
func (p *Producer) Close() {
	zap.S().Info("closing kafka producer...")
	if err := p.writer.Close(); err != nil {
		zap.S().Errorf("failed to close kafka writer: %v", err)
	}
	zap.S().Info("kafka producer closed")
}
