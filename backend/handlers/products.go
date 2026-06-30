package handlers

import (
	"net/http"
	"sync"

	"github.com/labstack/echo/v4"
	"tunisianet-scraper/models"
	"tunisianet-scraper/scraper"
)

// scrapeAllSources interroge Tunisianet et Mytek EN PARALLÈLE et combine leurs résultats.
// Une erreur sur une seule source n'empêche pas de retourner les résultats de l'autre.
func scrapeAllSources(query string, category string) []models.Product {
	var allProducts []models.Product
	var mu sync.Mutex
	var wg sync.WaitGroup

	wg.Add(2)

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

	products := scrapeAllSources(query, category)

	return c.JSON(http.StatusOK, map[string]interface{}{
		"products": products,
		"total":    len(products),
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