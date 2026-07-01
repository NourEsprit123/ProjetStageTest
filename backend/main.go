package main

import (
	"tunisianet-scraper/handlers"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware" // 💡 Ne pas oublier cet import !
)

func main() {
	e := echo.New()

	// 💡 AJOUTE CES LIGNES ICI (Juste après avoir créé "e")
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"http://localhost:3000"}, // Autorise ton frontend Next.js
		AllowMethods: []string{echo.GET, echo.PUT, echo.POST, echo.DELETE},
		AllowHeaders: []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept},
	}))

	// Tes routes actuelles...
	e.GET("/api/products", handlers.GetProducts)
	e.GET("/api/categories", handlers.GetCategories)
	e.GET("/api/products/:id", handlers.GetProductByID)

	e.Logger.Fatal(e.Start(":8080"))
}