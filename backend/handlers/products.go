package handlers

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"fmt"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/labstack/echo/v4"
	"github.com/lib/pq"
	"tunisianet-scraper/models"
	"tunisianet-scraper/scraper"
)

// ─── Utilitaires ────────────────────────────────────────────────────────────

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

func normalizeCacheKey(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.Join(strings.Fields(s), " ")
	return s
}

// ─── Sauvegarde ─────────────────────────────────────────────────────────────

// Utilitaire de normalisation
func normalizeString(s string) string {
    s = strings.ToLower(s)
    reg := regexp.MustCompile(`[^a-z0-9\s]`)
    s = reg.ReplaceAllString(s, "")
    return strings.Join(strings.Fields(s), " ")
}

func SaveOrUpdateProduct(db *sql.DB, es *elasticsearch.Client, p models.Product, category string, reference string) error {
    p.Reference = strings.TrimSpace(reference)
    p.Category = category

    // 1. Normalisation du nom
    normalizedName := normalizeString(p.Name)

    // 2. Insertion/Update dans PostgreSQL
    query := `
    INSERT INTO products (name, price, url, image_url, source, category, reference, updated_at)
    VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())
    ON CONFLICT (url)
    DO UPDATE SET
        price      = EXCLUDED.price,
        name       = EXCLUDED.name,
        image_url  = EXCLUDED.image_url,
        category   = EXCLUDED.category,
        reference  = EXCLUDED.reference,
        updated_at = NOW();`

    _, err := db.Exec(query, normalizedName, p.Price, p.URL, p.Image, p.Source, p.Category, p.Reference)
    if err != nil {
        return err
    }

    numericPrice := cleanPrice(p.Price)
    if numericPrice != "" {
        db.Exec("INSERT INTO price_history (product_url, price) VALUES ($1, $2)", p.URL, numericPrice)
    }

    // 3. Indexation synchrone dans ES
    if es != nil {
        p.Name = normalizedName 
        
        // --- APPEL NLP POUR RÉCUPÉRER LE VECTEUR ---
        vector, err := getVectorFromPython(normalizedName)
        if err == nil {
            p.Vector = vector // p.Vector doit exister dans votre struct Product
        } else {
            log.Printf("⚠️ Erreur NLP pour %s: %v", p.Name, err)
        }
        // -------------------------------------------
        
        data, _ := json.Marshal(p)
        res, err := es.Index(
            "products",
            bytes.NewReader(data),
            es.Index.WithDocumentID(models.DocIDFromURL(p.URL)),
        )
        if err != nil {
            log.Printf("⚠️ Erreur transport ES pour %s: %v", p.Name, err)
        } else if res.IsError() {
            bodyBytes, _ := io.ReadAll(res.Body)
            log.Printf("⚠️ ES rejeté '%s': %s", p.Name, string(bodyBytes))
            res.Body.Close()
        } else {
            res.Body.Close()
        }
    }

    return nil
}
// ─── Scraping à la demande ──────────────────────────────────────────────────

