package database

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"strings"
    "io"

	"github.com/elastic/go-elasticsearch/v8"
	"tunisianet-scraper/models"
)

func InitES() (*elasticsearch.Client, error) {
	cfg := elasticsearch.Config{
		Addresses: []string{"http://elasticsearch:9200"},
	}

	es, err := elasticsearch.NewClient(cfg)
	if err != nil {
		return nil, err
	}

	// Vérification que ES répond
	res, err := es.Info()
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	// Crée l'index avec le bon mapping s'il n'existe pas
	existsRes, err := es.Indices.Exists([]string{"products"})
	if err != nil || existsRes.StatusCode == 404 {
		// ⚠️ IMPORTANT : "reference" a maintenant un sous-champ "keyword"
		// pour permettre les recherches exactes (term query) par référence produit.
		mapping := `{
			"mappings": {
				"properties": {
					"name": {
						"type": "text",
						"fields": {
							"keyword": { "type": "keyword" }
						}
					},
					"reference": {
						"type": "text",
						"fields": {
							"keyword": { "type": "keyword" }
						}
					},
					"category":  { "type": "keyword" },
					"source":    { "type": "keyword" },
					"url":       { "type": "keyword" },
					"price":     { "type": "text" }
				}
			}
		}`
		createRes, err := es.Indices.Create(
			"products",
			es.Indices.Create.WithBody(strings.NewReader(mapping)),
		)
		if err != nil {
			log.Printf("⚠️ Erreur création index ES: %v", err)
		} else {
			defer createRes.Body.Close()
			log.Println("✅ Index ES 'products' créé")
		}
	}

	log.Println("🚀 Connecté à Elasticsearch !")
	return es, nil
}

func MigrateToES(db *sql.DB, es *elasticsearch.Client) {
	rows, err := db.Query("SELECT name, price, url, image_url, source, category, reference FROM products")
	if err != nil {
		log.Printf("❌ Erreur lecture BD pour migration ES: %v", err)
		return
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var p models.Product
		err := rows.Scan(&p.Name, &p.Price, &p.URL, &p.Image, &p.Source, &p.Category, &p.Reference)
		if err != nil {
			continue
		}
		data, _ := json.Marshal(p)
		res, err := es.Index(
			"products",
			bytes.NewReader(data),
			es.Index.WithDocumentID(models.DocIDFromURL(p.URL)),
			es.Index.WithContext(context.Background()),
		)
		if err != nil {
			log.Printf("⚠️ Erreur transport ES pour %s: %v", p.Name, err)
			continue
		}
		if res.IsError() {
			bodyBytes, _ := io.ReadAll(res.Body)
			log.Printf("⚠️ ES a rejeté le document '%s': status=%d body=%s", p.Name, res.StatusCode, string(bodyBytes))
			res.Body.Close()
			continue
		}
		res.Body.Close()
		count++
	}
	log.Printf("✅ Migration ES terminée : %d produits indexés", count)
}