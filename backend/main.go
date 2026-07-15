package main

import (
	"time"

	"tunisianet-scraper/db"
	"tunisianet-scraper/handlers"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func main() {
	db.InitDB()
	db.InitES()

	go func() {
		for {
			time.Sleep(24 * time.Hour)
			handlers.RunFullScrape()
		}
	}()

	e := echo.New()
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"http://localhost:3000"},
		AllowMethods: []string{echo.GET, echo.PUT, echo.POST, echo.DELETE},
		AllowHeaders: []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept},
	}))

	e.GET("/api/products", handlers.GetProducts)
	e.GET("/api/categories", handlers.GetCategories)
	e.GET("/api/products/:id", handlers.GetProductByID)
	e.POST("/api/scrape", handlers.TriggerScraping)

	e.Logger.Fatal(e.Start(":8080"))
}