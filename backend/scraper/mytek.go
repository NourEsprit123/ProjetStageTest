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

	"github.com/PuerkitoBio/goquery"
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
	"informatique":         "pc",
	"ordinateurs":          "ordinateur",
}

func ScrapeMytekProducts(query string, category string) ([]models.Product, error) {
	var allProducts []models.Product
	// 1. Augmentation du timeout à 10 secondes pour laisser le temps à l'API Magento
	client := &http.Client{Timeout: 10 * time.Second}
	page := 1

	searchQuery := query
	if searchQuery == "" {
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
		fmt.Printf("📥 [Mytek] Récupération de la page %d pour : %s (Filtre Catégorie: %s)\n", page, searchQuery, category)

		products, err := fetchMytekAPIPage(client, searchQuery, category, page)
		if err != nil {
			// Si l'API échoue ou expire, on tente IMMÉDIATEMENT le plan B (Scraping HTML)
			fmt.Printf("⚠️ [Mytek API] Échec API (Erreur: %v). Tentative de repli vers le scraping HTML...\n", err)
			products, err = fetchMytekViaStaticParsing(client, searchQuery, page)
			if err != nil {
				fmt.Printf("❌ [Mytek HTML] Échec du repli HTML à la page %d: %v\n", page, err)
				break
			}
		}

		if len(products) == 0 {
			fmt.Printf("🏁 [Mytek] Aucun produit trouvé à la page %d. Fin.\n", page)
			break
		}

		allProducts = append(allProducts, products...)
		page++
		time.Sleep(1 * time.Second)
	}

	return allProducts, nil
}

func fetchMytekAPIPage(client *http.Client, query string, category string, page int) ([]models.Product, error) {
	var results []models.Product

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
		return nil, err // Retourne l'erreur pour activer le bloc 'if err != nil' dans ScrapeMytekProducts
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("statut HTTP API non-valide: %d", resp.StatusCode)
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
		return nil, err
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

// 2. Implémentation du scraping HTML statique en cas de panne de l'API
func fetchMytekViaStaticParsing(client *http.Client, term string, page int) ([]models.Product, error) {
	var results []models.Product
	
	// URL de recherche publique HTML standard (très rapide grâce au cache vernish du site)
	searchURL := fmt.Sprintf("https://www.mytek.tn/catalogsearch/result/index/?q=%s&p=%d", url.QueryEscape(term), page)
	
	req, err := http.NewRequest("GET", searchURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("statut HTTP de repli non-valide: %d", resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, err
	}

	// Parsing des produits sur la page HTML standard de MyTek (Magento Luma Theme)
	doc.Find("li.product-item").Each(func(i int, s *goquery.Selection) {
		nameEl := s.Find(".product-item-link")
		name := strings.TrimSpace(nameEl.Text())
		
		prodURL, _ := nameEl.Attr("href")
		
		price := strings.TrimSpace(s.Find("[data-price-type='finalPrice'] .price").Text())
		if price == "" {
			price = strings.TrimSpace(s.Find(".price-box .price").Text())
		}
		
		imgEl := s.Find(".product-image-photo")
		imgURL, _ := imgEl.Attr("src")
		if imgURL == "" {
			imgURL, _ = imgEl.Attr("data-src")
		}

		// Extraction de l'ID depuis l'attribut data-product-id
		id, _ := s.Find(".price-box").Attr("data-product-id")
		if id == "" {
			id = fmt.Sprintf("mytek-html-%d-%d", page, i)
		} else {
			id = "mytek-" + id
		}

		if name != "" {
			results = append(results, models.Product{
				ID:       id,
				Name:     name,
				Price:    price,
				Image:    imgURL,
				URL:      prodURL,
				InStock:  true, // Par défaut dispo ou à affiner selon classe CSS
				Category: "",
			})
		}
	})

	fmt.Printf("[Mytek HTML Debug] Extrait %d produits depuis la page de recherche HTML (Page %d).\n", len(results), page)
	return results, nil
}