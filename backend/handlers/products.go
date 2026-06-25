package handlers

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"tunisianet-scraper/scraper"
)

func GetProducts(c echo.Context) error {
	query := c.QueryParam("search")
	category := c.QueryParam("category")

	products, err := scraper.ScrapeProducts(query, category)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

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

	products, err := scraper.ScrapeProducts(query, category)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	for _, product := range products {
		if product.ID == id {
			return c.JSON(http.StatusOK, product)
		}
	}

	return c.JSON(http.StatusNotFound, map[string]string{
		"error": "Produit non trouvé",
	})
}