package handlers

import (
	"time"

	"github.com/HAJO4KidsDE/hajo4kids-backend/internal/database"
	"github.com/HAJO4KidsDE/hajo4kids-backend/internal/middleware"
	"github.com/HAJO4KidsDE/hajo4kids-backend/internal/models"
	"github.com/HAJO4KidsDE/hajo4kids-backend/internal/utils"
	"github.com/HAJO4KidsDE/hajo4kids-backend/pkg/response"
	"github.com/gofiber/fiber/v2"
	"golang.org/x/crypto/bcrypt"
)

type RegisterRequest struct {
	Username string `json:"username" validate:"required,min=3,max=50,username"`
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=6"`
}

type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

type AuthResponse struct {
	Token string       `json:"token"`
	User  *models.User `json:"user"`
}

func Register(c *fiber.Ctx) error {
	var req RegisterRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Invalid request body")
	}

	// Validate input
	if err := utils.Validate(req); err != nil {
		return response.BadRequest(c, "Validation error: "+err.Error())
	}

	// Sanitize input
	req.Username = utils.SanitizeString(req.Username)
	req.Email = utils.SanitizeString(req.Email)

	// Check if user exists
	var existing models.User
	if database.DB.Where("email = ? OR username = ?", req.Email, req.Username).First(&existing).Error == nil {
		return response.Error(c, fiber.StatusConflict, "User already exists")
	}

	// Validate password strength
	if len(req.Password) < 8 {
		return response.BadRequest(c, "Password must be at least 8 characters")
	}

	// Hash password
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return response.InternalError(c, "Failed to hash password")
	}

	user := models.User{
		Username:     req.Username,
		Email:        req.Email,
		PasswordHash: string(hash),
		Role:         "user",
		Active:       true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if err := database.DB.Create(&user).Error; err != nil {
		return response.InternalError(c, "Failed to create user")
	}

	// Generate token
	token, err := middleware.GenerateToken(&user, middleware.GetConfig())
	if err != nil {
		return response.InternalError(c, "Failed to generate token")
	}

	return response.Created(c, AuthResponse{Token: token, User: &user})
}

func Login(c *fiber.Ctx) error {
	var req LoginRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Invalid request body")
	}

	// Validate input
	if err := utils.Validate(req); err != nil {
		return response.BadRequest(c, "Validation error: "+err.Error())
	}

	// Sanitize input
	req.Email = utils.SanitizeString(req.Email)

	var user models.User
	if err := database.DB.Where("email = ?", req.Email).First(&user).Error; err != nil {
		return response.Unauthorized(c, "Invalid credentials")
	}

	if !user.Active {
		return response.Unauthorized(c, "Account is disabled")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return response.Unauthorized(c, "Invalid credentials")
	}

	token, err := middleware.GenerateToken(&user, middleware.GetConfig())
	if err != nil {
		return response.InternalError(c, "Failed to generate token")
	}

	return response.Success(c, AuthResponse{Token: token, User: &user})
}

func GetMe(c *fiber.Ctx) error {
	user := middleware.GetUserFromContext(c)
	if user == nil {
		return response.Unauthorized(c, "Not authenticated")
	}

	// Fetch fresh user data from DB
	var freshUser models.User
	if err := database.DB.First(&freshUser, user.ID).Error; err != nil {
		return response.NotFound(c, "User not found")
	}

	return response.Success(c, freshUser)
}