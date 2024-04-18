package db

import (
	"fmt"
	"log"
	"wbstorage/internal/models"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

type DataBase struct {
	Db *sqlx.DB
}

func NewDB(dataSourceName string) (*DataBase, error) {
	db, err := sqlx.Connect("postgres", dataSourceName)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}
	return &DataBase{Db: db}, nil
}

func (base *DataBase) InsertOrder(order models.Order) error {

	orderQuery := `INSERT INTO orders (order_uid, track_number, entry, locale, internal_signature, customer_id, delivery_service, shardkey, sm_id, date_created, oof_shard)
	VALUES (:order_uid, :track_number, :entry, :locale, :internal_signature, :customer_id, :delivery_service, :shardkey, :sm_id, :date_created, :oof_shard)`
	deliveryQuery := `INSERT INTO delivery (order_uid, name, phone, zip, city, address, region, email) VALUES (:order_uid, :name, :phone, :zip, :city, :address, :region, :email)`
	paymentQuery := `INSERT INTO payment (order_uid, transaction, request_id, currency, provider, amount, payment_dt, bank, delivery_cost, goods_total, custom_fee) 
	VALUES (:order_uid, :transaction, :request_id, :currency, :provider, :amount, to_timestamp(:payment_dt), :bank, :delivery_cost, :goods_total, :custom_fee)`
	itemQuery := `INSERT INTO items (order_uid, chrt_id, track_number, price, rid, name, sale, size, total_price, nm_id, brand, status) 
	VALUES (:order_uid, :chrt_id, :track_number, :price, :rid, :name, :sale, :size, :total_price, :nm_id, :brand, :status)`

	tx, err := base.Db.Beginx()
	if err != nil {
		return err
	}

	if _, err := tx.NamedExec(orderQuery, order); err != nil {
		tx.Rollback()
		return err
	} else {
		fmt.Print("Order, ")
	}

	if _, err := tx.NamedExec(deliveryQuery, order.Delivery); err != nil {
		tx.Rollback()
		return err
	} else {
		fmt.Print("Delivery, ")
	}

	if _, err := tx.NamedExec(paymentQuery, order.Payment); err != nil {
		tx.Rollback()
		return err
	} else {
		fmt.Print("Payment, ")
	}

	nRows := 0
	for _, item := range order.Items {
		if _, err := tx.NamedExec(itemQuery, item); err != nil {
			tx.Rollback()
			return err
		} else {
			nRows++
		}
	}
	fmt.Print(nRows, " Items\n")

	return tx.Commit()
}

func (db *DataBase) SelectOrder(orderUID string) (*models.Order, error) {
	order := models.Order{}

	orderQuery := `SELECT * FROM orders WHERE order_uid = $1`
	deliveryQuery := `SELECT * FROM delivery WHERE order_uid = $1`
	paymentQuery := `SELECT order_uid, transaction, request_id, currency, provider, amount, FLOOR(EXTRACT(EPOCH FROM payment_dt))::BIGINT as payment_dt, bank, delivery_cost, goods_total, custom_fee 
FROM payment WHERE order_uid = $1`
	itemsQuery := `SELECT * FROM items WHERE order_uid = $1`

	err := db.Db.Get(&order, orderQuery, orderUID)
	if err != nil {
		return nil, fmt.Errorf("error fetching order: %v", err)
	}

	err = db.Db.Get(&order.Delivery, deliveryQuery, orderUID)
	if err != nil {
		return nil, fmt.Errorf("error fetching delivery data: %v", err)
	}

	err = db.Db.Get(&order.Payment, paymentQuery, orderUID)
	if err != nil {
		return nil, fmt.Errorf("error fetching payment data: %v", err)
	}

	err = db.Db.Select(&order.Items, itemsQuery, orderUID)
	if err != nil {
		return nil, fmt.Errorf("error fetching items: %v", err)
	}

	return &order, nil
}

func (db *DataBase) CreateTables() error {
	createTablesQuery := `CREATE TABLE IF NOT EXISTS orders (
		order_uid VARCHAR PRIMARY KEY,
		track_number VARCHAR,
		entry VARCHAR,
		locale VARCHAR,
		internal_signature VARCHAR,
		customer_id VARCHAR,
		delivery_service VARCHAR,
		shardkey VARCHAR,
		sm_id INT,
		date_created TIMESTAMP,
		oof_shard VARCHAR
	);
	
	CREATE TABLE IF NOT EXISTS delivery (
		id SERIAL PRIMARY KEY,
		order_uid VARCHAR REFERENCES orders(order_uid),
		name VARCHAR,
		phone VARCHAR,
		zip VARCHAR,
		city VARCHAR,
		address VARCHAR,
		region VARCHAR,
		email VARCHAR
	);
	
	CREATE TABLE IF NOT EXISTS payment (
		id SERIAL PRIMARY KEY,
		order_uid VARCHAR REFERENCES orders(order_uid),
		transaction VARCHAR,
		request_id VARCHAR,
		currency VARCHAR,
		provider VARCHAR,
		amount INT,
		payment_dt TIMESTAMP,
		bank VARCHAR,
		delivery_cost INT,
		goods_total INT,
		custom_fee INT
	);
	
	CREATE TABLE IF NOT EXISTS items (
		id SERIAL PRIMARY KEY,
		order_uid VARCHAR REFERENCES orders(order_uid),
		chrt_id INT,
		track_number VARCHAR,
		price INT,
		rid VARCHAR,
		name VARCHAR,
		sale INT,
		size VARCHAR,
		total_price INT,
		nm_id INT,
		brand VARCHAR,
		status INT
	);
	`

	if err := db.Db.Ping(); err != nil {
		log.Fatal(err)
	} else {
		log.Println("Successfully Connected")
	}

	_, err := db.Db.Exec(createTablesQuery)
	if err != nil {
		return fmt.Errorf("failed to create tables: %w", err)
	}
	return nil

}
