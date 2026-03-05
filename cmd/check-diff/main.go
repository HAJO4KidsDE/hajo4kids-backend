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

	// Check which IDs exist where
	fmt.Println("=== ID Analysis ===")
	fmt.Println("\nIDs in kategorien_old:")
	rows, _ := db.Query("SELECT id, name FROM kategorien_old ORDER BY id")
	for rows.Next() {
		var id int
		var name string
		rows.Scan(&id, &name)
		fmt.Printf("  %d: %s\n", id, name)
	}
	rows.Close()

	fmt.Println("\n\nIDs in kategories:")
	rows2, _ := db.Query("SELECT id, name FROM kategories ORDER BY id")
	for rows2.Next() {
		var id int
		var name string
		rows2.Scan(&id, &name)
		fmt.Printf("  %d: %s\n", id, name)
	}
	rows2.Close()

	// Find missing by name
	fmt.Println("\n\n=== Missing Categories (by name) ===")
	rows3, _ := db.Query(`
		SELECT ko.id, ko.name
		FROM kategorien_old ko
		WHERE ko.name NOT IN (SELECT name FROM kategories)
		ORDER BY ko.id
	`)
	for rows3.Next() {
		var id int
		var name string
		rows3.Scan(&id, &name)
		fmt.Printf("  %d: %s\n", id, name)
	}
	rows3.Close()

	// Check if IDs overlap
	fmt.Println("\n\n=== ID Overlap ===")
	var matchingIDs, missingIDs int
	db.QueryRow(`SELECT COUNT(*) FROM kategorien_old ko WHERE ko.id IN (SELECT id FROM kategories)`).Scan(&matchingIDs)
	db.QueryRow(`SELECT COUNT(*) FROM kategorien_old ko WHERE ko.id NOT IN (SELECT id FROM kategories)`).Scan(&missingIDs)
	fmt.Printf("  IDs matching: %d\n", matchingIDs)
	fmt.Printf("  IDs missing: %d\n", missingIDs)
}