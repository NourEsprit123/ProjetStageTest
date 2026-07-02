package main // 💡 VÉRIFIE BIEN CETTE LIGNE !

import (
	"database/sql"
	"log"
	"time"
	"tunisianet-scraper/database"
	"tunisianet-scraper/handlers"
	

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func startPeriodicWorker(db *sql.DB) {
	categories := []string{
		"informatique",  
		"ordinateurs",
		"smartphones",
		"composants",
		"reseaux",
		"peripheriques",
		"stockage",
		"electromenager",
	}

	go func() {
		log.Println("⚡ Initialisation : Lancement du premier scraping global en tâche de fond...")
		for _, cat := range categories {
			log.Printf("🔄 [Worker] Scraping de la catégorie : %s ...", cat)
			handlers.ScrapeAndSaveAll(db, cat)
		}
		log.Println("✅ [Worker] Premier scraping terminé pour toutes les catégories ! Base de données prête.")
	}()

	ticker := time.NewTicker(1 * time.Hour)
	go func() {
		for range ticker.C {
			log.Println("⏰ Worker : Lancement de la mise à jour horaire automatique...")
			for _, cat := range categories {
				handlers.ScrapeAndSaveAll(db, cat)
			}
			log.Println("✅ Worker : Fin de la mise à jour horaire complète.")
		}
	}()
}

func main() {
	e := echo.New()

	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"http://localhost:3000", "http://localhost:5000"},
		AllowMethods: []string{echo.GET, echo.POST},
	}))

	db := database.InitDB()
	defer db.Close()

	startPeriodicWorker(db)

	e.GET("/api/products", handlers.GetProductsFromDB(db))

	log.Println("⇨ API Server démarré sur http://localhost:8080")
	e.Logger.Fatal(e.Start(":8080"))
}