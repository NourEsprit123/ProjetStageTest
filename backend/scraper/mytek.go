package scraper

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"tunisianet-scraper/models"
)

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

	searchQuery := ""
	if category != "" {
		if kw, ok := mytekCategoryKeywords[category]; ok {
			searchQuery = kw
		} else {
			searchQuery = category
		}
	} else if query != "" {
		searchQuery = query
	} else {
		searchQuery = "pc"
	}

	fmt.Printf("🔍 [Mytek API] Lancement pour: '%s' | Catégorie: '%s'\n", searchQuery, category)

	for page := 1; page <= 3; page++ {
		products, err := fetchMytekAPIPage(client, searchQuery, category, page)
		if err != nil || len(products) == 0 {
			break
		}

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

	req, _ := http.NewRequest("GET", apiURL, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0")
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var data struct {
		Items []struct {
			ID               int     `json:"id"`
			Name             string  `json:"name"`
			Price            float64 `json:"price"`
			CustomAttributes []struct {
				AttributeCode string      `json:"attribute_code"`
				Value         interface{} `json:"value"`
			} `json:"custom_attributes"`
		} `json:"items"`
	}

	if err := json.Unmarshal(body, &data); err != nil {
		return nil, err
	}

	for _, item := range data.Items {
		p := models.Product{
			ID:       fmt.Sprintf("mytek-%d", item.ID),
			Name:     item.Name,
			Price:    fmt.Sprintf("%.2f TND", item.Price),
			Category: category,
			Source:   "mytek",
			InStock:  false, // Valeur par défaut prudente
		}

		for _, attr := range item.CustomAttributes {
			switch attr.AttributeCode {
			case "image":
				if valStr, ok := attr.Value.(string); ok {
					p.Image = "https://www.mytek.tn/media/catalog/product" + valStr
				}
			case "url_key":
				if valStr, ok := attr.Value.(string); ok {
					p.URL = "https://www.mytek.tn/" + valStr + ".html"
				}
			case "quantity_and_stock_status":
				// Extraction complexe du statut stock
				if stockMap, ok := attr.Value.(map[string]interface{}); ok {
					if is_in_stock, exists := stockMap["is_in_stock"]; exists {
						// Magento renvoie souvent 1 pour true
						if val, ok := is_in_stock.(bool); ok {
							p.InStock = val
						} else if valFloat, ok := is_in_stock.(float64); ok {
							p.InStock = (valFloat == 1)
						}
					}
				}
			}
		}

		fmt.Printf("DEBUG: %s | En stock: %v\n", p.Name, p.InStock)
		results = append(results, p)
	}

	return results, nil
}