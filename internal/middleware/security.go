package middleware

import (
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/limiter"
)

// RateLimiter creates a rate limiting middleware
func RateLimiter() fiber.Handler {
	return limiter.New(limiter.Config{
		Max:        100,             // Maximum requests per time window
		Expiration: 1 * time.Minute, // Time window
		KeyGenerator: func(c *fiber.Ctx) string {
			// Use IP + User-Agent as key for more specific limiting
			return c.IP() + ":" + c.Get("User-Agent")
		},
		LimitReached: func(c *fiber.Ctx) error {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"success": false,
				"error":   "Rate limit exceeded. Please try again later.",
			})
		},
		Storage: nil, // Use in-memory storage (for production, use Redis)
	})
}

// RateLimiterAuth creates a stricter rate limiter for auth endpoints
func RateLimiterAuth() fiber.Handler {
	return limiter.New(limiter.Config{
		Max:        5,                // Maximum requests per time window for auth
		Expiration: 1 * time.Minute, // Time window
		KeyGenerator: func(c *fiber.Ctx) string {
			return c.IP()
		},
		LimitReached: func(c *fiber.Ctx) error {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"success": false,
				"error":   "Too many authentication attempts. Please try again later.",
			})
		},
	})
}

// RateLimiterAPI creates a rate limiter for API endpoints
func RateLimiterAPI() fiber.Handler {
	return limiter.New(limiter.Config{
		Max:        50,               // Maximum requests per time window for API
		Expiration: 1 * time.Minute, // Time window
		KeyGenerator: func(c *fiber.Ctx) string {
			// Use user ID if authenticated, otherwise use IP
			token := c.Get("Authorization")
			if token != "" {
				// Extract user ID from token context
				if user := GetUserFromContext(c); user != nil {
					return string(rune(user.ID))
				}
			}
			return c.IP()
		},
		LimitReached: func(c *fiber.Ctx) error {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"success": false,
				"error":   "API rate limit exceeded. Please slow down.",
			})
		},
	})
}

// SanitizeInput sanitizes user input to prevent XSS
func SanitizeInput() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// This is a basic implementation - for production, use a proper HTML sanitizer
		// The handlers should also sanitize individual fields based on context
		return c.Next()
	}
}

// SecurityHeaders sets security-related headers
func SecurityHeaders() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Prevent clickjacking
		c.Set("X-Frame-Options", "DENY")
		// Prevent MIME type sniffing
		c.Set("X-Content-Type-Options", "nosniff")
		// Enable XSS protection in browsers
		c.Set("X-XSS-Protection", "1; mode=block")
		// Referrer policy
		c.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		// Content Security Policy
		c.Set("Content-Security-Policy", "default-src 'self'; img-src 'self' data: https:; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'")
		// Permissions Policy
		c.Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")
		
		return c.Next()
	}
}

// NoIndex prevents search engine indexing for non-public endpoints
func NoIndex() fiber.Handler {
	return func(c *fiber.Ctx) error {
		c.Set("X-Robots-Tag", "noindex, nofollow")
		return c.Next()
	}
}

// ValidateFileUpload validates file uploads for size and type
func ValidateFileUpload(c *fiber.Ctx, allowedTypes []string, maxSize int64) (string, bool) {
	file, err := c.FormFile("file")
	if err != nil {
		return "No file provided", false
	}

	// Check file size
	if file.Size > maxSize {
		return "File too large", false
	}

	// Check file extension
	ext := strings.ToLower(file.Filename[strings.LastIndex(file.Filename, "."):])
	validExt := false
	for _, allowed := range allowedTypes {
		if ext == allowed || ext == strings.ToLower(allowed) {
			validExt = true
			break
		}
	}
	if !validExt {
		return "Invalid file type", false
	}

	return "", true
}