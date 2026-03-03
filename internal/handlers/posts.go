package handlers

import (
	"strings"
	"time"

	"github.com/HAJO4KidsDE/hajo4kids-backend/internal/database"
	"github.com/HAJO4KidsDE/hajo4kids-backend/internal/middleware"
	"github.com/HAJO4KidsDE/hajo4kids-backend/internal/models"
	"github.com/HAJO4KidsDE/hajo4kids-backend/pkg/response"
	"github.com/gofiber/fiber/v2"
)

// ==================== POSTS ====================

func GetPosts(c *fiber.Ctx) error {
	query := database.DB.Model(&models.Post{})
	
	// Filter by status (admin sees all, public only published)
	user := middleware.GetUserFromContext(c)
	if user == nil || (user.Role != "admin" && user.Role != "reporter") {
		query = query.Where("status = ?", "published")
	} else if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	}
	
	// Filter by kategorie
	if kategorie := c.Query("kategorie"); kategorie != "" {
		query = query.Where("kategorie = ?", kategorie)
	}
	
	// Filter by tag
	if tag := c.Query("tag"); tag != "" {
		query = query.Where("tags LIKE ?", "%\""+tag+"\"%")
	}
	
	// Filter by author
	if autorID := c.Query("autor_id"); autorID != "" {
		query = query.Where("autor_id = ?", autorID)
	}
	
	// Search
	if search := c.Query("q"); search != "" {
		searchPattern := "%" + strings.ToLower(search) + "%"
		query = query.Where("LOWER(title) LIKE ? OR LOWER(excerpt) LIKE ? OR LOWER(inhalt) LIKE ?",
			searchPattern, searchPattern, searchPattern)
	}
	
	// Get total count
	var total int64
	query.Count(&total)
	
	// Pagination
	limit := c.QueryInt("limit", 10)
	offset := c.QueryInt("offset", 0)
	
	var posts []models.Post
	query.Preload("Autor").Order("published_at desc, created_at desc").Offset(offset).Limit(limit).Find(&posts)
	
	return response.Paginated(c, posts, int(total), limit, offset)
}

func GetPost(c *fiber.Ctx) error {
	var post models.Post
	if err := database.DB.Preload("Autor").
		Preload("Comments", "approved = ?", true).
		First(&post, c.Params("id")).Error; err != nil {
		return response.NotFound(c, "Post not found")
	}
	
	// Check access for non-published posts
	user := middleware.GetUserFromContext(c)
	if post.Status != "published" && (user == nil || (user.Role != "admin" && user.Role != "reporter")) {
		return response.NotFound(c, "Post not found")
	}
	
	// Increment view count
	database.DB.Model(&post).Update("view_count", post.ViewCount+1)
	
	return response.Success(c, post)
}

func GetPostBySlug(c *fiber.Ctx) error {
	var post models.Post
	if err := database.DB.Preload("Autor").
		Preload("Comments", "approved = ?", true).
		Where("slug = ?", c.Params("slug")).First(&post).Error; err != nil {
		return response.NotFound(c, "Post not found")
	}
	
	// Check access for non-published posts
	user := middleware.GetUserFromContext(c)
	if post.Status != "published" && (user == nil || (user.Role != "admin" && user.Role != "reporter")) {
		return response.NotFound(c, "Post not found")
	}
	
	// Increment view count
	database.DB.Model(&post).Update("view_count", post.ViewCount+1)
	
	return response.Success(c, post)
}

func CreatePost(c *fiber.Ctx) error {
	user := middleware.GetUserFromContext(c)
	
	var post models.Post
	if err := c.BodyParser(&post); err != nil {
		return response.BadRequest(c, "Invalid request body")
	}
	
	post.AutorID = &user.ID
	post.CreatedAt = time.Now()
	post.UpdatedAt = time.Now()
	
	// Set published_at if status is published
	if post.Status == "published" && post.PublishedAt == nil {
		now := time.Now()
		post.PublishedAt = &now
	}
	
	if err := database.DB.Create(&post).Error; err != nil {
		return response.InternalError(c, "Failed to create post")
	}
	return response.Created(c, post)
}

