package database

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/lib/pq"
)

func InitDB() *sql.DB {
	// 1. On essaie d'abord de récupérer la chaîne de connexion de Render (Neon)
	connStr := os.Getenv("DB_URL")

	// 2. Si elle est vide, on utilise la configuration locale par défaut (Docker Compose)
	if connStr == "" {
		log.Println("⚠️ DB_URL non trouvée, utilisation de la configuration locale...")
		host := "db"
		port := 5432
		user := "postgres"
		password := "nour"
		dbname := "postgres"

		connStr = fmt.Sprintf(
			"host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
			host, port, user, password, dbname,
		)
	}

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("❌ Erreur ouverture BD: %v", err)
	}
	if err = db.Ping(); err != nil {
		log.Fatalf("❌ Impossible de ping PostgreSQL: %v", err)
	}

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS products (
			id          SERIAL PRIMARY KEY,
			name        TEXT NOT NULL,
			price       TEXT,
			url         TEXT UNIQUE NOT NULL,
			image_url   TEXT,
			source      TEXT,
			category    TEXT,
			reference   TEXT,
			updated_at  TIMESTAMP DEFAULT NOW()
		);

		CREATE TABLE IF NOT EXISTS price_history (
			id          SERIAL PRIMARY KEY,
			product_url TEXT NOT NULL,
			price       TEXT,
			recorded_at TIMESTAMP DEFAULT NOW()
		);

		CREATE INDEX IF NOT EXISTS idx_products_reference ON products(reference);
		CREATE INDEX IF NOT EXISTS idx_products_category  ON products(category);
		CREATE INDEX IF NOT EXISTS idx_price_history_url  ON price_history(product_url);
	`)
	if err != nil {
		log.Fatalf("❌ Erreur création tables: %v", err)
	}

	log.Println("🚀 Connecté à PostgreSQL avec succès !")
	return db
}