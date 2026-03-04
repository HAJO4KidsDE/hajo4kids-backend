package models

import (
	"time"

	"gorm.io/gorm"
)

// ==================== USER ====================

// User represents a user
type User struct {
	ID           uint           `json:"id" gorm:"primaryKey"`
	Username     string         `json:"username" gorm:"size:50;uniqueIndex;not null"`
	Email        string         `json:"email" gorm:"size:255;uniqueIndex;not null"`
	PasswordHash string         `json:"-" gorm:"size:255;not null"`
	Role         string         `json:"role" gorm:"size:20;default:'user'"` // guest, user, reporter, admin
	FirstName    string         `json:"first_name,omitempty" gorm:"size:100"`
	LastName     string         `json:"last_name,omitempty" gorm:"size:100"`
	Active       bool           `json:"active" gorm:"default:true"`
	CreatedAt    time.Time      `json:"created"`
	UpdatedAt    time.Time      `json:"updated"`
	DeletedAt    gorm.DeletedAt `json:"-" gorm:"index"`
}

// PasswordReset represents a password reset token
type PasswordReset struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	UserID    uint      `json:"user_id" gorm:"not null;index"`
	User      *User     `json:"-" gorm:"foreignKey:UserID"`
	Token     string    `json:"token" gorm:"size:255;uniqueIndex;not null"`
	ExpiresAt time.Time `json:"expires_at"`
	Used      bool      `json:"used" gorm:"default:false"`
	CreatedAt time.Time `json:"created"`
}

// ==================== ZIELE ====================

// Ziel represents a destination/location
type Ziel struct {
	ID            uint           `json:"id" gorm:"primaryKey"`
	MarketerID    *uint          `json:"marketer,omitempty" gorm:"index"`
	Marketer      *Vermarkter    `json:"marketer_data,omitempty" gorm:"foreignKey:MarketerID"`
	Name          string         `json:"name" gorm:"size:255;not null"`
	Slugname      string         `json:"slugname" gorm:"size:255;uniqueIndex"`
	PlaceID       string         `json:"placeid,omitempty" gorm:"size:255"`
	Webseite      string         `json:"webseite,omitempty" gorm:"size:500"`
	Facebook      string         `json:"facebook,omitempty" gorm:"size:255"`
	Adresse       string         `json:"adresse,omitempty" gorm:"size:500"`
	Stadt         string         `json:"stadt,omitempty" gorm:"size:255;index"`
	Latitude      float64        `json:"latitude"`
	Longitude     float64        `json:"longitude"`
	Auszug        string         `json:"auszug,omitempty" gorm:"type:text"`
	Beschreibung  string         `json:"beschreibung,omitempty" gorm:"type:text"`
	Besucht       bool           `json:"besucht" gorm:"default:false"`
	Vorteile      string         `json:"vorteile,omitempty" gorm:"type:text"`
	Oeffnungszeiten string       `json:"oeffnungszeiten,omitempty" gorm:"type:text"`
	Telefonnummer string         `json:"telefonnummer,omitempty" gorm:"size:50"`
	Status        string         `json:"status" gorm:"size:20;default:'DESIGN'"` // DESIGN, PUBLISHED, ARCHIVED
	CreatedAt     time.Time      `json:"created"`
	UpdatedAt     time.Time      `json:"updated"`
	CreatedBy     *uint          `json:"createdby,omitempty"`
	UpdatedBy     *uint          `json:"updatedby,omitempty"`
	DeletedAt     gorm.DeletedAt `json:"-" gorm:"index"`
	
	// Relations
	Kategorien    []Kategorie    `json:"kategorien,omitempty" gorm:"many2many:ziel_kategorien;"`
	Bilder        []Bild         `json:"bilder,omitempty" gorm:"many2many:ziel_bilder;"`
	Ratings       []Rating       `json:"ratings,omitempty" gorm:"foreignKey:ZielID"`
	Favoriten     []Favorit      `json:"-" gorm:"foreignKey:ZielID"`
	Veranstaltungen []Veranstaltung `json:"veranstaltungen,omitempty" gorm:"foreignKey:ZielID"`
	
	// Computed fields (not stored in DB)
	Distanz       *float64       `json:"distanz,omitempty" gorm:"-"`
	Rating        float64         `json:"rating" gorm:"-"`
	Favorit       bool           `json:"favorit" gorm:"-"`
	Favorits      int            `json:"favorits" gorm:"-"`
}

