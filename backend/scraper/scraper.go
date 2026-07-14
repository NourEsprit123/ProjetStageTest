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
var categories = map[string]string{
	// Informatique & matériel
	"informatique":         "https://www.tunisianet.com.tn/301-informatique",
	"composants":           "https://www.tunisianet.com.tn/305-composants-pc-tunisie",
	"ordinateurs":          "https://www.tunisianet.com.tn/302-ordinateurs-portables",
	"reseaux":              "https://www.tunisianet.com.tn/306-reseaux-informatiques",
	"peripheriques":        "https://www.tunisianet.com.tn/309-peripheriques-informatiques",
	"stockage":             "https://www.tunisianet.com.tn/307-stockage",

	// Téléphonie
	"smartphones":          "https://www.tunisianet.com.tn/596-smartphone-tunisie",
	"telephonie portables": "https://www.tunisianet.com.tn/377-telephone-portable-tunisie",
	"accessoires telephonie":"https://www.tunisianet.com.tn/379-accessoires-telephonie",
	"telephones fixes":     "https://www.tunisianet.com.tn/378-telephone-fixe-tunisie",
	"smartwatch":           "https://www.tunisianet.com.tn/624-montres-connectees",

	// Santé & Beauté
	"sante beaute":         "https://www.tunisianet.com.tn/469-sante-beaute",
	"soins personnels":     "https://www.tunisianet.com.tn/471-soins-personnels",
	"moniteurs sante":      "https://www.tunisianet.com.tn/470-moniteurs-de-sante",

	// Électroménager
	"electromenager":       "https://www.tunisianet.com.tn/238-electromenager-tunisie",
	"aspirateurs":          "https://www.tunisianet.com.tn/243-aspirateur-tunisie",
	"machine a laver":      "https://www.tunisianet.com.tn/241-lave-linge-tunisie",
	"seche linge":          "https://www.tunisianet.com.tn/634-seche-linge",
	"lave vaisselle":       "https://www.tunisianet.com.tn/242-lave-vaisselle-tunisie",
	"fours":                "https://www.tunisianet.com.tn/250-fours-encastrables",
}

func getHTTPClient() *http.Client {
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
		DialContext: (&net.Dialer{
			Timeout:   90 * time.Second,
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

// ScrapeProducts récupère TOUTES les pages de résultats disponibles de manière dynamique.
func ScrapeProducts(query string, category string) ([]models.Product, error) {
	var allProducts []models.Product
	baseURL := buildURL(query, category)
	client := getHTTPClient()

	page := 1
	for {
		pageURL := addPageParam(baseURL, page)
		fmt.Printf("📥 [Tunisianet] Scraping Page %d: %s\n", page, pageURL)

		products, htmlDoc, err := scrapePageWithDoc(client, pageURL, category)
		if err != nil {
			// En cas d'erreur sur une page (ex: 404), on renvoie ce qu'on a déjà pour ne pas tout perdre
			fmt.Printf("⚠️ [Tunisianet] Arrêt ou erreur à la page %d: %v\n", page, err)
			return allProducts, nil
		}

		// Si la page ne contient aucun produit, on arrête la pagination
		if len(products) == 0 {
			fmt.Printf("🏁 [Tunisianet] Plus de produits trouvés. Fin à la page %d.\n", page)
			break
		}

		allProducts = append(allProducts, products...)

		// 🔍 CONDITION D'ARRÊT : On vérifie si le bouton "Suivant" est présent sur la page.
		// PrestaShop utilise généralement la classe 'a.next' ou l'attribut 'rel="next"'
		hasNextPage := htmlDoc.Find("a.next, a[rel='next']").Length() > 0
		if !hasNextPage {
			fmt.Printf("🏁 [Tunisianet] Fin du catalogue atteinte naturellement à la page %d.\n", page)
			break
		}

		page++
		
		// Pause de sécurité pour éviter d'être bloqué par le serveur
		time.Sleep(300 * time.Millisecond)
	}

	return allProducts, nil
}

// scrapePageWithDoc effectue la requête et extrait les données tout en retournant le document HTML complet pour analyse.
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

	// Code de débug original conservé
	debugHTML, _ := doc.Html()
	os.WriteFile("debug_page.html", []byte(debugHTML), 0644)
	fmt.Println("[Tunisianet Debug] Taille HTML reçu:", len(debugHTML), "octets")
	fmt.Println("[Tunisianet Debug] Articles 'article.product-miniature' trouvés:", doc.Find("article.product-miniature").Length())
	fmt.Println("[Tunisianet Debug] Titre de la page <title>:", strings.TrimSpace(doc.Find("title").Text()))

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
		product.Source = "Tunisianet"

		// Vérification basique de la disponibilité dans le DOM (optionnel selon le thème)
		product.InStock = !strings.Contains(strings.ToLower(s.Text()), "hors stock")

		if product.Name != "" {
			products = append(products, product)
		}
	})

	return products, doc, nil
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
	// Si l'utilisateur tape un texte, c'est ce texte précis qu'on cherche en priorité !
	if query != "" {
		return fmt.Sprintf("https://www.tunisianet.com.tn/recherche?controller=search&s=%s", url.QueryEscape(query))
	}

	// Sinon, s'il n'y a pas de texte mais une catégorie, on charge l'URL de la catégorie
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