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
	// 1. Initialisation du framework Echo
	e := echo.New()

	// Configuration CORS pour autoriser le front-end
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"http://localhost:3000", "http://localhost:5000"},
		AllowMethods: []string{http.MethodGet, http.MethodPost},
	}))

	// 2. Initialisation unique de la DB Postgres
	db := database.InitDB()
	defer db.Close()

	// 3. Initialisation d'Elasticsearch
	// On le fait une seule fois au démarrage
	es, err := database.InitES()
	if err != nil {
		  log.Printf("⚠️ Elasticsearch non disponible, recherche via PostgreSQL uniquement: %v", err)
          es = nil
	}

	// 4. Routes 
	// On passe 'db' ET 'es' aux handlers pour qu'ils puissent chercher/sauvegarder
	e.GET("/api/products", handlers.GetProductsFromDB(db, es))
	e.GET("/api/categories", handlers.GetCategories)

	// Démarrage du serveur
	log.Println("🚀 API Server démarré sur http://localhost:8080")
	e.Logger.Fatal(e.Start(":8080"))
}