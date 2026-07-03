package handlers

import (
	"database/sql"
	"log"
	"net/http"
	"regexp"
	"strings"
	"sync"

	"github.com/labstack/echo/v4"
	"tunisianet-scraper/models"
	"tunisianet-scraper/scraper"
)

func normalizeText(text string) string {
	text = strings.ToLower(text)
	reg := regexp.MustCompile(`[\/\-_—–\|\+,\.]`)
	text = reg.ReplaceAllString(text, " ")
	return strings.Join(strings.Fields(text), " ")
}

func cleanPrice(priceStr string) string {
	priceStr = strings.ReplaceAll(priceStr, ",", ".")
	reg := regexp.MustCompile(`[^0-9.]`)
	cleaned := reg.ReplaceAllString(priceStr, "")
	parts := strings.Split(cleaned, ".")
	if len(parts) > 2 {
		cleaned = strings.Join(parts[:len(parts)-1], "") + "." + parts[len(parts)-1]
	}
	return cleaned
}

func SaveOrUpdateProduct(db *sql.DB, p models.Product, category string, reference string) error {
	query := `
	INSERT INTO products (name, price, url, image_url, source, category, reference, updated_at)
	VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())
	ON CONFLICT (url)
	DO UPDATE SET
		price      = EXCLUDED.price,
		name       = EXCLUDED.name,
		image_url  = EXCLUDED.image_url,
		reference  = EXCLUDED.reference,
		updated_at = NOW();`

	_, err := db.Exec(query, p.Name, p.Price, p.URL, p.Image, p.Source, category, reference)
	if err != nil {
		return err
	}

	numericPrice := cleanPrice(p.Price)
	if numericPrice != "" {
		db.Exec("INSERT INTO price_history (product_url, price) VALUES ($1, $2)", p.URL, numericPrice)
	}
	return nil
}

func scrapeByReference(db *sql.DB, reference string, category string) []models.Product {
	var allProducts []models.Product
	var mu sync.Mutex
	var wg sync.WaitGroup

	wg.Add(3)
	scrapeAndSave := func(fn func(string, string) ([]models.Product, error), source string) {
		defer wg.Done()
		products, err := fn(reference, category)
		if err != nil {
			log.Printf("⚠️ [%s] Erreur scraping: %v", source, err)
			return
		}
		for _, p := range products {
			p.Source = source
			if err := SaveOrUpdateProduct(db, p, category, reference); err == nil {
				mu.Lock()
				allProducts = append(allProducts, p)
				mu.Unlock()
			}
		}
	}

	go scrapeAndSave(scraper.ScrapeProducts, "tunisianet")
	go scrapeAndSave(scraper.ScrapeMytekProducts, "mytek")
	go scrapeAndSave(scraper.ScrapeWikiProducts, "wiki")
	wg.Wait()
	return allProducts
}

func GetProductsFromDB(db *sql.DB) echo.HandlerFunc {
    return func(c echo.Context) error {
        query := strings.TrimSpace(c.QueryParam("search"))
        category := c.QueryParam("category") // 1. Récupération de la catégorie

        // Requête SQL avec filtrage optionnel par catégorie
        // On utilise $1 pour la recherche et $2 pour la catégorie
        sqlQuery := `
            SELECT id, name, price, url, image_url, source, category, 
                   similarity(name, $1) AS score, updated_at
            FROM products
            WHERE (name % $1 OR reference % $1)
            AND ($2 = '' OR category = $2) -- 2. Filtre ici : si $2 est vide, tout est pris, sinon filtré
            ORDER BY score DESC
            LIMIT 50;`

        rows, err := db.Query(sqlQuery, query, category) // 3. Passage du paramètre $2
        
        var results []models.Product
        if err == nil {
            defer rows.Close()
            for rows.Next() {
                var p models.Product
                err := rows.Scan(&p.ID, &p.Name, &p.Price, &p.URL, &p.Image, &p.Source, &p.Category, &p.Score, &p.UpdatedAt)
                if err == nil {
                    results = append(results, p)
                }
            }
        }

        if len(results) > 0 {
            return c.JSON(http.StatusOK, map[string]interface{}{
                "products": results,
                "from_cache": true,
            })
        }

        // Si rien trouvé, on passe la catégorie au scraper
        scraped := scrapeByReference(db, query, category) // 4. Transmission de la catégorie au scraper
        
        return c.JSON(http.StatusOK, map[string]interface{}{
            "products": scraped,
            "from_cache": false,
        })
    }
}
func GetCategories(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]interface{}{
		"categories": scraper.GetCategories(),
	})
}