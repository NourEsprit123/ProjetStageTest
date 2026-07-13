package main

// Dans main.go, assurez-vous d'avoir cet import pour que le driver Postgres soit chargé
import (
    "log"
    "net/http"
    "tunisianet-scraper/database"
    "tunisianet-scraper/handlers"

    _ "github.com/lib/pq" // <--- Ajoutez cette ligne avec le underscore devant
    "github.com/labstack/echo/v4"
    "github.com/labstack/echo/v4/middleware"
)

func main() {
	e := echo.New()

	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"http://localhost:3000", "http://localhost:5000","https://frontend-tunisianet.onrender.com",},
		AllowMethods: []string{http.MethodGet, http.MethodPost, http.MethodDelete},
	}))

	db := database.InitDB()
	defer db.Close()

	es, err := database.InitES()
	if err != nil {
		log.Printf("⚠️ Elasticsearch non disponible: %v", err)
		es = nil
	} else {
		log.Println("🔄 Migration PostgreSQL → Elasticsearch en cours...")
		go database.MigrateToES(db, es)
	}

	e.GET("/api/products", handlers.GetProductsFromDB(db, es))
	e.GET("/api/products/:id", handlers.GetProductDetail(db))
	e.GET("/api/categories", handlers.GetCategories)
	// Enregistrez la nouvelle route
    e.GET("/api/autocomplete", handlers.GetAutocompleteSuggestions(es))

	// Endpoint pour forcer la migration manuellement si besoin
	e.POST("/api/migrate-es", func(c echo.Context) error {
		if es == nil {
			return c.JSON(http.StatusServiceUnavailable, map[string]string{
				"error": "Elasticsearch non disponible",
			})
		}
		go database.MigrateToES(db, es)
		return c.JSON(http.StatusOK, map[string]string{
			"message": "Migration lancée — vérifiez les logs",
		})
	})

	log.Println("🚀 API Server démarré sur http://localhost:8080")
	e.Logger.Fatal(e.Start(":8081"))
}