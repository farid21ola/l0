package postgres

import (
	"context"
	"errors"
	"fmt"
	"l0/internal/models"
	"l0/pkg/er"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

// Postgres - структура для работы с БД
type Postgres struct {
	pool *pgxpool.Pool
}

// checkPostgresError проверяет специфичные ошибки PostgreSQL
func checkPostgresError(err error) error {
	if err == nil {
		return nil
	}

	// Проверка на "no rows found"
	if errors.Is(err, pgx.ErrNoRows) {
		return er.ErrOrderNotFound
	}

	// Проверка специфичных ошибок PostgreSQL по кодам
	// https://www.postgresql.org/docs/11/errcodes-appendix.html
	var pgerr *pgconn.PgError
	if errors.As(err, &pgerr) {
		switch pgerr.Code {
		case "23505":
			return fmt.Errorf("%w: violation of uniqueness", er.ErrOrderExists)
		case "23503":
			return fmt.Errorf("%w: foreign key violation", er.ErrInvalidData)
		case "23502":
			return fmt.Errorf("%w: required field not filled", er.ErrInvalidData)
		case "23514":
			return fmt.Errorf("%w: check constraint violation", er.ErrInvalidData)
		case "42P01":
			return fmt.Errorf("%w: table does not exist", er.ErrDatabaseError)
		case "42703":
			return fmt.Errorf("%w: column does not exist", er.ErrDatabaseError)
		case "22001":
			return fmt.Errorf("%w: string data too long", er.ErrInvalidData)
		case "22003":
			return fmt.Errorf("%w: numeric value out of range", er.ErrInvalidData)
		case "22008":
			return fmt.Errorf("%w: datetime field overflow", er.ErrInvalidData)
		default:
			return fmt.Errorf("%w: %s", er.ErrDatabaseError, pgerr.Message)
		}
	}

	return fmt.Errorf("%w: %v", er.ErrDatabaseError, err)
}

func NewPostgres(connString string) (*Postgres, error) {
	pool, err := pgxpool.New(context.Background(), connString)
	if err != nil {
		return nil, fmt.Errorf("failed to create pgx pool: %w", err)
	}
	return &Postgres{pool: pool}, nil
}

// Публичные CRUD для Order
func (p *Postgres) CreateOrder(ctx context.Context, order models.Order) error {
	if order.OrderUID == "" {
		return fmt.Errorf("%w: order_uid cannot be empty", er.ErrInvalidData)
	}

	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", checkPostgresError(err))
	}

	defer func() {
		if err != nil {
			if rbErr := tx.Rollback(ctx); rbErr != nil {
				zap.S().Errorf("rollback error: %v\n", rbErr)
			}
		}
	}()

	deliveryID, err := p.createDeliveryTx(ctx, tx, order.Delivery)
	if err != nil {
		return fmt.Errorf("delivery creation error: %w", checkPostgresError(err))
	}

	paymentID, err := p.createPaymentTx(ctx, tx, order.Payment)
	if err != nil {
		return fmt.Errorf("payment creation error: %w", checkPostgresError(err))
	}

	query := `INSERT INTO orders (
		order_uid, track_number, entry, delivery_id, payment_id, locale, internal_signature, customer_id, delivery_service, shardkey, sm_id, date_created, oof_shard
	) VALUES (
		$1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13
	)`

	_, err = tx.Exec(ctx, query,
		order.OrderUID, order.TrackNumber, order.Entry, deliveryID, paymentID, order.Locale, order.InternalSignature, order.CustomerID, order.DeliveryService, order.Shardkey, order.SmID, order.DateCreated, order.OofShard,
	)
	if err != nil {
		return fmt.Errorf("order creation error: %w", checkPostgresError(err))
	}

	for i, item := range order.Items {
		_, err := p.createItemTx(ctx, tx, item, order.OrderUID)
		if err != nil {
			return fmt.Errorf("item %d creation error: %w", i+1, checkPostgresError(err))
		}
	}

	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", checkPostgresError(err))
	}

	return nil
}

