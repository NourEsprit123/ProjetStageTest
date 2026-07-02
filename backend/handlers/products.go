package handlers

import (
	"database/sql"
	"fmt"
	"log"
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
	"peripheriques":        {"souris", "clavier", "casque", "webcam", "imprimante", "scanner", "sacoche", "sac"},
	"stockage":             {"disque dur", "ssd", "hdd", "stockage", "cle usb", "clé usb", "sandisk", "kingston"},
	"electromenager":       {"refrigerateur", "réfrigérateur", "congelateur", "congélateur", "climatiseur", "four", "lave", "seche", "sèche", "aspirateur"},
}

var intrusBlacklist = []string{
	"chaussures", "banquette", "lisseur", "distributeur", "pancake", "cupcake",
	"moule", "brosse chauffante", "epilateur", "rasoir", "tondeuse", "poele", "casserole",
}

func normalizeText(text string) string {
	text = strings.ToLower(text)
	reg := regexp.MustCompile(`[\/\-_—–\|\+,\.]`)
	text = reg.ReplaceAllString(text, " ")
	return strings.Join(strings.Fields(text), " ")
}

func matchesCategory(product models.Product, category string) bool {
    // Désactivation temporaire du filtre pour tester
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

func SaveOrUpdateProduct(db *sql.DB, p models.Product, category string) error {
	query := `
INSERT INTO products (name, price, url, image_url, source, category, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, NOW())
ON CONFLICT (url) 
DO UPDATE SET 
price = EXCLUDED.price,
name = EXCLUDED.name,
image_url = EXCLUDED.image_url,
updated_at = NOW();`
	_, err := db.Exec(query, p.Name, p.Price, p.URL, p.Image, p.Source, category)
	return err
}

func ScrapeAndSaveAll(db *sql.DB, category string) {
	var wg sync.WaitGroup
	wg.Add(3)

	// --- TUNISIANET ---
	go func() {
		defer wg.Done()
		// query="" → buildURL utilise la page catégorie dédiée si elle existe
		products, err := scraper.ScrapeProducts("", category)
		if err == nil {
			saved := 0
			for _, p := range products {
				p.Source = "tunisianet"
				if !isBlacklisted(p.Name) && matchesCategory(p, category) {
					if saveErr := SaveOrUpdateProduct(db, p, category); saveErr == nil {
						saved++
					}
				}
			}
			log.Printf("✅ [Tunisianet] %d produits sauvegardés pour '%s'", saved, category)
		} else {
			log.Printf("❌ [Tunisianet] Erreur pour '%s': %v", category, err)
		}
	}()

	// --- MYTEK ---
	go func() {
		defer wg.Done()
		// ✅ CORRECTION : query="" pour que Mytek utilise mytekCategoryKeywords
		products, err := scraper.ScrapeMytekProducts("", category)
		if err == nil {
			saved := 0
			for _, p := range products {
				p.Source = "mytek"
				if !isBlacklisted(p.Name) && matchesCategory(p, category) {
					if saveErr := SaveOrUpdateProduct(db, p, category); saveErr == nil {
						saved++
					}
				}
			}
			log.Printf("✅ [Mytek] %d produits sauvegardés pour '%s'", saved, category)
		} else {
			log.Printf("❌ [Mytek] Erreur pour '%s': %v", category, err)
		}
	}()

	// --- WIKI ---
	go func() {
		defer wg.Done()
		// ✅ CORRECTION : query="" pour que Wiki utilise wikiCategories
		products, err := scraper.ScrapeWikiProducts("", category)
		if err == nil {
			saved := 0
			for _, p := range products {
				p.Source = "wiki"
				if !isBlacklisted(p.Name) && matchesCategory(p, category) {
					if saveErr := SaveOrUpdateProduct(db, p, category); saveErr == nil {
						saved++
					}
				}
			}
			log.Printf("✅ [Wiki] %d produits sauvegardés pour '%s'", saved, category)
		} else {
			log.Printf("❌ [Wiki] Erreur pour '%s': %v", category, err)
		}
	}()

	wg.Wait()
	log.Printf("📥 [Worker] Fin du scraping pour la catégorie : %s", category)
}

func GetProductsFromDB(db *sql.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		category := strings.ToLower(strings.TrimSpace(c.QueryParam("category")))
		search := c.QueryParam("search")
		cleanedSearch := strings.ToLower(strings.TrimSpace(search))

		results := []models.Product{}

      sqlQuery := "SELECT url, name, price, url, image_url, source, category FROM products WHERE 1=1"	
	  	var args []interface{}
		argIdx := 1

		if category != "" {
			sqlQuery += fmt.Sprintf(" AND LOWER(category) = $%d", argIdx)
			args = append(args, category)
			argIdx++
		}

		if cleanedSearch != "" {
			words := strings.Fields(cleanedSearch)
			for _, word := range words {
				if len(word) > 1 {
					sqlQuery += fmt.Sprintf(" AND name ILIKE $%d", argIdx)
					args = append(args, "%"+word+"%")
					argIdx++
				}
			}
		}

		sqlQuery += " ORDER BY updated_at DESC"

		log.Printf("DEBUG SQL: Query=%s", sqlQuery)
        log.Printf("DEBUG SQL: Args=%v", args)

		
  

		rows, err := db.Query(sqlQuery, args...)
		if err != nil {
			log.Printf("❌ Erreur SQL : %v", err)
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Erreur SQL interne"})
		}
		defer rows.Close()

		for rows.Next() {
			var p models.Product
err := rows.Scan(&p.ID, &p.Name, &p.Price, &p.URL, &p.Image, &p.Source, &p.Category)
			if err != nil {
				log.Printf("⚠️ Erreur Scan SQL: %v", err)
				continue
			}
			results = append(results, p)
		}

		log.Printf("🔍 Catégorie: '%s' | Recherche: '%s' | Résultats: %d", category, search, len(results))

		return c.JSON(http.StatusOK, map[string]interface{}{
			"products": results,
		})
	}
}

func GetCategories(c echo.Context) error {
	categories := scraper.GetCategories()
	return c.JSON(http.StatusOK, map[string]interface{}{
		"categories": categories,
	})
}