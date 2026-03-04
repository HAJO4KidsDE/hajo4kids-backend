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

	log.Println("Connected to database")

	// 1. Copy users (with column transformations)
	log.Println("\n=== Migrating users ===")
	result, err := db.Exec(`
		INSERT IGNORE INTO users (username, email, password_hash, role, first_name, active, created, updated)
		SELECT username, email, password, 
			CASE WHEN role = '' THEN 'user' ELSE role END,
			COALESCE(fullname, ''),
			CASE WHEN state = 'ENABLED' THEN 1 ELSE 0 END,
			COALESCE(created, NOW()),
			COALESCE(updated, NOW())
		FROM users_old
	`)
	if err != nil {
		log.Printf("Warning: %v", err)
	} else {
		affected, _ := result.RowsAffected()
		log.Printf("✅ Migrated %d users", affected)
	}

	// 2. Copy kategorien
	log.Println("\n=== Migrating kategorien ===")
	result, err = db.Exec(`
		INSERT IGNORE INTO kategorien (id, name, beschreibung, sort_order, created, updated)
		SELECT id, name, COALESCE(beschreibung, ''), 0, COALESCE(created, NOW()), COALESCE(updated, NOW())
		FROM kategorien_old
	`)
	if err != nil {
		log.Printf("Warning: %v", err)
	} else {
		affected, _ := result.RowsAffected()
		log.Printf("✅ Migrated %d kategorien", affected)
	}

	// 3. Copy ziele
	log.Println("\n=== Migrating ziele ===")
	result, err = db.Exec(`
		INSERT IGNORE INTO ziele (id, marketer_id, name, slugname, placeid, webseite, facebook, adresse, stadt, latitude, longitude, auszug, beschreibung, besucht, vorteile, status, oeffnungszeiten, telefonnummer, created, updated)
		SELECT id, marketer, name, slugname, placeid, webseite, facebook, adresse, stadt, latitude, longitude, auszug, beschreibung, besucht, vorteile, status, oeffnungszeiten, telefonnummer, COALESCE(created, NOW()), COALESCE(updated, NOW())
		FROM ziele_old
	`)
	if err != nil {
		log.Printf("Warning: %v", err)
	} else {
		affected, _ := result.RowsAffected()
		log.Printf("✅ Migrated %d ziele", affected)
	}

	// 4. Copy bilder
	log.Println("\n=== Migrating bilder ===")
	result, err = db.Exec(`
		INSERT IGNORE INTO bilder (id, filename, original_name, alt, path, created, updated)
		SELECT id, name, name, beschreibung, name, COALESCE(zeitstempel, NOW()), COALESCE(updated, NOW())
		FROM bilder_old
	`)
	if err != nil {
		log.Printf("Warning: %v", err)
	} else {
		affected, _ := result.RowsAffected()
		log.Printf("✅ Migrated %d bilder", affected)
	}

	// 5. Copy marketers → vermarkter
	log.Println("\n=== Migrating vermarkter ===")
	result, err = db.Exec(`
		INSERT IGNORE INTO vermarkter (id, name, beschreibung, logo, webseite, email, telefon, created, updated)
		SELECT id, name, COALESCE(beschreibung, ''), logo, webseite, email, telefon, NOW(), NOW()
		FROM marketers_old
	`)
	if err != nil {
		log.Printf("Warning: %v", err)
	} else {
		affected, _ := result.RowsAffected()
		log.Printf("✅ Migrated %d vermarkter", affected)
	}

	// 6. Copy events → veranstaltungen
	log.Println("\n=== Migrating veranstaltungen ===")
	result, err = db.Exec(`
		INSERT IGNORE INTO veranstaltungen (id, ziel_id, title, beschreibung, start_datum, created, updated)
		SELECT id, ziel_id, title, description, 
			COALESCE(STR_TO_DATE(CONCAT(date, ' ', COALESCE(begin, '00:00:00')), '%Y-%m-%d %H:%i:%s'), NOW()),
			NOW(), NOW()
		FROM events_old
	`)
	if err != nil {
		log.Printf("Warning: %v", err)
	} else {
		affected, _ := result.RowsAffected()
		log.Printf("✅ Migrated %d veranstaltungen", affected)
	}

	// 7. Copy trip → trips
	log.Println("\n=== Migrating trips ===")
	result, err = db.Exec(`
		INSERT IGNORE INTO trips (id, user_id, title, beschreibung, is_public, created, updated)
		SELECT t.id, u.id, t.title, t.description, CASE WHEN t.state = 'public' THEN 1 ELSE 0 END, t.date, t.date
		FROM trip_old t
		LEFT JOIN users u ON u.username = t.username
	`)
	if err != nil {
		log.Printf("Warning: %v", err)
	} else {
		affected, _ := result.RowsAffected()
		log.Printf("✅ Migrated %d trips", affected)
	}

	// 8. Copy favoriten
	log.Println("\n=== Migrating favoriten ===")
	result, err = db.Exec(`
		INSERT IGNORE INTO favoriten (user_id, ziel_id, created)
		SELECT u.id, f.ziel, NOW()
		FROM favoriten_old f
		LEFT JOIN users u ON u.username = f.username
	`)
	if err != nil {
		log.Printf("Warning: %v", err)
	} else {
		affected, _ := result.RowsAffected()
		log.Printf("✅ Migrated %d favoriten", affected)
	}

	// 9. Copy rating
	log.Println("\n=== Migrating rating ===")
	result, err = db.Exec(`
		INSERT IGNORE INTO rating (ziel_id, user_id, score, comment, created, updated)
		SELECT r.ziel_id, u.id, r.value, r.comment, COALESCE(r.created, NOW()), COALESCE(r.created, NOW())
		FROM rating_old r
		LEFT JOIN users u ON u.username = r.username
	`)
	if err != nil {
		log.Printf("Warning: %v", err)
	} else {
		affected, _ := result.RowsAffected()
		log.Printf("✅ Migrated %d ratings", affected)
	}

	// 10. Copy ziel_kategorien
	log.Println("\n=== Migrating ziel_kategorien ===")
	result, err = db.Exec(`
		INSERT IGNORE INTO ziel_kategorien (ziel_id, kategorie_id)
		SELECT ziel_id, kategorie_id FROM ziele_kategorien_old
	`)
	if err != nil {
		log.Printf("Warning: %v", err)
	} else {
		affected, _ := result.RowsAffected()
		log.Printf("✅ Migrated %d ziel_kategorien", affected)
	}

	// 11. Copy ziel_bilder
	log.Println("\n=== Migrating ziel_bilder ===")
	result, err = db.Exec(`
		INSERT IGNORE INTO ziel_bilder (ziel_id, bild_id)
		SELECT ziel_id, bild_id FROM ziele_bilder_old
	`)
	if err != nil {
		log.Printf("Warning: %v", err)
	} else {
		affected, _ := result.RowsAffected()
		log.Printf("✅ Migrated %d ziel_bilder", affected)
	}

	// 12. Copy trip_ziele
	log.Println("\n=== Migrating trip_ziele ===")
	result, err = db.Exec(`
		INSERT IGNORE INTO trip_ziele (trip_id, ziel_id)
		SELECT trip_id, ziel_id FROM ziele_trip_old
	`)
	if err != nil {
		log.Printf("Warning: %v", err)
	} else {
		affected, _ := result.RowsAffected()
		log.Printf("✅ Migrated %d trip_ziele", affected)
	}

	log.Println("\n==================================================")
	log.Println("✅ Migration completed!")
	log.Println("==================================================")
}