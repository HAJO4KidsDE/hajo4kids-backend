package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"

	_ "github.com/go-sql-driver/mysql"
)

func main() {
	host := flag.String("host", "localhost", "Database host")
	port := flag.String("port", "3306", "Database port")
	user := flag.String("user", "root", "Database user")
	pass := flag.String("pass", "", "Database password")
	dbName := flag.String("db", "hajo4kids", "Database name")
	flag.Parse()

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local&multiStatements=true",
		*user, *pass, *host, *port, *dbName)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatalf("❌ Failed to connect: %v", err)
	}
	defer db.Close()

	log.Println("✅ Connected to database")

	// Step 1: Rename old tables
	log.Println("\n📦 Step 1: Renaming old tables...")

	renames := []struct {
		oldName string
		newName string
	}{
		{"users", "users_old"},
		{"kategorien", "kategorien_old"},
		{"ziele", "ziele_old"},
		{"bilder", "bilder_old"},
		{"marketers", "marketers_old"},
		{"events", "events_old"},
		{"trip", "trip_old"},
		{"favoriten", "favoriten_old"},
		{"rating", "rating_old"},
		{"ziele_kategorien", "ziele_kategorien_old"},
		{"ziele_bilder", "ziele_bilder_old"},
		{"ziele_trip", "ziele_trip_old"},
	}

	for _, r := range renames {
		_, err := db.Exec(fmt.Sprintf("RENAME TABLE `%s` TO `%s`", r.oldName, r.newName))
		if err != nil {
			log.Printf("   ⚠️  Could not rename %s: %v (might not exist)", r.oldName, err)
		} else {
			log.Printf("   ✅ Renamed %s → %s", r.oldName, r.newName)
		}
	}

	log.Println("\n✅ Old tables renamed with _old suffix")
	log.Println("\n⚠️  Now:")
	log.Println("   1. Restart the backend - GORM will create new tables")
	log.Println("   2. Run this tool again with -copy flag to copy data")
}

/*
Step 2 script - run after backend creates new tables:

package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"

	_ "github.com/go-sql-driver/mysql"
)

func main() {
	host := flag.String("host", "localhost", "Database host")
	port := flag.String("port", "3306", "Database port")
	user := flag.String("user", "root", "Database user")
	pass := flag.String("pass", "", "Database password")
	dbName := flag.String("db", "hajo4kids", "Database name")
	flag.Parse()

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		*user, *pass, *host, *port, *dbName)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer db.Close()

	log.Println("Copying data from _old tables to new tables...")

	// Copy users
	result, _ := db.Exec(`
		INSERT IGNORE INTO users (username, email, password_hash, role, first_name, active, created, updated)
		SELECT username, email, password,
			CASE WHEN role = '' THEN 'user' ELSE role END,
			COALESCE(fullname, ''),
			CASE WHEN state = 'ENABLED' THEN 1 ELSE 0 END,
			COALESCE(created, NOW()), COALESCE(updated, NOW())
		FROM users_old
	`)
	affected, _ := result.RowsAffected()
	log.Printf("✅ Copied %d users", affected)

	// Copy kategorien
	result, _ = db.Exec(`
		INSERT IGNORE INTO kategorien (id, name, beschreibung, sort_order, created, updated)
		SELECT id, name, COALESCE(beschreibung, ''), 0, COALESCE(created, NOW()), COALESCE(updated, NOW())
		FROM kategorien_old
	`)
	affected, _ = result.RowsAffected()
	log.Printf("✅ Copied %d kategorien", affected)

	// Copy ziele
	result, _ = db.Exec(`
		INSERT IGNORE INTO ziele (id, marketer_id, name, slugname, placeid, webseite, facebook, adresse, stadt, latitude, longitude, auszug, beschreibung, besucht, vorteile, status, oeffnungszeiten, telefonnummer, created, updated)
		SELECT id, marketer, name, slugname, placeid, webseite, facebook, adresse, stadt, latitude, longitude, auszug, beschreibung, besucht, vorteile, status, oeffnungszeiten, telefonnummer, COALESCE(created, NOW()), COALESCE(updated, NOW())
		FROM ziele_old
	`)
	affected, _ = result.RowsAffected()
	log.Printf("✅ Copied %d ziele", affected)

	// Copy more tables...
	log.Println("✅ Done!")
}
*/