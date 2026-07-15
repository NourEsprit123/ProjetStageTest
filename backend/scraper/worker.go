package scraper

import (
	"sync"
	"tunisianet-scraper/models"
)

// ScrapeAllSources lance les 3 scrapers en parallèle et agrège les résultats bruts
func ScrapeAllSources(query string, category string) []models.Product {
	var allProducts []models.Product
	var mu sync.Mutex
	var wg sync.WaitGroup

	wg.Add(3)

	go func() {
		defer wg.Done()
		products, err := ScrapeProducts(query, category)
		if err == nil && len(products) > 0 {
			mu.Lock()
			allProducts = append(allProducts, products...)
			mu.Unlock()
		}
	}()

	go func() {
		defer wg.Done()
		products, err := ScrapeMytekProducts(query, category)
		if err == nil && len(products) > 0 {
			mu.Lock()
			allProducts = append(allProducts, products...)
			mu.Unlock()
		}
	}()

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