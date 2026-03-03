package middleware

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/HAJO4KidsDE/hajo4kids-backend/internal/config"
	"github.com/HAJO4KidsDE/hajo4kids-backend/internal/models"
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"
)

var (
	ErrMissingToken = errors.New("missing or invalid authorization token")
	ErrInvalidToken = errors.New("invalid token")
	ErrTokenExpired = errors.New("token expired")
)

// DefaultConfig for JWT (should be overridden by actual config in production)
var DefaultConfig = config.Config{
	JWT: config.JWTConfig{
		Secret: "change-me-in-production",
		Expiry: 24,
		Issuer: "hajo4kids.de",
	},
}

type Claims struct {
	UserID   uint   `json:"user_id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

// GenerateToken creates a new JWT token for a user
func GenerateToken(user *models.User, cfg *config.Config) (string, error) {
	expiry := time.Duration(cfg.JWT.Expiry) * time.Hour
	
	claims := &Claims{
		UserID:   user.ID,
		Username: user.Username,
		Email:    user.Email,
		Role:     user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(expiry)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    cfg.JWT.Issuer,
		},
	}
	
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(cfg.JWT.Secret))
}

// ValidateToken parses and validates a JWT token
func ValidateToken(tokenString string, cfg *config.Config) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(cfg.JWT.Secret), nil
	})
	
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrTokenExpired
		}
		return nil, ErrInvalidToken
	}
	
	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}
	
	return nil, ErrInvalidToken
}

// AuthMiddleware validates JWT and sets user info in context
func AuthMiddleware(db *gorm.DB, cfg *config.Config) fiber.Handler {
	return func(c *fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Missing authorization header",
			})
		}
		
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid authorization header format",
			})
		}
		
		claims, err := ValidateToken(parts[1], cfg)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		
		// Verify user still exists and is active
		var user models.User
		if err := db.First(&user, claims.UserID).Error; err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "User not found",
			})
		}
		
		if !user.Active {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "User account is disabled",
			})
		}
		
		// Store user info in context
		c.Locals("user", &user)
		c.Locals("claims", claims)
		
		return c.Next()
	}
}

// OptionalAuthMiddleware validates JWT if present but doesn't require it
func OptionalAuthMiddleware(db *gorm.DB, cfg *config.Config) fiber.Handler {
	return func(c *fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return c.Next()
		}
		
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			return c.Next()
		}
		
		claims, err := ValidateToken(parts[1], cfg)
		if err != nil {
			return c.Next()
		}
		
		var user models.User
		if err := db.First(&user, claims.UserID).Error; err != nil {
			return c.Next()
		}
		
		if user.Active {
			c.Locals("user", &user)
			c.Locals("claims", claims)
		}
		
		return c.Next()
	}
}

// RoleMiddleware checks if user has required role
func RoleMiddleware(allowedRoles ...string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		user, ok := c.Locals("user").(*models.User)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Unauthorized",
			})
		}
		
		for _, role := range allowedRoles {
			if user.Role == role {
				return c.Next()
			}
		}
		
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Insufficient permissions",
		})
	}
}

// GetUserFromContext extracts the current user from context
func GetUserFromContext(c *fiber.Ctx) *models.User {
	if user, ok := c.Locals("user").(*models.User); ok {
		return user
	}
	return nil
}

// GetUserIDFromContext extracts the current user ID from context
func GetUserIDFromContext(c *fiber.Ctx) uint {
	if user := GetUserFromContext(c); user != nil {
		return user.ID
	}
	return 0
}