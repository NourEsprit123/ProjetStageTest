package handlers

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
	"tunisianet-scraper/db"
	"tunisianet-scraper/scraper"
)

func TriggerScraping(c echo.Context) error {
	go RunFullScrape()
	return c.JSON(http.StatusAccepted, map[string]string{
		"message": "Scraping lancé en arrière-plan. Cela peut prendre plusieurs minutes.",
	})
}

func RunFullScrape() {
	fmt.Println("🚀 Scraping complet démarré")
	total := 0
	for _, cat := range scraper.GetCategories() {
		fmt.Printf("📦 Scraping catégorie: %s\n", cat)
		raw := scraper.ScrapeAllSources("", cat)
		cleaned := scraper.CleanProducts(raw, cat)
		db.SaveProducts(cleaned)
		if err := db.BulkIndexProducts(cleaned); err != nil {
			fmt.Printf("⚠️ ES pour %s: %v\n", cat, err)
		}
		total += len(cleaned)
	}
	fmt.Printf("✅ Scraping terminé. %d produits traités.\n", total)
}