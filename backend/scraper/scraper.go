package scraper

import (
	"compress/gzip"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"tunisianet-scraper/models"
)

// IMPORTANT : ces URLs doivent être vérifiées une par une en naviguant sur
// https://www.tunisianet.com.tn (menu "Toutes nos catégories"), puis en copiant
// l'URL exacte affichée dans la barre d'adresse pour chaque catégorie.
// Les ID numériques Tunisianet ne suivent aucun schéma prévisible.
var categories = map[string]string{
	"informatique":         "https://www.tunisianet.com.tn/301-informatique",         // confirmé OK
	"smartphones":          "https://www.tunisianet.com.tn/596-smartphone-tunisie",    // confirmé OK
	"telephonie portables": "https://www.tunisianet.com.tn/377-telephone-portable-tunisie", // confirmé OK

	// ⚠️ À VÉRIFIER MANUELLEMENT — probablement incorrectes, retirées du mapping
	// pour éviter les pages 404 silencieuses. Une fois vérifiées, remets-les ici.
	// "composants":             "https://www.tunisianet.com.tn/226-composants",
	// "ordinateurs":            "https://www.tunisianet.com.tn/228-ordinateurs",
	// "reseaux":                "https://www.tunisianet.com.tn/235-reseaux-et-connectivite",
	// "peripheriques":          "https://www.tunisianet.com.tn/234-peripheriques",
	// "stockage":               "https://www.tunisianet.com.tn/233-stockages",
	// "accessoires telephonie": "https://www.tunisianet.com.tn/accessoires-telephonie",
	// "telephones fixes":       "https://www.tunisianet.com.tn/telephones-fixes",
	// "smartwatch":             "https://www.tunisianet.com.tn/smartwatch",
	// "electromenager":         "https://www.tunisianet.com.tn/303-electromenager",
}

func getHTTPClient() *http.Client {
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		TLSHandshakeTimeout:   30 * time.Second,
		ResponseHeaderTimeout: 30 * time.Second,
	}
	return &http.Client{
		Timeout:   60 * time.Second,
		Transport: transport,
	}
}

// ScrapeProducts récupère jusqu'à maxPages pages de résultats (24 produits/page environ).
// maxPages=0 ou 1 => comportement identique à avant (une seule page).
func ScrapeProducts(query string, category string) ([]models.Product, error) {
	const maxPages = 5 // ajuste selon le nombre de produits que tu veux récupérer

	var allProducts []models.Product
	baseURL := buildURL(query, category)

	client := getHTTPClient()

	for page := 1; page <= maxPages; page++ {
		pageURL := addPageParam(baseURL, page)
		fmt.Println("Scraping URL:", pageURL)

		products, err := scrapePage(client, pageURL, category)
		if err != nil {
			return allProducts, err
		}

		if len(products) == 0 {
			// Plus de produits => on a atteint la dernière page, on arrête.
			break
		}

		allProducts = append(allProducts, products...)
	}

	return allProducts, nil
}

func scrapePage(client *http.Client, pageURL string, category string) ([]models.Product, error) {
	var products []models.Product

	req, err := http.NewRequest("GET", pageURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Language", "fr-FR,fr;q=0.9")
	req.Header.Set("Accept-Encoding", "gzip, deflate")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Upgrade-Insecure-Requests", "1")
	req.Header.Set("Cache-Control", "max-age=0")

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

	// DEBUG TEMPORAIRE — à retirer une fois le diagnostic terminé
	debugHTML, _ := doc.Html()
	os.WriteFile("debug_page.html", []byte(debugHTML), 0644)
	fmt.Println("Taille HTML reçu:", len(debugHTML), "octets")
	fmt.Println("Nombre d'articles 'article.product-miniature' trouvés:", doc.Find("article.product-miniature").Length())
	fmt.Println("Titre de la page <title>:", strings.TrimSpace(doc.Find("title").Text()))

	doc.Find("article.product-miniature").Each(func(i int, s *goquery.Selection) {
		product := models.Product{}

		product.ID = s.AttrOr("data-id-product", fmt.Sprintf("%d", i+1))
		product.Name = strings.TrimSpace(s.Find("h2.product-title a").Text())
		product.URL, _ = s.Find("h2.product-title a").Attr("href")

		product.Image, _ = s.Find(".wb-image-block img").Attr("src")
		if product.Image == "" {
			product.Image, _ = s.Find(".wb-image-block img").Attr("data-src")
		}

		product.Price = strings.TrimSpace(s.Find(".price").First().Text())
		product.Category = category

		if product.Name != "" {
			products = append(products, product)
		}
	})

	return products, nil
}

// addPageParam ajoute ou met à jour le paramètre "page" dans l'URL.
func addPageParam(baseURL string, page int) string {
	if page <= 1 {
		return baseURL
	}
	parsed, err := url.Parse(baseURL)
	if err != nil {
		return baseURL
	}
	q := parsed.Query()
	q.Set("page", fmt.Sprintf("%d", page))
	parsed.RawQuery = q.Encode()
	return parsed.String()
}

func buildURL(query string, category string) string {
	if category != "" {
		if catURL, ok := categories[category]; ok {
			return catURL
		}
		return fmt.Sprintf("https://www.tunisianet.com.tn/recherche?controller=search&s=%s", url.QueryEscape(category))
	}

	if query != "" {
		return fmt.Sprintf("https://www.tunisianet.com.tn/recherche?controller=search&s=%s", url.QueryEscape(query))
	}

	return "https://www.tunisianet.com.tn/recherche?controller=search&s=informatique"
}

func GetCategories() []string {
	return []string{
		"informatique",
		"composants",
		"ordinateurs",
		"reseaux",
		"peripheriques",
		"stockage",
		"telephonie portables",
		"smartphones",
		"accessoires telephonie",
		"telephones fixes",
		"smartwatch",
		"sante beaute",
		"toiletries",
		"moniteurs sante",
		"bebe enfants",
		"pharmaceutiques",
		"soins personnels",
		"electromenager",
		"aspirateurs",
		"machine a laver",
		"seche linge",
		"lave vaisselle",
		"fours",
	}
}