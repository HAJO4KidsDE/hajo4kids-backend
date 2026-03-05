package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/go-sql-driver/mysql"
)

func main() {
	dsn := "hajo4kids:[_wEahKPxkI[2h1P@tcp(172.17.0.1:3367)/hajo4kids?charset=utf8mb4&parseTime=True&loc=Local"
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Step 1: Check current state
	fmt.Println("=== Current State ===")
	var oldCount, newCount int
	db.QueryRow("SELECT COUNT(*) FROM kategorien_old").Scan(&oldCount)
	db.QueryRow("SELECT COUNT(*) FROM kategories").Scan(&newCount)
	fmt.Printf("  kategorien_old: %d rows\n", oldCount)
	fmt.Printf("  kategories: %d rows\n", newCount)

	// Step 2: Add bild_id column if not exists
	fmt.Println("\n=== Adding bild_id column ===")
	_, err = db.Exec("ALTER TABLE kategories ADD COLUMN bild_id INT(11) NULL AFTER beschreibung")
	if err != nil {
		if err.Error() == "Error 1060 (42S21): Duplicate column name 'bild_id'" {
			fmt.Println("  bild_id column already exists")
		} else {
			fmt.Printf("  Error (may already exist): %v\n", err)
		}
	} else {
		fmt.Println("  ✓ bild_id column added")
	}

	// Step 3: Check and add foreign key constraint to bilds table
	fmt.Println("\n=== Checking bilds table ===")
	var bildsCount int
	db.QueryRow("SELECT COUNT(*) FROM bilds").Scan(&bildsCount)
	fmt.Printf("  bilds: %d rows\n", bildsCount)

	// Step 4: Update existing categories with bild_id from old table
	fmt.Println("\n=== Updating bild_id from kategorien_old ===")
	result, err := db.Exec(`
		UPDATE kategories k
		JOIN kategorien_old ko ON k.id = ko.id
		SET k.bild_id = ko.bild
		WHERE ko.bild IS NOT NULL AND ko.bild > 0
	`)
	if err != nil {
		fmt.Printf("  Error updating: %v\n", err)
	} else {
		affected, _ := result.RowsAffected()
		fmt.Printf("  ✓ Updated %d categories with bild_id\n", affected)
	}

	// Step 5: Check what's missing
	fmt.Println("\n=== Checking for missing categories ===")
	rows, err := db.Query(`
		SELECT ko.id, ko.name, ko.bild, ko.status, ko.beschreibung
		FROM kategorien_old ko
		WHERE ko.id NOT IN (SELECT id FROM kategories)
	`)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	var missingCategories []struct {
		ID          int
		Name        string
		Bild        sql.NullInt64
		Status      string
		Beschreibung sql.NullString
	}
	for rows.Next() {
		var c struct {
			ID          int
			Name        string
			Bild        sql.NullInt64
			Status      string
			Beschreibung sql.NullString
		}
		rows.Scan(&c.ID, &c.Name, &c.Bild, &c.Status, &c.Beschreibung)
		missingCategories = append(missingCategories, c)
	}
	rows.Close()

	fmt.Printf("  Found %d missing categories\n", len(missingCategories))

	// Step 6: Insert missing categories
	if len(missingCategories) > 0 {
		fmt.Println("\n=== Inserting missing categories ===")
		for _, c := range missingCategories {
			beschreibung := ""
			if c.Beschreibung.Valid {
				beschreibung = c.Beschreibung.String
			}
			bildID := "NULL"
			if c.Bild.Valid && c.Bild.Int64 > 0 {
				bildID = fmt.Sprintf("%d", c.Bild.Int64)
			}

			// Use original ID and dates from old table
			_, err := db.Exec(`
				INSERT INTO kategories (id, name, beschreibung, bild_id, sort_order, created_at, updated_at)
				SELECT ?, ?, ?, ?, 0, COALESCE(ko.created, NOW()), COALESCE(ko.updated, NOW())
				FROM kategorien_old ko WHERE ko.id = ?
			`, c.ID, c.Name, beschreibung, bildID, c.ID)
			if err != nil {
				// Try without bild_id if it failed
				_, err2 := db.Exec(`
					INSERT INTO kategories (id, name, beschreibung, sort_order, created_at, updated_at)
					SELECT ?, ?, ?, 0, COALESCE(ko.created, NOW()), COALESCE(ko.updated, NOW())
					FROM kategorien_old ko WHERE ko.id = ?
				`, c.ID, c.Name, beschreibung, c.ID)
				if err2 != nil {
					fmt.Printf("  ✗ Failed to insert '%s': %v\n", c.Name, err)
				} else {
					fmt.Printf("  ✓ Inserted '%s' (ID %d) without bild_id\n", c.Name, c.ID)
				}
			} else {
				fmt.Printf("  ✓ Inserted '%s' (ID %d) with bild_id=%s\n", c.Name, c.ID, bildID)
			}
		}
	}

	// Step 7: Final verification
	fmt.Println("\n=== Final Verification ===")
	db.QueryRow("SELECT COUNT(*) FROM kategories").Scan(&newCount)
	var withBildID int
	db.QueryRow("SELECT COUNT(*) FROM kategories WHERE bild_id IS NOT NULL AND bild_id > 0").Scan(&withBildID)
	fmt.Printf("  Total categories: %d\n", newCount)
	fmt.Printf("  With bild_id set: %d\n", withBildID)
	fmt.Printf("  Without bild_id: %d\n", newCount-withBildID)

	// Show sample
	fmt.Println("\n=== Sample Categories ===")
	rows2, _ := db.Query(`
		SELECT k.id, k.name, k.bild_id, b.filename
		FROM kategories k
		LEFT JOIN bilds b ON k.bild_id = b.id
		WHERE k.bild_id IS NOT NULL AND k.bild_id > 0
		LIMIT 5
	`)
	fmt.Println("  ID | Name | bild_id | Image File")
	for rows2.Next() {
		var id int
		var name string
		var bildID sql.NullInt64
		var filename sql.NullString
		rows2.Scan(&id, &name, &bildID, &filename)
		bildStr := "NULL"
		if bildID.Valid {
			bildStr = fmt.Sprintf("%d", bildID.Int64)
		}
		fileStr := "NULL"
		if filename.Valid {
			fileStr = filename.String
		}
		fmt.Printf("  %d | %s | %s | %s\n", id, name, bildStr, fileStr)
	}
	rows2.Close()

	fmt.Println("\n✅ Migration complete!")
}