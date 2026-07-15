package handlers

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
	"tunisianet-scraper/db"
	"tunisianet-scraper/scraper"
)

func GetProducts(c echo.Context) error {
	category := c.QueryParam("category")
	search := c.QueryParam("search")

	if search != "" {
		esProducts, err := db.SearchProductsES(search, category)
		if err == nil {
			return c.JSON(http.StatusOK, map[string]interface{}{"products": esProducts})
		}
		fmt.Printf("⚠️ ES indisponible, repli Postgres: %v\n", err)
	}

	products, _ := db.GetProductsFromDB(search, category)

	if (len(products) == 0) && category != "" {
		fmt.Printf("📦 Scraping à la demande: %s\n", category)
		scrapedProducts := scraper.ScrapeAllSources("", category)
		cleaned := scraper.CleanProducts(scrapedProducts, category)
		db.SaveProducts(cleaned)
		db.BulkIndexProducts(cleaned)
		products, _ = db.GetProductsFromDB(search, category)
	}

	return c.JSON(http.StatusOK, map[string]interface{}{"products": products})
}

func GetCategories(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]interface{}{"categories": scraper.GetCategories()})
}

func GetProductByID(c echo.Context) error {
	id := c.Param("id")
	query := c.QueryParam("search")
	category := c.QueryParam("category")

	products := scraper.ScrapeAllSources(query, category)
	for _, product := range products {
		if product.ID == id {
			return c.JSON(http.StatusOK, product)
		}
	}
	return c.JSON(http.StatusNotFound, map[string]string{"error": "Produit non trouvé"})
}