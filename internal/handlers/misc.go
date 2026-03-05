package handlers

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/HAJO4KidsDE/hajo4kids-backend/internal/database"
	"github.com/HAJO4KidsDE/hajo4kids-backend/internal/middleware"
	"github.com/HAJO4KidsDE/hajo4kids-backend/internal/models"
	"github.com/HAJO4KidsDE/hajo4kids-backend/pkg/response"
	"github.com/gofiber/fiber/v2"
)

// Helper: Check if user can edit trip (owner or admin)
func canEditTrip(user *models.User, trip models.Trip) bool {
	if user == nil {
		return false
	}
	if user.ID == trip.UserID {
		return true
	}
	if user.Role == "admin" {
		return true
	}
	return false
}

// Kategorien
func GetKategorien(c *fiber.Ctx) error {
	var kategorien []models.Kategorie
	if err := database.DB.Preload("BildData").Order("sort_order, name").Find(&kategorien).Error; err != nil {
		return response.InternalError(c, "Failed to fetch kategorien")
	}
	return response.Success(c, kategorien)
}

func GetKategorie(c *fiber.Ctx) error {
	id := c.Params("id")
	var kategorie models.Kategorie
	if err := database.DB.Preload("BildData").First(&kategorie, id).Error; err != nil {
		return response.NotFound(c, "Kategorie not found")
	}
	return response.Success(c, kategorie)
}

func CreateKategorie(c *fiber.Ctx) error {
	var kategorie models.Kategorie
	if err := c.BodyParser(&kategorie); err != nil {
		return response.BadRequest(c, "Invalid request body")
	}
	if err := database.DB.Create(&kategorie).Error; err != nil {
		return response.InternalError(c, "Failed to create kategorie")
	}
	return response.Created(c, kategorie)
}

func UpdateKategorie(c *fiber.Ctx) error {
	id := c.Params("id")
	var kategorie models.Kategorie
	if err := database.DB.First(&kategorie, id).Error; err != nil {
		return response.NotFound(c, "Kategorie not found")
	}
	var updates map[string]interface{}
	c.BodyParser(&updates)
	database.DB.Model(&kategorie).Updates(updates)
	return response.Success(c, kategorie)
}

func DeleteKategorie(c *fiber.Ctx) error {
	database.DB.Delete(&models.Kategorie{}, c.Params("id"))
	return response.NoContent(c)
}

// Vermarkter
func GetVermarkter(c *fiber.Ctx) error {
	var vermarkter []models.Vermarkter
	database.DB.Find(&vermarkter)
	return response.Success(c, vermarkter)
}

func GetVermarkterByID(c *fiber.Ctx) error {
	var v models.Vermarkter
	if err := database.DB.First(&v, c.Params("id")).Error; err != nil {
		return response.NotFound(c, "Vermarkter not found")
	}
	return response.Success(c, v)
}

// Favoriten
func GetFavoriten(c *fiber.Ctx) error {
	user := middleware.GetUserFromContext(c)
	var favoriten []models.Favorit
	database.DB.Preload("Ziel.Bilder").Preload("Ziel.Kategorien").Where("user_id = ?", user.ID).Find(&favoriten)
	return response.Success(c, favoriten)
}

func AddFavorit(c *fiber.Ctx) error {
	user := middleware.GetUserFromContext(c)
	zielID := c.Params("zielid")
	
	fav := models.Favorit{UserID: user.ID, ZielID: parseUint(zielID), CreatedAt: time.Now()}
	if err := database.DB.Create(&fav).Error; err != nil {
		return response.Error(c, fiber.StatusConflict, "Already favorited")
	}
	return response.Created(c, fav)
}

func RemoveFavorit(c *fiber.Ctx) error {
	user := middleware.GetUserFromContext(c)
	database.DB.Where("user_id = ? AND ziel_id = ?", user.ID, c.Params("zielid")).Delete(&models.Favorit{})
	return response.NoContent(c)
}

// Ratings
func CreateRating(c *fiber.Ctx) error {
	user := middleware.GetUserFromContext(c)
	var rating models.Rating
	c.BodyParser(&rating)
	rating.UserID = user.ID
	rating.CreatedAt = time.Now()
	if err := database.DB.Create(&rating).Error; err != nil {
		return response.InternalError(c, "Failed to create rating")
	}
	return response.Created(c, rating)
}

