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

	// Show columns for old tables with errors
	tables := []string{"marketers_old", "rating_old", "favoriten_old", "users_old"}
	for _, table := range tables {
		fmt.Printf("\n=== %s ===\n", table)
		rows, _ := db.Query(fmt.Sprintf("DESCRIBE `%s`", table))
		for rows.Next() {
			var field, typ, null, key string
			var def, extra interface{}
			rows.Scan(&field, &typ, &null, &key, &def, &extra)
			fmt.Printf("  %s: %s\n", field, typ)
		}
		rows.Close()
	}

	// Show row counts
	fmt.Println("\n=== Row Counts ===")
	countTables := []string{"users_old", "kategorien_old", "ziele_old", "bilder_old", 
		"marketers_old", "events_old", "trip_old", "favoriten_old", "rating_old"}
	for _, table := range countTables {
		var count int
		db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM `%s`", table)).Scan(&count)
		fmt.Printf("  %s: %d rows\n", table, count)
	}
}