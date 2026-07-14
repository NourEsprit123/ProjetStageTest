CREATE TABLE IF NOT EXISTS products (
    id SERIAL PRIMARY KEY,
    external_id TEXT UNIQUE NOT NULL,
    name TEXT,
    price TEXT,
    image TEXT,
    category TEXT,
    url TEXT,
    in_stock BOOLEAN,
    source TEXT,              -- 👈 ajouté
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_products_category ON products(category);
CREATE INDEX IF NOT EXISTS idx_products_name ON products(name);