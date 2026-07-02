package scraper

import (
	"compress/gzip"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"tunisianet-scraper/models"
)

var categories = map[string]string{
	"informatique":         "https://www.tunisianet.com.tn/301-informatique",
	"smartphones":          "https://www.tunisianet.com.tn/596-smartphone-tunisie",
	"telephonie portables": "https://www.tunisianet.com.tn/377-telephone-portable-tunisie",
}

func getHTTPClient() *http.Client {
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		TLSHandshakeTimeout:   30 * time.Second,
		ResponseHeaderTimeout: 30 * time.Second,
	}
	return &http.Client{Timeout: 60 * time.Second, Transport: transport}
}

func ScrapeProducts(query string, category string) ([]models.Product, error) {
	const maxPages = 5 // ← Limite de sécurité : évite les 31 pages pour "stockage"

	var allProducts []models.Product
	baseURL := buildURL(query, category)
	client := getHTTPClient()

	for page := 1; page <= maxPages; page++ {
		pageURL := addPageParam(baseURL, page)
		fmt.Printf("📥 [Tunisianet] Scraping Page %d: %s\n", page, pageURL)

		products, htmlDoc, err := scrapePageWithDoc(client, pageURL, category)
		if err != nil {
			fmt.Printf("⚠️ [Tunisianet] Arrêt à la page %d: %v\n", page, err)
			return allProducts, nil
		}

		if len(products) == 0 {
			fmt.Printf("🏁 [Tunisianet] Plus de produits. Fin à la page %d.\n", page)
			break
		}

		allProducts = append(allProducts, products...)

		hasNextPage := htmlDoc.Find("a.next, a[rel='next']").Length() > 0
		if !hasNextPage {
			fmt.Printf("🏁 [Tunisianet] Fin naturelle à la page %d.\n", page)
			break
		}

		time.Sleep(300 * time.Millisecond)
	}

	return allProducts, nil
}

func scrapePageWithDoc(client *http.Client, pageURL string, category string) ([]models.Product, *goquery.Document, error) {
	var products []models.Product

	req, err := http.NewRequest("GET", pageURL, nil)
	if err != nil {
		return nil, nil, err
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
		return nil, nil, err
	}
	defer resp.Body.Close()

	var reader io.Reader
	if resp.Header.Get("Content-Encoding") == "gzip" {
		gzReader, err := gzip.NewReader(resp.Body)
		if err != nil {
			return nil, nil, err
		}
		defer gzReader.Close()
		reader = gzReader
	} else {
		reader = resp.Body
	}

	doc, err := goquery.NewDocumentFromReader(reader)
	if err != nil {
		return nil, nil, err
	}

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
		product.InStock = !strings.Contains(strings.ToLower(s.Text()), "hors stock")

		if product.Name != "" {
			products = append(products, product)
		}
	})

	return products, doc, nil
}

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
	if query != "" {
		return fmt.Sprintf("https://www.tunisianet.com.tn/recherche?controller=search&s=%s", url.QueryEscape(query))
	}
	if category != "" {
		if catURL, ok := categories[category]; ok {
			return catURL
		}
		return fmt.Sprintf("https://www.tunisianet.com.tn/recherche?controller=search&s=%s", url.QueryEscape(category))
	}
	return "https://www.tunisianet.com.tn/recherche?controller=search&s=informatique"
}

func GetCategories() []string {
	return []string{
		"informatique", "composants", "ordinateurs", "reseaux", "peripheriques",
		"stockage", "telephonie portables", "smartphones", "accessoires telephonie",
		"telephones fixes", "smartwatch", "sante beaute", "toiletries",
		"moniteurs sante", "bebe enfants", "pharmaceutiques", "soins personnels",
		"electromenager", "aspirateurs", "machine a laver", "seche linge",
		"lave vaisselle", "fours",
	}
}