func scrapeByReference(db *sql.DB, es *elasticsearch.Client, reference string, category string) []models.Product {
	var allProducts []models.Product
	var mu sync.Mutex
	var wg sync.WaitGroup

	cacheKey := normalizeCacheKey(reference)

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

			products, err := source.fn(reference, category)
			if err != nil {
				log.Printf("❌ Erreur scraper %s: %v", source.name, err)
				return
			}

			for _, p := range products {
				p.Source = source.name
				log.Printf("📥 Tentative de sauvegarde : %s", p.Name)
				err := SaveOrUpdateProduct(db, es, p, category, cacheKey)
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

// ─── Recherche Elasticsearch ─────────────────────────────────────────────────

func searchViaElasticsearch(es *elasticsearch.Client, db *sql.DB, query string, category string) []models.Product {
	if es == nil {
		return nil
	}

	var buf bytes.Buffer
	// Configuration de la recherche plus stricte avec 'AND'
	esQuery := map[string]interface{}{
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"must": []interface{}{ // Utilisation de 'must' pour AND strict
					map[string]interface{}{
						"match": map[string]interface{}{
							"name": map[string]interface{}{
								"query":                query,
								"operator":             "and", // Force la correspondance de TOUS les mots
								"minimum_should_match": "100%",
								"fuzziness":            "AUTO:3,6",
								"boost":                3,
							},
						},
					},
				},
			},
		},
		"size": 200,
	}

	if category != "" {
		esQuery["query"].(map[string]interface{})["bool"].(map[string]interface{})["filter"] = map[string]interface{}{
			"term": map[string]interface{}{
				"category": strings.ToLower(category),
			},
		}
	}

	json.NewEncoder(&buf).Encode(esQuery)

	res, err := es.Search(
		es.Search.WithIndex("products"),
		es.Search.WithBody(&buf),
	)
	if err != nil {
		log.Printf("⚠️ Erreur ES: %v", err)
		return nil
	}
	defer res.Body.Close()

	bodyBytes, _ := io.ReadAll(res.Body)
	var r map[string]interface{}
	json.Unmarshal(bodyBytes, &r)

	hitsWrapper, ok := r["hits"].(map[string]interface{})
	if !ok {
		return nil
	}
	hits, ok := hitsWrapper["hits"].([]interface{})
	if !ok || len(hits) == 0 {
		return nil
	}

	type esHit struct {
		url   string
		score float64
	}
	var esHits []esHit
	scoreMap := map[string]float64{}

	for _, hit := range hits {
		h, _ := hit.(map[string]interface{})
		score, _ := h["_score"].(float64)
		src, _ := h["_source"].(map[string]interface{})
		
		// FILTRE : On ignore les résultats avec un score trop bas (ex: < 5.0)
		if u, _ := src["url"].(string); u != "" && score > 5.0 {
			esHits = append(esHits, esHit{url: u, score: score})
			scoreMap[u] = score
		}
	}

	if len(esHits) == 0 {
		return nil
	}

	log.Printf("⚡ [ES] %d résultats pertinents pour '%s' (score max: %.2f)", len(esHits), query, esHits[0].score)

	// Suite du code pour récupérer les détails depuis PostgreSQL
	urls := make([]string, len(esHits))
	for i, h := range esHits {
		urls[i] = h.url
	}

	sqlStr := `SELECT id, name, price, url, image_url, source, category FROM products WHERE url = ANY($1)`
	args := []interface{}{pq.Array(urls)}
	
	rows, err := db.Query(sqlStr, args...)
	if err != nil {
		return nil
	}
	defer rows.Close()

	productMap := map[string]models.Product{}
	for rows.Next() {
		var p models.Product
		if err := rows.Scan(&p.ID, &p.Name, &p.Price, &p.URL, &p.Image, &p.Source, &p.Category); err == nil {
			p.Score = scoreMap[p.URL]
			productMap[p.URL] = p
		}
	}

	var products []models.Product
	for _, h := range esHits {
		if p, ok := productMap[h.url]; ok {
			products = append(products, p)
		}
	}
	return products
}

// ─── Récupération par catégorie paginée ──────────────────────────────────────

func fetchFromPostgresByCategoryPaginated(db *sql.DB, category string, page int, limit int) ([]models.Product, int, error) {
	offset := (page - 1) * limit

	var totalCount int
	err := db.QueryRow(
		`SELECT COUNT(*) FROM products WHERE LOWER(category) = $1`,
		strings.ToLower(category),
	).Scan(&totalCount)
	if err != nil {
		return nil, 0, err
	}

	rows, err := db.Query(
		`SELECT id, name, price, url, image_url, source, category
		 FROM products
		 WHERE LOWER(category) = $1
		 ORDER BY updated_at DESC
		 LIMIT $2 OFFSET $3`,
		strings.ToLower(category), limit, offset,
	)
	if err != nil {
		return nil, totalCount, err
	}
	defer rows.Close()

	var products []models.Product
	for rows.Next() {
		var p models.Product
		if err := rows.Scan(&p.ID, &p.Name, &p.Price, &p.URL, &p.Image, &p.Source, &p.Category); err == nil {
			products = append(products, p)
		}
	}
	return products, totalCount, nil
}

// ─── Handlers Echo ───────────────────────────────────────────────────────────

