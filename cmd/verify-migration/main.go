package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/go-sql-driver/mysql"
)

func main() {
	dsn := "hajo4kids:[_wEahKPxkI[2h1P@tcp(172.17.0.1:3367)/hajo4kids?charset=utf8mb4"
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Check unique IDs
	fmt.Println("=== Duplicate Analysis ===")
	var totalRows, uniqueIDs int
	db.QueryRow("SELECT COUNT(*) FROM kategorien_old").Scan(&totalRows)
	db.QueryRow("SELECT COUNT(DISTINCT id) FROM kategorien_old").Scan(&uniqueIDs)
	fmt.Printf("  kategorien_old: %d total rows, %d unique IDs\n", totalRows, uniqueIDs)

	// Categories in new table
	var newCount int
	db.QueryRow("SELECT COUNT(*) FROM kategories").Scan(&newCount)
	fmt.Printf("  kategories: %d rows\n", newCount)

	// Bild_id status
	var withBildID int
	db.QueryRow("SELECT COUNT(*) FROM kategories WHERE bild_id IS NOT NULL AND bild_id > 0").Scan(&withBildID)
	fmt.Printf("  With bild_id: %d\n", withBildID)
	fmt.Printf("  Without bild_id: %d\n", newCount-withBildID)

	// Verify bild_id links to actual images
	fmt.Println("\n=== Bild Verification ===")
	rows, _ := db.Query(`
		SELECT k.id, k.name, k.bild_id, b.id as bild_exists, b.filename
		FROM kategories k
		LEFT JOIN bilds b ON k.bild_id = b.id
		ORDER BY k.id
	`)
	fmt.Println("  ID | Name | bild_id | Image Found")
	for rows.Next() {
		var id int
		var name string
		var bildID sql.NullInt64
		var bildExists sql.NullInt64
		var filename sql.NullString
		rows.Scan(&id, &name, &bildID, &bildExists, &filename)
		bidStr := "NULL"
		if bildID.Valid {
			bidStr = fmt.Sprintf("%d", bildID.Int64)
		}
		status := "❌ NOT FOUND"
		if bildExists.Valid {
			status = "✓"
		}
		if !bildID.Valid || bildID.Int64 == 0 {
			status = "(no bild_id)"
		}
		fmt.Printf("  %d | %s | %s | %s\n", id, name, bidStr, status)
	}
	rows.Close()

	// Check how many bild_ids point to non-existent images
	fmt.Println("\n=== Orphaned bild_ids ===")
	var orphanCount int
	db.QueryRow(`
		SELECT COUNT(*) FROM kategories k
		WHERE k.bild_id IS NOT NULL AND k.bild_id > 0
		AND k.bild_id NOT IN (SELECT id FROM bilds)
	`).Scan(&orphanCount)
	fmt.Printf("  Orphaned bild_ids: %d\n", orphanCount)

	fmt.Println("\n✅ Summary:")
	fmt.Printf("  - %d unique categories migrated\n", newCount)
	fmt.Printf("  - %d/%d have bild_id set\n", withBildID, newCount)
}