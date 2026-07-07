package models

import (
	"crypto/sha256"
	"encoding/hex"
)

type Product struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Price       string `json:"price"`
	Image       string `json:"image"`
	Category    string `json:"category"`
	URL         string `json:"url"`
	Description string `json:"description"`
	InStock     bool   `json:"in_stock"`
	Source      string `json:"source"` // 💡 Le S majuscule ici est obligatoire pour être visible par le handler !
	Score       float64 `json:"score"`
	UpdatedAt string  `json:"updated_at"`
	 Reference   string  `json:"reference"`
}

type SearchRequest struct {
	Query    string `json:"query"`
	Category string `json:"category"`
}


// DocIDFromURL génère un ID de document Elasticsearch stable et valide
// (sans "/" ni caractères spéciaux) à partir d'une URL produit.
// C'est nécessaire car ES rejette les URL brutes contenant des "/" comme _id.
func DocIDFromURL(url string) string {
	h := sha256.Sum256([]byte(url))
	return hex.EncodeToString(h[:])
}