func UpdatePost(c *fiber.Ctx) error {
	user := middleware.GetUserFromContext(c)
	id := c.Params("id")
	
	var post models.Post
	if err := database.DB.First(&post, id).Error; err != nil {
		return response.NotFound(c, "Post not found")
	}
	
	// Only owner or admin can update
	if post.AutorID != nil && *post.AutorID != user.ID && user.Role != "admin" {
		return response.Forbidden(c, "Not authorized to update this post")
	}
	
	var updates map[string]interface{}
	c.BodyParser(&updates)
	updates["updated_at"] = time.Now()
	
	// Set published_at if status changes to published
	if status, ok := updates["status"].(string); ok && status == "published" && post.PublishedAt == nil {
		now := time.Now()
		updates["published_at"] = &now
	}
	
	database.DB.Model(&post).Updates(updates)
	return response.Success(c, post)
}

func DeletePost(c *fiber.Ctx) error {
	user := middleware.GetUserFromContext(c)
	id := c.Params("id")
	
	var post models.Post
	if err := database.DB.First(&post, id).Error; err != nil {
		return response.NotFound(c, "Post not found")
	}
	
	// Only owner or admin can delete
	if post.AutorID != nil && *post.AutorID != user.ID && user.Role != "admin" {
		return response.Forbidden(c, "Not authorized to delete this post")
	}
	
	database.DB.Delete(&post)
	return response.NoContent(c)
}

// ==================== POST COMMENTS ====================

func GetPostComments(c *fiber.Ctx) error {
	var comments []models.PostComment
	database.DB.Where("post_id = ? AND approved = ?", c.Params("id"), true).
		Order("created_at desc").
		Find(&comments)
	return response.Success(c, comments)
}

func CreatePostComment(c *fiber.Ctx) error {
	var comment models.PostComment
	if err := c.BodyParser(&comment); err != nil {
		return response.BadRequest(c, "Invalid request body")
	}
	
	// Check if post exists
	var post models.Post
	if err := database.DB.First(&post, comment.PostID).Error; err != nil {
		return response.NotFound(c, "Post not found")
	}
	
	// Set user ID if logged in
	user := middleware.GetUserFromContext(c)
	if user != nil {
		comment.UserID = &user.ID
		comment.Name = user.Username
		comment.Email = user.Email
	}
	
	// Auto-approve comments from logged-in users, require approval for guests
	comment.Approved = user != nil
	comment.CreatedAt = time.Now()
	
	// Validate required fields
	if comment.Name == "" || comment.Inhalt == "" {
		return response.BadRequest(c, "Name and content are required")
	}
	
	if err := database.DB.Create(&comment).Error; err != nil {
		return response.InternalError(c, "Failed to create comment")
	}
	
	return response.Created(c, fiber.Map{
		"comment": comment,
		"message": func() string {
			if comment.Approved {
				return "Comment posted successfully"
			}
			return "Comment submitted for review"
		}(),
	})
}

func GetAllComments(c *fiber.Ctx) error {
	// Admin only - get all comments including unapproved
	query := database.DB.Model(&models.PostComment{}).Preload("Post").Preload("User")
	
	// Filter by approval status
	if approved := c.Query("approved"); approved != "" {
		query = query.Where("approved = ?", approved == "true")
	}
	
	// Filter by post
	if postID := c.Query("post_id"); postID != "" {
		query = query.Where("post_id = ?", postID)
	}
	
	var comments []models.PostComment
	query.Order("created_at desc").Find(&comments)
	return response.Success(c, comments)
}

func ApproveComment(c *fiber.Ctx) error {
	database.DB.Model(&models.PostComment{}).Where("id = ?", c.Params("id")).Update("approved", true)
	return response.Success(c, fiber.Map{"approved": true})
}

func DeleteComment(c *fiber.Ctx) error {
	database.DB.Delete(&models.PostComment{}, c.Params("id"))
	return response.NoContent(c)
}