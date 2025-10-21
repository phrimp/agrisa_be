DROP DATABASE IF EXISTS payment_service;
CREATE DATABASE payment_service;

CREATE TYPE payment_status AS ENUM ('pending', 'completed', 'failed', 'refunded', 'cancelled', 'expired');

CREATE TABLE payments (
    id VARCHAR PRIMARY KEY,
    amount DECIMAL(12,2) NOT NULL,
    description VARCHAR(255) NOT NULL,
    status payment_status NOT NULL DEFAULT 'pending',
    user_id VARCHAR NOT NULL,
    checkout_url VARCHAR(255),
    type VARCHAR(50),
    order_code VARCHAR(255),
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expired_at TIMESTAMP,
    paid_at TIMESTAMP,
    deleted_at TIMESTAMP
);

CREATE TABLE order_items (
    id VARCHAR PRIMARY KEY,
    payment_id VARCHAR NOT NULL,
    item_id VARCHAR,
    name VARCHAR NOT NULL,
    price DECIMAL(12,2) NOT NULL,
    quantity INT NOT NULL DEFAULT 1,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP,
    FOREIGN KEY (payment_id) REFERENCES payments(id) ON DELETE CASCADE
);

CREATE TABLE configurations (
    id VARCHAR PRIMARY KEY,
    payos_client_id VARCHAR NOT NULL,
    payos_api_key VARCHAR NOT NULL,
    payos_checksum_key VARCHAR NOT NULL,
    payos_expired_duration VARCHAR,
    payos_order_code_length INT,
    payment_cron_expression VARCHAR,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP
);