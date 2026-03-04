package main

import (
	"bufio"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	_ "github.com/go-sql-driver/mysql"
)

func main() {
	dumpFile := flag.String("dump-file", "", "Path to SQL dump file")
	newHost := flag.String("new-host", "localhost", "New database host")
	newPort := flag.String("new-port", "3306", "New database port")
	newUser := flag.String("new-user", "root", "New database user")
	newPass := flag.String("new-pass", "", "New database password")
	newDB := flag.String("new-db", "hajo4kids", "New database name")

	flag.Parse()

	if *dumpFile == "" {
		log.Fatal("Please provide -dump-file")
	}

	log.SetFlags(log.LstdFlags)

	// Connect to new database
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local&multiStatements=true",
		*newUser, *newPass, *newHost, *newPort, *newDB)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}
	log.Println("Connected to new database")

	// Open dump file
	file, err := os.Open(*dumpFile)
	if err != nil {
		log.Fatalf("Failed to open dump file: %v", err)
	}
	defer file.Close()

	log.Println("Opened dump file")

	// Parse and execute
	scanner := bufio.NewScanner(file)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 10*1024*1024)

	var statement strings.Builder
	var tableCount int
	var insertCount int
	var skipCount int

	for scanner.Scan() {
		line := scanner.Text()

		// Skip comments
		if strings.HasPrefix(line, "--") || strings.HasPrefix(line, "/*") || line == "" {
			continue
		}

		statement.WriteString(line)
		statement.WriteString("\n")

		// Execute on semicolon
		if strings.HasSuffix(strings.TrimSpace(line), ";") {
			sql := statement.String()
			statement.Reset()

			// Handle CREATE TABLE
			if strings.Contains(sql, "CREATE TABLE") {
				tableName := extractTableName(sql)
				if tableName != "" {
					tableCount++
					log.Printf("Creating table: %s", tableName)
				}
				_, err := db.Exec(sql)
				if err != nil && !strings.Contains(err.Error(), "already exists") {
					log.Printf("CREATE TABLE warning: %v", err)
				}
				continue
			}

			// Handle INSERT INTO
			if strings.Contains(sql, "INSERT INTO") {
				tableName := extractInsertTable(sql)
				if tableName == "" {
					continue
				}

				// Transform INSERT for incompatible tables
				transformedSQL := transformInsert(sql, tableName)

				_, err := db.Exec(transformedSQL)
				if err != nil {
					if strings.Contains(err.Error(), "Duplicate entry") {
						skipCount++
					} else if insertCount%500 == 0 {
						log.Printf("INSERT warning (table %s): %v", tableName, err)
					}
				} else {
					insertCount++
					if insertCount%100 == 0 {
						log.Printf("Inserted %d rows...", insertCount)
					}
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		log.Printf("Scanner error: %v", err)
	}

	log.Println("")
	log.Println("MIGRATION SUMMARY")
	log.Println("==================================================")
	log.Printf("Tables created: %d", tableCount)
	log.Printf("Rows inserted: %d", insertCount)
	log.Printf("Skipped (duplicates): %d", skipCount)
	log.Println("")
	log.Println("Migration completed!")
}

func extractTableName(sql string) string {
	start := strings.Index(sql, "`")
	if start == -1 {
		return ""
	}
	end := strings.Index(sql[start+1:], "`")
	if end == -1 {
		return ""
	}
	return sql[start+1 : start+1+end]
}

func extractInsertTable(sql string) string {
	marker := "INSERT INTO `"
	start := strings.Index(sql, marker)
	if start == -1 {
		return ""
	}
	rest := sql[start+len(marker):]
	end := strings.Index(rest, "`")
	if end == -1 {
		return ""
	}
	return rest[:end]
}

func transformInsert(sql string, tableName string) string {
	// For users table, transform column names
	if tableName == "users" {
		sql = strings.ReplaceAll(sql, "`fullname`", "`first_name`")
		sql = strings.ReplaceAll(sql, "`password`", "`password_hash`")
		sql = strings.ReplaceAll(sql, "`picture`", "`picture_id`")
		sql = strings.ReplaceAll(sql, "'ENABLED'", "true")
		sql = strings.ReplaceAll(sql, "'DISABLED'", "false")
	}

	// For ziele table
	if tableName == "ziele" {
		sql = strings.ReplaceAll(sql, "`marketer`", "`marketer_id`")
	}

	// For trip table
	if tableName == "trip" {
		sql = strings.ReplaceAll(sql, "`state`", "`is_public`")
		sql = strings.ReplaceAll(sql, "'public'", "true")
		sql = strings.ReplaceAll(sql, "'private'", "false")
	}

	return sql
}