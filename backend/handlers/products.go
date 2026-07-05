package handlers

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"regexp"
	"strings"
	"sync"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/labstack/echo/v4"
	"github.com/lib/pq"
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

func SaveOrUpdateProduct(db *sql.DB, es *elasticsearch.Client, p models.Product, category string, reference string) error {
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

	// Indexation dans Elasticsearch en arrière-plan
	if es != nil {
		go func(product models.Product) {
			data, err := json.Marshal(product)
			if err != nil {
				return
			}
			res, err := es.Index(
				"products",
				bytes.NewReader(data),
				es.Index.WithDocumentID(product.URL),
				es.Index.WithContext(context.Background()),
			)
			if err != nil {
				log.Printf("⚠️ Erreur indexation ES: %v", err)
			} else {
				res.Body.Close()
			}
		}(p)
	}

	return nil
}

func scrapeByReference(db *sql.DB, es *elasticsearch.Client, reference string, category string) []models.Product {
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
			if err := SaveOrUpdateProduct(db, es, p, category, reference); err == nil {
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

// searchViaElasticsearch cherche via ES puis hydrate depuis PostgreSQL
func searchViaElasticsearch(es *elasticsearch.Client, db *sql.DB, query string, category string) []models.Product {
	if es == nil {
		return nil
	}

	var buf bytes.Buffer
	esQuery := map[string]interface{}{
		"query": map[string]interface{}{
			"multi_match": map[string]interface{}{
				"query":     query,
				"fields":    []string{"name", "reference"},
				"fuzziness": "AUTO",
			},
		},
		"size": 50,
	}
	json.NewEncoder(&buf).Encode(esQuery)

	res, err := es.Search(
		es.Search.WithIndex("products"),
		es.Search.WithBody(&buf),
	)
	if err != nil {
		log.Printf("⚠️ Erreur recherche ES: %v", err)
		return nil
	}
	defer res.Body.Close()

	var r map[string]interface{}
	json.NewDecoder(res.Body).Decode(&r)

	hitsWrapper, ok := r["hits"].(map[string]interface{})
	if !ok {
		return nil
	}
	hits, ok := hitsWrapper["hits"].([]interface{})
	if !ok || len(hits) == 0 {
		return nil
	}

	var urls []string
	for _, hit := range hits {
		source := hit.(map[string]interface{})["_source"].(map[string]interface{})
		if url, ok := source["url"].(string); ok {
			urls = append(urls, url)
		}
	}

	if len(urls) == 0 {
		return nil
	}

	// Filtre catégorie optionnel
	sqlQuery := "SELECT id, name, price, url, image_url, source, category FROM products WHERE url = ANY($1)"
	args := []interface{}{pq.Array(urls)}
	if category != "" {
		sqlQuery += " AND LOWER(category) = $2"
		args = append(args, strings.ToLower(category))
	}

	rows, err := db.Query(sqlQuery, args...)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var products []models.Product
	for rows.Next() {
		var p models.Product
		if err := rows.Scan(&p.ID, &p.Name, &p.Price, &p.URL, &p.Image, &p.Source, &p.Category); err == nil {
			products = append(products, p)
		}
	}
	return products
}

// fetchFromPostgresByCategory récupère les produits d'une catégorie sans recherche textuelle
func fetchFromPostgresByCategory(db *sql.DB, category string) []models.Product {
	rows, err := db.Query(
		"SELECT id, name, price, url, image_url, source, category FROM products WHERE LOWER(category) = $1 ORDER BY updated_at DESC LIMIT 50",
		strings.ToLower(category),
	)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var products []models.Product
	for rows.Next() {
		var p models.Product
		if err := rows.Scan(&p.ID, &p.Name, &p.Price, &p.URL, &p.Image, &p.Source, &p.Category); err == nil {
			products = append(products, p)
		}
	}
	return products
}

func GetProductsFromDB(db *sql.DB, es *elasticsearch.Client) echo.HandlerFunc {
	return func(c echo.Context) error {
		query    := strings.TrimSpace(c.QueryParam("search"))
		category := c.QueryParam("category")

		var results []models.Product

		if query != "" {
			// Recherche via Elasticsearch si disponible
			results = searchViaElasticsearch(es, db, query, category)
		} else if category != "" {
			// Pas de recherche textuelle → catégorie seule depuis PostgreSQL
			results = fetchFromPostgresByCategory(db, category)
		}

		if len(results) > 0 {
			return c.JSON(http.StatusOK, map[string]interface{}{
				"products":   results,
				"from_cache": true,
			})
		}

		// Rien trouvé → scraping à la demande
		scraped := scrapeByReference(db, es, query, category)
		return c.JSON(http.StatusOK, map[string]interface{}{
			"products":   scraped,
			"from_cache": false,
		})
	}
}

func GetCategories(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]interface{}{
		"categories": scraper.GetCategories(),
	})
}