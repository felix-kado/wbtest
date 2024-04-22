package db

import (
	"context"
	"fmt"
	"sync"
	"wbstorage/internal/models"
)

type CachedClient struct {
	mu    sync.Mutex
	cache map[string]*models.Order
	db    Database
}

func NewCachedClient(ctx context.Context, db Database, n int) (*CachedClient, error) {
	client := &CachedClient{
		db:    db,
		cache: make(map[string]*models.Order),
	}

	if err := client.cacheWarming(ctx, n); err != nil {
		return client, fmt.Errorf("failed to warmup cache: %w", err)
	}
	return client, nil
}

func (c *CachedClient) InsertOrder(ctx context.Context, order models.Order) error {
	err := c.db.InsertOrder(ctx, order)
	if err != nil {
		return err
	}

	c.mu.Lock()
	c.cache[order.OrderUID] = &order
	c.mu.Unlock()

	return nil
}

func (c *CachedClient) SelectOrder(ctx context.Context, orderUID string) (*models.Order, error) {
	c.mu.Lock()
	if order, found := c.cache[orderUID]; found {
		c.mu.Unlock()
		return order, nil
	}
	c.mu.Unlock()

	order, err := c.db.SelectOrder(ctx, orderUID)
	if err != nil {
		return nil, err
	}

	c.mu.Lock()
	c.cache[orderUID] = order
	c.mu.Unlock()

	return order, nil
}

func (c *CachedClient) InsertCacheInfo(ctx context.Context, orderUID string) error {
	return c.db.InsertCacheInfo(ctx, orderUID)
}

func (c *CachedClient) UpdateCacheLoadDate(ctx context.Context, orderUID string) error {
	return c.db.UpdateCacheLoadDate(ctx, orderUID)
}

func (c *CachedClient) GetRecentOrders(ctx context.Context, n int) ([]string, error) {
	return c.db.GetRecentOrders(ctx, n)
}

func (c *CachedClient) cacheWarming(ctx context.Context, n int) error {
	UIDsList, err := c.GetRecentOrders(ctx, n)
	if err != nil {
		return err
	}
	for _, orderUID := range UIDsList {
		if _, err := c.SelectOrder(ctx, orderUID); err != nil {
			return err
		}
		if err := c.UpdateCacheLoadDate(ctx, orderUID); err != nil {
			return err
		}
	}
	return nil
}
