CREATE TABLE IF NOT EXISTS orders (
    id                BIGSERIAL PRIMARY KEY,
    customer_name     VARCHAR(255) NOT NULL,
    customer_email    VARCHAR(255) NOT NULL,
    customer_company  VARCHAR(255),
    customer_phone    VARCHAR(50),
    customer_country  VARCHAR(100),
    report_id         BIGINT NOT NULL REFERENCES reports(id) ON DELETE RESTRICT,
    report_title      VARCHAR(500) NOT NULL,
    report_slug       VARCHAR(500) NOT NULL,
    amount            DECIMAL(10,2) NOT NULL,
    currency          VARCHAR(3) NOT NULL DEFAULT 'USD',
    paypal_order_id   VARCHAR(100) UNIQUE,
    paypal_capture_id VARCHAR(100),
    status            VARCHAR(30) NOT NULL DEFAULT 'pending_payment',
    fulfilled_at      TIMESTAMP,
    fulfilled_by      BIGINT REFERENCES users(id) ON DELETE SET NULL,
    admin_notes       TEXT,
    created_at        TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at        TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT chk_order_status CHECK (status IN (
        'pending_payment','payment_received','processing','delivered','cancelled','refunded'
    ))
);

CREATE INDEX idx_orders_status         ON orders(status);
CREATE INDEX idx_orders_report_id      ON orders(report_id);
CREATE INDEX idx_orders_customer_email ON orders(customer_email);
CREATE INDEX idx_orders_created_at     ON orders(created_at DESC);
