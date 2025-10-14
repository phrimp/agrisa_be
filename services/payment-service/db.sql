CREATE DATABASE IF NOT EXISTS payment_service;

CREATE TABLE payments (
    id VARCHAR PRIMARY KEY,
    amount DECIMAL(12,2) NOT NULL,
    description VARCHAR(255) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    user_id VARCHAR NOT NULL,
    checkout_url VARCHAR(255),
    order_code VARCHAR(255),
    created_at TIMESTAMP NOT NULL DEFAULT now(),
    updated_at TIMESTAMP NOT NULL DEFAULT now(),
    expired_at TIMESTAMP,
    paid_at TIMESTAMP,
    deleted_at TIMESTAMP
);

CREATE TABLE order_items (
    id VARCHAR PRIMARY KEY,
    payment_id VARCHAR NOT NULL,
    item_id VARCHAR NOT NULL,
    item_name VARCHAR(255) NOT NULL,
    item_price DECIMAL(12,2) NOT NULL,
    type VARCHAR(50),
    created_at TIMESTAMP NOT NULL DEFAULT now(),
    updated_at TIMESTAMP NOT NULL DEFAULT now(),
    deleted_at TIMESTAMP,
    FOREIGN KEY (payment_id) REFERENCES payments(id)
);

CREATE TABLE configurations (
    id VARCHAR PRIMARY KEY,
    payos_client_id VARCHAR NOT NULL,
    payos_api_key VARCHAR NOT NULL,
    payos_checksum_key VARCHAR NOT NULL,
    payos_expired_duration VARCHAR,
    payos_order_code_length INT,
    payment_cron_expression VARCHAR,
    created_at TIMESTAMP NOT NULL DEFAULT now(),
    updated_at TIMESTAMP NOT NULL DEFAULT now(),
    deleted_at TIMESTAMP
);