func UpdateRating(c *fiber.Ctx) error {
	user := middleware.GetUserFromContext(c)
	id := c.Params("id")
	var rating models.Rating
	if err := database.DB.First(&rating, id).Error; err != nil {
		return response.NotFound(c, "Rating not found")
	}
	if rating.UserID != user.ID {
		return response.Forbidden(c, "Not your rating")
	}
	var updates map[string]interface{}
	c.BodyParser(&updates)
	database.DB.Model(&rating).Updates(updates)
	return response.Success(c, rating)
}

func DeleteRating(c *fiber.Ctx) error {
	user := middleware.GetUserFromContext(c)
	result := database.DB.Where("id = ? AND user_id = ?", c.Params("id"), user.ID).Delete(&models.Rating{})
	if result.RowsAffected == 0 {
		return response.NotFound(c, "Rating not found")
	}
	return response.NoContent(c)
}

// Kontakt
func SubmitKontakt(c *fiber.Ctx) error {
	var msg models.KontaktMessage
	if err := c.BodyParser(&msg); err != nil {
		return response.BadRequest(c, "Invalid request body")
	}
	msg.CreatedAt = time.Now()
	msg.Read = false
	database.DB.Create(&msg)
	return response.Created(c, fiber.Map{"message": "Message sent successfully"})
}

// Admin
func GetUsers(c *fiber.Ctx) error {
	var users []models.User
	database.DB.Select("id, username, email, role, active, created_at").Find(&users)
	return response.Success(c, users)
}

func UpdateUser(c *fiber.Ctx) error {
	var updates map[string]interface{}
	c.BodyParser(&updates)
	database.DB.Model(&models.User{}).Where("id = ?", c.Params("id")).Updates(updates)
	return response.Success(c, fiber.Map{"updated": true})
}

func DeleteUser(c *fiber.Ctx) error {
	database.DB.Delete(&models.User{}, c.Params("id"))
	return response.NoContent(c)
}

func GetKontaktMessages(c *fiber.Ctx) error {
	var messages []models.KontaktMessage
	database.DB.Order("created_at desc").Find(&messages)
	return response.Success(c, messages)
}

func MarkKontaktRead(c *fiber.Ctx) error {
	database.DB.Model(&models.KontaktMessage{}).Where("id = ?", c.Params("id")).Update("read", true)
	return response.Success(c, fiber.Map{"read": true})
}

func parseUint(s string) uint {
	var i uint
	for _, c := range s {
		if c >= '0' && c <= '9' {
			i = i*10 + uint(c-'0')
		}
	}
	return i
}

// ==================== VERANSTALTUNGEN (Events) ====================

func GetVeranstaltungen(c *fiber.Ctx) error {
	query := database.DB.Preload("Ziel").Model(&models.Veranstaltung{})
	
	// Filter by Ziel
	if zielID := c.Query("ziel_id"); zielID != "" {
		query = query.Where("ziel_id = ?", zielID)
	}
	
	// Filter by date range
	if from := c.Query("from"); from != "" {
		if t, err := time.Parse("2006-01-02", from); err == nil {
			query = query.Where("start_datum >= ?", t)
		}
	}
	if to := c.Query("to"); to != "" {
		if t, err := time.Parse("2006-01-02", to); err == nil {
			query = query.Where("start_datum <= ?", t)
		}
	}
	
	// Upcoming only
	if c.Query("upcoming") == "true" {
		query = query.Where("start_datum >= ?", time.Now())
	}
	
	var veranstaltungen []models.Veranstaltung
	query.Order("start_datum asc").Find(&veranstaltungen)
	return response.Success(c, veranstaltungen)
}

func GetVeranstaltung(c *fiber.Ctx) error {
	var v models.Veranstaltung
	if err := database.DB.Preload("Ziel").First(&v, c.Params("id")).Error; err != nil {
		return response.NotFound(c, "Veranstaltung not found")
	}
	return response.Success(c, v)
}

func CreateVeranstaltung(c *fiber.Ctx) error {
	var v models.Veranstaltung
	if err := c.BodyParser(&v); err != nil {
		return response.BadRequest(c, "Invalid request body")
	}
	v.CreatedAt = time.Now()
	v.UpdatedAt = time.Now()
	if err := database.DB.Create(&v).Error; err != nil {
		return response.InternalError(c, "Failed to create veranstaltung")
	}
	return response.Created(c, v)
}

