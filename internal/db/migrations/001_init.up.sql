DO $$
BEGIN
    IF NOT EXISTS (SELECT FROM pg_database WHERE datname = 'wbstore') THEN
        CREATE DATABASE wbstore;
    END IF;
END
$$;

DO $$
BEGIN
    IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'felix') THEN
        CREATE ROLE felix WITH LOGIN;
    END IF;
END
$$;

GRANT CREATE ON DATABASE wbstore TO felix;


CREATE TABLE IF NOT EXISTS orders (
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

CREATE TABLE cache_info (
    order_uid VARCHAR REFERENCES orders(order_uid),
    creation_date TIMESTAMP NOT NULL,
    last_cache_load_date TIMESTAMP NOT NULL
);