package handlers

import (
	"crypto/rand"
	"encoding/hex"
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

type UpdateProfileRequest struct {
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Username  string `json:"username" validate:"omitempty,min=3,max=50,username"`
}

type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" validate:"required"`
	NewPassword     string `json:"new_password" validate:"required,min=8"`
}

type ForgotPasswordRequest struct {
	Email string `json:"email" validate:"required,email"`
}

type ResetPasswordRequest struct {
	Token    string `json:"token" validate:"required"`
	Password string `json:"password" validate:"required,min=8"`
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

// UpdateProfile updates the current user's profile
func UpdateProfile(c *fiber.Ctx) error {
	user := middleware.GetUserFromContext(c)
	if user == nil {
		return response.Unauthorized(c, "Not authenticated")
	}

	var req UpdateProfileRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Invalid request body")
	}

	// Validate
	if err := utils.Validate(req); err != nil {
		return response.BadRequest(c, "Validation error: "+err.Error())
	}

	// Sanitize
	req.FirstName = utils.SanitizeString(req.FirstName)
	req.LastName = utils.SanitizeString(req.LastName)
	req.Username = utils.SanitizeString(req.Username)

	// Check username uniqueness if changing
	if req.Username != "" && req.Username != user.Username {
		var existing models.User
		if database.DB.Where("username = ? AND id != ?", req.Username, user.ID).First(&existing).Error == nil {
			return response.Error(c, fiber.StatusConflict, "Username already taken")
		}
		user.Username = req.Username
	}

	user.FirstName = req.FirstName
	user.LastName = req.LastName
	user.UpdatedAt = time.Now()

	if err := database.DB.Save(user).Error; err != nil {
		return response.InternalError(c, "Failed to update profile")
	}

	return response.Success(c, user)
}

// ChangePassword changes the current user's password
func ChangePassword(c *fiber.Ctx) error {
	user := middleware.GetUserFromContext(c)
	if user == nil {
		return response.Unauthorized(c, "Not authenticated")
	}

	var req ChangePasswordRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Invalid request body")
	}

	// Validate
	if err := utils.Validate(req); err != nil {
		return response.BadRequest(c, "Validation error: "+err.Error())
	}

	// Verify current password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.CurrentPassword)); err != nil {
		return response.Unauthorized(c, "Current password is incorrect")
	}

	// Hash new password
	hash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return response.InternalError(c, "Failed to hash password")
	}

	user.PasswordHash = string(hash)
	user.UpdatedAt = time.Now()

	if err := database.DB.Save(user).Error; err != nil {
		return response.InternalError(c, "Failed to update password")
	}

	return response.Success(c, fiber.Map{"message": "Password changed successfully"})
}

// ForgotPassword initiates password reset
func ForgotPassword(c *fiber.Ctx) error {
	var req ForgotPasswordRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Invalid request body")
	}

	// Validate
	if err := utils.Validate(req); err != nil {
		return response.BadRequest(c, "Validation error: "+err.Error())
	}

	// Find user
	var user models.User
	if err := database.DB.Where("email = ?", req.Email).First(&user).Error; err != nil {
		// Don't reveal if user exists or not
		return response.Success(c, fiber.Map{"message": "If the email exists, a reset link has been sent"})
	}

	// Generate reset token
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return response.InternalError(c, "Failed to generate token")
	}
	token := hex.EncodeToString(tokenBytes)

	// Create reset record
	reset := models.PasswordReset{
		UserID:    user.ID,
		Token:     token,
		ExpiresAt: time.Now().Add(1 * time.Hour),
		CreatedAt: time.Now(),
	}

	if err := database.DB.Create(&reset).Error; err != nil {
		return response.InternalError(c, "Failed to create reset token")
	}

	// TODO: Send email with reset link
	// In production, this would send an email via SMTP/service
	// resetLink := fmt.Sprintf("%s/reset-password?token=%s", config.FrontendURL, token)

	return response.Success(c, fiber.Map{
		"message": "If the email exists, a reset link has been sent",
		// Development only - remove in production:
		"dev_token": token,
	})
}

// ResetPassword completes password reset
func ResetPassword(c *fiber.Ctx) error {
	var req ResetPasswordRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Invalid request body")
	}

	// Validate
	if err := utils.Validate(req); err != nil {
		return response.BadRequest(c, "Validation error: "+err.Error())
	}

	// Find reset token
	var reset models.PasswordReset
	if err := database.DB.Where("token = ? AND used = ? AND expires_at > ?", req.Token, false, time.Now()).
		Preload("User").First(&reset).Error; err != nil {
		return response.BadRequest(c, "Invalid or expired reset token")
	}

	// Hash new password
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return response.InternalError(c, "Failed to hash password")
	}

	// Update user password
	reset.User.PasswordHash = string(hash)
	reset.User.UpdatedAt = time.Now()

	if err := database.DB.Save(&reset.User).Error; err != nil {
		return response.InternalError(c, "Failed to update password")
	}

	// Mark token as used
	reset.Used = true
	database.DB.Save(&reset)

	return response.Success(c, fiber.Map{"message": "Password reset successfully"})
}