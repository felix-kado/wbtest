package db

import (
	"context"
	"fmt"
	"log/slog" // Ensure you import the slog package
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
		slog.Error("Failed to warm up cache", "error", err)
		return client, fmt.Errorf("failed to warmup cache: %w", err)
	}
	slog.Info("Cache warmed up successfully")
	return client, nil
}

func (c *CachedClient) InsertOrder(ctx context.Context, order models.Order) error {
	err := c.db.InsertOrder(ctx, order)
	if err != nil {
		slog.Error("Failed to insert order into database", "error", err)
		return err
	}

	c.mu.Lock()
	c.cache[order.OrderUID] = &order
	c.mu.Unlock()
	slog.Info("Order cached successfully", "orderUID", order.OrderUID)

	return nil
}

func (c *CachedClient) SelectOrder(ctx context.Context, orderUID string) (*models.Order, error) {
	c.mu.Lock()
	if order, found := c.cache[orderUID]; found {
		c.mu.Unlock()
		slog.Info("Order retrieved from cache", "orderUID", orderUID)
		return order, nil
	}
	c.mu.Unlock()

	order, err := c.db.SelectOrder(ctx, orderUID)
	if err != nil {
		slog.Error("Failed to select order from database", "error", err)
		return nil, err
	}

	c.mu.Lock()
	c.cache[orderUID] = order
	c.mu.Unlock()
	slog.Info("Order cached after database retrieval", "orderUID", orderUID)

	return order, nil
}

func (c *CachedClient) GetRecentOrders(ctx context.Context, n int) ([]string, error) {
	return c.db.GetRecentOrders(ctx, n)
}

func (c *CachedClient) cacheWarming(ctx context.Context, n int) error {
	UIDsList, err := c.GetRecentOrders(ctx, n)
	if err != nil {
		slog.Error("Failed to retrieve recent orders for cache warming", "error", err)
		return err
	}
	for _, orderUID := range UIDsList {
		if _, err := c.SelectOrder(ctx, orderUID); err != nil {
			slog.Error("Failed to select order during cache warming", "error", err)
			return err
		}
	}
	return nil
}
