package handlers

import (
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/HAJO4KidsDE/hajo4kids-backend/internal/database"
	"github.com/HAJO4KidsDE/hajo4kids-backend/internal/middleware"
	"github.com/HAJO4KidsDE/hajo4kids-backend/internal/models"
	"github.com/HAJO4KidsDE/hajo4kids-backend/pkg/response"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// Allowed image extensions
var allowedExtensions = map[string]bool{
	".jpg":  true,
	".jpeg": true,
	".png":  true,
	".webp": true,
}

// Upload type constants
const (
	UploadTypeDestination = "DESTINATION"
	UploadTypeCategory    = "CATEGORY"
	UploadTypeEvent       = "EVENT"
	UploadTypeProfile     = "PROFILE"
	UploadTypeMarketer    = "MARKETER"
)

// BildUploadRequest represents the upload request
type BildUploadRequest struct {
	File         multipart.File
	Header       *multipart.FileHeader
	Type         string `form:"type"`       // DESTINATION, CATEGORY, EVENT, PROFILE, MARKETER
	TargetID     string `form:"id"`         // Target entity ID
	Alt          string `form:"alt"`        // Alt text
	IsPrimary    bool   `form:"is_primary"` // Set as primary image
}

// GetBilder returns list of images
func GetBilder(c *fiber.Ctx) error {
	page, _ := strconv.Atoi(c.Query("page", "1"))
	perPage, _ := strconv.Atoi(c.Query("per_page", "50"))
	imageType := c.Query("type")
	targetID := c.Query("target_id")
	withAuthor := c.Query("with_author")

	query := database.DB.Model(&models.Bild{})

	if imageType != "" {
		// Filter by type via association would require join
		// For now, simple filter
	}
	if targetID != "" {
		// Filter by target via association
	}

	// Filter for images with author (for credits page)
	if withAuthor == "true" {
		query = query.Where("autor IS NOT NULL AND autor != ''")
	}

	var total int64
	query.Count(&total)

	var bilder []models.Bild
	offset := (page - 1) * perPage
	if err := query.Order("created_at DESC").Offset(offset).Limit(perPage).Find(&bilder).Error; err != nil {
		return response.InternalError(c, "Failed to fetch images")
	}

	return response.SuccessWithMeta(c, bilder, &response.Meta{
		Total:    total,
		Page:     page,
		PerPage:  perPage,
		LastPage: int((total + int64(perPage) - 1) / int64(perPage)),
	})
}

// GetBild returns a single image by ID
func GetBild(c *fiber.Ctx) error {
	id := c.Params("id")

	var bild models.Bild
	if err := database.DB.First(&bild, id).Error; err != nil {
		return response.NotFound(c, "Image not found")
	}

	return response.Success(c, bild)
}

// ServeBild serves the actual image file
func ServeBild(c *fiber.Ctx) error {
	id := c.Params("id")

	var bild models.Bild
	if err := database.DB.First(&bild, id).Error; err != nil {
		return response.NotFound(c, "Image not found")
	}

	// Check if file exists
	if bild.Path == "" {
		return response.NotFound(c, "Image file not found")
	}

	// Set content type and send file
	c.Set("Content-Type", bild.MimeType)
	c.Set("Cache-Control", "public, max-age=31536000") // 1 year cache
	return c.SendFile(bild.Path)
}

// ServeBildThumbnail serves a thumbnail of the image
func ServeBildThumbnail(c *fiber.Ctx) error {
	id := c.Params("id")
	width, _ := strconv.Atoi(c.Params("width", "200"))
	height, _ := strconv.Atoi(c.Params("height", "200"))

	var bild models.Bild
	if err := database.DB.First(&bild, id).Error; err != nil {
		return response.NotFound(c, "Image not found")
	}

	// Check if thumbnail exists
	if bild.Thumbnail != "" {
		thumbPath := bild.Thumbnail
		if _, err := os.Stat(thumbPath); err == nil {
			c.Set("Content-Type", "image/webp")
			c.Set("Cache-Control", "public, max-age=31536000")
			return c.SendFile(thumbPath)
		}
	}

	// Generate thumbnail on-the-fly (simplified - in production use imaging library)
	// For now, just serve the original
	_ = width
	_ = height
	return ServeBild(c)
}

// UploadBild handles image upload
func UploadBild(c *fiber.Ctx) error {
	user := middleware.GetUserFromContext(c)
	if user == nil {
		return response.Unauthorized(c, "Authentication required")
	}

	// Parse multipart form
	file, err := c.FormFile("file")
	if err != nil {
		return response.BadRequest(c, "No file provided")
	}

	uploadType := c.FormValue("type", "DESTINATION")
	targetID := c.FormValue("id")
	alt := c.FormValue("alt")
	autor := c.FormValue("autor")
	beschreibung := c.FormValue("beschreibung")
	isPrimary := c.FormValue("is_primary") == "true"

	// Validate upload type
	if !isValidUploadType(uploadType) {
		return response.BadRequest(c, "Invalid upload type. Allowed: DESTINATION, CATEGORY, EVENT, PROFILE, MARKETER")
	}

	// Check permissions
	if !canUploadType(user.Role, uploadType) {
		return response.Forbidden(c, "Insufficient permissions for this upload type")
	}

	// Validate file extension
	ext := strings.ToLower(filepath.Ext(file.Filename))
	if !allowedExtensions[ext] {
		return response.BadRequest(c, "Invalid file type. Allowed: JPG, JPEG, PNG, WEBP")
	}

	// Validate file size (max 10MB)
	if file.Size > 10*1024*1024 {
		return response.BadRequest(c, "File too large. Maximum size: 10MB")
	}

	// Create upload directory
	uploadDir := getUploadDir(uploadType, targetID)
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		return response.InternalError(c, "Failed to create upload directory")
	}

	// Generate unique filename
	filename := generateFilename(ext)
	fullPath := filepath.Join(uploadDir, filename)
	relativePath := filepath.Join(getRelativeUploadDir(uploadType, targetID), filename)

	// Open uploaded file
	src, err := file.Open()
	if err != nil {
		return response.InternalError(c, "Failed to open uploaded file")
	}
	defer src.Close()

	// Create destination file
	dst, err := os.Create(fullPath)
	if err != nil {
		return response.InternalError(c, "Failed to create file")
	}
	defer dst.Close()

	// Copy file
	if _, err := io.Copy(dst, src); err != nil {
		return response.InternalError(c, "Failed to save file")
	}

	// Determine MIME type
	mimeType := getMimeType(ext)

	// Create database record
	bild := models.Bild{
		Filename:     filename,
		OriginalName: file.Filename,
		MimeType:     mimeType,
		Size:         file.Size,
		Path:         relativePath,
		Alt:          alt,
		Autor:        autor,
		Beschreibung: beschreibung,
		IsPrimary:    isPrimary,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if err := database.DB.Create(&bild).Error; err != nil {
		// Clean up file on database error
		os.Remove(fullPath)
		return response.InternalError(c, "Failed to create image record")
	}

	// Associate with target entity if specified
	if targetID != "" {
		if err := associateBildWithTarget(bild.ID, uploadType, targetID); err != nil {
			// Log but don't fail - image is uploaded
			fmt.Printf("Warning: failed to associate image: %v\n", err)
		}
	}

	// Set as primary if requested (for destinations)
	if isPrimary && uploadType == UploadTypeDestination && targetID != "" {
		setPrimaryImage(bild.ID, targetID)
	}

	return response.Created(c, bild)
}

// UpdateBild updates image metadata
func UpdateBild(c *fiber.Ctx) error {
	id := c.Params("id")

	var bild models.Bild
	if err := database.DB.First(&bild, id).Error; err != nil {
		return response.NotFound(c, "Image not found")
	}

	var input map[string]interface{}
	if err := c.BodyParser(&input); err != nil {
		return response.BadRequest(c, "Invalid request body")
	}

	// Only allow updating certain fields
	allowedFields := map[string]bool{
		"alt":          true,
		"autor":        true,
		"beschreibung": true,
		"is_primary":   true,
	}

	updates := make(map[string]interface{})
	for key, value := range input {
		if allowedFields[key] {
			updates[key] = value
		}
	}

	updates["updated_at"] = time.Now()

	if err := database.DB.Model(&bild).Updates(updates).Error; err != nil {
		return response.InternalError(c, "Failed to update image")
	}

	return response.Success(c, bild)
}

// DeleteBild deletes an image
func DeleteBild(c *fiber.Ctx) error {
	id := c.Params("id")

	var bild models.Bild
	if err := database.DB.First(&bild, id).Error; err != nil {
		return response.NotFound(c, "Image not found")
	}

	// Delete file from filesystem
	if bild.Path != "" {
		if err := os.Remove(bild.Path); err != nil && !os.IsNotExist(err) {
			fmt.Printf("Warning: failed to delete file %s: %v\n", bild.Path, err)
		}
	}

	// Delete thumbnail if exists
	if bild.Thumbnail != "" {
		os.Remove(bild.Thumbnail)
	}

	// Delete database record
	if err := database.DB.Delete(&bild).Error; err != nil {
		return response.InternalError(c, "Failed to delete image")
	}

	return response.NoContent(c)
}

// Helper functions

func isValidUploadType(t string) bool {
	switch t {
	case UploadTypeDestination, UploadTypeCategory, UploadTypeEvent, UploadTypeProfile, UploadTypeMarketer:
		return true
	default:
		return false
	}
}

func canUploadType(role string, uploadType string) bool {
	switch uploadType {
	case UploadTypeProfile:
		return true // Users can upload their own profile pictures
	case UploadTypeDestination, UploadTypeCategory, UploadTypeEvent, UploadTypeMarketer:
		return role == "admin" || role == "reporter"
	default:
		return false
	}
}

func getUploadDir(uploadType string, targetID string) string {
	baseDir := "./uploads" // Should be from config
	return filepath.Join(baseDir, getRelativeUploadDir(uploadType, targetID))
}

func getRelativeUploadDir(uploadType string, targetID string) string {
	switch uploadType {
	case UploadTypeDestination:
		return filepath.Join("destination", targetID)
	case UploadTypeCategory:
		return filepath.Join("category", targetID)
	case UploadTypeEvent:
		return filepath.Join("event", targetID)
	case UploadTypeProfile:
		return filepath.Join("profile", targetID)
	case UploadTypeMarketer:
		return filepath.Join("marketer", targetID)
	default:
		return "misc"
	}
}

func generateFilename(ext string) string {
	return fmt.Sprintf("%s%s", uuid.New().String(), ext)
}

func getMimeType(ext string) string {
	switch ext {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".webp":
		return "image/webp"
	default:
		return "application/octet-stream"
	}
}

func associateBildWithTarget(bildID uint, uploadType string, targetID string) error {
	id, err := strconv.ParseUint(targetID, 10, 32)
	if err != nil {
		return err
	}

	switch uploadType {
	case UploadTypeDestination:
		// Add to ziel_bilder junction table
		return database.DB.Exec(
			"INSERT INTO ziel_bilder (ziel_id, bild_id) VALUES (?, ?)",
			uint(id), bildID,
		).Error
	case UploadTypeCategory:
		// Categories might have images too
		return errors.New("category image association not implemented")
	case UploadTypeEvent:
		// Events might have images
		return errors.New("event image association not implemented")
	case UploadTypeMarketer:
		// Marketer logos
		return database.DB.Model(&models.Vermarkter{}).Where("id = ?", id).Update("logo", bildID).Error
	default:
		return nil
	}
}

func setPrimaryImage(bildID uint, zielID string) error {
	// Reset all primary flags for this destination
	database.DB.Exec(
		"UPDATE bilder b JOIN ziel_bilder zb ON b.id = zb.bild_id SET b.is_primary = FALSE WHERE zb.ziel_id = ?",
		zielID,
	)
	// Set new primary
	return database.DB.Model(&models.Bild{}).Where("id = ?", bildID).Update("is_primary", true).Error
}