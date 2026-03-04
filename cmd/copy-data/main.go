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
		log.Fatalf("❌ Failed to connect: %v", err)
	}
	defer db.Close()

	log.Println("✅ Connected to database")
	log.Println("\n📦 Copying remaining data...")

	// 1. Copy marketers_old -> vermarkters (correct columns)
	log.Println("\n=== Copying vermarkters ===")
	result, err := db.Exec(`
		INSERT IGNORE INTO vermarkters (id, name, logo, created_at, updated_at)
		SELECT id, name, logo, COALESCE(created, NOW()), COALESCE(updated, NOW())
		FROM marketers_old
	`)
	if err != nil {
		log.Printf("   ⚠️  Error: %v", err)
	} else {
		affected, _ := result.RowsAffected()
		log.Printf("   ✅ Copied %d vermarkters", affected)
	}

	// 2. Copy rating_old -> ratings (correct columns)
	// Note: rating_old has 0 rows, but let's do it right
	log.Println("\n=== Copying ratings ===")
	result, err = db.Exec(`
		INSERT IGNORE INTO ratings (ziel_id, user_id, score, comment, created_at, updated_at)
		SELECT r.ziel_id, u.id, r.value, r.comment, COALESCE(r.timestamp, NOW()), COALESCE(r.timestamp, NOW())
		FROM rating_old r
		LEFT JOIN users_old u ON BINARY u.username = BINARY r.username
		WHERE u.id IS NOT NULL
	`)
	if err != nil {
		log.Printf("   ⚠️  Error: %v", err)
	} else {
		affected, _ := result.RowsAffected()
		log.Printf("   ✅ Copied %d ratings", affected)
	}

	// 3. Copy favoriten_old -> favorits (correct columns)
	// Note: No users yet, so this won't copy
	log.Println("\n=== Copying favorits ===")
	result, err = db.Exec(`
		INSERT IGNORE INTO favorits (user_id, ziel_id, created_at)
		SELECT u.id, f.ziel, NOW()
		FROM favoriten_old f
		LEFT JOIN users_old u ON BINARY u.username = BINARY f.username
		WHERE u.id IS NOT NULL
	`)
	if err != nil {
		log.Printf("   ⚠️  Error: %v", err)
	} else {
		affected, _ := result.RowsAffected()
		log.Printf("   ✅ Copied %d favorits", affected)
	}

	// Final counts
	log.Println("\n" + "==================================================")
	log.Println("📊 Final row counts:")
	tables := []struct {
		name string
	}{
		{"users"}, {"kategories"}, {"ziels"}, {"bilds"},
		{"vermarkters"}, {"veranstaltungs"}, {"trips"},
		{"favorits"}, {"ratings"},
		{"ziel_kategorien"}, {"ziel_bilder"}, {"trip_ziele"},
	}

	for _, t := range tables {
		var count int
		db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM `%s`", t.name)).Scan(&count)
		log.Printf("  %s: %d rows", t.name, count)
	}

	log.Println("\n✅ Done!")
}