func (p *Postgres) GetOrder(ctx context.Context, orderUID string) (models.Order, error) {
	var order models.Order
	var deliveryID, paymentID int

	// Валидация входных данных
	if orderUID == "" {
		return order, fmt.Errorf("%w: order_uid cannot be empty", er.ErrInvalidData)
	}

	query := `SELECT order_uid, track_number, entry, delivery_id, payment_id, locale, internal_signature, customer_id, delivery_service, shardkey, sm_id, date_created, oof_shard FROM orders WHERE order_uid = $1`
	err := p.pool.QueryRow(ctx, query, orderUID).
		Scan(&order.OrderUID, &order.TrackNumber, &order.Entry, &deliveryID, &paymentID, &order.Locale, &order.InternalSignature, &order.CustomerID, &order.DeliveryService, &order.Shardkey, &order.SmID, &order.DateCreated, &order.OofShard)
	if err != nil {
		return order, fmt.Errorf("order retrieval error: %w", checkPostgresError(err))
	}

	order.Delivery, err = p.getDelivery(ctx, deliveryID)
	if err != nil {
		return order, fmt.Errorf("delivery data retrieval error: %w", checkPostgresError(err))
	}

	order.Payment, err = p.getPayment(ctx, paymentID)
	if err != nil {
		return order, fmt.Errorf("payment data retrieval error: %w", checkPostgresError(err))
	}

	order.Items, err = p.getItemsByOrderUID(ctx, orderUID)
	if err != nil {
		return order, fmt.Errorf("order items retrieval error: %w", checkPostgresError(err))
	}

	return order, nil
}

func (p *Postgres) GetOrders(ctx context.Context) ([]models.Order, error) {
	query := `SELECT order_uid, track_number, entry, delivery_id, payment_id, locale, internal_signature, customer_id, delivery_service, shardkey, sm_id, date_created, oof_shard FROM orders`
	rows, err := p.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("order query error: %w", checkPostgresError(err))
	}
	defer rows.Close()

	var orders []models.Order
	for rows.Next() {
		var order models.Order
		var deliveryID, paymentID int
		err := rows.Scan(&order.OrderUID, &order.TrackNumber, &order.Entry, &deliveryID, &paymentID, &order.Locale, &order.InternalSignature, &order.CustomerID, &order.DeliveryService, &order.Shardkey, &order.SmID, &order.DateCreated, &order.OofShard)
		if err != nil {
			return nil, fmt.Errorf("order scanning error: %w", checkPostgresError(err))
		}

		order.Delivery, err = p.getDelivery(ctx, deliveryID)
		if err != nil {
			return nil, fmt.Errorf("delivery data retrieval error for order %s: %w", order.OrderUID, checkPostgresError(err))
		}

		order.Payment, err = p.getPayment(ctx, paymentID)
		if err != nil {
			return nil, fmt.Errorf("payment data retrieval error for order %s: %w", order.OrderUID, checkPostgresError(err))
		}

		order.Items, err = p.getItemsByOrderUID(ctx, order.OrderUID)
		if err != nil {
			return nil, fmt.Errorf("items retrieval error for order %s: %w", order.OrderUID, checkPostgresError(err))
		}

		orders = append(orders, order)
	}

	// Проверка на ошибки итерации
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("order iteration error: %w", checkPostgresError(err))
	}

	return orders, nil
}

// Методы для работы с транзакциями
func (p *Postgres) createDeliveryTx(ctx context.Context, tx pgx.Tx, d models.Delivery) (int, error) {
	// Валидация данных доставки
	if d.Name == "" {
		return 0, fmt.Errorf("%w: recipient name is required", er.ErrInvalidData)
	}
	if d.Phone == "" {
		return 0, fmt.Errorf("%w: recipient phone is required", er.ErrInvalidData)
	}

	query := `INSERT INTO delivery (name, phone, zip, city, address, region, email) VALUES ($1,$2,$3,$4,$5,$6,$7) RETURNING id`
	var id int
	err := tx.QueryRow(ctx, query,
		d.Name, d.Phone, d.Zip, d.City, d.Address, d.Region, d.Email).Scan(&id)
	if err != nil {
		return 0, checkPostgresError(err)
	}
	return id, nil
}

func (p *Postgres) getDelivery(ctx context.Context, id int) (models.Delivery, error) {
	if id <= 0 {
		return models.Delivery{}, fmt.Errorf("%w: invalid delivery ID", er.ErrInvalidData)
	}

	query := `SELECT name, phone, zip, city, address, region, email FROM delivery WHERE id=$1`
	var d models.Delivery
	err := p.pool.QueryRow(ctx, query, id).
		Scan(&d.Name, &d.Phone, &d.Zip, &d.City, &d.Address, &d.Region, &d.Email)
	if err != nil {
		return d, checkPostgresError(err)
	}
	return d, nil
}

