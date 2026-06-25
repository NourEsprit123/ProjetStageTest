package models

type Product struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Price       string  `json:"price"`
	Image       string  `json:"image"`
	Category    string  `json:"category"`
	URL         string  `json:"url"`
	Description string  `json:"description"`
	InStock     bool    `json:"in_stock"`
}

type SearchRequest struct {
	Query    string `json:"query"`
	Category string `json:"category"`
}