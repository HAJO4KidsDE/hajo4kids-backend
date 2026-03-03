package handlers

import (
	"time"

	"github.com/HAJO4KidsDE/hajo4kids-backend/internal/database"
	"github.com/HAJO4KidsDE/hajo4kids-backend/internal/middleware"
	"github.com/HAJO4KidsDE/hajo4kids-backend/internal/models"
	"github.com/HAJO4KidsDE/hajo4kids-backend/pkg/response"
	"github.com/gofiber/fiber/v2"
)

// ==================== SHOP KATEGORIEN ====================

func GetShopKategorien(c *fiber.Ctx) error {
	var kategorien []models.ShopKategorie
	if err := database.DB.Preload("Items", "status = ?", "published").
		Order("sort_order, name").
		Find(&kategorien).Error; err != nil {
		return response.InternalError(c, "Failed to fetch shop kategorien")
	}
	return response.Success(c, kategorien)
}

func GetShopKategorie(c *fiber.Ctx) error {
	var kategorie models.ShopKategorie
	if err := database.DB.Preload("Items").First(&kategorie, c.Params("id")).Error; err != nil {
		return response.NotFound(c, "Shop kategorie not found")
	}
	return response.Success(c, kategorie)
}

func CreateShopKategorie(c *fiber.Ctx) error {
	var kategorie models.ShopKategorie
	if err := c.BodyParser(&kategorie); err != nil {
		return response.BadRequest(c, "Invalid request body")
	}
	kategorie.CreatedAt = time.Now()
	kategorie.UpdatedAt = time.Now()
	if err := database.DB.Create(&kategorie).Error; err != nil {
		return response.InternalError(c, "Failed to create shop kategorie")
	}
	return response.Created(c, kategorie)
}

func UpdateShopKategorie(c *fiber.Ctx) error {
	id := c.Params("id")
	var kategorie models.ShopKategorie
	if err := database.DB.First(&kategorie, id).Error; err != nil {
		return response.NotFound(c, "Shop kategorie not found")
	}
	var updates map[string]interface{}
	c.BodyParser(&updates)
	updates["updated_at"] = time.Now()
	database.DB.Model(&kategorie).Updates(updates)
	return response.Success(c, kategorie)
}

func DeleteShopKategorie(c *fiber.Ctx) error {
	database.DB.Delete(&models.ShopKategorie{}, c.Params("id"))
	return response.NoContent(c)
}

// ==================== SHOP ITEMS ====================

func GetShopItems(c *fiber.Ctx) error {
	query := database.DB.Model(&models.ShopItem{}).Preload("Kategorie")
	
	// Filter by status (admin sees all, public only published)
	user := middleware.GetUserFromContext(c)
	if user == nil || (user.Role != "admin" && user.Role != "reporter") {
		query = query.Where("status = ?", "published")
	} else if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	}
	
	// Filter by kategorie
	if kategorieID := c.Query("kategorie_id"); kategorieID != "" {
		query = query.Where("kategorie_id = ?", kategorieID)
	}
	
	// Filter by availability
	if c.Query("verfuegbar") == "true" {
		query = query.Where("verfuegbar = ?", true)
	}
	
	// Get total count
	var total int64
	query.Count(&total)
	
	// Pagination
	limit := c.QueryInt("limit", 20)
	offset := c.QueryInt("offset", 0)
	
	var items []models.ShopItem
	query.Order("sort_order, created_at desc").Offset(offset).Limit(limit).Find(&items)
	
	return response.Paginated(c, items, int(total), limit, offset)
}

func GetShopItem(c *fiber.Ctx) error {
	var item models.ShopItem
	if err := database.DB.Preload("Kategorie").First(&item, c.Params("id")).Error; err != nil {
		return response.NotFound(c, "Shop item not found")
	}
	
	// Check access for non-published items
	user := middleware.GetUserFromContext(c)
	if item.Status != "published" && (user == nil || (user.Role != "admin" && user.Role != "reporter")) {
		return response.NotFound(c, "Shop item not found")
	}
	
	return response.Success(c, item)
}

func GetShopItemBySlug(c *fiber.Ctx) error {
	var item models.ShopItem
	if err := database.DB.Preload("Kategorie").Where("slug = ?", c.Params("slug")).First(&item).Error; err != nil {
		return response.NotFound(c, "Shop item not found")
	}
	
	// Check access for non-published items
	user := middleware.GetUserFromContext(c)
	if item.Status != "published" && (user == nil || (user.Role != "admin" && user.Role != "reporter")) {
		return response.NotFound(c, "Shop item not found")
	}
	
	return response.Success(c, item)
}

func CreateShopItem(c *fiber.Ctx) error {
	var item models.ShopItem
	if err := c.BodyParser(&item); err != nil {
		return response.BadRequest(c, "Invalid request body")
	}
	
	user := middleware.GetUserFromContext(c)
	if user != nil {
		item.CreatedBy = &user.ID
	}
	
	item.CreatedAt = time.Now()
	item.UpdatedAt = time.Now()
	
	if err := database.DB.Create(&item).Error; err != nil {
		return response.InternalError(c, "Failed to create shop item")
	}
	return response.Created(c, item)
}

func UpdateShopItem(c *fiber.Ctx) error {
	id := c.Params("id")
	var item models.ShopItem
	if err := database.DB.First(&item, id).Error; err != nil {
		return response.NotFound(c, "Shop item not found")
	}
	var updates map[string]interface{}
	c.BodyParser(&updates)
	updates["updated_at"] = time.Now()
	database.DB.Model(&item).Updates(updates)
	return response.Success(c, item)
}

func DeleteShopItem(c *fiber.Ctx) error {
	database.DB.Delete(&models.ShopItem{}, c.Params("id"))
	return response.NoContent(c)
}