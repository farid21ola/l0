# L0 - Система обработки заказов

Система для приема заказов через Kafka, сохранения в PostgreSQL и предоставления REST API для их получения.

## Структура проекта

```
L0/
├── cmd/
│   └── l0/
│       ├── server/
│           └── server.go          # Основное приложение (consumer + HTTP server)
│       └── producer/
│           └── producer.go  # Producer для отправки заказов в Kafka
├── internal/
│   ├── broker/              # Kafka producer/consumer
│   ├── models/              # Модели данных
│   ├── repository/          # Работа с БД
│   ├── service/             # Бизнес-логика
│   └── transport/           # HTTP сервер
├── config/                  # Конфигурация
├── migrations/              # Миграции БД
├── docker-compose.yml       # Docker контейнеры
└── pkg/                     # Общие пакеты
```

## Быстрый старт с Docker

### 1. Предварительные требования

- Docker и Docker Compose
- Go 1.23+

### 2. Настройка окружения

Создайте файл `.env` в корне проекта:

```env
# База данных PostgreSQL
DB_CONNECTION_STRING=postgres://l0_user:l0_password@localhost:5432/l0_db?sslmode=disable

# Kafka настройки
KAFKA_BROKERS=localhost:9092
KAFKA_TOPIC=orders
KAFKA_GROUPID=l0-consumer-group

# HTTP Server
HTTP_PORT=8081

# Логирование
LOG_LEVEL=info
```

### 3. Запуск инфраструктуры

```bash
# Запуск всех сервисов (Kafka, PostgreSQL, Kafka UI)
make run-all-docker

# Или по шагам:
make docker-up
make create-topic
```

### 4. Запуск приложений

В разных терминалах:

```bash
# Терминал 1: Producer (отправка заказов в Kafka)
make run-producer

# Терминал 2: Consumer + HTTP Server
make run-consumer
```

## Ручная установка (без Docker)

### 1. Установка зависимостей

```bash
go mod download
go mod tidy
```

### 2. Настройка PostgreSQL

1. Установите PostgreSQL
2. Создайте базу данных:
   ```sql
   CREATE DATABASE l0_db;
   CREATE USER l0_user WITH PASSWORD 'l0_password';
   GRANT ALL PRIVILEGES ON DATABASE l0_db TO l0_user;
   ```

3. Примените миграции:
   ```bash
   # Установите golang-migrate
   go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
   
   # Примените миграции
   migrate -path migrations -database "postgres://l0_user:l0_password@localhost:5432/l0_db?sslmode=disable" up
   ```

### 3. Настройка Kafka

1. Установите Apache Kafka
2. Запустите Zookeeper и Kafka
3. Создайте топик:
   ```bash
   kafka-topics --create --topic orders --bootstrap-server localhost:9092 --partitions 1 --replication-factor 1
   ```

## API Endpoints

### Получение заказа по ID

```http
GET /order/{order_uid}
```

**Пример ответа:**
```json
{
  "status": "ok",
  "data": {
    "order_uid": "test-order-123",
    "track_number": "WBILMTESTTRACK",
    "delivery": {
      "name": "Test Testov",
      "phone": "+9720000000",
      "zip": "2639809",
      "city": "Kiryat Mozkin",
      "address": "Ploshad Mira 15",
      "region": "Kraiot",
      "email": "test@gmail.com"
    },
    "payment": {
      "currency": "USD",
      "provider": "wbpay",
      "amount": 1817,
      "payment_dt": 1637907727,
      "bank": "alpha",
      "delivery_cost": 1500,
      "goods_total": 317,
      "custom_fee": 0
    },
    "items": [
      {
        "track_number": "WBILMTESTTRACK",
        "price": 453,
        "name": "Mascaras",
        "sale": 30,
        "size": "0",
        "total_price": 317,
        "brand": "Vivienne Sabo"
      }
    ],
    "locale": "en",
    "delivery_service": "meest",
    "date_created": "2021-11-26T06:22:19Z"
  }
}
```

## Мониторинг

### Kafka UI
- Доступен по адресу: http://localhost:8080
- Позволяет просматривать топики, сообщения и конфигурацию

### Логи
```bash
# Просмотр логов всех сервисов
make docker-logs

# Логи конкретного сервиса
docker-compose logs -f kafka
```

## Безопасность данных

API возвращает только безопасные данные пользователю, скрывая:
- Внутренние технические поля (`InternalSignature`, `Shardkey`, `SmID`, `OofShard`, `Entry`)
- Конфиденциальные данные (`CustomerID`, `Transaction`, `RequestID`)
- Внутренние ID товаров (`ChrtID`, `Rid`, `NmID`, `Status`)

## Тестирование

```bash
# Запуск всех тестов
make test

# Запуск тестов конкретного пакета
go test ./internal/service -v
```

## Разработка

```bash
# Форматирование кода
make fmt

# Проверка линтером
make lint

# Очистка
make clean

# Перезапуск Docker сервисов
make docker-restart
```

## Полезные команды

```bash
# Проверка статуса Docker контейнеров
docker-compose ps

# Остановка всех сервисов
make docker-down

# Создание топика Kafka
make create-topic

# Просмотр топиков
make list-topics

```
