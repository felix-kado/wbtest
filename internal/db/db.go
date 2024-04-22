package db

import (
	"context"
	"fmt"
	"strings"
	"time"
	"wbstorage/internal/models"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

type Database interface {
	InsertOrder(ctx context.Context, order models.Order) error
	SelectOrder(ctx context.Context, orderUID string) (*models.Order, error)
	InsertCacheInfo(ctx context.Context, orderUID string) error
	UpdateCacheLoadDate(ctx context.Context, orderUID string) error
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
	ctx, cancel := context.WithTimeout(ctx, time.Second*15)
	defer cancel()

	orderQuery := `
	INSERT INTO orders 
		(order_uid, track_number, entry, locale, internal_signature, 
		customer_id, delivery_service, shardkey, sm_id, date_created, oof_shard)
		VALUES 
		(:order_uid, :track_number, :entry, :locale, :internal_signature, 
		:customer_id, :delivery_service, :shardkey, :sm_id, :date_created, :oof_shard)
	`

	deliveryQuery := `
	INSERT INTO delivery 
		(order_uid, name, phone, zip, city, address, region, email) 
		VALUES 
		(:order_uid, :name, :phone, :zip, :city, :address, :region, :email)
	`

	paymentQuery := `
	INSERT INTO payment 
		(order_uid, transaction, request_id, currency, provider, 
		amount, payment_dt, bank, delivery_cost, goods_total, custom_fee) 
		VALUES 
		(:order_uid, :transaction, :request_id, :currency, :provider, 
		:amount, to_timestamp(:payment_dt), :bank, :delivery_cost, :goods_total, :custom_fee)
	`
	itemsQuery := `
	INSERT INTO items
		(order_uid, chrt_id, track_number,
		price, rid, name, sale, size, total_price,
		nm_id, brand, status)
	VALUES %s
	`

	valueStrings := make([]string, 0, len(order.Items))
	valueArgs := make([]interface{}, 0, len(order.Items)*12) // 12 полей в Item

	for i, item := range order.Items {
		num := fmt.Sprintf("($%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d)", i*12+1, i*12+2, i*12+3, i*12+4, i*12+5, i*12+6, i*12+7, i*12+8, i*12+9, i*12+10, i*12+11, i*12+12)
		valueStrings = append(valueStrings, num)
		valueArgs = append(valueArgs, item.OrderUID, item.ChrtID, item.TrackNumber, item.Price, item.RID, item.Name, item.Sale, item.Size, item.TotalPrice, item.NmID, item.Brand, item.Status)
	}
	stmt := fmt.Sprintf(itemsQuery, strings.Join(valueStrings, ","))

	tx, err := c.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() // Ensure rollback is called on error.

	// NamedExecContext is used to ensure the context is respected.
	if _, err := tx.NamedExecContext(ctx, orderQuery, order); err != nil {
		return err
	}
	if _, err := tx.NamedExecContext(ctx, deliveryQuery, order.Delivery); err != nil {
		return err
	}
	if _, err := tx.NamedExecContext(ctx, paymentQuery, order.Payment); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, stmt, valueArgs...); err != nil {
		return err
	}

	return tx.Commit()
}

func (c *Client) SelectOrder(ctx context.Context, orderUID string) (*models.Order, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Second*15)
	defer cancel()

	order := models.Order{}

	orderQuery := `
	SELECT * 
	FROM orders 
	WHERE order_uid = $1
	`

	deliveryQuery := `
	SELECT * 
	FROM delivery 
	WHERE order_uid = $1
	`

	paymentQuery := `
	SELECT order_uid, transaction, request_id, currency, provider, 
		   amount, FLOOR(EXTRACT(EPOCH FROM payment_dt))::BIGINT AS payment_dt, 
		   bank, delivery_cost, goods_total, custom_fee 
	FROM payment 
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

func (c *Client) InsertCacheInfo(ctx context.Context, orderUID string) error {
	ctx, cancel := context.WithTimeout(ctx, time.Second*15)
	defer cancel()

	query := `
	INSERT INTO cache_info 
		(order_uid, creation_date, last_cache_load_date) 
	VALUES 
		($1, NOW(), NOW())
	`
	_, err := c.db.ExecContext(ctx, query, orderUID)
	if err != nil {
		return err
	}
	return nil
}

func (c *Client) UpdateCacheLoadDate(ctx context.Context, orderUID string) error {
	ctx, cancel := context.WithTimeout(ctx, time.Second*15)
	defer cancel()

	query := `
	UPDATE cache_info 
	SET last_cache_load_date = NOW() 
	WHERE order_uid = $1
	`
	_, err := c.db.ExecContext(ctx, query, orderUID)
	if err != nil {
		return err
	}
	return nil
}

func (c *Client) GetRecentOrders(ctx context.Context, n int) ([]string, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Second*15)
	defer cancel()

	var orderUIDs []string
	query := `
	SELECT order_uid 
	FROM cache_info 
	ORDER BY last_cache_load_date DESC 
	LIMIT $1
	`
	err := c.db.SelectContext(ctx, &orderUIDs, query, n)
	if err != nil {
		return nil, err
	}
	return orderUIDs, nil
}
