package handlers

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strings"
	"sync"

	"github.com/labstack/echo/v4"
	"tunisianet-scraper/models"
	"tunisianet-scraper/scraper"
)

// --- Fonctions utilitaires ---

func normalizeText(text string) string {
	text = strings.ToLower(text)
	reg := regexp.MustCompile(`[\/\-_—–\|\+,\.]`)
	text = reg.ReplaceAllString(text, " ")
	return strings.Join(strings.Fields(text), " ")
}

func isBlacklisted(name string) bool {
	intrusBlacklist := []string{
		"chaussures", "banquette", "lisseur", "distributeur", "pancake", "cupcake",
		"moule", "brosse chauffante", "epilateur", "rasoir", "tondeuse", "poele", "casserole",
	}
	nameLower := strings.ToLower(name)
	for _, word := range intrusBlacklist {
		if strings.Contains(nameLower, word) {
			return true
		}
	}
	return false
}

func cleanPrice(priceStr string) string {
	reg := regexp.MustCompile(`[^0-9.]`)
	cleaned := reg.ReplaceAllString(priceStr, "")
	return cleaned
}

// --- Logique de Base de données ---

func SaveOrUpdateProduct(db *sql.DB, p models.Product, category string) error {
	query := `
	INSERT INTO products (name, price, url, image_url, source, category, updated_at)
	VALUES ($1, $2, $3, $4, $5, $6, NOW())
	ON CONFLICT (url) 
	DO UPDATE SET 
		price = EXCLUDED.price,
		name = EXCLUDED.name,
		image_url = EXCLUDED.image_url,
		updated_at = NOW();`

	_, err := db.Exec(query, p.Name, p.Price, p.URL, p.Image, p.Source, category)
	if err != nil {
		return err
	}

	numericPrice := cleanPrice(p.Price)
	_, err = db.Exec("INSERT INTO price_history (product_url, price) VALUES ($1, $2)", p.URL, numericPrice)
	return err
}

// --- Worker de Scraping ---

func ScrapeAndSaveAll(db *sql.DB, category string) {
	var wg sync.WaitGroup
	wg.Add(3)

	scrapeTask := func(fn func(string, string) ([]models.Product, error), source string) {
		defer wg.Done()
		products, err := fn("", category)
		if err == nil {
			saved := 0
			for _, p := range products {
				p.Source = source
				if !isBlacklisted(p.Name) {
					if saveErr := SaveOrUpdateProduct(db, p, category); saveErr == nil {
						saved++
					}
				}
			}
			log.Printf("✅ [%s] %d produits sauvegardés pour '%s'", source, saved, category)
		}
	}

	go scrapeTask(scraper.ScrapeProducts, "tunisianet")
	go scrapeTask(scraper.ScrapeMytekProducts, "mytek")
	go scrapeTask(scraper.ScrapeWikiProducts, "wiki")

	wg.Wait()
}

// --- API Handlers ---

func GetProductsFromDB(db *sql.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		category := strings.ToLower(strings.TrimSpace(c.QueryParam("category")))
		search := strings.TrimSpace(c.QueryParam("search"))

		sqlQuery := "SELECT id, name, price, url, image_url, source, category FROM products WHERE 1=1"
		var args []interface{}
		argIdx := 1

		if category != "" {
			sqlQuery += fmt.Sprintf(" AND LOWER(category) = $%d", argIdx)
			args = append(args, category)
			argIdx++
		}

		// Recherche flexible : on découpe par mots-clés
		if search != "" {
			words := strings.Fields(strings.ToLower(search))
			for _, word := range words {
				sqlQuery += fmt.Sprintf(" AND name ILIKE $%d", argIdx)
				args = append(args, "%"+word+"%")
				argIdx++
			}
		}

		sqlQuery += " ORDER BY updated_at DESC"

		rows, err := db.Query(sqlQuery, args...)
		if err != nil {
			log.Printf("❌ Erreur SQL : %v", err)
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Erreur SQL"})
		}
		defer rows.Close()

		results := []models.Product{}
		for rows.Next() {
			var p models.Product
			err := rows.Scan(&p.ID, &p.Name, &p.Price, &p.URL, &p.Image, &p.Source, &p.Category)
			if err != nil {
				continue
			}
			results = append(results, p)
		}

		return c.JSON(http.StatusOK, map[string]interface{}{"products": results})
	}
}

func GetCategories(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]interface{}{"categories": scraper.GetCategories()})
}