func UpdateVeranstaltung(c *fiber.Ctx) error {
	id := c.Params("id")
	var v models.Veranstaltung
	if err := database.DB.First(&v, id).Error; err != nil {
		return response.NotFound(c, "Veranstaltung not found")
	}
	var updates map[string]interface{}
	c.BodyParser(&updates)
	updates["updated_at"] = time.Now()
	database.DB.Model(&v).Updates(updates)
	return response.Success(c, v)
}

func DeleteVeranstaltung(c *fiber.Ctx) error {
	database.DB.Delete(&models.Veranstaltung{}, c.Params("id"))
	return response.NoContent(c)
}

// ==================== VERMARKTER CRUD ====================

func CreateVermarkter(c *fiber.Ctx) error {
	var v models.Vermarkter
	if err := c.BodyParser(&v); err != nil {
		return response.BadRequest(c, "Invalid request body")
	}
	v.CreatedAt = time.Now()
	v.UpdatedAt = time.Now()
	if err := database.DB.Create(&v).Error; err != nil {
		return response.InternalError(c, "Failed to create vermarkter")
	}
	return response.Created(c, v)
}

func UpdateVermarkter(c *fiber.Ctx) error {
	id := c.Params("id")
	var v models.Vermarkter
	if err := database.DB.First(&v, id).Error; err != nil {
		return response.NotFound(c, "Vermarkter not found")
	}
	var updates map[string]interface{}
	c.BodyParser(&updates)
	updates["updated_at"] = time.Now()
	database.DB.Model(&v).Updates(updates)
	return response.Success(c, v)
}

func DeleteVermarkter(c *fiber.Ctx) error {
	database.DB.Delete(&models.Vermarkter{}, c.Params("id"))
	return response.NoContent(c)
}

// ==================== TRIPS (Stories) ====================

func GetTrips(c *fiber.Ctx) error {
	query := database.DB.Model(&models.Trip{})
	
	// Public only for non-authenticated users
	user := middleware.GetUserFromContext(c)
	if user == nil {
		query = query.Where("is_public = ?", true)
	} else if c.Query("my") == "true" {
		query = query.Where("user_id = ?", user.ID)
	}
	
	// Preload Ziele
	query = query.Preload("Ziele")
	
	var trips []models.Trip
	query.Order("created_at desc").Find(&trips)
	return response.Success(c, trips)
}

func GetTrip(c *fiber.Ctx) error {
	var trip models.Trip
	if err := database.DB.Preload("Ziele").First(&trip, c.Params("id")).Error; err != nil {
		return response.NotFound(c, "Trip not found")
	}
	
	// Check access
	user := middleware.GetUserFromContext(c)
	if !trip.IsPublic && (user == nil || user.ID != trip.UserID) {
		return response.Forbidden(c, "This trip is private")
	}
	
	return response.Success(c, trip)
}

func CreateTrip(c *fiber.Ctx) error {
	user := middleware.GetUserFromContext(c)
	var trip models.Trip
	if err := c.BodyParser(&trip); err != nil {
		return response.BadRequest(c, "Invalid request body")
	}
	trip.UserID = user.ID
	trip.CreatedAt = time.Now()
	trip.UpdatedAt = time.Now()
	
	// Handle Ziel associations
	var zielIDs []uint
	if ids := c.Body(); len(ids) > 0 {
		var body struct {
			ZielIDs []uint `json:"ziel_ids"`
		}
		if err := json.Unmarshal(c.Body(), &body); err == nil {
			zielIDs = body.ZielIDs
		}
	}
	
	if err := database.DB.Create(&trip).Error; err != nil {
		return response.InternalError(c, "Failed to create trip")
	}
	
	// Associate Ziele
	if len(zielIDs) > 0 {
		var ziele []models.Ziel
		database.DB.Find(&ziele, zielIDs)
		database.DB.Model(&trip).Association("Ziele").Replace(ziele)
	}
	
	return response.Created(c, trip)
}

func UpdateTrip(c *fiber.Ctx) error {
	user := middleware.GetUserFromContext(c)
	id := c.Params("id")
	var trip models.Trip
	if err := database.DB.First(&trip, id).Error; err != nil {
		return response.NotFound(c, "Trip not found")
	}
	if !canEditTrip(user, trip) {
		return response.Forbidden(c, "Not your trip")
	}
	
	var updates map[string]interface{}
	c.BodyParser(&updates)
	updates["updated_at"] = time.Now()
	database.DB.Model(&trip).Updates(updates)
	
	// Handle Ziel associations
	var body struct {
		ZielIDs []uint `json:"ziel_ids"`
	}
	if err := json.Unmarshal(c.Body(), &body); err == nil && len(body.ZielIDs) > 0 {
		var ziele []models.Ziel
		database.DB.Find(&ziele, body.ZielIDs)
		database.DB.Model(&trip).Association("Ziele").Replace(ziele)
	}
	
	return response.Success(c, trip)
}

