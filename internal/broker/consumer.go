package broker

import (
	"context"
	"encoding/json"
	"errors"
	"l0/internal/models"
	"time"

	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"
)

type Consumer struct {
	reader *kafka.Reader
	topic  string
}

// NewConsumer создает новый Kafka consumer
func NewConsumer(config Config) (*Consumer, error) {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:     config.Brokers,
		Topic:       config.Topic,
		GroupID:     config.GroupID,
		ErrorLogger: kafka.LoggerFunc(zap.S().Errorf),
	})

	return &Consumer{reader: reader, topic: config.Topic}, nil
}

// ConsumeOrders читает заказы из Kafka и передает их в handler
func (c *Consumer) ConsumeOrders(ctx context.Context, handler func(*models.Order) error) error {
	zap.S().Infof("starting to consume orders from topic: %s", c.topic)

	for {
		select {
		case <-ctx.Done():
			zap.S().Info("consumer context cancelled, stopping...")
			return ctx.Err()
		default:
			// Используем таймаут для чтения сообщений
			readCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
			msg, err := c.reader.ReadMessage(readCtx)
			cancel()

			if err != nil {
				if errors.Is(err, context.DeadlineExceeded) {
					continue
				}
				if errors.Is(err, context.Canceled) {
					zap.S().Info("consumer context canceled, stopping...")
					return err
				}
				zap.S().Warnf("failed to read message: %v", err)
				continue
			}

			var order models.Order
			if err := json.Unmarshal(msg.Value, &order); err != nil {
				zap.S().Warnf("failed to unmarshal order: %v", err)
				continue
			}

			if err := handler(&order); err != nil {
				zap.S().Warnf("handler error: %v", err)
				continue
			}

			zap.S().Debugf("processed order: %s", order.OrderUID)
		}
	}
}

// ConsumeOrdersBatch читает заказы пакетами из Kafka
func (c *Consumer) ConsumeOrdersBatch(ctx context.Context, batchSize int, handler func([]*models.Order) error) error {
	zap.S().Infof("starting to consume orders in batches from topic: %s", c.topic)

	for {
		select {
		case <-ctx.Done():
			zap.S().Info("consumer context cancelled, stopping...")
			return ctx.Err()
		default:
			// Читаем пакет сообщений
			readCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
			msgs, err := c.reader.ReadMessage(readCtx)
			cancel()

			if err != nil {
				if err == context.DeadlineExceeded {
					// Это нормально - просто нет сообщений
					continue
				}
				if err == context.Canceled {
					zap.S().Info("consumer context canceled, stopping...")
					return err
				}
				zap.S().Warnf("failed to read message: %v", err)
				continue
			}

			orders := make([]*models.Order, 0, batchSize)
			var order models.Order
			if err := json.Unmarshal(msgs.Value, &order); err != nil {
				zap.S().Warnf("failed to unmarshal order: %v", err)
				continue
			}
			orders = append(orders, &order)

			// Пытаемся прочитать еще сообщения для пакета
			for len(orders) < batchSize {
				ctx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
				msg, err := c.reader.ReadMessage(ctx)
				cancel()

				if err != nil {
					break // Больше сообщений нет
				}

				var order models.Order
				if err := json.Unmarshal(msg.Value, &order); err != nil {
					zap.S().Warnf("failed to unmarshal order: %v", err)
					continue
				}
				orders = append(orders, &order)
			}

			if len(orders) > 0 {
				if err := handler(orders); err != nil {
					zap.S().Warnf("batch handler error: %v", err)
					continue
				}
				zap.S().Debugf("processed batch of %d orders", len(orders))
			}
		}
	}
}

// Close закрывает consumer
func (c *Consumer) Close() {
	zap.S().Info("closing kafka consumer...")
	if err := c.reader.Close(); err != nil {
		zap.S().Errorf("failed to close kafka reader: %v", err)
	}
	zap.S().Info("kafka consumer closed")
}
