package db

import (
	"sync"
	"wbstorage/internal/models"
)

type OrderDB interface {
	InsertOrder(order models.Order) error
	SelectOrder(orderUID string) (*models.Order, error)
	CreateTables() error
}

func NewCachedClient(db OrderDB) *CachedClient {
	return &CachedClient{
		db:    db,
		cache: make(map[string]*models.Order),
	}
}

type CachedClient struct {
	mu    sync.Mutex
	cache map[string]*models.Order
	db    OrderDB
}

func (c *CachedClient) GetOrderFromCache(orderUID string) (*models.Order, error) {
	c.mu.Lock()
	order, ok := c.cache[orderUID]
	c.mu.Unlock()

	if ok {
		return order, nil
	}

	order, err := c.db.SelectOrder(orderUID)
	if err != nil {
		return nil, err
	}

	c.mu.Lock()
	c.cache[orderUID] = order
	c.mu.Unlock()

	return order, nil
}

func (c *CachedClient) PutOrderIntoDbAndCache(order models.Order) error {

	err := c.db.InsertOrder(order)
	if err != nil {
		return err
	}

	_, err = c.GetOrderFromCache(order.OrderUID)
	return err
}
