package scraper

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
	"strings"

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

var mytekCategoryKeywords = map[string]string{
	"smartphones":          "telephone",
	"telephonie portables": "telephone",
	"informatique":         "pc portable",
	"ordinateurs":          "ordinateur",
	"composants":           "composant",
	"reseaux":              "reseau",
	"peripheriques":        "peripherique",
	"stockage":             "stockage",
	"electromenager":       "electromenager",
}

func ScrapeMytekProducts(query string, category string) ([]models.Product, error) {
	var allProducts []models.Product
	client := &http.Client{Timeout: 15 * time.Second}

	// 1. Logique de priorité :
	// Si une catégorie est définie, on force le mot-clé correspondant.
	// On ignore la 'query' car elle vient souvent d'une recherche textuelle globale.
	searchQuery := ""
	if category != "" {
		if kw, ok := mytekCategoryKeywords[category]; ok {
			searchQuery = kw
		} else {
			searchQuery = category
		}
	} else if query != "" {
		// On n'utilise la query que si aucune catégorie n'est précisée
		searchQuery = query
	} else {
		searchQuery = "pc" // Valeur de repli
	}

	fmt.Printf("🔍 [Mytek API] Lancement pour: '%s' | Catégorie: '%s'\n", searchQuery, category)

	for page := 1; page <= 5; page++ {
		products, err := fetchMytekAPIPage(client, searchQuery, category, page)
		if err != nil {
			break
		}
		if len(products) == 0 {
			break
		}

		// 2. FILTRAGE POST-RÉCUPÉRATION
		// Même si l'API Mytek renvoie des résultats, on vérifie manuellement 
		// si le nom du produit contient bien le mot-clé de la catégorie.
		if category != "" {
			for _, p := range products {
				if strings.Contains(strings.ToLower(p.Name), strings.ToLower(searchQuery)) {
					allProducts = append(allProducts, p)
				}
			}
		} else {
			allProducts = append(allProducts, products...)
		}

		time.Sleep(1 * time.Second)
	}

	return allProducts, nil
}
func fetchMytekAPIPage(client *http.Client, query string, category string, page int) ([]models.Product, error) {
	var results []models.Product

	apiURL := fmt.Sprintf(
		"https://www.mytek.tn/rest/V1/products?searchCriteria[filter_groups][0][filters][0][field]=name&searchCriteria[filter_groups][0][filters][0][value]=%%25%s%%25&searchCriteria[filter_groups][0][filters][0][condition_type]=like&searchCriteria[pageSize]=24&searchCriteria[currentPage]=%d",
		url.QueryEscape(query), page,
	)

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

	fmt.Printf("[Mytek API Debug] %d produits extraits (Page %d).\n", len(results), page)
	return results, nil
}

func fetchMytekViaStaticParsing(client *http.Client, term string, page int) ([]models.Product, error) {
	return []models.Product{}, nil
}