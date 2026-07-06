package handlers

import (
	"bytes"
	
	"database/sql"
	"encoding/json"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"log"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/labstack/echo/v4"
	"github.com/lib/pq"
	"tunisianet-scraper/models"
	"tunisianet-scraper/scraper"
)

// Nettoie le prix pour le convertir en format numérique (ex: "1.250 DT" -> "1.250")
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

// Vérifie si la requête ressemble à une référence produit (alphanumérique)
func isReference(query string) bool {
	query = strings.TrimSpace(query)
	reg := regexp.MustCompile(`^[A-Za-z0-9\-_\.]{3,}$`)
	return reg.MatchString(query)
}

// Sauvegarde ou met à jour le produit en Postgres
func SaveOrUpdateProduct(db *sql.DB, es *elasticsearch.Client, p models.Product, category string, reference string) error {
	p.Reference = strings.TrimSpace(reference)
	p.Category = category

	query := `
	INSERT INTO products (name, price, url, image_url, source, category, reference, updated_at)
	VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())
	ON CONFLICT (url)
	DO UPDATE SET
		price     = EXCLUDED.price,
		name      = EXCLUDED.name,
		image_url = EXCLUDED.image_url,
		category  = EXCLUDED.category,
		reference = EXCLUDED.reference,
		updated_at = NOW();`

	_, err := db.Exec(query, p.Name, p.Price, p.URL, p.Image, p.Source, p.Category, p.Reference)
	if err != nil {
		return err
	}

	numericPrice := cleanPrice(p.Price)
	if numericPrice != "" {
		db.Exec("INSERT INTO price_history (product_url, price) VALUES ($1, $2)", p.URL, numericPrice)
	}

	// Indexation asynchrone dans ES
	if es != nil {
		go func(product models.Product) {
			data, _ := json.Marshal(product)
			res, err := es.Index("products", bytes.NewReader(data), es.Index.WithDocumentID(product.URL))
			if err == nil {
				res.Body.Close()
			}
		}(p)
	}
	return nil
}

// Scrape les sites et filtre les résultats pour ne garder que ceux correspondant à la référence
func scrapeByReference(db *sql.DB, es *elasticsearch.Client, reference string, category string) []models.Product {
    var allProducts []models.Product
    var mu sync.Mutex
    var wg sync.WaitGroup

    sources := []struct {
        fn   func(string, string) ([]models.Product, error)
        name string
    }{
        {scraper.ScrapeProducts, "tunisianet"},
        {scraper.ScrapeMytekProducts, "mytek"},
        {scraper.ScrapeWikiProducts, "wiki"},
    }

    log.Printf("🔍 Démarrage du scraping pour la référence : %s", reference)

    for _, s := range sources {
        wg.Add(1)
        go func(source struct {
            fn   func(string, string) ([]models.Product, error)
            name string
        }) {
            defer wg.Done()
            
            // Appel du scraper
            products, err := source.fn(reference, category)
            if err != nil {
                log.Printf("❌ Erreur scraper %s: %v", source.name, err)
                return
            }
            
            for _, p := range products {
        // SUPPRIME ou COMMENTE le 'if !strings.Contains...' qui bloque tout
        
        p.Source = source.name
        
        // AJOUTE CECI :
        log.Printf("📥 Tentative de sauvegarde : %s", p.Name)
        
        err := SaveOrUpdateProduct(db, es, p, category, reference)
        if err != nil {
            log.Printf("❌ ERREUR SAUVEGARDE : %v", err)
        } else {
            log.Printf("✅ SUCCÈS SAUVEGARDE : %s", p.Name)
            mu.Lock()
            allProducts = append(allProducts, p)
            mu.Unlock()
        }
    }
        }(s)
    }
    wg.Wait()
    log.Printf("🏁 Scraping terminé. Produits trouvés et sauvés : %d", len(allProducts))
    return allProducts
}