func DeleteTrip(c *fiber.Ctx) error {
	user := middleware.GetUserFromContext(c)
	id := c.Params("id")
	var trip models.Trip
	if err := database.DB.First(&trip, id).Error; err != nil {
		return response.NotFound(c, "Trip not found")
	}
	if !canEditTrip(user, trip) {
		return response.Forbidden(c, "Not your trip")
	}
	database.DB.Delete(&trip)
	return response.NoContent(c)
}

func AddTripZiel(c *fiber.Ctx) error {
	user := middleware.GetUserFromContext(c)
	var trip models.Trip
	if err := database.DB.First(&trip, c.Params("id")).Error; err != nil {
		return response.NotFound(c, "Trip not found")
	}
	if !canEditTrip(user, trip) {
		return response.Forbidden(c, "Not your trip")
	}
	
	var ziel models.Ziel
	if err := database.DB.First(&ziel, c.Params("zielid")).Error; err != nil {
		return response.NotFound(c, "Ziel not found")
	}
	
	database.DB.Model(&trip).Association("Ziele").Append(&ziel)
	return response.Success(c, fiber.Map{"added": true})
}

func RemoveTripZiel(c *fiber.Ctx) error {
	user := middleware.GetUserFromContext(c)
	var trip models.Trip
	if err := database.DB.First(&trip, c.Params("id")).Error; err != nil {
		return response.NotFound(c, "Trip not found")
	}
	if !canEditTrip(user, trip) {
		return response.Forbidden(c, "Not your trip")
	}
	
	var ziel models.Ziel
	database.DB.First(&ziel, c.Params("zielid"))
	database.DB.Model(&trip).Association("Ziele").Delete(&ziel)
	return response.NoContent(c)
}

// ==================== SEARCH ====================

type SearchRequest struct {
	Query      string `json:"query"`
	Stadt      string `json:"stadt"`
	Kategorie  string `json:"kategorie"`
	MarketerID *uint  `json:"marketer_id"`
	Latitude   float64 `json:"latitude"`
	Longitude  float64 `json:"longitude"`
	Radius     float64 `json:"radius"` // km
	Limit      int    `json:"limit"`
	Offset     int    `json:"offset"`
}

