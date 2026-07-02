package models

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
}

type SearchRequest struct {
	Query    string `json:"query"`
	Category string `json:"category"`
}