func (p *Postgres) createPaymentTx(ctx context.Context, tx pgx.Tx, pay models.Payment) (int, error) {
	// Валидация данных платежа
	if pay.Transaction == "" {
		return 0, fmt.Errorf("%w: transaction number is required", er.ErrInvalidData)
	}
	if pay.Provider == "" {
		return 0, fmt.Errorf("%w: payment provider is required", er.ErrInvalidData)
	}
	if pay.Amount <= 0 {
		return 0, fmt.Errorf("%w: payment amount must be greater than zero", er.ErrInvalidData)
	}

	query := `INSERT INTO payment (transaction, request_id, currency, provider, amount, payment_dt, bank, delivery_cost, goods_total, custom_fee) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10) RETURNING id`
	var id int
	err := tx.QueryRow(ctx, query,
		pay.Transaction, pay.RequestID, pay.Currency, pay.Provider, pay.Amount, pay.PaymentDt, pay.Bank, pay.DeliveryCost, pay.GoodsTotal, pay.CustomFee).Scan(&id)
	if err != nil {
		return 0, checkPostgresError(err)
	}
	return id, nil
}

func (p *Postgres) getPayment(ctx context.Context, id int) (models.Payment, error) {
	if id <= 0 {
		return models.Payment{}, fmt.Errorf("%w: invalid payment ID", er.ErrInvalidData)
	}

	query := `SELECT transaction, request_id, currency, provider, amount, payment_dt, bank, delivery_cost, goods_total, custom_fee FROM payment WHERE id=$1`
	var pay models.Payment
	err := p.pool.QueryRow(ctx, query, id).
		Scan(&pay.Transaction, &pay.RequestID, &pay.Currency, &pay.Provider, &pay.Amount, &pay.PaymentDt, &pay.Bank, &pay.DeliveryCost, &pay.GoodsTotal, &pay.CustomFee)
	if err != nil {
		return pay, checkPostgresError(err)
	}
	return pay, nil
}

func (p *Postgres) getItemsByOrderUID(ctx context.Context, orderUID string) ([]models.Item, error) {
	if orderUID == "" {
		return nil, fmt.Errorf("%w: order_uid cannot be empty", er.ErrInvalidData)
	}

	query := `SELECT chrt_id, track_number, price, rid, name, sale, size, total_price, nm_id, brand, status FROM item WHERE order_uid=$1`
	rows, err := p.pool.Query(ctx, query, orderUID)
	if err != nil {
		return nil, fmt.Errorf("item query error: %w", checkPostgresError(err))
	}
	defer rows.Close()

	var items []models.Item
	for rows.Next() {
		var item models.Item
		err := rows.Scan(&item.ChrtID, &item.TrackNumber, &item.Price, &item.Rid, &item.Name, &item.Sale, &item.Size, &item.TotalPrice, &item.NmID, &item.Brand, &item.Status)
		if err != nil {
			return nil, fmt.Errorf("item scanning error: %w", checkPostgresError(err))
		}
		items = append(items, item)
	}

	// Проверка на ошибки итерации
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("item iteration error: %w", checkPostgresError(err))
	}

	return items, nil
}

func (p *Postgres) createItemTx(ctx context.Context, tx pgx.Tx, item models.Item, orderUID string) (int, error) {
	// Валидация данных товара
	if item.Name == "" {
		return 0, fmt.Errorf("%w: item name is required", er.ErrInvalidData)
	}
	if item.Price <= 0 {
		return 0, fmt.Errorf("%w: item price must be greater than zero", er.ErrInvalidData)
	}
	if orderUID == "" {
		return 0, fmt.Errorf("%w: item order_uid is required", er.ErrInvalidData)
	}

	query := `INSERT INTO item (chrt_id, track_number, price, rid, name, sale, size, total_price, nm_id, brand, status, order_uid) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12) RETURNING id`
	var id int
	err := tx.QueryRow(ctx, query,
		item.ChrtID, item.TrackNumber, item.Price, item.Rid, item.Name, item.Sale, item.Size, item.TotalPrice, item.NmID, item.Brand, item.Status, orderUID).Scan(&id)
	if err != nil {
		return 0, checkPostgresError(err)
	}
	return id, nil
}
