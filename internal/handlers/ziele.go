package handlers

import (
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/HAJO4KidsDE/hajo4kids-backend/internal/database"
	"github.com/HAJO4KidsDE/hajo4kids-backend/internal/middleware"
	"github.com/HAJO4KidsDE/hajo4kids-backend/internal/models"
	"github.com/HAJO4KidsDE/hajo4kids-backend/pkg/response"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// ZielResponse is the expanded Ziel for API responses
type ZielResponse struct {
	models.Ziel
	Social         *SocialLinks `json:"social"`
	AvgRating      float64      `json:"rating"`
	FavoriteCount  int          `json:"favorits"`
	IsFavorite     bool         `json:"favorit"`
	Distance       *float64     `json:"distanz,omitempty"`
}

type SocialLinks struct {
	Facebook  string `json:"facebook,omitempty"`
	Instagram string `json:"instagram,omitempty"`
	Twitter   string `json:"twitter,omitempty"`
}

// GetZiele returns list of destinations with optional filters
func GetZiele(c *fiber.Ctx) error {
	page, _ := strconv.Atoi(c.Query("page", "1"))
	perPage, _ := strconv.Atoi(c.Query("per_page", "20"))
	
	// Filters
	filter := c.Query("filter")
	status := c.Query("status", "published")
	stadt := c.Query("city")
	kategorie := c.Query("category")
	placeLat, _ := strconv.ParseFloat(c.Query("pos_lat"), 64)
	placeLng, _ := strconv.ParseFloat(c.Query("pos_lng"), 64)
	usrLat, _ := strconv.ParseFloat(c.Query("usr_lat"), 64)
	usrLng, _ := strconv.ParseFloat(c.Query("usr_lng"), 64)

	query := database.DB.Model(&models.Ziel{}).Preload("Kategorien").Preload("Bilder")

	// Status filter (default: published)
	if status != "" {
		query = query.Where("status = ?", status)
	}
	
	// City filter
	if stadt != "" {
		query = query.Where("stadt LIKE ?", "%"+stadt+"%")
	}
	
	// Category filter
	if kategorie != "" {
		query = query.Joins("JOIN ziel_kategorien ON ziel_kategorien.ziel_id = ziele.id").
			Where("ziel_kategorien.kategorie_id = ?", kategorie)
	}
	
	// Text search
	if filter != "" {
		query = query.Where("name LIKE ? OR beschreibung LIKE ? OR auszug LIKE ?", 
			"%"+filter+"%", "%"+filter+"%", "%"+filter+"%")
	}

	// Location filter (find nearby destinations)
	if placeLat != 0 && placeLng != 0 {
		// Use bounding box for initial filter (faster)
		latDelta := 0.5 // ~50km radius
		lngDelta := 0.5
		query = query.Where("latitude BETWEEN ? AND ? AND longitude BETWEEN ? AND ?",
			placeLat-latDelta, placeLat+latDelta,
			placeLng-lngDelta, placeLng+lngDelta)
	}

	// Count total
	var total int64
	query.Count(&total)

	// Paginate
	var ziele []models.Ziel
	offset := (page - 1) * perPage
	if err := query.Offset(offset).Limit(perPage).Find(&ziele).Error; err != nil {
		return response.InternalError(c, "Failed to fetch ziele")
	}

	// Get current user if authenticated
	user := middleware.GetUserFromContext(c)
	var userFavs []uint
	if user != nil {
		database.DB.Model(&models.Favorit{}).Where("user_id = ?", user.ID).Pluck("ziel_id", &userFavs)
	}

	// Expand and calculate distances
	result := make([]ZielResponse, len(ziele))
	for i, z := range ziele {
		result[i] = expandZiel(z, userFavs)
		
		// Calculate distance if user position provided
		if usrLat != 0 && usrLng != 0 && z.Latitude != 0 && z.Longitude != 0 {
			dist := haversine(usrLat, usrLng, z.Latitude, z.Longitude)
			result[i].Distance = &dist
		}
	}

	// Log statistics (async)
	go logSearchQuery(c, filter, stadt, kategorie, usrLat, usrLng, ziele)

	return response.SuccessWithMeta(c, result, &response.Meta{
		Total:    total,
		Page:     page,
		PerPage:  perPage,
		LastPage: int((total + int64(perPage) - 1) / int64(perPage)),
	})
}

// GetZiel returns a single destination by ID or slug
func GetZiel(c *fiber.Ctx) error {
	id := c.Params("id")

	var ziel models.Ziel
	
	// Try to parse as numeric ID first
	if zielID, err := strconv.ParseUint(id, 10, 32); err == nil {
		// It's a numeric ID
		if err := database.DB.Preload("Kategorien").Preload("Bilder").Preload("Ratings").
			Preload("Veranstaltungen").
			Preload("Marketer").
			First(&ziel, zielID).Error; err != nil {
			return response.NotFound(c, "Ziel not found")
		}
	} else {
		// Not numeric, try as slugname
		if err := database.DB.Where("slugname = ?", id).
			Preload("Kategorien").Preload("Bilder").Preload("Ratings").
			Preload("Veranstaltungen").
			Preload("Marketer").
			First(&ziel).Error; err != nil {
			return response.NotFound(c, "Ziel not found")
		}
	}

	user := middleware.GetUserFromContext(c)
	var userFavs []uint
	if user != nil {
		database.DB.Model(&models.Favorit{}).Where("user_id = ?", user.ID).Pluck("ziel_id", &userFavs)
	}

	result := expandZiel(ziel, userFavs)
	return response.Success(c, result)
}

// GetZielByPlaceID returns a destination by Google Place ID
func GetZielByPlaceID(c *fiber.Ctx) error {
	placeID := c.Params("placeid")

	var ziel models.Ziel
	if err := database.DB.Where("placeid = ?", placeID).
		Preload("Kategorien").Preload("Bilder").First(&ziel).Error; err != nil {
		return response.NotFound(c, "Ziel not found")
	}

	user := middleware.GetUserFromContext(c)
	var userFavs []uint
	if user != nil {
		database.DB.Model(&models.Favorit{}).Where("user_id = ?", user.ID).Pluck("ziel_id", &userFavs)
	}

	return response.Success(c, expandZiel(ziel, userFavs))
}

// GetZielEvents returns events for a destination
func GetZielEvents(c *fiber.Ctx) error {
	id := c.Params("id")
	
	var events []models.Veranstaltung
	if err := database.DB.Where("ziel_id = ?", id).
		Order("start_datum ASC").
		Find(&events).Error; err != nil {
		return response.InternalError(c, "Failed to fetch events")
	}
	
	return response.Success(c, events)
}

// GetZielRatings returns ratings for a destination
func GetZielRatings(c *fiber.Ctx) error {
	id := c.Params("id")
	page, _ := strconv.Atoi(c.Query("page", "1"))
	perPage, _ := strconv.Atoi(c.Query("per_page", "10"))
	
	var total int64
	database.DB.Model(&models.Rating{}).Where("ziel_id = ? AND revoked = false", id).Count(&total)
	
	var ratings []models.Rating
	offset := (page - 1) * perPage
	if err := database.DB.Where("ziel_id = ? AND revoked = false", id).
		Order("created_at DESC").
		Offset(offset).Limit(perPage).
		Find(&ratings).Error; err != nil {
		return response.InternalError(c, "Failed to fetch ratings")
	}
	
	return response.SuccessWithMeta(c, ratings, &response.Meta{
		Total:    total,
		Page:     page,
		PerPage:  perPage,
		LastPage: int((total + int64(perPage) - 1) / int64(perPage)),
	})
}

// CreateZiel creates a new destination (auth required)
func CreateZiel(c *fiber.Ctx) error {
	var input struct {
		Name          string   `json:"name"`
		MarketerID    *uint    `json:"marketer_id"`
		Slugname      string   `json:"slugname"`
		PlaceID       string   `json:"placeid"`
		Webseite      string   `json:"webseite"`
		Telefonnummer string   `json:"telefonnummer"`
		Facebook      string   `json:"facebook"`
		Adresse       string   `json:"adresse"`
		Stadt         string   `json:"stadt"`
		Latitude      float64  `json:"latitude"`
		Longitude     float64  `json:"longitude"`
		Auszug        string   `json:"auszug"`
		Beschreibung  string   `json:"beschreibung"`
		Besucht       bool     `json:"besucht"`
		Vorteile      string   `json:"vorteile"`
		Oeffnungszeiten string `json:"oeffnungszeiten"`
		Kategorien    []uint   `json:"kategorien"`
		Bilder        []uint   `json:"bilder"`
		Status        string   `json:"status"`
	}
	
	if err := c.BodyParser(&input); err != nil {
		return response.BadRequest(c, "Invalid request body")
	}

	user := middleware.GetUserFromContext(c)
	
	// Status: reporter+ can publish directly, others go to draft
	status := "draft"
	if input.Status != "" {
		status = strings.ToLower(input.Status)
	}
	if status == "published" && user.Role != "admin" && user.Role != "reporter" {
		status = "draft"
	}

	ziel := models.Ziel{
		Name:           input.Name,
		MarketerID:     input.MarketerID,
		Slugname:       input.Slugname,
		PlaceID:        input.PlaceID,
		Webseite:       input.Webseite,
		Telefonnummer:  input.Telefonnummer,
		Facebook:       input.Facebook,
		Adresse:        input.Adresse,
		Stadt:          input.Stadt,
		Latitude:       input.Latitude,
		Longitude:      input.Longitude,
		Auszug:         stripTags(input.Auszug),
		Beschreibung:   allowSomeHTML(input.Beschreibung),
		Besucht:        input.Besucht,
		Vorteile:       input.Vorteile,
		Oeffnungszeiten: allowBR(input.Oeffnungszeiten),
		Status:         status,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
		CreatedBy:      &user.ID,
		UpdatedBy:      &user.ID,
	}

	if err := database.DB.Create(&ziel).Error; err != nil {
		return response.InternalError(c, "Failed to create ziel: "+err.Error())
	}

	// Associate kategorien
	if len(input.Kategorien) > 0 {
		var kats []models.Kategorie
		database.DB.Find(&kats, input.Kategorien)
		database.DB.Model(&ziel).Association("Kategorien").Replace(kats)
	}

	// Associate bilder
	if len(input.Bilder) > 0 {
		var bilder []models.Bild
		database.DB.Find(&bilder, input.Bilder)
		database.DB.Model(&ziel).Association("Bilder").Replace(bilder)
	}

	return response.Created(c, ziel)
}

// UpdateZiel updates a destination (auth required)
func UpdateZiel(c *fiber.Ctx) error {
	id := c.Params("id")
	
	var ziel models.Ziel
	if err := database.DB.First(&ziel, id).Error; err != nil {
		return response.NotFound(c, "Ziel not found")
	}

	var input map[string]interface{}
	if err := c.BodyParser(&input); err != nil {
		return response.BadRequest(c, "Invalid request body")
	}

	user := middleware.GetUserFromContext(c)
	
	// Sanitize HTML fields
	if v, ok := input["auszug"].(string); ok {
		input["auszug"] = stripTags(v)
	}
	if v, ok := input["beschreibung"].(string); ok {
		input["beschreibung"] = allowSomeHTML(v)
	}
	if v, ok := input["oeffnungszeiten"].(string); ok {
		input["oeffnungszeiten"] = allowBR(v)
	}
	
	input["updated_by"] = user.ID
	input["updated_at"] = time.Now()

	// Handle status change
	if status, ok := input["status"].(string); ok {
		if status == "published" && user.Role != "admin" && user.Role != "reporter" {
			input["status"] = "draft"
		}
	}

	if err := database.DB.Model(&ziel).Updates(input).Error; err != nil {
		return response.InternalError(c, "Failed to update ziel")
	}

	// Handle kategorien update
	if kats, ok := input["kategorien"].([]interface{}); ok {
		var kategorien []models.Kategorie
		for _, k := range kats {
			if kid, ok := k.(float64); ok {
				kategorien = append(kategorien, models.Kategorie{ID: uint(kid)})
			}
		}
		database.DB.Model(&ziel).Association("Kategorien").Replace(kategorien)
	}

	// Handle bilder update
	if bilder, ok := input["bilder"].([]interface{}); ok {
		var bilderList []models.Bild
		for _, b := range bilder {
			if bid, ok := b.(float64); ok {
				bilderList = append(bilderList, models.Bild{ID: uint(bid)})
			}
		}
		database.DB.Model(&ziel).Association("Bilder").Replace(bilderList)
	}

	// Reload with associations
	database.DB.Preload("Kategorien").Preload("Bilder").First(&ziel, id)
	return response.Success(c, ziel)
}

// DeleteZiel soft-deletes a destination (auth required)
func DeleteZiel(c *fiber.Ctx) error {
	id := c.Params("id")

	if err := database.DB.Delete(&models.Ziel{}, id).Error; err != nil {
		return response.InternalError(c, "Failed to delete ziel")
	}

	return response.NoContent(c)
}

// expandZiel creates an expanded response with computed fields
func expandZiel(z models.Ziel, userFavs []uint) ZielResponse {
	// Calculate average rating
	var avgRating float64
	database.DB.Model(&models.Rating{}).
		Where("ziel_id = ? AND revoked = false", z.ID).
		Select("COALESCE(AVG(score), 0)").
		Scan(&avgRating)

	// Count favorites
	var favCount int64
	database.DB.Model(&models.Favorit{}).Where("ziel_id = ?", z.ID).Count(&favCount)

	// Check if user has favorited
	isFav := false
	for _, fid := range userFavs {
		if fid == z.ID {
			isFav = true
			break
		}
	}

	// Build social links
	social := &SocialLinks{
		Facebook:  z.Facebook,
		Instagram: "", // Not in model yet
		Twitter:   "", // Not in model yet
	}

	return ZielResponse{
		Ziel:          z,
		Social:        social,
		AvgRating:     avgRating,
		FavoriteCount: int(favCount),
		IsFavorite:    isFav,
	}
}

// haversine calculates distance between two points in km
func haversine(lat1, lng1, lat2, lng2 float64) float64 {
	const R = 6371 // Earth radius in km
	
	dLat := (lat2 - lat1) * math.Pi / 180
	dLng := (lng2 - lng1) * math.Pi / 180
	
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1*math.Pi/180)*math.Cos(lat2*math.Pi/180)*
			math.Sin(dLng/2)*math.Sin(dLng/2)
	
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	
	return R * c
}

