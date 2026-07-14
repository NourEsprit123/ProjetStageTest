package main

import (
	"tunisianet-scraper/db"
	"tunisianet-scraper/handlers"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func main() {
	db.InitDB()

	e := echo.New()

	// Configuration CORS
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"http://localhost:3000"},
		AllowMethods: []string{echo.GET, echo.PUT, echo.POST, echo.DELETE},
		AllowHeaders: []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept},
	}))

	// Routes
	e.GET("/api/products", handlers.GetProducts)
	e.GET("/api/categories", handlers.GetCategories)
	e.GET("/api/products/:id", handlers.GetProductByID)

	e.Logger.Fatal(e.Start(":8080"))
}