func SearchZiele(c *fiber.Ctx) error {
	var req SearchRequest
	
	// Parse query params
	req.Query = c.Query("q")
	req.Stadt = c.Query("stadt")
	req.Kategorie = c.Query("kategorie")
	req.Limit = 50
	req.Offset = 0
	
	if limit := c.QueryInt("limit"); limit > 0 {
		req.Limit = limit
	}
	if offset := c.QueryInt("offset"); offset > 0 {
		req.Offset = offset
	}
	if marketerID := c.QueryInt("marketer_id"); marketerID > 0 {
		uid := uint(marketerID)
		req.MarketerID = &uid
	}
	if lat := c.QueryFloat("lat"); lat != 0 {
		req.Latitude = lat
	}
	if lng := c.QueryFloat("lng"); lng != 0 {
		req.Longitude = lng
	}
	if radius := c.QueryFloat("radius"); radius > 0 {
		req.Radius = radius
	}
	
	query := database.DB.Model(&models.Ziel{}).Where("status = ?", "published")
	
	// Search by text
	if req.Query != "" {
		search := "%" + strings.ToLower(req.Query) + "%"
		query = query.Where(
			"LOWER(name) LIKE ? OR LOWER(beschreibung) LIKE ? OR LOWER(stadt) LIKE ? OR LOWER(adresse) LIKE ?",
			search, search, search, search,
		)
	}
	
	// Filter by Stadt
	if req.Stadt != "" {
		query = query.Where("LOWER(stadt) = ?", strings.ToLower(req.Stadt))
	}
	
	// Filter by Kategorie
	if req.Kategorie != "" {
		query = query.Joins("JOIN ziel_kategorien ON ziel_kategorien.ziel_id = ziels.id").
			Joins("JOIN kategorien ON kategorien.id = ziel_kategorien.kategorie_id").
			Where("LOWER(kategorien.name) = ?", strings.ToLower(req.Kategorie))
	}
	
	// Filter by Marketer
	if req.MarketerID != nil {
		query = query.Where("marketer_id = ?", *req.MarketerID)
	}
	
	// Get total count
	var total int64
	query.Count(&total)
	
	// Get results with relations
	var ziele []models.Ziel
	query = query.Preload("Kategorien").Preload("Bilder")
	
	// Check favorites for logged-in user
	user := middleware.GetUserFromContext(c)
	if user != nil {
		query = query.Preload("Favoriten", "user_id = ?", user.ID)
	}
	
	query.Offset(req.Offset).Limit(req.Limit).Find(&ziele)
	
	// Add computed fields
	for i := range ziele {
		// Calculate distance if lat/lng provided
		if req.Latitude != 0 && req.Longitude != 0 {
			// Simple Euclidean distance (for small areas)
			// For production, use Haversine formula
			dlat := ziele[i].Latitude - req.Latitude
			dlng := ziele[i].Longitude - req.Longitude
			dist := (dlat*dlat + dlng*dlng) * 111 * 111 // rough km^2
			ziele[i].Distanz = &dist
		}
		
		// Check if favorited
		if user != nil && len(ziele[i].Favoriten) > 0 {
			ziele[i].Favorit = true
		}
		
		// Get rating average
		var avgRating float64
		database.DB.Model(&models.Rating{}).Where("ziel_id = ? AND revoked = ?", ziele[i].ID, false).Select("AVG(score)").Scan(&avgRating)
		ziele[i].Rating = avgRating
		
		// Get favorite count
		var favCount int64
		database.DB.Model(&models.Favorit{}).Where("ziel_id = ?", ziele[i].ID).Count(&favCount)
		ziele[i].Favorits = int(favCount)
	}
	
	// Sort by distance if location search
	if req.Latitude != 0 && req.Longitude != 0 && req.Radius > 0 {
		// Filter by radius
		var filtered []models.Ziel
		for _, z := range ziele {
			if z.Distanz != nil && *z.Distanz <= req.Radius {
				filtered = append(filtered, z)
			}
		}
		ziele = filtered
	}
	
	// Log search statistics (async)
	go func() {
		stat := models.StatisticsQuery{
			QueryText:  req.Query,
			Session:    c.Get("X-Session-ID"),
			Device:     c.Get("User-Agent"),
			GuestAccess: user == nil,
			CreatedAt:  time.Now(),
		}
		if req.Latitude != 0 {
			stat.LocationLatitude = req.Latitude
			stat.LocationLongitude = req.Longitude
		}
		if req.MarketerID != nil {
			stat.QueryMarketer = req.MarketerID
		}
		database.DB.Create(&stat)
	}()
	
	return response.Paginated(c, ziele, int(total), req.Limit, req.Offset)
}

// ==================== SITEMAP ====================

type SitemapURL struct {
	Loc        string `xml:"loc"`
	LastMod    string `xml:"lastmod,omitempty"`
	ChangeFreq string `xml:"changefreq,omitempty"`
	Priority   string `xml:"priority,omitempty"`
}

