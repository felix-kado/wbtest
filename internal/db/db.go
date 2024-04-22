package db

import (
	"context"
	"fmt"
	"log/slog"
	"wbstorage/internal/models"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

type Database interface {
	InsertOrder(ctx context.Context, order models.Order) error
	SelectOrder(ctx context.Context, orderUID string) (*models.Order, error)
	GetRecentOrders(ctx context.Context, n int) ([]string, error)
}
type Client struct {
	db *sqlx.DB
}

func NewDB(conntectionString string) (*Client, error) {
	db, err := sqlx.Connect("postgres", conntectionString)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}
	return &Client{db: db}, nil
}

func (c *Client) InsertOrder(ctx context.Context, order models.Order) error {

	orderQuery := `
	INSERT INTO orders 
		(order_uid, track_number, entry, locale, internal_signature, 
		customer_id, delivery_service, shardkey, sm_id, date_created, oof_shard, last_interaction)
		VALUES 
		(:order_uid, :track_number, :entry, :locale, :internal_signature, 
		:customer_id, :delivery_service, :shardkey, :sm_id, :date_created, :oof_shard, NOW())
	`

	deliveryQuery := `
	INSERT INTO deliveries 
		(order_uid, name, phone, zip, city, address, region, email) 
		VALUES 
		(:order_uid, :name, :phone, :zip, :city, :address, :region, :email)
	`

	paymentQuery := `
	INSERT INTO payments
		(order_uid, transaction, request_id, currency, provider, 
		amount, payment_dt, bank, delivery_cost, goods_total, custom_fee) 
		VALUES 
		(:order_uid, :transaction, :request_id, :currency, :provider, 
		:amount, to_timestamp(:payment_dt), :bank, :delivery_cost, :goods_total, :custom_fee)
	`
	ItemsQuery := `INSERT INTO items
		(order_uid, chrt_id, track_number, price, rid, name, sale, size, total_price, nm_id, brand, status)
		VALUES (:order_uid, :chrt_id, :track_number, :price, :rid, :name, :sale, :size, :total_price, :nm_id, :brand, :status)`

	tx, err := c.db.BeginTxx(ctx, nil)
	if err != nil {
		errRollback := tx.Rollback()
		if errRollback != nil {
			slog.Error("error while rollback", errRollback)
		}
	}

	if _, err := tx.NamedExecContext(ctx, orderQuery, order); err != nil {
		errRollback := tx.Rollback()
		if errRollback != nil {
			slog.Error("error while rollback", "error", errRollback)
		}
		return err
	}
	if _, err := tx.NamedExecContext(ctx, deliveryQuery, order.Delivery); err != nil {
		errRollback := tx.Rollback()
		if errRollback != nil {
			slog.Error("error while rollback", "error", errRollback)
		}
		return err
	}
	if _, err := tx.NamedExecContext(ctx, paymentQuery, order.Payment); err != nil {
		errRollback := tx.Rollback()
		if errRollback != nil {
			slog.Error("error while rollback", "error", errRollback)
		}
		return err
	}
	if _, err = tx.NamedExecContext(ctx, ItemsQuery, order.Items); err != nil {
		errRollback := tx.Rollback()
		if errRollback != nil {
			slog.Error("error while rollback", "error", errRollback)
		}
		return err
	}

	return tx.Commit()
}

func (c *Client) SelectOrder(ctx context.Context, orderUID string) (*models.Order, error) {
	updateQuery := `
	UPDATE orders
	SET last_interaction = NOW()
	WHERE order_uid = $1
	`
	if _, err := c.db.ExecContext(ctx, updateQuery, orderUID); err != nil {
		return nil, fmt.Errorf("error updating last interaction time: %v", err)
	}

	order := models.Order{}

	orderQuery := `
	SELECT * 
	FROM orders 
	WHERE order_uid = $1
	`

	deliveryQuery := `
	SELECT * 
	FROM deliveries 
	WHERE order_uid = $1
	`

	paymentQuery := `
	SELECT order_uid, transaction, request_id, currency, provider, 
		   amount, FLOOR(EXTRACT(EPOCH FROM payment_dt))::BIGINT AS payment_dt, 
		   bank, delivery_cost, goods_total, custom_fee 
	FROM payments
	WHERE order_uid = $1
	`

	itemsQuery := `
	SELECT * 
	FROM items 
	WHERE order_uid = $1
	`

	if err := c.db.GetContext(ctx, &order, orderQuery, orderUID); err != nil {
		return nil, fmt.Errorf("error fetching order: %v", err)
	}
	if err := c.db.GetContext(ctx, &order.Delivery, deliveryQuery, orderUID); err != nil {
		return nil, fmt.Errorf("error fetching delivery data: %v", err)
	}
	if err := c.db.GetContext(ctx, &order.Payment, paymentQuery, orderUID); err != nil {
		return nil, fmt.Errorf("error fetching payment data: %v", err)
	}
	if err := c.db.SelectContext(ctx, &order.Items, itemsQuery, orderUID); err != nil {
		return nil, fmt.Errorf("error fetching items: %v", err)
	}

	return &order, nil
}

func (c *Client) GetRecentOrders(ctx context.Context, n int) ([]string, error) {
	var orderUIDs []string
	query := `
	SELECT order_uid 
	FROM orders 
	ORDER BY last_interaction DESC 
	LIMIT $1
	`

	err := c.db.SelectContext(ctx, &orderUIDs, query, n)
	if err != nil {
		return nil, fmt.Errorf("error fetching recent orders: %v", err)
	}
	return orderUIDs, nil
}
