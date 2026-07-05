package main

import (
    "database/sql"
    "log"
    "tunisianet-scraper/database"
    _ "github.com/lib/pq" // N'oublie pas l'import du driver postgres
)

func main() {
    // 1. Initialiser la connexion DB
    db := database.InitDB()
    defer db.Close()

    // 2. Initialiser la connexion ES
    es, err := database.InitES()
    if err != nil {
        log.Fatalf("Impossible de se connecter à ES: %v", err)
    }

    // 3. Lancer la migration
    database.MigrateToES(db, es)
}