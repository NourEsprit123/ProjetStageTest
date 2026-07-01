package scraper

import (
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"tunisianet-scraper/models"
)

var wikiCategories = map[string]string{
	"smartphones":            "https://wiki.tn/smartphones/",
	"telephonie portables":   "https://wiki.tn/telephones-portables/",
	"informatique":           "https://wiki.tn/pc-portables/",
	"ordinateurs":            "https://wiki.tn/ordinateur-de-bureau/",
	"composants":             "https://wiki.tn/composants-pc/",
	"peripheriques":          "https://wiki.tn/accessoires-ordinateur/",
	"stockage":               "https://wiki.tn/stockage/",
	"reseaux":                "https://wiki.tn/cable-et-adaptateur/",
	"electromenager":         "https://wiki.tn/electromenager-cuisine/",
	"machine a laver":        "https://wiki.tn/machine-a-laver/",
	"lave vaisselle":         "https://wiki.tn/lave-vaisselle/",
	"aspirateurs":            "https://wiki.tn/aspirateur-nettoyeur-a-vapeur/",
	"fours":                  "https://wiki.tn/four-electrique/",
	"smartwatch":             "https://wiki.tn/montre-connectee/",
	"accessoires telephonie": "https://wiki.tn/accessoires-telephone/",
}

func getWikiHTTPClient() *http.Client {
	return &http.Client{Timeout: 20 * time.Second}
}

func ScrapeWikiProducts(query string, category string) ([]models.Product, error) {
	const maxPages = 2
	var allProducts []models.Product
	client := getWikiHTTPClient()

	for page := 1; page <= maxPages; page++ {
		pageURL := buildWikiURL(query, category, page)
		fmt.Printf("📥 [Wiki.tn] Scraping Page %d: %s\n", page, pageURL)

		products, err := scrapeWikiPage(client, pageURL, category)
		if err != nil {
			fmt.Printf("⚠️ [Wiki.tn] Erreur page %d: %v\n", page, err)
			break
		}
		if len(products) == 0 {
			fmt.Printf("🏁 [Wiki.tn] Fin à la page %d\n", page)
			break
		}
		fmt.Printf("✅ [Wiki.tn] %d produits récupérés à la page %d\n", len(products), page)
		allProducts = append(allProducts, products...)
		if len(products) < 12 {
			break
		}
	}
	return allProducts, nil
}

func scrapeWikiPage(client *http.Client, pageURL string, category string) ([]models.Product, error) {
	var products []models.Product

	req, err := http.NewRequest("GET", pageURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "fr-FR,fr;q=0.9")
	req.Header.Set("Accept-Encoding", "gzip, deflate")
	req.Header.Set("Connection", "keep-alive")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var reader io.Reader
	if resp.Header.Get("Content-Encoding") == "gzip" {
		gzReader, err := gzip.NewReader(resp.Body)
		if err != nil {
			return nil, err
		}
		defer gzReader.Close()
		reader = gzReader
	} else {
		reader = resp.Body
	}

	doc, err := goquery.NewDocumentFromReader(reader)
	if err != nil {
		return nil, err
	}

	// data-product_id est sur le bouton "Ajouter au panier".
	// On remonte au conteneur parent qui contient tout le produit (nom, image, prix).
	seen := map[string]bool{}

	doc.Find("a.add_to_cart_button[data-product_id]").Each(func(i int, btn *goquery.Selection) {
		productID := btn.AttrOr("data-product_id", fmt.Sprintf("%d", i+1))

		// Éviter les doublons si plusieurs boutons pour le même produit
		if seen[productID] {
			return
		}
		seen[productID] = true

		// Remonter au conteneur parent qui contient image + titre + prix
		// Bricks Builder imbrique typiquement : div.brxe > div(produit) > [image, titre, prix, bouton]
		// On remonte jusqu'à trouver un ancêtre qui contient une image ET un texte de titre
		container := btn.Parent()
		for depth := 0; depth < 6; depth++ {
			hasImg := container.Find("img").Length() > 0
			hasTitle := container.Find("h2, h3, h4, [class*='title'], [class*='name']").Length() > 0
			if hasImg && hasTitle {
				break
			}
			container = container.Parent()
		}

		product := models.Product{}
		product.ID = "wiki-" + productID

		// Nom
		product.Name = strings.TrimSpace(container.Find("h2, h3, h4").First().Text())
		if product.Name == "" {
			product.Name = strings.TrimSpace(container.Find("[class*='title'], [class*='name']").First().Text())
		}

		// URL : cherche le premier lien qui n'est PAS le bouton panier
		container.Find("a").Each(func(_ int, a *goquery.Selection) {
			if product.URL == "" && !a.HasClass("add_to_cart_button") {
				product.URL, _ = a.Attr("href")
			}
		})

		// Image
		product.Image, _ = container.Find("img").First().Attr("src")
		if product.Image == "" {
			product.Image, _ = container.Find("img").First().Attr("data-src")
		}

		// Prix
		product.Price = strings.TrimSpace(container.Find(".price ins .amount").First().Text())
		if product.Price == "" {
			product.Price = strings.TrimSpace(container.Find(".woocommerce-Price-amount").First().Text())
		}
		if product.Price == "" {
			product.Price = strings.TrimSpace(container.Find(".price").First().Text())
		}

		// Stock
		stockClass, _ := btn.Attr("class")
		product.InStock = !strings.Contains(stockClass, "outofstock")
		product.Category = category

		// Debug du 1er produit
		if i == 0 {
			fmt.Printf("[Wiki.tn Debug] 1er produit → Nom: %q | Prix: %q | URL: %q\n",
				product.Name, product.Price, product.URL)
		}

		if product.Name != "" {
			products = append(products, product)
		}
	})

	fmt.Printf("[Wiki.tn Debug] Produits extraits: %d\n", len(products))
	return products, nil
}

func buildWikiURL(query string, category string, page int) string {
	if query != "" {
		base := fmt.Sprintf("https://wiki.tn/?s=%s&post_type=product", url.QueryEscape(query))
		if page > 1 {
			base += fmt.Sprintf("&paged=%d", page)
		}
		return base
	}
	if category != "" {
		if catURL, ok := wikiCategories[category]; ok {
			if page > 1 {
				return catURL + fmt.Sprintf("page/%d/", page)
			}
			return catURL
		}
		base := fmt.Sprintf("https://wiki.tn/?s=%s&post_type=product", url.QueryEscape(category))
		if page > 1 {
			base += fmt.Sprintf("&paged=%d", page)
		}
		return base
	}
	return "https://wiki.tn/shop/"
}