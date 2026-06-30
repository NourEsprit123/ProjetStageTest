package handlers

import (
	"net/http"
	"strings"
	"sync"

	"github.com/labstack/echo/v4"
	"tunisianet-scraper/models"
	"tunisianet-scraper/scraper"
)

// categoryKeywords : mots-clés utilisés pour vérifier qu'un produit appartient
// réellement à la catégorie sélectionnée, même quand le scraping vient d'une
// recherche texte (cas où query ET category sont fournis en même temps).
// Complète cette liste au fur et à mesure que tu ajoutes des catégories.
var categoryKeywords = map[string][]string{
	"smartphones": {
		"smartphone", "telephone", "téléphone", "mobile", "coque",
		"silicone", "ecran", "écran", "galaxy", "iphone", "redmi",
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

// matchesCategory vérifie si un produit correspond réellement à la catégorie demandée,
// en se basant sur son nom (et son champ Category s'il est déjà renseigné correctement).
func matchesCategory(product models.Product, category string) bool {
	catLower := strings.ToLower(category)
	nameLower := strings.ToLower(product.Name)

	keywords, ok := categoryKeywords[catLower]
	if !ok {
		// Pas de mots-clés définis pour cette catégorie : on ne filtre pas
		// (on fait confiance au scraping, qui a déjà ciblé la bonne page/catégorie).
		return true
	}

	for _, kw := range keywords {
		if strings.Contains(nameLower, kw) {
			return true
		}
	}
	return false
}

// matchesQuery vérifie que le nom du produit contient bien le terme recherché.
// Utile quand category+query sont combinés, car certains scrapers (page catégorie)
// ne filtrent pas eux-mêmes par texte libre.
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

	wg.Add(2)

	// IMPORTANT : on passe query et category SÉPARÉMENT et tels quels aux scrapers.
	// C'est buildURL (côté scraper) qui décide intelligemment :
	//   - s'il y a un texte de recherche -> recherche texte
	//   - sinon, s'il y a une catégorie connue -> page catégorie dédiée
	// Ne JAMAIS remplacer query par category ici, ça casse cette logique.
	go func() {
		defer wg.Done()
		products, err := scraper.ScrapeProducts(query, category)
		if err != nil {
			println("Erreur Tunisianet (résultats partiels conservés):", err.Error())
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
			println("Erreur Mytek (résultats partiels conservés):", err.Error())
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
	query := c.QueryParam("search")
	category := c.QueryParam("category")

	rawProducts := scrapeAllSources(query, category)

	var filteredProducts []models.Product
	for _, product := range rawProducts {
		if category != "" && !matchesCategory(product, category) {
			continue
		}
		if query != "" && !matchesQuery(product, query) {
			continue
		}
		filteredProducts = append(filteredProducts, product)
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"products": filteredProducts,
		"total":    len(filteredProducts),
		"query":    query,
		"category": category,
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