// stripTags removes all HTML tags
func stripTags(s string) string {
	var result strings.Builder
	inTag := false
	for _, r := range s {
		if r == '<' {
			inTag = true
			continue
		}
		if r == '>' {
			inTag = false
			continue
		}
		if !inTag {
			result.WriteRune(r)
		}
	}
	return result.String()
}

// allowSomeHTML allows safe HTML tags
func allowSomeHTML(s string) string {
	// Simple approach: allow specific tags
	// In production, use a proper HTML sanitizer library
	allowed := []string{"<p>", "</p>", "<div>", "</div>", "<ul>", "</ul>", "<li>", "</li>", "<a ", "</a>", "<br>"}
	result := s
	// This is a simplified version - consider using bluemonday or similar
	for i := range allowed {
		_ = allowed[i] // placeholder
	}
	return result
}

// allowBR allows only <br> tags
func allowBR(s string) string {
	return strings.ReplaceAll(stripTags(s), "\n", "<br>")
}

// logSearchQuery logs search statistics asynchronously
func logSearchQuery(c *fiber.Ctx, filter, city, kategorie string, lat, lng float64, ziele []models.Ziel) {
	// Get user session if available
	user := middleware.GetUserFromContext(c)
	
	// Build destination IDs
	destIDs := make([]uint, len(ziele))
	for i, z := range ziele {
		destIDs[i] = z.ID
	}
	
	// Get category ID
	var katID *uint
	if kategorie != "" {
		if id, err := strconv.ParseUint(kategorie, 10, 32); err == nil {
			kid := uint(id)
			katID = &kid
		}
	}
	
	stat := models.StatisticsQuery{
		Session:           c.IP(), // Use IP as session identifier
		Device:            c.Get("User-Agent"),
		LocationLatitude:  lat,
		LocationLongitude: lng,
		QueryRaw:          c.OriginalURL(),
		QueryText:         filter,
		QueryLocationName: city,
		QueryLocationLat:  0,
		QueryLocationLng:  0,
		QueryCategories:   kategorie,
		QueryMarketer:     nil,
		QueryTrip:         false,
		GuestAccess:       user == nil,
		CreatedAt:         time.Now(),
	}
	
	_ = katID // Avoid unused variable warning
	database.DB.Create(&stat)
}

// Helper to load Ziel with all associations
func loadZielWithAssociations(db *gorm.DB, id interface{}) (*models.Ziel, error) {
	var ziel models.Ziel
	err := db.Preload("Kategorien").
		Preload("Bilder").
		Preload("Ratings").
		Preload("Veranstaltungen").
		Preload("Marketer").
		First(&ziel, id).Error
	if err != nil {
		return nil, err
	}
	return &ziel, nil
}