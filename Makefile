.PHONY: build build-producer run-producer run-consumer clean docker-up docker-down docker-logs

# Сборка основного приложения
build:
	go build -o bin/l0 ./cmd/L0/server/server.go

# Сборка producer'а
build-producer:
	go build -o bin/producer ./cmd/L0/producer

# Запуск producer'а
run-producer: build-producer
	./bin/producer

# Запуск основного приложения (consumer + HTTP server)
run-consumer: build
	./bin/l0

# Очистка
clean:
	rm -rf bin/

# Установка зависимостей
deps:
	go mod download
	go mod tidy

# Docker команды
docker-up:
	docker-compose up -d

docker-down:
	docker-compose down

docker-logs:
	docker-compose logs -f

docker-restart:
	docker-compose restart

# Создание топика Kafka
create-topic:
	docker exec -it kafka kafka-topics --create --topic orders --bootstrap-server localhost:9092 --partitions 1 --replication-factor 1

# Проверка топиков Kafka
list-topics:
	docker exec -it kafka kafka-topics --list --bootstrap-server localhost:9092

# Полный запуск с Docker
run-all-docker: docker-up
	@echo "Ожидание запуска сервисов..."
	@sleep 10
	@echo "Создание топика Kafka..."
	@make create-topic || true
	@echo "Сборка приложений..."
	@make build build-producer
	@echo "Готово! Теперь запустите:"
	@echo "1. Producer: make run-producer"
	@echo "2. Consumer: make run-consumer"

# Запуск всех сервисов (в разных терминалах)
run-all: build build-producer
	@echo "Запуск всех сервисов..."
	@echo "1. Запустите producer: make run-producer"
	@echo "2. Запустите consumer: make run-consumer" 