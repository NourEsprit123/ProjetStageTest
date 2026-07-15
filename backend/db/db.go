package db

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"
	"tunisianet-scraper/models"
)

var DB *sql.DB

func InitDB() {
	connStr := "user=postgres password=nour dbname=postgres host=tunisianet-db port=5432 sslmode=disable"
	var err error
	for i := 0; i < 15; i++ {
		DB, err = sql.Open("postgres", connStr)
		if err == nil {
			if pingErr := DB.Ping(); pingErr == nil {
				fmt.Println("✅ Connexion DB établie")
				return
			} else {
				err = pingErr
			}
		}
		fmt.Printf("⏳ DB non prête (%v), tentative %d/15...\n", err, i+1)
		time.Sleep(2 * time.Second)
	}
	panic(fmt.Sprintf("Connexion DB impossible: %v", err))
}

func SaveProducts(products []models.Product) {
	query := `
		INSERT INTO products (external_id, name, price, image, category, url, in_stock, source, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW())
		ON CONFLICT (external_id) DO UPDATE 
		SET price = EXCLUDED.price, 
		    in_stock = EXCLUDED.in_stock,
		    image = EXCLUDED.image,
		    category = EXCLUDED.category,
		    source = EXCLUDED.source,
		    created_at = NOW();`

	for _, p := range products {
		if p.ID == "" {
			continue
		}
		_, err := DB.Exec(query, p.ID, p.Name, p.Price, p.Image, p.Category, p.URL, p.InStock, p.Source)
		if err != nil {
			fmt.Printf("Erreur insertion %s: %v\n", p.Name, err)
		}
	}
}

func GetProductsFromDB(search, category string) ([]models.Product, error) {
	var products []models.Product
	query := "SELECT external_id, name, price, image, category, url, in_stock, source FROM products WHERE 1=1"
	var args []interface{}
	argCount := 1

	if category != "" {
		query += fmt.Sprintf(" AND category ILIKE $%d", argCount)
		args = append(args, category)
		argCount++
	}
	if search != "" {
		query += fmt.Sprintf(" AND name ILIKE $%d", argCount)
		args = append(args, "%"+search+"%")
		argCount++
	}
	query += " ORDER BY created_at DESC LIMIT 300"

	rows, err := DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var p models.Product
		if err := rows.Scan(&p.ID, &p.Name, &p.Price, &p.Image, &p.Category, &p.URL, &p.InStock, &p.Source); err != nil {
			fmt.Printf("Erreur scan: %v\n", err)
			continue
		}
		products = append(products, p)
	}
	return products, nil
}