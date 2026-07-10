package scraper

import (
	"log"
	"strings"
	"tunisianet-scraper/models"

	"github.com/gocolly/colly"
)

func ScrapeTechnopro(reference string, category string) ([]models.Product, error) {
	var products []models.Product
	c := colly.NewCollector(
		colly.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"),
	)

	// Logique de capture
	c.OnHTML(".product-container", func(e *colly.HTMLElement) {
		name := strings.TrimSpace(e.ChildText("h5 a"))
		price := strings.TrimSpace(e.ChildText(".price"))
		url := e.ChildAttr("h5 a", "href")
		image := e.ChildAttr("img.replace-2x", "src")

		log.Printf("DEBUG TECHNOPRO : Produit trouvé -> %s | Prix: %s", name, price)

		if name != "" {
			products = append(products, models.Product{
				Name:     name,
				Price:    price,
				URL:      url,
				Image:    image,
				Source:   "technopro",
				Category: category,
			})
		}
	})

	url := "https://www.technopro-online.com/recherche?controller=search&s=" + reference
	c.Visit(url)
	
	return products, nil
}