// Recherche dans ES : Utilise "filter" pour une recherche stricte sur référence
func searchViaElasticsearch(es *elasticsearch.Client, db *sql.DB, query string, category string) []models.Product {
	if es == nil { return nil }

	var esQuery map[string]interface{}
	q := strings.TrimSpace(query)

	if isReference(q) {
		esQuery = map[string]interface{}{
			"query": map[string]interface{}{
				"bool": map[string]interface{}{
					"filter": []map[string]interface{}{
						{"term": map[string]interface{}{"reference.keyword": q}},
					},
				},
			},
		}
	} else {
		esQuery = map[string]interface{}{
			"query": map[string]interface{}{
				"match": map[string]interface{}{
					"name": map[string]interface{}{"query": q, "fuzziness": "AUTO"},
				},
			},
		}
	}

	var buf bytes.Buffer
	json.NewEncoder(&buf).Encode(esQuery)
	res, err := es.Search(es.Search.WithIndex("products"), es.Search.WithBody(&buf))
	if err != nil { return nil }
	defer res.Body.Close()

	var r map[string]interface{}
	json.NewDecoder(res.Body).Decode(&r)

	hitsWrapper, ok := r["hits"].(map[string]interface{})
	if !ok { return nil }
	hits, ok := hitsWrapper["hits"].([]interface{})
	if !ok || len(hits) == 0 { return nil }

	var urls []string
	for _, hit := range hits {
		src := hit.(map[string]interface{})["_source"].(map[string]interface{})
		if u, ok := src["url"].(string); ok {
			urls = append(urls, u)
		}
	}

	rows, err := db.Query("SELECT id, name, price, url, image_url, source, category, reference FROM products WHERE url = ANY($1)", pq.Array(urls))
	if err != nil { return nil }
	defer rows.Close()

	var products []models.Product
	for rows.Next() {
		var p models.Product
		rows.Scan(&p.ID, &p.Name, &p.Price, &p.URL, &p.Image, &p.Source, &p.Category, &p.Reference)
		products = append(products, p)
	}
	return products
}

// Récupération par catégorie sans recherche textuelle
func fetchFromPostgresByCategory(db *sql.DB, category string) []models.Product {
	rows, err := db.Query(
		`SELECT id, name, price, url, image_url, source, category, reference
		 FROM products
		 WHERE LOWER(category) = $1
		 ORDER BY updated_at DESC
		 LIMIT 100`,
		strings.ToLower(category),
	)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var products []models.Product
	for rows.Next() {
		var p models.Product
		if err := rows.Scan(&p.ID, &p.Name, &p.Price, &p.URL, &p.Image, &p.Source, &p.Category, &p.Reference); err == nil {
			products = append(products, p)
		}
	}
	return products
}

// Handler Echo pour l'API
func GetProductsFromDB(db *sql.DB, es *elasticsearch.Client) echo.HandlerFunc {
	return func(c echo.Context) error {
		query := c.QueryParam("search")
		category := c.QueryParam("category")

		// 1. Tenter la recherche rapide
		results := searchViaElasticsearch(es, db, query, category)
		
		// 2. Si on a des résultats, on les renvoie IMMÉDIATEMENT
		if len(results) > 0 {
			log.Printf("⚡ [Cache] %d produits trouvés via ES pour '%s'", len(results), query)
			return c.JSON(http.StatusOK, map[string]interface{}{"products": results, "from_cache": true})
		}

		// 3. Sinon, on scrape (c'est ici que ça devient lent la première fois)
		log.Printf("🔍 [Scraping] Aucun résultat en cache pour '%s'. Démarrage du scraper...", query)
		scraped := scrapeByReference(db, es, query, category)
		
		return c.JSON(http.StatusOK, map[string]interface{}{"products": scraped, "from_cache": false})
	}
}


	func GetProductDetail(db *sql.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		id := c.Param("id")
		var p models.Product
		
		// Utilise ta table products
		err := db.QueryRow("SELECT id, name, price, url, image_url, source, category, reference FROM products WHERE id = $1", id).Scan(
			&p.ID, &p.Name, &p.Price, &p.URL, &p.Image, &p.Source, &p.Category, &p.Reference,
		)
		
		if err != nil {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "Produit introuvable"})
		}
		
		return c.JSON(http.StatusOK, p)
	}
}

// Handler pour les catégories
func GetCategories(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]interface{}{"categories": scraper.GetCategories()})
}