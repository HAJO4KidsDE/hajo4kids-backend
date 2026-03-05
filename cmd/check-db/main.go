package main

import (
	"fmt"
	"log"

	"github.com/HAJO4KidsDE/hajo4kids-backend/internal/config"
	"github.com/HAJO4KidsDE/hajo4kids-backend/internal/database"
)

func main() {
	cfg := config.Load()
	if err := database.Connect(cfg); err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}

	db := database.DB

	// Check kategorien_old table
	fmt.Println("=== Kategorien_OLD Table Schema ===")
	var oldColumns []struct {
		CID     int
		Name    string
		Type    string
		NotNull int
		PK      int
	}
	db.Raw("PRAGMA table_info(kategorien_old)").Scan(&oldColumns)
	for _, col := range oldColumns {
		fmt.Printf("  %s (%s)\n", col.Name, col.Type)
	}

	fmt.Println("\n=== Kategorien_OLD Data ===")
	var oldKategorien []map[string]interface{}
	db.Table("kategorien_old").Find(&oldKategorien)
	for i, k := range oldKategorien {
		fmt.Printf("  [%d] %v\n", i+1, k)
	}

	// Check kategories table schema (GORM names it 'kategories')
	var columns []struct {
		CID        int
		Name       string
		Type       string
		NotNull    int
		DefaultVal interface{}
		PK         int
	}
	db.Raw("PRAGMA table_info(kategories)").Scan(&columns)
	fmt.Println("\n=== Kategories Table Schema ===")
	for _, col := range columns {
		fmt.Printf("  %s (%s) - NotNull: %d, PK: %d\n", col.Name, col.Type, col.NotNull, col.PK)
	}

	// Check kategorien data
	var kategorien []struct {
		ID           uint
		Name         string
		Beschreibung string
		Bild         string
		BildID       *uint
		SortOrder    int
	}
	db.Table("kategories").Find(&kategorien)
	fmt.Println("\n=== Kategories Data ===")
	for _, k := range kategorien {
		fmt.Printf("  ID:%d Name:'%s' Bild:'%s' BildID:%v\n", k.ID, k.Name, k.Bild, k.BildID)
	}

	// Check bilder count
	var bildCount int64
	db.Table("bilds").Count(&bildCount)
	fmt.Printf("\n=== Bilds Count: %d ===\n", bildCount)

	// Check if bild_id column exists
	var hasBildID bool
	for _, col := range columns {
		if col.Name == "bild_id" {
			hasBildID = true
			break
		}
	}

	if !hasBildID {
		fmt.Println("\n*** bild_id column does NOT exist! Need to add it. ***")
	} else {
		fmt.Println("\n*** bild_id column exists! ***")
	}

	// Check ziel_kategorien (many-to-many)
	var zielKatCount int64
	db.Table("ziel_kategories").Count(&zielKatCount)
	fmt.Printf("\n=== Ziel_Kategories Count: %d ===\n", zielKatCount)

	// Check all tables
	var tables []string
	db.Raw("SELECT name FROM sqlite_master WHERE type='table' ORDER BY name").Scan(&tables)
	fmt.Println("\n=== All Tables ===")
	for _, t := range tables {
		var count int64
		db.Table(t).Count(&count)
		fmt.Printf("  %s: %d rows\n", t, count)
	}
}