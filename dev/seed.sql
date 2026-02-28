-- pglet dev seed data
-- This runs automatically on first container start (via docker-entrypoint-initdb.d)

-- ============================================================
-- Schema: public (default)
-- ============================================================

CREATE TABLE customers (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    email TEXT UNIQUE NOT NULL,
    tier TEXT NOT NULL DEFAULT 'free' CHECK (tier IN ('free', 'pro', 'enterprise')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE products (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT,
    price NUMERIC(10, 2) NOT NULL CHECK (price >= 0),
    stock INT NOT NULL DEFAULT 0,
    category TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE orders (
    id SERIAL PRIMARY KEY,
    customer_id INT NOT NULL REFERENCES customers(id),
    status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'paid', 'shipped', 'delivered', 'cancelled')),
    total NUMERIC(10, 2) NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE order_items (
    id SERIAL PRIMARY KEY,
    order_id INT NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    product_id INT NOT NULL REFERENCES products(id),
    quantity INT NOT NULL CHECK (quantity > 0),
    unit_price NUMERIC(10, 2) NOT NULL
);

CREATE INDEX idx_orders_customer ON orders(customer_id);
CREATE INDEX idx_orders_status ON orders(status);
CREATE INDEX idx_order_items_order ON order_items(order_id);

-- ============================================================
-- Schema: analytics
-- ============================================================

CREATE SCHEMA analytics;

CREATE TABLE analytics.page_views (
    id BIGSERIAL PRIMARY KEY,
    path TEXT NOT NULL,
    referrer TEXT,
    user_agent TEXT,
    customer_id INT REFERENCES customers(id),
    viewed_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_page_views_path ON analytics.page_views(path);

-- ============================================================
-- Views
-- ============================================================

CREATE VIEW order_summary AS
SELECT
    o.id AS order_id,
    c.name AS customer_name,
    o.status,
    o.total,
    COUNT(oi.id) AS item_count,
    o.created_at
FROM orders o
JOIN customers c ON c.id = o.customer_id
LEFT JOIN order_items oi ON oi.order_id = o.id
GROUP BY o.id, c.name, o.status, o.total, o.created_at;

-- ============================================================
-- Functions
-- ============================================================

CREATE FUNCTION customer_order_total(cid INT)
RETURNS NUMERIC AS $$
    SELECT COALESCE(SUM(total), 0) FROM orders WHERE customer_id = cid;
$$ LANGUAGE sql STABLE;

-- ============================================================
-- Seed data
-- ============================================================

INSERT INTO customers (name, email, tier) VALUES
    ('Alice Johnson', 'alice@example.com', 'pro'),
    ('Bob Smith', 'bob@example.com', 'free'),
    ('Carol Williams', 'carol@example.com', 'enterprise'),
    ('Dave Brown', 'dave@example.com', 'free'),
    ('Eve Davis', 'eve@example.com', 'pro');

INSERT INTO products (name, description, price, stock, category) VALUES
    ('Widget A', 'Standard widget', 9.99, 150, 'widgets'),
    ('Widget B', 'Premium widget', 24.99, 75, 'widgets'),
    ('Gadget X', 'Entry-level gadget', 49.99, 30, 'gadgets'),
    ('Gadget Y', 'Pro gadget with extras', 99.99, 12, 'gadgets'),
    ('Doohickey', 'Multi-purpose doohickey', 14.50, 200, 'accessories'),
    ('Thingamajig', NULL, 5.00, 500, 'accessories');

INSERT INTO orders (customer_id, status, total) VALUES
    (1, 'delivered', 59.97),
    (1, 'shipped', 99.99),
    (2, 'pending', 14.50),
    (3, 'paid', 174.97),
    (3, 'delivered', 9.99),
    (4, 'cancelled', 49.99),
    (5, 'pending', 49.98);

INSERT INTO order_items (order_id, product_id, quantity, unit_price) VALUES
    (1, 1, 3, 9.99),
    (1, 5, 2, 14.50),
    (2, 4, 1, 99.99),
    (3, 5, 1, 14.50),
    (4, 2, 3, 24.99),
    (4, 4, 1, 99.99),
    (5, 1, 1, 9.99),
    (6, 3, 1, 49.99),
    (7, 1, 2, 9.99),
    (7, 6, 6, 5.00);

INSERT INTO analytics.page_views (path, referrer, customer_id, viewed_at) VALUES
    ('/', 'https://google.com', 1, now() - interval '2 days'),
    ('/products', 'https://google.com', 1, now() - interval '2 days'),
    ('/products/1', NULL, 1, now() - interval '2 days'),
    ('/', NULL, NULL, now() - interval '1 day'),
    ('/products', '/', NULL, now() - interval '1 day'),
    ('/', 'https://twitter.com', 3, now() - interval '1 hour'),
    ('/products', '/', 3, now() - interval '1 hour'),
    ('/products/4', '/products', 3, now() - interval '30 minutes');

-- ============================================================
-- Second database for testing multi-db switching
-- ============================================================
-- Note: Postgres doesn't allow CREATE DATABASE inside a transaction,
-- so the second DB is created via a separate script below.
