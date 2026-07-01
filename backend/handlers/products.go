package handlers

import (
	"net/http"
	"regexp"
	"strings"
	"sync"

	"github.com/labstack/echo/v4"
	"tunisianet-scraper/models"
	"tunisianet-scraper/scraper"
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

// 💡 scrapeAllSources renvoie le flux brut accumulé sans filtrage prématuré
func scrapeAllSources(query string, category string) []models.Product {
	var allProducts []models.Product
	var mu sync.Mutex
	var wg sync.WaitGroup

	wg.Add(3)

	// --- TUNISIANET ---
	go func() {
		defer wg.Done()
		products, err := scraper.ScrapeProducts(query, category)
		if err == nil && len(products) > 0 {
			mu.Lock()
			allProducts = append(allProducts, products...)
			mu.Unlock()
		}
	}()

	// --- MYTEK ---
	go func() {
		defer wg.Done()
		products, err := scraper.ScrapeMytekProducts(query, category)
		if err == nil && len(products) > 0 {
			mu.Lock()
			allProducts = append(allProducts, products...)
			mu.Unlock()
		}
	}()

	// --- WIKI ---
	go func() {
		defer wg.Done()
		products, err := scraper.ScrapeWikiProducts(query, category)
		if err == nil && len(products) > 0 {
			mu.Lock()
			allProducts = append(allProducts, products...)
			mu.Unlock()
		}
	}()

	wg.Wait()
	return allProducts
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

func GetProducts(c echo.Context) error {
	category := c.QueryParam("category")
	search := c.QueryParam("search")

	cleanedSearch := cleanQueryForScrapers(search)
	if cleanedSearch == "" {
		cleanedSearch = search
	}

	broadSearch := limitQueryWords(cleanedSearch, 3)
	allProducts := scrapeAllSources(broadSearch, category)

	// --- AJOUT DU DEBUG ICI ---
	println("--- DEBUG FILTRAGE ---")
	println("Recherche nettoyée cible :", cleanedSearch)
	println("Nombre total de produits reçus des scrapers :", len(allProducts))
	// ---------------------------

	var filteredProducts []models.Product
	for _, p := range allProducts {
		matchQ := matchesQuery(p, cleanedSearch)
		matchC := matchesCategory(p, category)
		
		// Décommente la ligne ci-dessous si tu veux voir le comportement de chaque produit :
		// fmt.Printf("Produit: %s | MatchQuery: %t | MatchCategory: %t\n", p.Name, matchQ, matchC)

		if matchQ && matchC && !isBlacklisted(p.Name) {
			filteredProducts = append(filteredProducts, p)
		}
	}

	println("Nombre de produits après filtrage :", len(filteredProducts))
	println("----------------------")

	return c.JSON(http.StatusOK, map[string]interface{}{
		"products": filteredProducts,
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