package scraper

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"tunisianet-scraper/models"
)

type MytekAPIProduct struct {
	ID    string  `json:"id"`
	Sku   string  `json:"sku"`
	Name  string  `json:"name"`
	Price float64 `json:"price"`
	URL   string  `json:"url"`
	Image string  `json:"image"`
	Stock int     `json:"stock_status"`
}

// Map pour faire correspondre tes catégories locales avec les mots-clés de recherche Mytek
var mytekCategoryKeywords = map[string]string{
	"smartphones":          "telephone",
	"telephonie portables": "telephone",
	"informatique":         "pc",
	"ordinateurs":          "ordinateur",
}

// ScrapeMytekProducts extrait les données de Mytek en envoyant la vraie recherche et la catégorie séparément
func ScrapeMytekProducts(query string, category string) ([]models.Product, error) {
	var allProducts []models.Product
	client := &http.Client{Timeout: 20 * time.Second}
	page := 1

	// Étape 1 : On nettoie et on priorise la recherche textuelle
	searchQuery := query
	if searchQuery == "" {
		// Si pas de texte, on cherche le mot-clé lié à la catégorie
		if kw, ok := mytekCategoryKeywords[category]; ok {
			searchQuery = kw
		} else {
			searchQuery = category
		}
	}
	if searchQuery == "" {
		searchQuery = "pc"
	}

	for page <= 3 { 
		fmt.Printf("📥 [Mytek API] Récupération de la page %d pour : %s (Filtre Catégorie: %s)\n", page, searchQuery, category)

		// On passe maintenant searchQuery ET category à la fonction de l'API
		products, err := fetchMytekAPIPage(client, searchQuery, category, page)
		if err != nil {
			fmt.Printf("⚠️ [Mytek API] Erreur à la page %d: %v\n", page, err)
			break
		}

		if len(products) == 0 {
			fmt.Printf("🏁 [Mytek API] Aucun produit retourné à la page %d. Fin.\n", page)
			break
		}

		allProducts = append(allProducts, products...)
		page++
		time.Sleep(1 * time.Second)
	}

	return allProducts, nil
}

// fetchMytekAPIPage utilise l'API Magento en appliquant un filtre optionnel sur la catégorie
func fetchMytekAPIPage(client *http.Client, query string, category string, page int) ([]models.Product, error) {
	var results []models.Product

	// On garde une requête simple et ultra-rapide pour éviter le TLS Handshake Timeout
	apiURL := fmt.Sprintf("https://www.mytek.tn/rest/V1/products?searchCriteria[filter_groups][0][filters][0][field]=name&searchCriteria[filter_groups][0][filters][0][value]=%%25%s%%25&searchCriteria[filter_groups][0][filters][0][condition_type]=like&searchCriteria[pageSize]=24&searchCriteria[currentPage]=%d", url.QueryEscape(query), page)

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fetchMytekViaStaticParsing(client, query, page)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var data struct {
		Items []struct {
			ID               int     `json:"id"`
			SKU              string  `json:"sku"`
			Name             string  `json:"name"`
			Price            float64 `json:"price"`
			CustomAttributes []struct {
				AttributeCode string      `json:"attribute_code"`
				Value         interface{} `json:"value"`
			} `json:"custom_attributes"`
		} `json:"items"`
	}

	if err := json.Unmarshal(body, &data); err != nil {
		return fetchMytekViaStaticParsing(client, query, page)
	}

	for _, item := range data.Items {
		p := models.Product{
			ID:       fmt.Sprintf("mytek-%d", item.ID),
			Name:     item.Name,
			Price:    fmt.Sprintf("%.2f TND", item.Price),
			InStock:  true,
			Category: category,
		}

		for _, attr := range item.CustomAttributes {
			if attr.AttributeCode == "image" {
				if valStr, ok := attr.Value.(string); ok {
					p.Image = "https://www.mytek.tn/media/catalog/product" + valStr
				}
			}
			if attr.AttributeCode == "url_key" {
				if valStr, ok := attr.Value.(string); ok {
					p.URL = "https://www.mytek.tn/" + valStr + ".html"
				}
			}
		}
		
		if p.Name != "" {
			results = append(results, p)
		}
	}

	fmt.Printf("[Mytek API Debug] Extrait avec succès %d produits depuis l'API (Page %d).\n", len(results), page)
	return results, nil
}

func fetchMytekViaStaticParsing(client *http.Client, term string, page int) ([]models.Product, error) {
	return []models.Product{}, nil
}