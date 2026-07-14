package handlers

import (
    "net/http"
    "github.com/labstack/echo/v4"
    "tunisianet-scraper/db"
    "tunisianet-scraper/models"
    "regexp"
    "strings"
	"tunisianet-scraper/scraper"
	"fmt"
)
var categoryKeywords = map[string][]string{
	"smartphones":          {"smartphone", "telephone", "téléphone", "mobile", "coque", "silicone", "ecran", "écran", "galaxy", "iphone", "redmi", "honor", "vivo", "infinix"},
	"telephonie portables": {"telephone", "téléphone", "smartphone", "mobile"},
	"informatique":         {"ordinateur", "pc", "laptop", "portable", "informatique", "gamer", "lenovo", "hp", "dell", "asus", "acer", "msi"},
	"ordinateurs":          {"ordinateur", "pc", "laptop", "portable", "lenovo", "hp", "dell", "asus", "acer", "msi"},
	"composants":           {"carte mere", "carte graphique", "processeur", "ram", "ventilateur", "alimentation", "boitier", "composant"},
	"reseaux":              {"routeur", "switch", "reseau", "réseau", "wifi", "câble", "cable", "access point", "modem"},
	"peripheriques":        {"souris", "clavier", "casque", "webcam", "imprimante", "scanner"},
	"stockage":             {"disque dur", "ssd", "hdd", "stockage", "cle usb", "clé usb", "sandisk", "kingston"},
	"electromenager":       {"refrigerateur", "réfrigérateur", "congelateur", "congélateur", "climatiseur", "four", "lave", "seche", "sèche", "aspirateur"},
}

var intrusBlacklist = []string{
	"chaussures", "banquette", "lisseur", "distributeur", "pancake", "cupcake",
	"moule", "brosse chauffante", "epilateur", "rasoir", "tondeuse", "poele", "casserole",
}

// 💡 Version ultra-robuste de la normalisation (gère le "–" de Wiki.tn)
func normalizeText(text string) string {
	text = strings.ToLower(text)
	
	// Remplace tous les types de tirets, slashs, pipes, plus et points par un espace
	reg := regexp.MustCompile(`[\/\-_—–\|\+,\.]`)
	text = reg.ReplaceAllString(text, " ")
	
	// Supprime les espaces multiples et uniformise
	return strings.Join(strings.Fields(text), " ")
}

func matchesCategory(product models.Product, category string) bool {
	if category == "" {
		return true
	}

	catLower := strings.ToLower(category)
	nameNormalized := normalizeText(product.Name)

	keywords, ok := categoryKeywords[catLower]
	if !ok {
		return true
	}

	for _, kw := range keywords {
		if kw == "pc" {
			matched, _ := regexp.MatchString(`\bpc\b|\bpc-portable\b`, nameNormalized)
			if matched {
				return true
			}
		} else {
			if strings.Contains(nameNormalized, normalizeText(kw)) {
				return true
			}
		}
	}
	return false
}

// 💡 Validation finale par jetons (tokens)
func matchesQuery(product models.Product, query string) bool {
	if query == "" {
		return true
	}

	productNameNormalized := normalizeText(product.Name)
	queryWords := strings.Fields(normalizeText(query))

	// Chaque mot-clé important de la recherche doit se retrouver dans le nom du produit
	for _, word := range queryWords {
		// On ignore les petits bruits structurels
		if word == "avec" || word == "pour" || word == "dans" || len(word) <= 1 {
			continue
		}
		
		if !strings.Contains(productNameNormalized, word) {
			return false 
		}
	}
	return true
}

func isBlacklisted(name string) bool {
	nameLower := strings.ToLower(name)
	for _, word := range intrusBlacklist {
		if strings.Contains(nameLower, word) {
			return true
		}
	}
	return false
}



// 💡 Limiter le nombre de mots pour éviter de saturer les moteurs internes de Mytek et Wiki
func limitQueryWords(query string, maxWords int) string {
	words := strings.Fields(query)
	if len(words) > maxWords {
		return strings.Join(words[:maxWords], " ")
	}
	return query
}

// 💡 Nettoyage complet des bruits textuels d'origine
func cleanQueryForScrapers(query string) string {
	if query == "" {
		return ""
	}
	
	q := strings.ToLower(query)
	
	generics := []string{"pc portable", "ordinateur portable", "pc", "laptop", "avec", "sacoche", "souris"}
	for _, gen := range generics {
		q = strings.ReplaceAll(q, gen, "")
	}
	
	reg := regexp.MustCompile(`[\/\-_,\+]`)
	q = reg.ReplaceAllString(q, " ")
	
	return strings.Join(strings.Fields(q), " ")
}

// Remplacez votre ancienne fonction StreamProducts par celle-ci :

func GetProducts(c echo.Context) error {
    category := c.QueryParam("category")
    search := c.QueryParam("search")

	fmt.Printf("DEBUG: Recherche reçue - Catégorie: '%s', Search: '%s'\n", category, search) // Ajouté

    products, _ := db.GetProductsFromDB(search, category)

    if (len(products) == 0) && category != "" {
        fmt.Printf("📦 Scraping à la demande pour la catégorie: %s\n", category) // 👈 utilise fmt
        
        scrapedProducts := scraper.ScrapeAllSources("", category)
        db.SaveProducts(scrapedProducts)
        
        products, _ = db.GetProductsFromDB(search, category)
    }

    return c.JSON(http.StatusOK, map[string]interface{}{"products": products})
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

    // Modifiez ici pour utiliser le préfixe scraper.
    products := scraper.ScrapeAllSources(query, category)

    for _, product := range products {
        if product.ID == id {
            return c.JSON(http.StatusOK, product)
        }
    }
    return c.JSON(http.StatusNotFound, map[string]string{"error": "Produit non trouvé"})
}