// Kategorie represents a category
type Kategorie struct {
	ID          uint           `json:"id" gorm:"primaryKey"`
	Name        string         `json:"name" gorm:"size:255;not null"`
	Beschreibung string        `json:"beschreibung,omitempty" gorm:"type:text"`
	Bild        string         `json:"bild,omitempty" gorm:"size:500"` // Fallback image for ziele without images
	SortOrder   int            `json:"sort_order" gorm:"default:0"`
	CreatedAt   time.Time      `json:"created"`
	UpdatedAt   time.Time      `json:"updated"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`
}

// Bild represents an image
type Bild struct {
	ID          uint           `json:"id" gorm:"primaryKey"`
	Filename    string         `json:"filename" gorm:"size:255;not null"`
	OriginalName string        `json:"original_name,omitempty" gorm:"size:255"`
	MimeType    string         `json:"mime_type,omitempty" gorm:"size:100"`
	Size        int64          `json:"size"`
	Path        string         `json:"path" gorm:"size:500"`
	Thumbnail   string         `json:"thumbnail,omitempty" gorm:"size:500"`
	Alt         string         `json:"alt,omitempty" gorm:"size:255"`
	Autor       string         `json:"autor,omitempty" gorm:"size:255"`
	Beschreibung string        `json:"beschreibung,omitempty" gorm:"type:text"`
	IsPrimary   bool           `json:"is_primary" gorm:"default:false"`
	CreatedAt   time.Time      `json:"created"`
	UpdatedAt   time.Time      `json:"updated"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`
}

