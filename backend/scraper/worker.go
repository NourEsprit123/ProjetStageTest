package scraper

import (
	"fmt"
	"sync" // Nécessaire pour l'erreur "undefined: sync"
	"tunisianet-scraper/db"
	"tunisianet-scraper/models" // Nécessaire pour l'erreur "undefined: models"

)




// 💡 scrapeAllSources renvoie le flux brut accumulé sans filtrage prématuré
func ScrapeAllSources(query string, category string) []models.Product {
	var allProducts []models.Product
	var mu sync.Mutex
	var wg sync.WaitGroup

	wg.Add(3)

	// --- TUNISIANET ---
	go func() {
		defer wg.Done()
		products, err := ScrapeProducts(query, category)
		if err == nil && len(products) > 0 {
			mu.Lock()
			allProducts = append(allProducts, products...)
			mu.Unlock()
		}
	}()

	// --- MYTEK ---
	go func() {
		defer wg.Done()
		products, err := ScrapeMytekProducts(query, category)
		if err == nil && len(products) > 0 {
			mu.Lock()
			allProducts = append(allProducts, products...)
			mu.Unlock()
		}
	}()

	// --- WIKI ---
	go func() {
		defer wg.Done()
		products, err := ScrapeWikiProducts(query, category)
		if err == nil && len(products) > 0 {
			mu.Lock()
			allProducts = append(allProducts, products...)
			mu.Unlock()
		}
	}()

	wg.Wait()
	return allProducts
}

// RunBackgroundUpdate scrappe les sources principales et met à jour la base
func RunBackgroundUpdate() {
	fmt.Println("🚀 Démarrage du rafraîchissement complet des données...")
	
	// On définit les catégories à mettre à jour
	categories := []string{"informatique", "smartphones", "ordinateurs", "composants"}
	
	for _, cat := range categories {
		// Scrape de chaque catégorie
		products := ScrapeAllSources("", cat)
		// Sauvegarde immédiate
		db.SaveProducts(products)
	}
	
	fmt.Println("✅ Mise à jour terminée avec succès.")
}