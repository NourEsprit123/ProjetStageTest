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

// Normalise une chaîne pour servir de clé de cache stable (utilisée
// comme "reference" quand on sauvegarde les produits scrapés).
func normalizeCacheKey(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.Join(strings.Fields(s), " ") // espaces multiples -> un seul
	return s
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

	// Indexation SYNCHRONE dans ES.
	// ⚠️ IMPORTANT : on utilise models.DocIDFromURL(p.URL) comme _id,
	// PAS p.URL directement, car ES rejette silencieusement (400) les IDs
	// contenant des "/" bruts comme dans une URL complète.
	if es != nil {
		data, _ := json.Marshal(p)
		res, err := es.Index("products", bytes.NewReader(data), es.Index.WithDocumentID(models.DocIDFromURL(p.URL)))
		if err != nil {
			log.Printf("⚠️ Erreur transport ES pour %s: %v", p.Name, err)
		} else if res.IsError() {
			bodyBytes, _ := io.ReadAll(res.Body)
			log.Printf("⚠️ ES a rejeté le document '%s': status=%d body=%s", p.Name, res.StatusCode, string(bodyBytes))
			res.Body.Close()
		} else {
			res.Body.Close()
		}
	}
	return nil
}

// Scrape les sites et filtre les résultats pour ne garder que ceux correspondant à la référence
func scrapeByReference(db *sql.DB, es *elasticsearch.Client, reference string, category string) []models.Product {
	var allProducts []models.Product
	var mu sync.Mutex
	var wg sync.WaitGroup

	// Clé de cache normalisée : c'est CETTE valeur qui est sauvegardée comme
	// "reference" en base et dans ES, donc c'est aussi celle qu'on doit
	// utiliser pour chercher au prochain appel identique.
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

// Recherche dans ES : utilise "filter" pour une recherche stricte sur référence
// et applique aussi le filtre de catégorie quand il est fourni.
func searchViaElasticsearch(es *elasticsearch.Client, db *sql.DB, query string, category string) []models.Product {
	if es == nil {
		log.Printf("🔎 [ES DEBUG][v3] client ES nil")
		return nil
	}

	q := normalizeCacheKey(query)
	if q == "" {
		return nil
	}

	filters := []map[string]interface{}{}
	if category != "" {
		filters = append(filters, map[string]interface{}{
			"term": map[string]interface{}{"category": category},
		})
	}

	var mainClause map[string]interface{}
	if isReference(q) {
		filters = append(filters, map[string]interface{}{
			"term": map[string]interface{}{"reference.keyword": q},
		})
		mainClause = map[string]interface{}{"match_all": map[string]interface{}{}}
	} else {
		mainClause = map[string]interface{}{
			"match": map[string]interface{}{
				"name": map[string]interface{}{"query": q, "fuzziness": "AUTO"},
			},
		}
	}

	esQuery := map[string]interface{}{
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"must":   mainClause,
				"filter": filters,
			},
		},
		"size": 200,
	}

	var buf bytes.Buffer
	json.NewEncoder(&buf).Encode(esQuery)

	log.Printf("🔎 [ES DEBUG][v3] query envoyée: %s", buf.String())

	res, err := es.Search(es.Search.WithIndex("products"), es.Search.WithBody(&buf))
	if err != nil {
		log.Printf("⚠️ [ES DEBUG][v3] erreur recherche ES: %v", err)
		return nil
	}
	defer res.Body.Close()

	if res.IsError() {
		log.Printf("⚠️ [ES DEBUG][v3] réponse ES en erreur: %s", res.String())
		return nil
	}

	var r map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&r); err != nil {
		log.Printf("⚠️ [ES DEBUG][v3] erreur décodage réponse: %v", err)
		return nil
	}

	hitsWrapper, ok := r["hits"].(map[string]interface{})
	if !ok {
		log.Printf("⚠️ [ES DEBUG][v3] pas de champ 'hits' dans la réponse: %+v", r)
		return nil
	}
	hits, ok := hitsWrapper["hits"].([]interface{})
	if !ok {
		log.Printf("⚠️ [ES DEBUG][v3] hits.hits n'est pas un tableau")
		return nil
	}

	log.Printf("🔎 [ES DEBUG][v3] nombre de hits ES: %d", len(hits))

	if len(hits) == 0 {
		return nil
	}

	var urls []string
	for _, hit := range hits {
		src, ok := hit.(map[string]interface{})["_source"].(map[string]interface{})
		if !ok {
			continue
		}
		if u, ok := src["url"].(string); ok && u != "" {
			urls = append(urls, u)
		}
	}
	if len(urls) == 0 {
		log.Printf("⚠️ [ES DEBUG][v3] hits trouvés mais aucune URL extraite")
		return nil
	}

	rows, err := db.Query(
		"SELECT id, name, price, url, image_url, source, category, reference FROM products WHERE url = ANY($1)",
		pq.Array(urls),
	)
	if err != nil {
		log.Printf("⚠️ [ES DEBUG][v3] erreur requête Postgres: %v", err)
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
	log.Printf("🔎 [ES DEBUG][v3] produits récupérés depuis Postgres: %d", len(products))
	return products
}

// Récupération PAGINÉE par catégorie, avec le total réel de produits.
// Remplace l'ancienne fetchFromPostgresByCategory qui avait un LIMIT 100 fixe
// (ce qui donnait toujours 5 pages de pagination côté frontend, peu importe
// le nombre réel de produits dans la catégorie).
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
		`SELECT id, name, price, url, image_url, source, category, reference
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
		if err := rows.Scan(&p.ID, &p.Name, &p.Price, &p.URL, &p.Image, &p.Source, &p.Category, &p.Reference); err == nil {
			products = append(products, p)
		}
	}
	return products, totalCount, nil
}

// Handler Echo pour l'API
func GetProductsFromDB(db *sql.DB, es *elasticsearch.Client) echo.HandlerFunc {
	return func(c echo.Context) error {
		query := c.QueryParam("search")
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

		// 1. Recherche textuelle -> cache ES (comportement inchangé)
		if strings.TrimSpace(query) != "" {
			results := searchViaElasticsearch(es, db, query, category)
			if len(results) > 0 {
				log.Printf("⚡ [Cache] %d produits trouvés via ES pour '%s'", len(results), query)
				return c.JSON(http.StatusOK, map[string]interface{}{
					"products":    results,
					"total_count": len(results),
					"from_cache":  true,
				})
			}

			log.Printf("🔍 [Scraping] Aucun résultat en cache pour '%s'. Démarrage du scraper...", query)
			scraped := scrapeByReference(db, es, query, category)
			return c.JSON(http.StatusOK, map[string]interface{}{
				"products":    scraped,
				"total_count": len(scraped),
				"from_cache":  false,
			})
		}

		// 2. Navigation par catégorie seule -> pagination serveur, rapide et complète
		if category != "" {
			products, totalCount, err := fetchFromPostgresByCategoryPaginated(db, category, page, limit)
			if err != nil {
				return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Erreur serveur"})
			}
			log.Printf("⚡ [Cache catégorie] %d/%d produits (page %d) pour '%s'", len(products), totalCount, page, category)
			return c.JSON(http.StatusOK, map[string]interface{}{
				"products":    products,
				"total_count": totalCount,
				"page":        page,
				"limit":       limit,
				"from_cache":  true,
			})
		}

		// 3. Ni query ni catégorie -> rien à faire
		return c.JSON(http.StatusOK, map[string]interface{}{"products": []models.Product{}, "total_count": 0, "from_cache": true})
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

// Handler pour les catégories
func GetCategories(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]interface{}{"categories": scraper.GetCategories()})
}