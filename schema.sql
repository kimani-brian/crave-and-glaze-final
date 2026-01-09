-- Users (Admin)
CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    username VARCHAR(50) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Categories (Wedding, Birthday, etc.)
CREATE TABLE IF NOT EXISTS categories (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    slug VARCHAR(100) UNIQUE NOT NULL
);

-- Products (The general cake info)
CREATE TABLE IF NOT EXISTS products (
    id SERIAL PRIMARY KEY,
    category_id INT REFERENCES categories(id),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    image_url VARCHAR(255),
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Product Variants (The specific logic: 1KG = 4000sh, 2KG = 7500sh)
CREATE TABLE IF NOT EXISTS product_variants (
    id SERIAL PRIMARY KEY,
    product_id INT REFERENCES products(id) ON DELETE CASCADE,
    weight_label VARCHAR(50) NOT NULL, -- e.g. "1 Kg", "2 Kg"
    price DECIMAL(10, 2) NOT NULL,     -- e.g. 4000.00
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Orders
CREATE TABLE IF NOT EXISTS orders (
    id SERIAL PRIMARY KEY,
    customer_name VARCHAR(100) NOT NULL,
    customer_phone VARCHAR(20) NOT NULL, -- Crucial for MPESA
    total_amount DECIMAL(10, 2) NOT NULL,
    status VARCHAR(20) DEFAULT 'PENDING', -- PENDING, PAID, FAILED
    mpesa_receipt VARCHAR(50),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Order Items (Linking order to specific cake variant)
CREATE TABLE IF NOT EXISTS order_items (
    id SERIAL PRIMARY KEY,
    order_id INT REFERENCES orders(id),
    product_variant_id INT REFERENCES product_variants(id),
    quantity INT DEFAULT 1,
    icing_flavor VARCHAR(100), -- "Whipping Cream", etc.
    custom_message TEXT,
    price_at_purchase DECIMAL(10, 2) NOT NULL
);

-- Seed some initial data for testing
INSERT INTO categories (name, slug) VALUES ('Birthday Cakes', 'birthday-cakes') ON CONFLICT DO NOTHING;