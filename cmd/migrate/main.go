package main

import (
	"flag"
	"fmt"
	"log"
	"strings"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// OldDB models (matching old database structure)
type OldZiel struct {
	ID            uint    `gorm:"primaryKey"`
	Name          string
	Marketer      *uint
	Slugname      string
	PlaceID       string  `gorm:"column:placeid"`
	Webseite      string
	Facebook      string
	Adresse       string
	Stadt         string
	Latitude      float64
	Longitude     float64
	Auszug        string
	Beschreibung  string
	Besucht       bool
	Vorteile      string
	Status        string
	Oeffnungszeiten string `gorm:"column:oeffnungszeiten"`
	Telefonnummer string
	Created       string
	CreatedBy     string `gorm:"column:createdby"`
	Updated       string
	UpdatedBy     string `gorm:"column:updatedby"`
}

func (OldZiel) TableName() string { return "ziele" }

type OldKategorie struct {
	ID              uint   `gorm:"primaryKey"`
	Name            string
	Slugname        string
	Bild            *uint
	Tags            string
	Beschreibung    string
	RequiresWebsite bool
	Status          string
	Created         string
	CreatedBy       string `gorm:"column:createdby"`
	Updated         string
	UpdatedBy       string `gorm:"column:updatedby"`
}

func (OldKategorie) TableName() string { return "kategorien" }

type OldBild struct {
	ID          uint   `gorm:"primaryKey"`
	Name        string
	Autor       string
	Beschreibung string
	Status      string
	Zeitstempel string
	Created     string
	CreatedBy   string `gorm:"column:createdby"`
	Updated     string
	UpdatedBy   string `gorm:"column:updatedby"`
}

func (OldBild) TableName() string { return "bilder" }

type OldUser struct {
	Username string `gorm:"primaryKey"`
	Fullname string
	Email    string
	Role     string
	Password string
	Picture  *uint
}

func (OldUser) TableName() string { return "users" }

type OldFavorit struct {
	ID       uint `gorm:"primaryKey;autoIncrement"`
	Username string
	Ziel     uint
}

func (OldFavorit) TableName() string { return "favoriten" }

type OldRating struct {
	ID       uint   `gorm:"primaryKey;autoIncrement"`
	ZielID   uint   `gorm:"column:ziel_id"`
	Username string
	Value    int
	Comment  string
	Created  string
}

func (OldRating) TableName() string { return "rating" }

type OldMarketer struct {
	ID           uint   `gorm:"primaryKey"`
	Name         string
	Slugname     string
	Username     string
	Logo         *uint
	Beschreibung string
	Webseite     string
	Email        string
	Telefon      string
}

func (OldMarketer) TableName() string { return "marketers" }

type OldEvent struct {
	ID           uint   `gorm:"primaryKey"`
	Title        string
	BildID       *uint  `gorm:"column:bild_id"`
	Date         string
	Begin        string
	End          string
	ZielID       *uint  `gorm:"column:ziel_id"`
	VermarkterID *uint  `gorm:"column:vermarkter_id"`
	Description  string
	Website      string
	Activated    bool
}

func (OldEvent) TableName() string { return "events" }

type OldTrip struct {
	ID          uint   `gorm:"primaryKey"`
	Username    string
	Title       string
	Tags        string
	Description string
	Date        string
	State       string
}

func (OldTrip) TableName() string { return "trip" }

type OldZielKategorie struct {
	ZielID      uint `gorm:"column:ziel_id"`
	KategorieID uint `gorm:"column:kategorie_id"`
}

func (OldZielKategorie) TableName() string { return "ziele_kategorien" }

type OldZielBild struct {
	ZielID uint `gorm:"column:ziel_id"`
	BildID uint `gorm:"column:bild_id"`
}

func (OldZielBild) TableName() string { return "ziele_bilder" }

type OldZielTrip struct {
	TripID uint `gorm:"column:trip_id"`
	ZielID uint `gorm:"column:ziel_id"`
}

func (OldZielTrip) TableName() string { return "ziele_trip" }

// NewDB models (for import)
type NewUser struct {
	ID           uint      `gorm:"primaryKey"`
	Username     string    `gorm:"size:50;uniqueIndex;not null"`
	Email        string    `gorm:"size:255;uniqueIndex;not null"`
	PasswordHash string    `gorm:"size:255;not null"`
	Role         string    `gorm:"size:20;default:'user'"`
	FirstName    string    `gorm:"size:100"`
	LastName     string    `gorm:"size:100"`
	Active       bool      `gorm:"default:true"`
	CreatedAt    time.Time `gorm:"column:created"`
	UpdatedAt    time.Time `gorm:"column:updated"`
}

func (NewUser) TableName() string { return "users" }

// BildLookup maps old bild IDs to paths
var bildLookup = make(map[uint]string)

type NewKategorie struct {
	ID           uint      `gorm:"primaryKey"`
	Name         string    `gorm:"size:255;not null"`
	Beschreibung string    `gorm:"type:text"`
	Bild         string    `gorm:"size:500"`
	SortOrder    int       `gorm:"default:0"`
	CreatedAt    time.Time `gorm:"column:created"`
	UpdatedAt    time.Time `gorm:"column:updated"`
}

func (NewKategorie) TableName() string { return "kategorien" }

type NewZiel struct {
	ID              uint      `gorm:"primaryKey"`
	MarketerID      *uint     `gorm:"index"`
	Name            string    `gorm:"size:255;not null"`
	Slugname        string    `gorm:"size:255;uniqueIndex"`
	PlaceID         string    `gorm:"size:255;column:placeid"`
	Webseite        string    `gorm:"size:500"`
	Facebook        string    `gorm:"size:255"`
	Adresse         string    `gorm:"size:500"`
	Stadt           string    `gorm:"size:255;index"`
	Latitude        float64
	Longitude       float64
	Auszug          string    `gorm:"type:text"`
	Beschreibung    string    `gorm:"type:text"`
	Besucht         bool      `gorm:"default:false"`
	Vorteile        string    `gorm:"type:text"`
	Oeffnungszeiten string    `gorm:"type:text"`
	Telefonnummer   string    `gorm:"size:50"`
	Status          string    `gorm:"size:20;default:'draft'"`
	CreatedAt       time.Time `gorm:"column:created"`
	UpdatedAt       time.Time `gorm:"column:updated"`
	CreatedBy       *uint
	UpdatedBy       *uint
}

func (NewZiel) TableName() string { return "ziele" }

type NewZielKategorie struct {
	ZielID      uint `gorm:"primaryKey;column:ziel_id"`
	KategorieID uint `gorm:"primaryKey;column:kategorie_id"`
}

func (NewZielKategorie) TableName() string { return "ziel_kategorien" }

type NewZielBild struct {
	ZielID uint `gorm:"primaryKey;column:ziel_id"`
	BildID uint `gorm:"primaryKey;column:bild_id"`
}

func (NewZielBild) TableName() string { return "ziel_bilder" }

type NewBild struct {
	ID           uint      `gorm:"primaryKey"`
	Filename     string    `gorm:"size:255;not null"`
	OriginalName string    `gorm:"size:255;column:original_name"`
	MimeType     string    `gorm:"size:100"`
	Size         int64
	Path         string    `gorm:"size:500"`
	Thumbnail    string    `gorm:"size:500"`
	Alt          string    `gorm:"size:255"`
	Autor        string    `gorm:"size:255"`
	Beschreibung string    `gorm:"type:text"`
	IsPrimary    bool      `gorm:"default:false"`
	CreatedAt    time.Time `gorm:"column:created"`
	UpdatedAt    time.Time `gorm:"column:updated"`
}

func (NewBild) TableName() string { return "bilder" }

type NewMarketer struct {
	ID           uint      `gorm:"primaryKey"`
	Name         string    `gorm:"size:255;not null"`
	Beschreibung string    `gorm:"type:text"`
	Logo         string    `gorm:"size:500"`
	Webseite     string    `gorm:"size:500"`
	Email        string    `gorm:"size:255"`
	Telefon      string    `gorm:"size:50"`
	CreatedAt    time.Time `gorm:"column:created"`
	UpdatedAt    time.Time `gorm:"column:updated"`
}

func (NewMarketer) TableName() string { return "vermarkter" }

type NewEvent struct {
	ID           uint       `gorm:"primaryKey"`
	ZielID       *uint      `gorm:"index"`
	Title        string     `gorm:"size:255;not null"`
	Beschreibung string     `gorm:"type:text"`
	StartDatum   time.Time  `gorm:"column:start_datum"`
	EndDatum     *time.Time `gorm:"column:end_datum"`
	Ort          string     `gorm:"size:255"`
	CreatedAt    time.Time  `gorm:"column:created"`
	UpdatedAt    time.Time  `gorm:"column:updated"`
}

func (NewEvent) TableName() string { return "veranstaltungen" }

type NewFavorit struct {
	ID        uint      `gorm:"primaryKey"`
	UserID    uint      `gorm:"not null;uniqueIndex:idx_user_ziel"`
	ZielID    uint      `gorm:"not null;uniqueIndex:idx_user_ziel"`
	CreatedAt time.Time `gorm:"column:created"`
}

func (NewFavorit) TableName() string { return "favoriten" }

type NewRating struct {
	ID        uint      `gorm:"primaryKey"`
	ZielID    uint      `gorm:"not null;index"`
	UserID    uint      `gorm:"not null;index"`
	Score     int       `gorm:"not null"`
	Comment   string    `gorm:"type:text"`
	CreatedAt time.Time `gorm:"column:created"`
	UpdatedAt time.Time `gorm:"column:updated"`
}

func (NewRating) TableName() string { return "rating" }

type NewTrip struct {
	ID          uint      `gorm:"primaryKey"`
	UserID      uint      `gorm:"not null;index"`
	Title       string    `gorm:"size:255;not null"`
	Beschreibung string   `gorm:"type:text"`
	IsPublic    bool      `gorm:"default:false;column:is_public"`
	CreatedAt   time.Time `gorm:"column:created"`
	UpdatedAt   time.Time `gorm:"column:updated"`
}

func (NewTrip) TableName() string { return "trips" }

type NewTripZiel struct {
	TripID uint `gorm:"primaryKey;column:trip_id"`
	ZielID uint `gorm:"primaryKey;column:ziel_id"`
}

func (NewTripZiel) TableName() string { return "trip_ziele" }

// User lookup map
var userLookup = make(map[string]uint)

func main() {
	oldHost := flag.String("old-host", "localhost", "Old database host")
	oldPort := flag.String("old-port", "3306", "Old database port")
	oldUser := flag.String("old-user", "root", "Old database user")
	oldPass := flag.String("old-pass", "", "Old database password")
	oldDBName := flag.String("old-db", "hajo4kids", "Old database name")
	
	newHost := flag.String("new-host", "localhost", "New database host")
	newPort := flag.String("new-port", "3306", "New database port")
	newUser := flag.String("new-user", "root", "New database user")
	newPass := flag.String("new-pass", "", "New database password")
	newDBName := flag.String("new-db", "hajo4kids", "New database name")
	
	dryRun := flag.Bool("dry-run", false, "Only show what would be migrated")
	
	flag.Parse()

	log.SetFlags(log.LstdFlags | log.Lmicroseconds)

	// Connect to old database
	oldDSN := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		*oldUser, *oldPass, *oldHost, *oldPort, *oldDBName)
	
	oldDB, err := gorm.Open(mysql.Open(oldDSN), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		log.Fatalf("❌ Failed to connect to old database: %v", err)
	}
	log.Println("✅ Connected to old database")

	// Connect to new database
	newDSN := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		*newUser, *newPass, *newHost, *newPort, *newDBName)
	
	newDB, err := gorm.Open(mysql.Open(newDSN), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		log.Fatalf("❌ Failed to connect to new database: %v", err)
	}
	log.Println("✅ Connected to new database")

	if *dryRun {
		log.Println("🔍 DRY RUN MODE - No data will be written")
	}

	// 1. Migrate Users
	log.Println("\n📧 Migrating Users...")
	var oldUsers []OldUser
	if err := oldDB.Find(&oldUsers).Error; err != nil {
		log.Printf("❌ Error fetching users: %v", err)
	} else {
		log.Printf("   Found %d users", len(oldUsers))
		for _, u := range oldUsers {
			if *dryRun {
				userLookup[u.Username] = 0
				continue
			}
			newUser := NewUser{
				Username: u.Username,
				Email: u.Email,
				PasswordHash: u.Password,
				Role: mapRole(u.Role),
				FirstName: u.Fullname,
				Active: true,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}
			if err := newDB.Create(&newUser).Error; err != nil {
				if !strings.Contains(err.Error(), "Duplicate entry") {
					log.Printf("   ❌ Error creating user %s: %v", u.Username, err)
				}
			} else {
				userLookup[u.Username] = newUser.ID
				log.Printf("   ✅ Created user: %s (ID: %d)", u.Username, newUser.ID)
			}
		}
	}

	// 2. Migrate Bilder FIRST (needed for Kategorie bild lookup)
	log.Println("\n🖼️  Migrating Bilder...")
	var oldBilder []OldBild
	if err := oldDB.Find(&oldBilder).Error; err != nil {
		log.Printf("❌ Error fetching bilder: %v", err)
	} else {
		log.Printf("   Found %d bilder", len(oldBilder))
		for _, b := range oldBilder {
			// Build path - construct from ID
			bildPath := fmt.Sprintf("2016/10/%d.jpg", b.ID)
			bildLookup[b.ID] = bildPath
			
			if *dryRun {
				continue
			}
			newBild := NewBild{
				ID:           b.ID,
				Filename:     b.Name,
				OriginalName: b.Name,
				Alt:          b.Beschreibung,
				Autor:        b.Autor,
				Beschreibung: b.Beschreibung,
				Path:         bildPath,
				CreatedAt:    parseTime(b.Created),
				UpdatedAt:    parseTime(b.Updated),
			}
			if err := newDB.Create(&newBild).Error; err != nil {
				if !strings.Contains(err.Error(), "Duplicate entry") {
					log.Printf("   ❌ Error creating bild %d: %v", b.ID, err)
				}
			}
		}
	}

	// 3. Migrate Kategorien (after Bilder, so bildLookup is populated)
	log.Println("\n📁 Migrating Kategorien...")
	var oldKategorien []OldKategorie
	if err := oldDB.Find(&oldKategorien).Error; err != nil {
		log.Printf("❌ Error fetching kategorien: %v", err)
	} else {
		log.Printf("   Found %d kategorien", len(oldKategorien))
		for _, k := range oldKategorien {
			if *dryRun {
				continue
			}
			// Convert bild ID to path if set
			var bildPath string
			if k.Bild != nil && *k.Bild > 0 {
				if path, ok := bildLookup[*k.Bild]; ok {
					bildPath = path
				}
			}
			newKat := NewKategorie{
				ID: k.ID,
				Name: k.Name,
				Beschreibung: k.Beschreibung,
				Bild: bildPath,
				SortOrder: 0,
				CreatedAt: parseTime(k.Created),
				UpdatedAt: parseTime(k.Updated),
			}
			if err := newDB.Create(&newKat).Error; err != nil {
				if !strings.Contains(err.Error(), "Duplicate entry") {
					log.Printf("   ❌ Error creating kategorie %s: %v", k.Name, err)
				}
			} else {
				log.Printf("   ✅ Created kategorie: %s (ID: %d, Bild: %s)", k.Name, k.ID, bildPath)
			}
		}
	}

	// 4. Migrate Marketer
	log.Println("\n🏢 Migrating Marketer...")
	var oldMarketer []OldMarketer
	if err := oldDB.Find(&oldMarketer).Error; err != nil {
		log.Printf("❌ Error fetching marketer: %v", err)
	} else {
		log.Printf("   Found %d marketer", len(oldMarketer))
		for _, m := range oldMarketer {
			if *dryRun {
				continue
			}
			newMark := NewMarketer{
				ID: m.ID,
				Name: m.Name,
				Beschreibung: m.Beschreibung,
				Webseite: m.Webseite,
				Email: m.Email,
				Telefon: m.Telefon,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}
			if err := newDB.Create(&newMark).Error; err != nil {
				if !strings.Contains(err.Error(), "Duplicate entry") {
					log.Printf("   ❌ Error creating marketer %s: %v", m.Name, err)
				}
			}
		}
	}

	// 5. Migrate Ziele
	log.Println("\n🎯 Migrating Ziele...")
	var oldZiele []OldZiel
	if err := oldDB.Find(&oldZiele).Error; err != nil {
		log.Printf("❌ Error fetching ziele: %v", err)
	} else {
		log.Printf("   Found %d ziele", len(oldZiele))
		for _, z := range oldZiele {
			if *dryRun {
				continue
			}
			newZiel := NewZiel{
				ID: z.ID,
				MarketerID: z.Marketer,
				Name: z.Name,
				Slugname: z.Slugname,
				PlaceID: z.PlaceID,
				Webseite: z.Webseite,
				Facebook: z.Facebook,
				Adresse: z.Adresse,
				Stadt: z.Stadt,
				Latitude: z.Latitude,
				Longitude: z.Longitude,
				Auszug: z.Auszug,
				Beschreibung: z.Beschreibung,
				Besucht: z.Besucht,
				Vorteile: z.Vorteile,
				Status: z.Status,
				Oeffnungszeiten: z.Oeffnungszeiten,
				Telefonnummer: z.Telefonnummer,
				CreatedAt: parseTime(z.Created),
				UpdatedAt: parseTime(z.Updated),
			}
			if err := newDB.Create(&newZiel).Error; err != nil {
				if !strings.Contains(err.Error(), "Duplicate entry") {
					log.Printf("   ❌ Error creating ziel %s: %v", z.Name, err)
				}
			}
		}
	}

	// 6. Migrate Ziel-Kategorien
	log.Println("\n🔗 Migrating Ziel-Kategorien...")
	var oldZielKategorien []OldZielKategorie
	if err := oldDB.Find(&oldZielKategorien).Error; err != nil {
		log.Printf("❌ Error fetching ziel_kategorien: %v", err)
	} else {
		log.Printf("   Found %d relations", len(oldZielKategorien))
		for _, zk := range oldZielKategorien {
			if *dryRun {
				continue
			}
			newZK := NewZielKategorie{
				ZielID: zk.ZielID,
				KategorieID: zk.KategorieID,
			}
			if err := newDB.Create(&newZK).Error; err != nil {
				// Ignore duplicate errors
			}
		}
	}

	// 7. Migrate Ziel-Bilder
	log.Println("\n🖼️  Migrating Ziel-Bilder...")
	var oldZielBilder []OldZielBild
	if err := oldDB.Find(&oldZielBilder).Error; err != nil {
		log.Printf("❌ Error fetching ziel_bilder: %v", err)
	} else {
		log.Printf("   Found %d relations", len(oldZielBilder))
		for _, zb := range oldZielBilder {
			if *dryRun {
				continue
			}
			newZB := NewZielBild{
				ZielID: zb.ZielID,
				BildID: zb.BildID,
			}
			if err := newDB.Create(&newZB).Error; err != nil {
				// Ignore duplicate errors
			}
		}
	}

	// 8. Migrate Events
	log.Println("\n📅 Migrating Events...")
	var oldEvents []OldEvent
	if err := oldDB.Find(&oldEvents).Error; err != nil {
		log.Printf("❌ Error fetching events: %v", err)
	} else {
		log.Printf("   Found %d events", len(oldEvents))
		for _, e := range oldEvents {
			if *dryRun {
				continue
			}
			startDate := parseDate(e.Date, e.Begin)
			var endDate *time.Time
			if e.End != "" {
				t := parseDate(e.Date, e.End)
				endDate = &t
			}
			newEvent := NewEvent{
				ID: e.ID,
				ZielID: e.ZielID,
				Title: e.Title,
				Beschreibung: e.Description,
				StartDatum: startDate,
				EndDatum: endDate,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}
			if err := newDB.Create(&newEvent).Error; err != nil {
				if !strings.Contains(err.Error(), "Duplicate entry") {
					log.Printf("   ❌ Error creating event %s: %v", e.Title, err)
				}
			}
		}
	}

	// 9. Migrate Trips
	log.Println("\n🚗 Migrating Trips...")
	var oldTrips []OldTrip
	if err := oldDB.Find(&oldTrips).Error; err != nil {
		log.Printf("❌ Error fetching trips: %v", err)
	} else {
		log.Printf("   Found %d trips", len(oldTrips))
		for _, t := range oldTrips {
			if *dryRun {
				continue
			}
			userID := userLookup[t.Username]
			newTrip := NewTrip{
				ID: t.ID,
				UserID: userID,
				Title: t.Title,
				Beschreibung: t.Description,
				IsPublic: t.State == "public",
				CreatedAt: parseTime(t.Date),
				UpdatedAt: parseTime(t.Date),
			}
			if err := newDB.Create(&newTrip).Error; err != nil {
				if !strings.Contains(err.Error(), "Duplicate entry") {
					log.Printf("   ❌ Error creating trip %s: %v", t.Title, err)
				}
			}
		}
	}

	// 10. Migrate Trip-Ziele
	log.Println("\n🔗 Migrating Trip-Ziele...")
	var oldZielTrips []OldZielTrip
	if err := oldDB.Find(&oldZielTrips).Error; err != nil {
		log.Printf("❌ Error fetching ziel_trip: %v", err)
	} else {
		log.Printf("   Found %d relations", len(oldZielTrips))
		for _, zt := range oldZielTrips {
			if *dryRun {
				continue
			}
			newZT := NewTripZiel{
				TripID: zt.TripID,
				ZielID: zt.ZielID,
			}
			if err := newDB.Create(&newZT).Error; err != nil {
				// Ignore duplicate errors
			}
		}
	}

	// Summary
	log.Println("\n" + strings.Repeat("=", 50))
	log.Println("📊 MIGRATION SUMMARY")
	log.Println(strings.Repeat("=", 50))
	log.Printf("  Users:       %d", len(oldUsers))
	log.Printf("  Kategorien: %d", len(oldKategorien))
	log.Printf("  Bilder:      %d", len(oldBilder))
	log.Printf("  Marketer:    %d", len(oldMarketer))
	log.Printf("  Ziele:       %d", len(oldZiele))
	log.Printf("  Events:      %d", len(oldEvents))
	log.Printf("  Trips:       %d", len(oldTrips))
	
	if *dryRun {
		log.Println("\n🔍 Run without --dry-run to perform actual migration")
	} else {
		log.Println("\n✅ Migration completed!")
	}
}

func mapRole(oldRole string) string {
	switch strings.ToLower(oldRole) {
	case "admin":
		return "admin"
	case "reporter":
		return "reporter"
	default:
		return "user"
	}
}

func parseTime(s string) time.Time {
	formats := []string{
		"2006-01-02 15:04:05",
		"2006-01-02",
		time.RFC3339,
	}
	for _, format := range formats {
		if t, err := time.Parse(format, s); err == nil {
			return t
		}
	}
	return time.Now()
}

func parseDate(dateStr, timeStr string) time.Time {
	dt := dateStr
	if timeStr != "" {
		dt = dateStr + " " + timeStr
	}
	return parseTime(dt)
}