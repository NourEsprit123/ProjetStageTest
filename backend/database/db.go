package database

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/lib/pq"
)

func InitDB() *sql.DB {
	// Configuration calquée exactement sur ton Docker
	host := "db"
	port := 5432
	user := "postgres"
	password := "nour" // Le mot de passe que tu as mis dans ton terminal
	dbname := "postgres" // Par défaut sous Docker, la base initiale s'appelle 'postgres'

	connStr := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("❌ Erreur d'ouverture de la base de données: %v", err)
	}

	// Vérification de la connectivité réelle
	err = db.Ping()
	if err != nil {
		log.Fatalf("❌ Impossible de ping PostgreSQL (Vérifie que Docker tourne): %v", err)
	}

	fmt.Println("🚀 Connecté avec succès à PostgreSQL via Docker !")
	return db
}