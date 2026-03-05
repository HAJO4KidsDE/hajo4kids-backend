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

	// Show kategorien_old structure
	fmt.Println("=== kategorien_old Structure ===")
	rows, _ := db.Query("DESCRIBE kategorien_old")
	for rows.Next() {
		var field, typ, null, key string
		var def, extra interface{}
		rows.Scan(&field, &typ, &null, &key, &def, &extra)
		fmt.Printf("  %s: %s\n", field, typ)
	}
	rows.Close()

	// Show kategories (new) structure
	fmt.Println("\n=== kategories Structure ===")
	rows2, _ := db.Query("DESCRIBE kategories")
	for rows2.Next() {
		var field, typ, null, key string
		var def, extra interface{}
		rows2.Scan(&field, &typ, &null, &key, &def, &extra)
		fmt.Printf("  %s: %s\n", field, typ)
	}
	rows2.Close()

	// Count old vs new
	var oldCount, newCount int
	db.QueryRow("SELECT COUNT(*) FROM kategorien_old").Scan(&oldCount)
	db.QueryRow("SELECT COUNT(*) FROM kategories").Scan(&newCount)
	fmt.Printf("\n=== Row Counts ===\n")
	fmt.Printf("  kategorien_old: %d\n", oldCount)
	fmt.Printf("  kategories: %d\n", newCount)

	// Show sample from old
	fmt.Println("\n=== kategorien_old Sample (with bild FK) ===")
	rows3, _ := db.Query("SELECT id, name, bild, status FROM kategorien_old WHERE bild > 0 LIMIT 5")
	for rows3.Next() {
		var id int
		var name, status string
		var bild int
		rows3.Scan(&id, &name, &bild, &status)
		fmt.Printf("  ID:%d Name:'%s' bild_id:%d status:'%s'\n", id, name, bild, status)
	}
	rows3.Close()

	// Show sample from new
	fmt.Println("\n=== kategories Sample (current state) ===")
	rows4, _ := db.Query("SELECT * FROM kategories LIMIT 5")
	cols, _ := rows4.Columns()
	fmt.Printf("  Columns: %v\n", cols)
	for rows4.Next() {
		vals := make([]interface{}, len(cols))
		ptrs := make([]interface{}, len(cols))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		rows4.Scan(ptrs...)
		fmt.Printf("  ")
		for i, col := range cols {
			switch v := vals[i].(type) {
			case []byte:
				fmt.Printf("%s='%s' ", col, string(v))
			case nil:
				fmt.Printf("%s=NULL ", col)
			default:
				fmt.Printf("%s=%v ", col, v)
			}
		}
		fmt.Println()
	}
	rows4.Close()

	// Check bild_id column exists in kategories
	fmt.Println("\n=== Checking bild_id in kategories ===")
	var bildIDCount int
	err = db.QueryRow("SELECT COUNT(*) FROM kategories WHERE bild_id IS NOT NULL").Scan(&bildIDCount)
	if err != nil {
		fmt.Printf("  bild_id column may not exist: %v\n", err)
	} else {
		fmt.Printf("  Categories with bild_id set: %d\n", bildIDCount)
	}
}