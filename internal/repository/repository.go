package repository

import (
	"context"
	"l0/internal/models"
	"l0/internal/repository/db/postgres"
)

type Config struct {
	ConnectionString string `env:"DB_CONNECTION_STRING,required"`
}

type Repository struct {
	db *postgres.Postgres
}

func NewRepository(cfg Config) (*Repository, error) {
	db, err := postgres.NewPostgres(cfg.ConnectionString)
	if err != nil {
		return nil, err
	}
	repo := &Repository{db: db}
	return repo, nil
}

func (r *Repository) GetOrder(ctx context.Context, orderUID string) (models.Order, error) {
	return r.db.GetOrder(ctx, orderUID)
}

func (r *Repository) CreateOrder(ctx context.Context, order models.Order) error {
	return r.db.CreateOrder(ctx, order)
}

func (r *Repository) GetAllOrders(ctx context.Context) ([]models.Order, error) {
	return r.db.GetOrders(ctx)
}
