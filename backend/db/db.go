package db

import (
	"database/sql"
	"fmt"
	"tunisianet-scraper/models"
	_ "github.com/lib/pq" // Driver PostgreSQL
	 "time"
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
		fmt.Printf("⏳ DB non prête (%v), nouvelle tentative dans 2s... (%d/15)\n", err, i+1)
		time.Sleep(2 * time.Second)
	}
	panic(fmt.Sprintf("Impossible de se connecter à la base de données: %v", err))
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
    var args []interface{}
    argCount := 1

    // Base de la requête
    query := "SELECT external_id, name, price, image, category, url, in_stock, source FROM products WHERE 1=1"

    // Filtre catégorie
    if category != "" {
        query += fmt.Sprintf(" AND category ILIKE $%d", argCount)
        args = append(args, category)
        argCount++
    }
    
    // Filtre recherche optimisé
    if search != "" {
        // Utilise tsvector pour la pertinence linguistique et le % pour la similarité (fautes de frappe)
       query += fmt.Sprintf(` AND (to_tsvector('french', name) @@ phraseto_tsquery('french', $%d) 
                      OR name %% $%d)`, argCount, argCount)
        args = append(args, search)
        argCount++
        
        // Ajout du tri par pertinence (ts_rank) avant la limite
        query += fmt.Sprintf(" ORDER BY ts_rank(to_tsvector('french', name), phraseto_tsquery('french', $%d)) DESC", argCount-1)
    } else {
        query += " ORDER BY created_at DESC"
    }

    query += " LIMIT 300"

    rows, err := DB.Query(query, args...)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    for rows.Next() {
        var p models.Product
        err := rows.Scan(&p.ID, &p.Name, &p.Price, &p.Image, &p.Category, &p.URL, &p.InStock, &p.Source)
        if err != nil {
            fmt.Printf("Erreur lors du scan d'un produit : %v\n", err)
            continue
        }
        products = append(products, p)
    }

    return products, nil
}