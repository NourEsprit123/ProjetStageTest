package database

import (
    "bytes"
    "context"
    "database/sql"
    "encoding/json"
    "log"
    "tunisianet-scraper/models" // Assure-toi que cet import est correct
    "github.com/elastic/go-elasticsearch/v8"
)

// InitES initialise la connexion avec le conteneur Elasticsearch
func InitES() (*elasticsearch.Client, error) {
    cfg := elasticsearch.Config{
        Addresses: []string{"http://elasticsearch:9200"},
    }

    es, err := elasticsearch.NewClient(cfg)
    if err != nil {
        return nil, err
    }

    res, err := es.Info()
    if err != nil {
        // ← Retourne l'erreur au lieu de planter
        return nil, err
    }
    defer res.Body.Close()

    log.Println("🚀 Connecté à Elasticsearch !")
    return es, nil
}


// MigrateToES copie tous les produits de Postgres vers Elasticsearch
func MigrateToES(db *sql.DB, es *elasticsearch.Client) {
    rows, err := db.Query("SELECT name, price, url, image_url, source, category, reference FROM products")
    if err != nil {
        log.Fatalf("❌ Erreur lors de la lecture de la base de données: %v", err)
    }
    defer rows.Close()

    count := 0
    for rows.Next() {
        var p models.Product
        err := rows.Scan(&p.Name, &p.Price, &p.URL, &p.Image, &p.Source, &p.Category, &p.Reference)
        if err != nil {
            log.Printf("⚠️ Erreur lors du scan: %v", err)
            continue
        }

        data, _ := json.Marshal(p)
        _, err = es.Index(
            "products",
            bytes.NewReader(data),
            es.Index.WithDocumentID(p.URL),
            es.Index.WithContext(context.Background()),
        )
        if err != nil {
            log.Printf("⚠️ Erreur d'indexation pour %s: %v", p.URL, err)
        }
        count++
    }
    log.Printf("✅ Migration terminée : %d produits ont été indexés dans Elasticsearch !", count)
}