DROP DATABASE IF EXISTS payment_service;
CREATE DATABASE payment_service;

-- Connect to the database first
\c payment_service

CREATE TYPE payment_status AS ENUM ('pending', 'completed', 'failed', 'refunded', 'cancelled', 'expired');
CREATE TYPE payout_status AS ENUM ('pending', 'completed');

CREATE TABLE payments (
    id VARCHAR PRIMARY KEY,
    amount DECIMAL(12,2) NOT NULL,
    description VARCHAR(255) NOT NULL,
    status payment_status NOT NULL DEFAULT 'pending',
    user_id VARCHAR NOT NULL,
    checkout_url VARCHAR(255),
    type VARCHAR(50),
    order_code VARCHAR(255),
    created_at TIMESTAMP NOT NULL DEFAULT (CURRENT_TIMESTAMP AT TIME ZONE 'Asia/Ho_Chi_Minh'),
    updated_at TIMESTAMP NOT NULL DEFAULT (CURRENT_TIMESTAMP AT TIME ZONE 'Asia/Ho_Chi_Minh'),
    expired_at TIMESTAMP,
    paid_at TIMESTAMP,
    deleted_at TIMESTAMP
);

CREATE TABLE items (
    id VARCHAR PRIMARY KEY,
    payment_id VARCHAR NOT NULL,
    item_id VARCHAR,
    name VARCHAR NOT NULL,
    price DECIMAL(12,2) NOT NULL,
    quantity INT NOT NULL DEFAULT 1,
    created_at TIMESTAMP NOT NULL DEFAULT (CURRENT_TIMESTAMP AT TIME ZONE 'Asia/Ho_Chi_Minh'),
    updated_at TIMESTAMP NOT NULL DEFAULT (CURRENT_TIMESTAMP AT TIME ZONE 'Asia/Ho_Chi_Minh'),
    deleted_at TIMESTAMP,
    FOREIGN KEY (payment_id) REFERENCES payments(id) ON DELETE CASCADE
);

CREATE TABLE payouts (
    id VARCHAR PRIMARY KEY,
    amount DECIMAL(12,2) NOT NULL,
    description VARCHAR(255) NOT NULL,
    status payout_status NOT NULL DEFAULT 'pending',
    user_id VARCHAR NOT NULL,
    bank_code VARCHAR(255),
    account_number VARCHAR(255),
    created_at TIMESTAMP NOT NULL DEFAULT (CURRENT_TIMESTAMP AT TIME ZONE 'Asia/Ho_Chi_Minh'),
    updated_at TIMESTAMP NOT NULL DEFAULT (CURRENT_TIMESTAMP AT TIME ZONE 'Asia/Ho_Chi_Minh'),
    deleted_at TIMESTAMP,
    completed_at TIMESTAMP
);