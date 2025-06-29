CREATE TABLE delivery (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255),
    phone VARCHAR(50),
    zip VARCHAR(20),
    city VARCHAR(100),
    address VARCHAR(255),
    region VARCHAR(100),
    email VARCHAR(255)
);

CREATE TABLE payment (
    id SERIAL PRIMARY KEY,
    transaction VARCHAR(255),
    request_id VARCHAR(255),
    currency VARCHAR(10),
    provider VARCHAR(100),
    amount INTEGER,
    payment_dt INTEGER,
    bank VARCHAR(100),
    delivery_cost INTEGER,
    goods_total INTEGER,
    custom_fee INTEGER
);

CREATE TABLE orders (
    order_uid VARCHAR(255) PRIMARY KEY,
    track_number VARCHAR(255),
    entry VARCHAR(100),
    delivery_id INTEGER REFERENCES delivery(id),
    payment_id INTEGER REFERENCES payment(id),
    locale VARCHAR(10),
    internal_signature VARCHAR(255),
    customer_id VARCHAR(255),
    delivery_service VARCHAR(100),
    shardkey VARCHAR(50),
    sm_id INTEGER,
    date_created TIMESTAMP,
    oof_shard VARCHAR(50)
);

CREATE TABLE item (
    id SERIAL PRIMARY KEY,
    chrt_id INTEGER,
    track_number VARCHAR(255),
    price INTEGER,
    rid VARCHAR(255),
    name VARCHAR(255),
    sale INTEGER,
    size VARCHAR(50),
    total_price INTEGER,
    nm_id INTEGER,
    brand VARCHAR(255),
    status INTEGER,
    order_uid VARCHAR(255) REFERENCES orders(order_uid)
);