// Rating represents a user rating for a Ziel
type Rating struct {
	ID        uint           `json:"id" gorm:"primaryKey"`
	ZielID    uint           `json:"ziel_id" gorm:"not null;index"`
	UserID    uint           `json:"user_id" gorm:"not null;index"`
	Score     int            `json:"score" gorm:"not null"` // 1-5
	Comment   string         `json:"comment,omitempty" gorm:"type:text"`
	Revoked   bool           `json:"revoked" gorm:"default:false"`
	CreatedAt time.Time      `json:"created"`
	UpdatedAt time.Time      `json:"updated"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

// Favorit represents a user's favorite Ziel
type Favorit struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	UserID    uint      `json:"user_id" gorm:"not null;uniqueIndex:idx_user_ziel"`
	ZielID    uint      `json:"ziel_id" gorm:"not null;uniqueIndex:idx_user_ziel"`
	CreatedAt time.Time `json:"created"`
}

// Vermarkter represents a marketer/organization
type Vermarkter struct {
	ID          uint           `json:"id" gorm:"primaryKey"`
	Name        string         `json:"name" gorm:"size:255;not null"`
	Beschreibung string        `json:"beschreibung,omitempty" gorm:"type:text"`
	Logo        string         `json:"logo,omitempty" gorm:"size:500"`
	Webseite    string         `json:"webseite,omitempty" gorm:"size:500"`
	Email       string         `json:"email,omitempty" gorm:"size:255"`
	Telefon     string         `json:"telefon,omitempty" gorm:"size:50"`
	CreatedAt   time.Time      `json:"created"`
	UpdatedAt   time.Time      `json:"updated"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`
}

// Veranstaltung represents an event
type Veranstaltung struct {
	ID          uint           `json:"id" gorm:"primaryKey"`
	ZielID      *uint          `json:"ziel_id,omitempty" gorm:"index"`
	Ziel        *Ziel          `json:"ziel,omitempty" gorm:"foreignKey:ZielID"`
	Title       string         `json:"title" gorm:"size:255;not null"`
	Beschreibung string        `json:"beschreibung,omitempty" gorm:"type:text"`
	StartDatum  time.Time      `json:"start_datum"`
	EndDatum    *time.Time     `json:"end_datum,omitempty"`
	Ort         string         `json:"ort,omitempty" gorm:"size:255"`
	CreatedAt   time.Time      `json:"created"`
	UpdatedAt   time.Time      `json:"updated"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`
}

// Trip represents a user's journey/story
type Trip struct {
	ID          uint           `json:"id" gorm:"primaryKey"`
	UserID      uint           `json:"user_id" gorm:"not null;index"`
	Title       string         `json:"title" gorm:"size:255;not null"`
	Beschreibung string        `json:"beschreibung,omitempty" gorm:"type:text"`
	IsPublic    bool           `json:"is_public" gorm:"default:false"`
	CreatedAt   time.Time      `json:"created"`
	UpdatedAt   time.Time      `json:"updated"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`
	
	// Relations
	Ziele       []Ziel         `json:"ziele,omitempty" gorm:"many2many:trip_ziele;"`
}

// KontaktMessage represents a contact form submission
type KontaktMessage struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	Name      string    `json:"name" gorm:"size:255;not null"`
	Email     string    `json:"email" gorm:"size:255;not null"`
	Betreff   string    `json:"betreff,omitempty" gorm:"size:255"`
	Nachricht string    `json:"nachricht" gorm:"type:text;not null"`
	Read      bool      `json:"read" gorm:"default:false"`
	CreatedAt time.Time `json:"created"`
}

// StatisticsQuery represents search statistics
type StatisticsQuery struct {
	ID                   uint      `json:"id" gorm:"primaryKey"`
	Session              string    `json:"session,omitempty" gorm:"size:100"`
	Device               string    `json:"device,omitempty" gorm:"size:255"`
	LocationLatitude     float64   `json:"location_latitude,omitempty"`
	LocationLongitude    float64   `json:"location_longitude,omitempty"`
	QueryRaw             string    `json:"query_raw,omitempty" gorm:"type:text"`
	QueryText            string    `json:"query_text,omitempty" gorm:"type:text"`
	QueryLocationName    string    `json:"query_location_name,omitempty" gorm:"size:255"`
	QueryLocationLat     float64   `json:"query_location_latitude,omitempty"`
	QueryLocationLng     float64   `json:"query_location_longitude,omitempty"`
	QueryCategories      string    `json:"query_categories,omitempty" gorm:"type:text"`
	QueryMarketer        *uint     `json:"query_marketer,omitempty"`
	QueryTrip            bool      `json:"query_trip" gorm:"default:false"`
	GuestAccess          bool      `json:"guest_access" gorm:"default:true"`
	Destinations         string    `json:"destinations,omitempty" gorm:"type:text"` // JSON array of IDs
	CreatedAt            time.Time `json:"created"`
}

// ==================== SHOP ====================

// ShopKategorie represents a shop category
type ShopKategorie struct {
	ID          uint           `json:"id" gorm:"primaryKey"`
	Name        string         `json:"name" gorm:"size:255;not null"`
	Slug        string         `json:"slug" gorm:"size:255;uniqueIndex"`
	Beschreibung string        `json:"beschreibung,omitempty" gorm:"type:text"`
	SortOrder   int            `json:"sort_order" gorm:"default:0"`
	CreatedAt   time.Time      `json:"created"`
	UpdatedAt   time.Time      `json:"updated"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`
	
	// Relations
	Items       []ShopItem     `json:"items,omitempty" gorm:"foreignKey:KategorieID"`
}

// ShopItem represents a product in the shop
type ShopItem struct {
	ID            uint           `json:"id" gorm:"primaryKey"`
	KategorieID   *uint          `json:"kategorie_id,omitempty" gorm:"index"`
	Kategorie     *ShopKategorie `json:"kategorie,omitempty" gorm:"foreignKey:KategorieID"`
	Name          string         `json:"name" gorm:"size:255;not null"`
	Slug          string         `json:"slug" gorm:"size:255;uniqueIndex"`
	Beschreibung  string         `json:"beschreibung,omitempty" gorm:"type:text"`
	Preis         float64        `json:"preis" gorm:"not null"`
	PreisAlt      float64        `json:"preis_alt,omitempty"` // Original price for discounts
	Bild          string         `json:"bild,omitempty" gorm:"size:500"`
	Bilder        string         `json:"bilder,omitempty" gorm:"type:text"` // JSON array of image URLs
	Lagerbestand  int            `json:"lagerbestand" gorm:"default:0"`
	Verfuegbar    bool           `json:"verfuegbar" gorm:"default:true"`
	Highlights    string         `json:"highlights,omitempty" gorm:"type:text"` // JSON array
	Status        string         `json:"status" gorm:"size:20;default:'DESIGN'"` // DESIGN, PUBLISHED, ARCHIVED
	SortOrder     int            `json:"sort_order" gorm:"default:0"`
	CreatedAt     time.Time      `json:"created"`
	UpdatedAt     time.Time      `json:"updated"`
	CreatedBy     *uint          `json:"createdby,omitempty"`
	DeletedAt     gorm.DeletedAt `json:"-" gorm:"index"`
}

// ==================== POSTS/BLOG ====================

// Post represents a blog article
type Post struct {
	ID           uint           `json:"id" gorm:"primaryKey"`
	Title        string         `json:"title" gorm:"size:255;not null"`
	Slug         string         `json:"slug" gorm:"size:255;uniqueIndex"`
	Excerpt      string         `json:"excerpt,omitempty" gorm:"type:text"`
	Inhalt       string         `json:"inhalt" gorm:"type:text;not null"`
	Bild         string         `json:"bild,omitempty" gorm:"size:500"`
	AutorID      *uint          `json:"autor_id,omitempty" gorm:"index"`
	Autor        *User          `json:"autor,omitempty" gorm:"foreignKey:AutorID"`
	Kategorie    string         `json:"kategorie,omitempty" gorm:"size:100"`
	Tags         string         `json:"tags,omitempty" gorm:"type:text"` // JSON array
	Status       string         `json:"status" gorm:"size:20;default:'DESIGN'"` // DESIGN, PUBLISHED, ARCHIVED
	PublishedAt  *time.Time     `json:"published_at,omitempty"`
	ViewCount    int            `json:"view_count" gorm:"default:0"`
	CreatedAt    time.Time      `json:"created"`
	UpdatedAt    time.Time      `json:"updated"`
	DeletedAt    gorm.DeletedAt `json:"-" gorm:"index"`
	
	// Relations
	Comments     []PostComment  `json:"comments,omitempty" gorm:"foreignKey:PostID"`
}

// PostComment represents a comment on a post
type PostComment struct {
	ID        uint           `json:"id" gorm:"primaryKey"`
	PostID    uint           `json:"post_id" gorm:"not null;index"`
	Post      *Post          `json:"-" gorm:"foreignKey:PostID"`
	UserID    *uint          `json:"user_id,omitempty" gorm:"index"`
	User      *User          `json:"user,omitempty" gorm:"foreignKey:UserID"`
	Name      string         `json:"name,omitempty" gorm:"size:255"` // For guest comments
	Email     string         `json:"email,omitempty" gorm:"size:255"` // For guest comments
	Inhalt    string         `json:"inhalt" gorm:"type:text;not null"`
	Approved  bool           `json:"approved" gorm:"default:false"`
	CreatedAt time.Time      `json:"created"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}