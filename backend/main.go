package main

import (
	"log"
	"net/http"
	"tunisianet-scraper/database"
	"tunisianet-scraper/handlers"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func main() {
	e := echo.New()

	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"http://localhost:3000", "http://localhost:5000"},
		AllowMethods: []string{http.MethodGet, http.MethodPost},
	}))

	db := database.InitDB()
	defer db.Close()

	es, err := database.InitES()
	if err != nil {
		log.Printf("⚠️ Elasticsearch non disponible: %v", err)
		es = nil
	}




	e.GET("/api/products", handlers.GetProductsFromDB(db, es))
	e.GET("/api/products/:id", handlers.GetProductDetail(db))
	e.GET("/api/categories", handlers.GetCategories)

	log.Println("🚀 API Server démarré sur http://localhost:8080")
	e.Logger.Fatal(e.Start(":8080"))
}