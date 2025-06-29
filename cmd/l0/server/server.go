package main

import (
	"context"
	"errors"
	"l0/config"
	"l0/internal/broker"
	"l0/internal/models"
	"l0/internal/repository"
	"l0/internal/service"
	"l0/internal/transport/rest"
	"l0/pkg/logger"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"go.uber.org/zap"
)

func main() {
	// Читаем конфиг
	cfg, err := config.NewConfig()
	if err != nil {
		log.Fatalf("failed to initialize config: %v", err)
	}

	// Создаем логирование
	logger.SetupLogger(cfg.LoggerConfig)
	zap.S().Info("logger initialized")

	// Создаем репозиторий для работы с БД
	repo, err := repository.NewRepository(cfg.DbConfig)
	if err != nil {
		zap.S().Errorf("failed to initialize repository: %v", err)
		panic(err)
	}
	zap.S().Info("repository initialized")

	// Инициализируем сервис
	svc, err := service.NewService(repo)
	if err != nil {
		zap.S().Errorf("failed to initialize service: %v", err)
	}
	zap.S().Info("service initialized")

	// Создаем kafka consumer
	consumer, err := broker.NewConsumer(cfg.KafkaConfig)
	if err != nil {
		zap.S().Fatalf("failed to create kafka consumer: %v", err)
	}
	defer consumer.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Создаем HTTP сервер
	httpHandlers := rest.NewHandler(svc)
	server := rest.CreateServer(cfg.ServerConfig, httpHandlers)

	var wg sync.WaitGroup

	// graceful shutdown
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		zap.S().Info("received shutdown signal, starting graceful shutdown...")

		// Отменяем контекст для остановки consumer'а
		cancel()

		// Graceful shutdown HTTP сервера
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer shutdownCancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			zap.S().Errorf("HTTP server shutdown error: %v", err)
		} else {
			zap.S().Info("HTTP server shutdown completed")
		}
	}()

	// Запускаем consumer в отдельной горутине
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := consumer.ConsumeOrders(ctx, func(order *models.Order) error {
			// Сохраняем заказ через сервис
			if err := svc.CreateOrder(ctx, order); err != nil {
				zap.S().Errorf("failed to save order from kafka: %v", err)
				return err
			}
			zap.S().Infof("order saved from kafka: %s", order.OrderUID)
			return nil
		})
		if err != nil && !errors.Is(err, context.Canceled) {
			zap.S().Errorf("kafka consumer stopped with error: %v", err)
		}
	}()

	// Запускаем HTTP сервер
	zap.S().Infof("starting HTTP server on %s", cfg.ServerConfig.Port)
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			zap.S().Errorf("HTTP server error: %v", err)
		}
	}()

	// Ждем завершения всех горутин
	wg.Wait()
	zap.S().Info("application shutdown completed")
}
