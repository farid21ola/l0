package service

import (
	"context"
	"l0/internal/models"
	"l0/internal/repository"
	"sync"
)

type Service struct {
	repo  *repository.Repository
	cache map[string]*models.Order // [order_uid]Order
	mu    sync.RWMutex
}

func NewService(repo *repository.Repository) (*Service, error) {
	s := &Service{
		repo:  repo,
		cache: make(map[string]*models.Order),
	}
	if err := s.RestoreCache(context.Background()); err != nil {
		return nil, err
	}
	return s, nil
}

// RestoreCache загружает все заказы из БД в кеш
func (s *Service) RestoreCache(ctx context.Context) error {
	orders, err := s.repo.GetAllOrders(ctx)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, order := range orders {
		orderCopy := order
		s.cache[order.OrderUID] = &orderCopy
	}
	return nil
}

// GetOrder возвращает заказ по ID (сначала из кеша, если нет — из БД)
func (s *Service) GetOrder(ctx context.Context, orderUID string) (*models.Order, error) {
	s.mu.RLock()
	order, ok := s.cache[orderUID]
	s.mu.RUnlock()
	if ok {
		return order, nil
	}
	orderDB, err := s.repo.GetOrder(ctx, orderUID)
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	orderCopy := orderDB
	s.cache[orderUID] = &orderCopy
	s.mu.Unlock()
	return &orderCopy, nil
}

// GetOrderResponse возвращает безопасную версию заказа для пользователя
func (s *Service) GetOrderResponse(ctx context.Context, orderUID string) (*models.OrderResponse, error) {
	order, err := s.GetOrder(ctx, orderUID)
	if err != nil {
		return nil, err
	}
	return s.convertToOrderResponse(order), nil
}

// CreateOrder сохраняет заказ в БД и кеш
func (s *Service) CreateOrder(ctx context.Context, order *models.Order) error {
	if err := s.repo.CreateOrder(ctx, *order); err != nil {
		return err
	}
	s.mu.Lock()
	s.cache[order.OrderUID] = order
	s.mu.Unlock()
	return nil
}

// convertToOrderResponse конвертирует Order в OrderResponse
func (s *Service) convertToOrderResponse(order *models.Order) *models.OrderResponse {
	itemsResponse := make(models.ItemsResponse, len(order.Items))
	for i, item := range order.Items {
		itemsResponse[i] = models.ItemResponse{
			TrackNumber: item.TrackNumber,
			Price:       item.Price,
			Name:        item.Name,
			Sale:        item.Sale,
			Size:        item.Size,
			TotalPrice:  item.TotalPrice,
			Brand:       item.Brand,
		}
	}

	return &models.OrderResponse{
		OrderUID:    order.OrderUID,
		TrackNumber: order.TrackNumber,
		Delivery:    order.Delivery,
		Payment: models.PaymentResponse{
			Currency:     order.Payment.Currency,
			Provider:     order.Payment.Provider,
			Amount:       order.Payment.Amount,
			PaymentDt:    order.Payment.PaymentDt,
			Bank:         order.Payment.Bank,
			DeliveryCost: order.Payment.DeliveryCost,
			GoodsTotal:   order.Payment.GoodsTotal,
			CustomFee:    order.Payment.CustomFee,
		},
		Items:           itemsResponse,
		Locale:          order.Locale,
		DeliveryService: order.DeliveryService,
		DateCreated:     order.DateCreated,
	}
}
