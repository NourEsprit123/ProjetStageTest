package main

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"tunisianet-scraper/handlers"
)

func main() {
	e := echo.New()

	// Middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	// CORS - permettre les requêtes depuis Next.js
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"http://localhost:3000"},
		AllowMethods: []string{http.MethodGet, http.MethodPost},
		AllowHeaders: []string{echo.HeaderContentType},
	}))

	// Routes
	api := e.Group("/api")

	api.GET("/products", handlers.GetProducts)
	api.GET("/products/:id", handlers.GetProductByID)
	api.GET("/categories", handlers.GetCategories)

	// Démarrer le serveur sur le port 8080
	e.Logger.Fatal(e.Start(":8080"))
}