func GetProductsFromDB(db *sql.DB, es *elasticsearch.Client) echo.HandlerFunc {
	return func(c echo.Context) error {
		query    := c.QueryParam("search")
		category := c.QueryParam("category")

		page := 1
		if p := c.QueryParam("page"); p != "" {
			if parsed, err := strconv.Atoi(p); err == nil && parsed > 0 {
				page = parsed
			}
		}
		limit := 24
		if l := c.QueryParam("limit"); l != "" {
			if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
				limit = parsed
			}
		}

		// Recherche textuelle → ES
		if strings.TrimSpace(query) != "" {
			results := searchViaElasticsearch(es, db, query, category)
			if len(results) > 0 {
				log.Printf("⚡ [Cache] %d produits via ES pour '%s'", len(results), query)
				return c.JSON(http.StatusOK, map[string]interface{}{
					"products":    results,
					"total_count": len(results),
					"total_pages": 1,
					"from_cache":  true,
				})
			}

			// Cache miss → scraping en arrière-plan
			log.Printf("🔍 [Scraping] '%s' non trouvé → scraping...", query)
			go scrapeByReference(db, es, query, category)

			return c.JSON(http.StatusAccepted, map[string]interface{}{
				"message":     "Recherche en cours, réessayez dans quelques secondes...",
				"products":    []models.Product{},
				"total_count": 0,
				"total_pages": 1,
			})
		}

		// Navigation par catégorie → PostgreSQL paginé
		if category != "" {
			products, totalCount, err := fetchFromPostgresByCategoryPaginated(db, category, page, limit)
			if err != nil {
				return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Erreur serveur"})
			}

			totalPages := (totalCount + limit - 1) / limit
			if totalPages == 0 {
				totalPages = 1
			}

			log.Printf("⚡ [Catégorie] %d/%d produits (page %d) pour '%s'", len(products), totalCount, page, category)

			return c.JSON(http.StatusOK, map[string]interface{}{
				"products":    products,
				"total_count": totalCount,
				"page":        page,
				"limit":       limit,
				"total_pages": totalPages,
				"from_cache":  true,
			})
		}

		return c.JSON(http.StatusOK, map[string]interface{}{
			"products":    []models.Product{},
			"total_count": 0,
			"total_pages": 1,
		})
	}
}

func GetProductDetail(db *sql.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		rawID := c.Param("id")
		productURL, _ := url.QueryUnescape(rawID)

		var p models.Product
		err := db.QueryRow(
			`SELECT id, name, price, url, image_url, source, category
             FROM products WHERE url = $1`,
			productURL,
		).Scan(&p.ID, &p.Name, &p.Price, &p.URL, &p.Image, &p.Source, &p.Category)

		if err == sql.ErrNoRows {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "Produit non trouvé"})
		}
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Erreur serveur"})
		}

		return c.JSON(http.StatusOK, p)
	}
}

func GetCategories(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]interface{}{"categories": scraper.GetCategories()})
}



func GetAutocompleteSuggestions(es *elasticsearch.Client) echo.HandlerFunc {
    return func(c echo.Context) error {
        query := c.QueryParam("q")
        if query == "" {
            return c.JSON(http.StatusOK, []string{})
        }

        // Requête Elasticsearch pour suggérer les noms de produits
        // Utilisation de match_phrase_prefix pour l'autocomplete
        searchQuery := fmt.Sprintf(`{
            "query": {
                "match_phrase_prefix": {
                    "name": "%s"
                }
            },
            "size": 5,
            "_source": ["name"]
        }`, query)

        res, err := es.Search(
            es.Search.WithIndex("products"),
            es.Search.WithBody(strings.NewReader(searchQuery)),
        )
        if err != nil || res.IsError() {
            return c.JSON(http.StatusInternalServerError, "Erreur ES")
        }
        defer res.Body.Close()

        var r map[string]interface{}
        json.NewDecoder(res.Body).Decode(&r)

        var suggestions []string
        for _, hit := range r["hits"].(map[string]interface{})["hits"].([]interface{}) {
            name := hit.(map[string]interface{})["_source"].(map[string]interface{})["name"].(string)
            suggestions = append(suggestions, name)
        }

        return c.JSON(http.StatusOK, suggestions)
    }
}



func getVectorFromPython(text string) ([]float32, error) {
    jsonData, _ := json.Marshal(map[string]string{"text": text})
    // Notez l'URL : on utilise le nom du service "nlp-service" défini dans docker-compose
    resp, err := http.Post("http://nlp-service:8000/vectorize", "application/json", bytes.NewBuffer(jsonData))
    if err != nil { return nil, err }
    defer resp.Body.Close()
    
    var result struct { Vector []float32 `json:"vector"` }
    json.NewDecoder(resp.Body).Decode(&result)
    return result.Vector, nil
}




