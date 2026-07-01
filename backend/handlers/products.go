package handlers

import (
	"net/http"
	"strings"
	"sync"

	"github.com/labstack/echo/v4"
	"tunisianet-scraper/models"
	"tunisianet-scraper/scraper"
)

var categoryKeywords = map[string][]string{
	"smartphones": {
		"smartphone", "telephone", "téléphone", "mobile", "coque",
		"silicone", "ecran", "écran", "galaxy", "iphone", "redmi", "honor", "vivo", "infinix",
	},
	"telephonie portables": {
		"telephone", "téléphone", "smartphone", "mobile",
	},
	"informatique": {
		"ordinateur", "pc", "laptop", "portable", "informatique",
	},
	"ordinateurs": {
		"ordinateur", "pc", "laptop", "portable",
	},
	"composants": {
		"carte mere", "carte graphique", "processeur", "ram", "ventilateur",
		"alimentation", "boitier", "composant",
	},
	"reseaux": {
		"routeur", "switch", "reseau", "réseau", "wifi", "câble", "cable",
		"access point", "modem",
	},
	"peripheriques": {
		"souris", "clavier", "casque", "webcam", "imprimante", "scanner",
	},
	"stockage": {
		"disque dur", "ssd", "hdd", "stockage", "cle usb", "clé usb",
	},
	"electromenager": {
		"refrigerateur", "réfrigérateur", "congelateur", "congélateur",
		"climatiseur", "four", "lave", "seche", "sèche", "aspirateur",
	},
}

func matchesCategory(product models.Product, category string) bool {
	catLower := strings.ToLower(category)
	nameLower := strings.ToLower(product.Name)

	keywords, ok := categoryKeywords[catLower]
	if !ok {
		return true
	}

	for _, kw := range keywords {
		if strings.Contains(nameLower, kw) {
			return true
		}
	}
	return false
}

func matchesQuery(product models.Product, query string) bool {
	if query == "" {
		return true
	}
	return strings.Contains(strings.ToLower(product.Name), strings.ToLower(query))
}

func scrapeAllSources(query string, category string) []models.Product {
	var allProducts []models.Product
	var mu sync.Mutex
	var wg sync.WaitGroup

	wg.Add(3)

	go func() {
		defer wg.Done()
		products, err := scraper.ScrapeProducts(query, category)
		if err != nil {
			println("Erreur Tunisianet:", err.Error())
		}
		if len(products) > 0 {
			mu.Lock()
			allProducts = append(allProducts, products...)
			mu.Unlock()
		}
	}()

	go func() {
		defer wg.Done()
		products, err := scraper.ScrapeMytekProducts(query, category)
		if err != nil {
			println("Erreur Mytek:", err.Error())
		}
		if len(products) > 0 {
			mu.Lock()
			allProducts = append(allProducts, products...)
			mu.Unlock()
		}
	}()

	go func() {
		defer wg.Done()
		products, err := scraper.ScrapeWikiProducts(query, category)
		if err != nil {
			println("Erreur Wiki.tn:", err.Error())
		}
		if len(products) > 0 {
			mu.Lock()
			allProducts = append(allProducts, products...)
			mu.Unlock()
		}
	}()

	wg.Wait()
	return allProducts
}

func GetProducts(c echo.Context) error {
	category := c.QueryParam("category")
	search := c.QueryParam("search")

	allProducts := scrapeAllSources(search, category)

	return c.JSON(http.StatusOK, map[string]interface{}{
		"products": allProducts,
	})
}

func GetCategories(c echo.Context) error {
	categories := scraper.GetCategories()
	return c.JSON(http.StatusOK, map[string]interface{}{
		"categories": categories,
	})
}

func GetProductByID(c echo.Context) error {
	id := c.Param("id")
	query := c.QueryParam("search")
	category := c.QueryParam("category")

	products := scrapeAllSources(query, category)

	for _, product := range products {
		if product.ID == id {
			return c.JSON(http.StatusOK, product)
		}
	}

	return c.JSON(http.StatusNotFound, map[string]string{
		"error": "Produit non trouvé",
	})
}