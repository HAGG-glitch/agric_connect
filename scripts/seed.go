package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"

	"github.com/agriconnect-ai/internal/config"
	"github.com/agriconnect-ai/internal/database"
	"github.com/agriconnect-ai/internal/models"
	"github.com/agriconnect-ai/internal/repositories"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	db, err := database.Connect(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	migrationsDir := findMigrationsDir()
	migrator := database.NewMigrationRunner(db, migrationsDir)
	if err := migrator.Up(); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	seedFile := findSeedFile()
	data, err := os.ReadFile(seedFile)
	if err != nil {
		log.Fatalf("Failed to read seed file: %v", err)
	}

	var docs []models.AgriculturalDocument
	if err := json.Unmarshal(data, &docs); err != nil {
		log.Fatalf("Failed to parse seed data: %v", err)
	}

	// Validate required fields
	for i, doc := range docs {
		if doc.Category == "" || doc.Title == "" || doc.Content == "" {
			log.Printf("Warning: document %d missing required fields (category, title, or content)", i)
		}
	}

	repo := repositories.NewKnowledgeRepository(db)
	if err := repo.SeedDocuments(context.Background(), docs); err != nil {
		log.Fatalf("Failed to seed documents: %v", err)
	}

	fmt.Printf("Successfully seeded %d agricultural documents.\n", len(docs))
}

func findSeedFile() string {
	_, filename, _, _ := runtime.Caller(0)
	dir := filepath.Dir(filename)
	candidates := []string{
		filepath.Join(dir, "../seed/agricultural_documents.json"),
		"/app/seed/agricultural_documents.json",
		"seed/agricultural_documents.json",
	}
	for _, c := range candidates {
		if p, err := filepath.Abs(c); err == nil {
			if _, err := os.Stat(p); err == nil {
				return p
			}
		}
	}
	return "seed/agricultural_documents.json"
}

func findMigrationsDir() string {
	_, filename, _, _ := runtime.Caller(0)
	dir := filepath.Dir(filename)
	candidates := []string{
		filepath.Join(dir, "../migrations"),
		"/app/migrations",
		"migrations",
	}
	for _, c := range candidates {
		if p, err := filepath.Abs(c); err == nil {
			if _, err := os.Stat(p); err == nil {
				return p
			}
		}
	}
	return "migrations"
}
