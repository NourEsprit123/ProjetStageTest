package scraper

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"tunisianet-scraper/models"
)

var categoryKeywords = map[string][]string{
	"smartphones":          {"smartphone", "telephone", "téléphone", "mobile", "coque", "silicone", "ecran", "écran", "galaxy", "iphone", "redmi", "honor", "vivo", "infinix"},
	"telephonie portables": {"telephone", "téléphone", "smartphone", "mobile"},
	"informatique":         {"ordinateur", "pc", "laptop", "portable", "informatique", "gamer", "lenovo", "hp", "dell", "asus", "acer", "msi"},
	"ordinateurs":          {"ordinateur", "pc", "laptop", "portable", "lenovo", "hp", "dell", "asus", "acer", "msi"},
	"composants":           {"carte mere", "carte graphique", "processeur", "ram", "ventilateur", "alimentation", "boitier", "composant"},
	"reseaux":              {"routeur", "switch", "reseau", "réseau", "wifi", "câble", "cable", "access point", "modem"},
	"peripheriques":        {"souris", "clavier", "casque", "webcam", "imprimante", "scanner"},
	"stockage":             {"disque dur", "ssd", "hdd", "stockage", "cle usb", "clé usb", "sandisk", "kingston"},
	"electromenager":       {"refrigerateur", "réfrigérateur", "congelateur", "congélateur", "climatiseur", "four", "lave", "seche", "sèche", "aspirateur"},
}

var intrusBlacklist = []string{
	"chaussures", "banquette", "lisseur", "distributeur", "pancake", "cupcake",
	"moule", "brosse chauffante", "epilateur", "rasoir", "tondeuse", "poele", "casserole",
}

func normalizeText(text string) string {
	text = strings.ToLower(text)
	reg := regexp.MustCompile(`[\/\-_—–\|\+,\.]`)
	text = reg.ReplaceAllString(text, " ")
	return strings.Join(strings.Fields(text), " ")
}

func matchesCategory(product models.Product, category string) bool {
	if category == "" {
		return true
	}
	catLower := strings.ToLower(category)
	nameNormalized := normalizeText(product.Name)

	keywords, ok := categoryKeywords[catLower]
	if !ok {
		return true
	}
	for _, kw := range keywords {
		if kw == "pc" {
			matched, _ := regexp.MatchString(`\bpc\b|\bpc-portable\b`, nameNormalized)
			if matched {
				return true
			}
		} else if strings.Contains(nameNormalized, normalizeText(kw)) {
			return true
		}
	}
	return false
}

func normalizeForDedup(name string) string {
	name = strings.ToLower(name)
	reg := regexp.MustCompile(`[^a-z0-9]+`)
	return strings.TrimSpace(reg.ReplaceAllString(name, " "))
}

func normalizePriceKey(price string) string {
	price = strings.ToLower(price)
	price = strings.ReplaceAll(price, " ", "")
	price = strings.ReplaceAll(price, "tnd", "")
	price = strings.ReplaceAll(price, "د.ت", "")
	price = strings.ReplaceAll(price, ",", ".")
	return price
}

func formatPrice(price string) string {
	cleaned := strings.ReplaceAll(price, " ", "")
	cleaned = strings.ReplaceAll(cleaned, ",", ".")
	cleaned = strings.ReplaceAll(cleaned, "TND", "")
	cleaned = strings.ReplaceAll(cleaned, "tnd", "")
	cleaned = strings.TrimSpace(cleaned)
	if val, err := strconv.ParseFloat(cleaned, 64); err == nil {
		return fmt.Sprintf("%.2f TND", val)
	}
	return price
}

// CleanProducts nettoie les données brutes : entrées vides, doublons (par ID
// et par nom+prix inter-sources), intrus hors-sujet, pertinence catégorie, prix normalisé.
func CleanProducts(products []models.Product, category string) []models.Product {
	seenID := make(map[string]bool)
	seenNamePrice := make(map[string]bool)
	cleaned := make([]models.Product, 0, len(products))

	for _, p := range products {
		p.Name = strings.TrimSpace(p.Name)
		p.Price = strings.TrimSpace(p.Price)

		if p.Name == "" || p.ID == "" || p.Image == "" {
			continue
		}
		if seenID[p.ID] {
			continue
		}

		nameLower := strings.ToLower(p.Name)
		blacklisted := false
		for _, word := range intrusBlacklist {
			if strings.Contains(nameLower, word) {
				blacklisted = true
				break
			}
		}
		if blacklisted {
			continue
		}

		if !matchesCategory(p, category) {
			continue
		}

		dedupKey := normalizeForDedup(p.Name) + "|" + normalizePriceKey(p.Price)
		if seenNamePrice[dedupKey] {
			continue
		}
		seenNamePrice[dedupKey] = true

		p.Price = formatPrice(p.Price)
		seenID[p.ID] = true
		cleaned = append(cleaned, p)
	}
	return cleaned
}