func GetSitemap(c *fiber.Ctx) error {
	baseURL := c.Protocol() + "://" + c.Hostname()
	
	var urls []SitemapURL
	
	// Static pages
	urls = append(urls,
		SitemapURL{Loc: baseURL + "/", ChangeFreq: "weekly", Priority: "1.0"},
		SitemapURL{Loc: baseURL + "/ziele", ChangeFreq: "daily", Priority: "0.9"},
		SitemapURL{Loc: baseURL + "/kategorien", ChangeFreq: "weekly", Priority: "0.7"},
		SitemapURL{Loc: baseURL + "/veranstaltungen", ChangeFreq: "daily", Priority: "0.8"},
		SitemapURL{Loc: baseURL + "/trips", ChangeFreq: "weekly", Priority: "0.6"},
	)
	
	// Published Ziele
	var ziele []models.Ziel
	database.DB.Where("status = ?", "published").Select("id, slugname, updated_at").Find(&ziele)
	for _, z := range ziele {
		urls = append(urls, SitemapURL{
			Loc:        baseURL + "/ziele/" + z.Slugname,
			LastMod:    z.UpdatedAt.Format("2006-01-02"),
			ChangeFreq: "weekly",
			Priority:   "0.8",
		})
	}
	
	// Kategorien
	var kategorien []models.Kategorie
	database.DB.Select("id, name, updated_at").Find(&kategorien)
	for _, k := range kategorien {
		urls = append(urls, SitemapURL{
			Loc:        baseURL + "/kategorie/" + strings.ToLower(strings.ReplaceAll(k.Name, " ", "-")),
			LastMod:    k.UpdatedAt.Format("2006-01-02"),
			ChangeFreq: "weekly",
			Priority:   "0.7",
		})
	}
	
	// Public Trips
	var trips []models.Trip
	database.DB.Where("is_public = ?", true).Select("id, updated_at").Find(&trips)
	for _, t := range trips {
		urls = append(urls, SitemapURL{
			Loc:        baseURL + "/trips/" + string(rune(t.ID)),
			LastMod:    t.UpdatedAt.Format("2006-01-02"),
			ChangeFreq: "monthly",
			Priority:   "0.5",
		})
	}
	
	// Build XML
	var sb strings.Builder
	sb.WriteString(`<?xml version="1.0" encoding="UTF-8"?>`)
	sb.WriteString(`<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">`)
	for _, u := range urls {
		sb.WriteString("<url>")
		sb.WriteString("<loc>" + u.Loc + "</loc>")
		if u.LastMod != "" {
			sb.WriteString("<lastmod>" + u.LastMod + "</lastmod>")
		}
		if u.ChangeFreq != "" {
			sb.WriteString("<changefreq>" + u.ChangeFreq + "</changefreq>")
		}
		if u.Priority != "" {
			sb.WriteString("<priority>" + u.Priority + "</priority>")
		}
		sb.WriteString("</url>")
	}
	sb.WriteString(`</urlset>`)
	
	c.Set("Content-Type", "application/xml")
	return c.SendString(sb.String())
}

// ==================== STATISTICS ====================

func GetStatisticsOverview(c *fiber.Ctx) error {
	var stats struct {
		ZieleTotal       int64 `json:"ziele_total"`
		ZielePublished   int64 `json:"ziele_published"`
		ZieleDraft       int64 `json:"ziele_draft"`
		UsersTotal       int64 `json:"users_total"`
		KategorienCount  int64 `json:"kategorien_count"`
		RatingsCount     int64 `json:"ratings_count"`
		FavoritenCount   int64 `json:"favoriten_count"`
		TripsCount       int64 `json:"trips_count"`
		EventsUpcoming   int64 `json:"events_upcoming"`
	}
	
	database.DB.Model(&models.Ziel{}).Count(&stats.ZieleTotal)
	database.DB.Model(&models.Ziel{}).Where("status = ?", "published").Count(&stats.ZielePublished)
	database.DB.Model(&models.Ziel{}).Where("status = ?", "draft").Count(&stats.ZieleDraft)
	database.DB.Model(&models.User{}).Count(&stats.UsersTotal)
	database.DB.Model(&models.Kategorie{}).Count(&stats.KategorienCount)
	database.DB.Model(&models.Rating{}).Where("revoked = ?", false).Count(&stats.RatingsCount)
	database.DB.Model(&models.Favorit{}).Count(&stats.FavoritenCount)
	database.DB.Model(&models.Trip{}).Count(&stats.TripsCount)
	database.DB.Model(&models.Veranstaltung{}).Where("start_datum >= ?", time.Now()).Count(&stats.EventsUpcoming)
	
	return response.Success(c, stats)
}

func GetSearchStatistics(c *fiber.Ctx) error {
	var result struct {
		TopSearches []struct {
			Query string `json:"query"`
			Count int    `json:"count"`
		} `json:"top_searches"`
		SearchesPerDay []struct {
			Date  string `json:"date"`
			Count int    `json:"count"`
		} `json:"searches_per_day"`
	}
	
	// Top searches (last 30 days)
	database.DB.Model(&models.StatisticsQuery{}).
		Select("query_text as query, COUNT(*) as count").
		Where("query_text IS NOT NULL AND query_text != '' AND created_at > ?", time.Now().AddDate(0, 0, -30)).
		Group("query_text").
		Order("count desc").
		Limit(20).
		Scan(&result.TopSearches)
	
	// Searches per day (last 7 days)
	database.DB.Model(&models.StatisticsQuery{}).
		Select("DATE(created_at) as date, COUNT(*) as count").
		Where("created_at > ?", time.Now().AddDate(0, 0, -7)).
		Group("DATE(created_at)").
		Order("date desc").
		Scan(&result.SearchesPerDay)
	
	return response.Success(c, result)
}