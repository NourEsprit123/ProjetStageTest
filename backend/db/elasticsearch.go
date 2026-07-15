package db

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/elastic/go-elasticsearch/v8"
	"tunisianet-scraper/models"
)

var ES *elasticsearch.Client

const productsIndexMapping = `{
  "mappings": {
    "properties": {
      "name":     { "type": "text", "analyzer": "french" },
      "price":    { "type": "keyword" },
      "category": { "type": "keyword" },
      "source":   { "type": "keyword" },
      "in_stock": { "type": "boolean" },
      "url":      { "type": "keyword" },
      "image":    { "type": "keyword" }
    }
  }
}`

func InitES() {
	cfg := elasticsearch.Config{Addresses: []string{"http://elasticsearch:9200"}}
	client, err := elasticsearch.NewClient(cfg)
	if err != nil {
		fmt.Printf("⚠️ Client ES: %v\n", err)
		return
	}
	ES = client

	res, err := ES.Info()
	if err != nil {
		fmt.Printf("⚠️ ES injoignable: %v\n", err)
		return
	}
	defer res.Body.Close()
	fmt.Println("✅ Connexion Elasticsearch établie")
	ensureProductsIndex()
}

func ensureProductsIndex() {
	res, err := ES.Indices.Exists([]string{"products"})
	if err != nil {
		return
	}
	defer res.Body.Close()
	if res.StatusCode == 200 {
		return
	}
	createRes, err := ES.Indices.Create("products", ES.Indices.Create.WithBody(strings.NewReader(productsIndexMapping)))
	if err != nil {
		fmt.Printf("Erreur création index ES: %v\n", err)
		return
	}
	defer createRes.Body.Close()
	fmt.Println("✅ Index Elasticsearch 'products' créé")
}

func BulkIndexProducts(products []models.Product) error {
	if ES == nil || len(products) == 0 {
		return nil
	}
	var buf bytes.Buffer
	for _, p := range products {
		if p.ID == "" {
			continue
		}
		meta, _ := json.Marshal(map[string]interface{}{"index": map[string]interface{}{"_index": "products", "_id": p.ID}})
		doc, _ := json.Marshal(p)
		buf.Write(meta)
		buf.WriteByte('\n')
		buf.Write(doc)
		buf.WriteByte('\n')
	}
	res, err := ES.Bulk(bytes.NewReader(buf.Bytes()), ES.Bulk.WithIndex("products"), ES.Bulk.WithRefresh("true"))
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.IsError() {
		return fmt.Errorf("erreur bulk ES: %s", res.String())
	}
	fmt.Printf("✅ %d produits indexés dans Elasticsearch\n", len(products))
	return nil
}

func SearchProductsES(search, category string) ([]models.Product, error) {
	if ES == nil {
		return nil, fmt.Errorf("client ES non initialisé")
	}
	must := []map[string]interface{}{}
	if search != "" {
		must = append(must, map[string]interface{}{"match": map[string]interface{}{"name": map[string]interface{}{"query": search, "fuzziness": "AUTO"}}})
	} else {
		must = append(must, map[string]interface{}{"match_all": map[string]interface{}{}})
	}
	filter := []map[string]interface{}{}
	if category != "" {
		filter = append(filter, map[string]interface{}{"term": map[string]interface{}{"category": category}})
	}
	query := map[string]interface{}{"size": 300, "query": map[string]interface{}{"bool": map[string]interface{}{"must": must, "filter": filter}}}

	var buf bytes.Buffer
	json.NewEncoder(&buf).Encode(query)

	res, err := ES.Search(ES.Search.WithIndex("products"), ES.Search.WithBody(&buf))
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.IsError() {
		return nil, fmt.Errorf("erreur recherche ES: %s", res.String())
	}

	var result struct {
		Hits struct {
			Hits []struct {
				Source models.Product `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
	}
	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		return nil, err
	}
	products := make([]models.Product, 0, len(result.Hits.Hits))
	for _, hit := range result.Hits.Hits {
		products = append(products, hit.Source)
	}
	return products, nil
}